package runtime

import (
	"context"
	"errors"
	"fmt"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"golang.org/x/sync/errgroup"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

type varHandler struct {
	lifecycleMixin
	suspendableMixin
	stoppableMixin
	id          string
	inputs      map[string]<-chan *pubsub.Message
	pubsub      executor.PubSub
	topic       string
	program     cel.Program
	transforms  *transformPipeline
	transformPS executor.PubSub
	adapter     types.Adapter
	cache       *cacheBackend
}

func (h *varHandler) Run(ctx context.Context) error {
	if h.transforms != nil {
		return h.runWithTransforms(ctx)
	}

	var evalCount uint64
	for {
		act := h.cache.newActivation(ctx, h.inputs, h.adapter, h.SuspendChan(), h.StopChan())
		vars, err := act.Resolve()
		if errors.Is(err, errOperatorStopped) {
			break
		}
		if errors.Is(err, errOperatorSuspended) {
			res := h.waitForResume(ctx, h.StopChan())
			if res == suspendCancelled {
				return ctx.Err()
			}
			if res == suspendStopped {
				break
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("var %s resolve: %w", h.id, err)
		}
		if act.AnyEOF() {
			break
		}

		// cache:true producer post-capture: drain upstream events but
		// skip eval/publish. ClearCache resets the flag.
		if h.cache.isCaptured() {
			h.checkLifecycle(vars)
			continue
		}

		result, err := evalCEL(h.program, vars)
		if err != nil {
			return fmt.Errorf("var %s eval: %w", h.id, err)
		}
		val, err := refValToExpr(result)
		if err != nil {
			return fmt.Errorf("var %s convert: %w", h.id, err)
		}

		evalCount++
		node := flowv1beta2.RunSnapshot_VarNode_builder{
			Id:        h.id,
			Value:     val,
			EvalCount: evalCount,
			Phase:     flowv1beta2.RunSnapshot_PHASE_RUNNING,
		}.Build()
		if err := publishNode(h.pubsub, h.topic, node); err != nil {
			return err
		}
		h.cache.markCaptured()
		h.checkLifecycle(vars)
	}
	return publishNode(h.pubsub, h.topic, flowv1beta2.RunSnapshot_VarNode_builder{
		Id:        h.id,
		Value:     newEOFValue(),
		EvalCount: evalCount,
		Phase:     flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
	}.Build())
}

func (h *varHandler) runWithTransforms(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	var sinkCount uint64
	sink := func(_ context.Context, val *expr.Value, eof bool) error {
		// cache:true: only the FIRST post-transform value reaches
		// consumers. The main loop may feed multiple values into the
		// pipeline before captured flips (filter could drop several
		// inputs before one passes; map could process several before
		// the first sink callback fires). Drop subsequent sink
		// emissions silently so consumers see exactly one value.
		if !eof && h.cache.isCaptured() {
			return nil
		}
		sinkCount++
		phase := flowv1beta2.RunSnapshot_PHASE_RUNNING
		if eof {
			phase = flowv1beta2.RunSnapshot_PHASE_SUCCEEDED
		}
		if err := publishNode(h.pubsub, h.topic, flowv1beta2.RunSnapshot_VarNode_builder{
			Id:        h.id,
			Value:     val,
			EvalCount: sinkCount,
			Phase:     phase,
		}.Build()); err != nil {
			return err
		}
		if !eof {
			h.cache.markCaptured()
		}
		return nil
	}
	onState := newStateCallback(h.pubsub, h.topic, len(h.transforms.steps),
		func(t []*flowv1beta2.RunSnapshot_Transform) executor.StateNode {
			return flowv1beta2.RunSnapshot_VarNode_builder{
				Id:         h.id,
				Transforms: t,
				Phase:      flowv1beta2.RunSnapshot_PHASE_RUNNING,
			}.Build()
		})

	inputTopic, err := h.transforms.Start(ctx, g, h.transformPS, h.topic, sink, onState)
	if err != nil {
		return err
	}

	g.Go(func() error {
		for {
			act := h.cache.newActivation(ctx, h.inputs, h.adapter, h.SuspendChan(), h.StopChan())
			vars, err := act.Resolve()
			if errors.Is(err, errOperatorStopped) {
				break
			}
			if errors.Is(err, errOperatorSuspended) {
				res := h.waitForResume(ctx, h.StopChan())
				if res == suspendCancelled {
					return ctx.Err()
				}
				if res == suspendStopped {
					break
				}
				continue
			}
			if err != nil {
				return fmt.Errorf("var %s resolve: %w", h.id, err)
			}
			if act.AnyEOF() {
				break
			}
			// cache:true producer post-capture: drain upstream events
			// and skip eval + transform pipeline publish.
			if h.cache.isCaptured() {
				continue
			}
			result, err := evalCEL(h.program, vars)
			if err != nil {
				return fmt.Errorf("var %s eval: %w", h.id, err)
			}
			val, err := refValToExpr(result)
			if err != nil {
				return fmt.Errorf("var %s convert: %w", h.id, err)
			}
			if err := h.transformPS.Publish(inputTopic, pubsub.NewMessage(val)); err != nil {
				return err
			}
			// markCaptured fires inside the sink (post-transforms) so the
			// cached value matches what consumers actually see.
		}
		return h.transformPS.Publish(inputTopic, pubsub.NewMessage(newEOFValue()))
	})

	return g.Wait()
}
