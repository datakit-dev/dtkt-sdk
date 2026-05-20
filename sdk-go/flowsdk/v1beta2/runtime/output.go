package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"golang.org/x/sync/errgroup"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// outputHandler receives messages, evaluates a CEL expression, and publishes the result to the output PubSub topic.
type outputHandler struct {
	lifecycleMixin
	suspendableMixin
	stoppableMixin
	id          string
	inputs      map[string]<-chan *pubsub.Message
	program     cel.Program
	transforms  *transformPipeline
	transformPS executor.PubSub
	pubsub      executor.PubSub
	outputTopic string
	throttle    time.Duration
	env         shared.Env
	cache       *cacheBackend
}

func (h *outputHandler) Run(ctx context.Context) error {
	if h.transforms != nil {
		return h.runWithTransforms(ctx)
	}

	var (
		iterCount int
		eofSeen   bool
	)
loop:
	for {
		if h.throttle > 0 && iterCount > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-h.StopChan():
				break loop // graceful stop; post-loop EOF publish fires
			case <-time.After(h.throttle):
			}
		}

		act := h.cache.newActivation(ctx, h.inputs, h.env, h.SuspendChan(), h.StopChan())
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
			if ctx.Err() != nil {
				break
			}
			return fmt.Errorf("output %s resolve: %w", h.id, err)
		}
		if act.AnyEOF() {
			eofSeen = true
			break
		}
		// Once Resolve has consumed a message, we always finish the
		// iteration (eval + publish) before the next loop pass. The
		// suspend signal is only honored at the TOP of the loop, never
		// mid-iteration. This guarantees no consumed-but-unpublished
		// message can be dropped on suspend.
		result, err := evalCEL(h.program, vars)
		if err != nil {
			return fmt.Errorf("output %s eval: %w", h.id, err)
		}

		val, err := refValToExpr(result)
		if err != nil {
			return fmt.Errorf("output %s: converting result to cel value: %w", h.id, err)
		}

		iterCount++
		node := flowv1beta2.RunSnapshot_OutputNode_builder{
			Id:    h.id,
			Value: val,
			Phase: flowv1beta2.RunSnapshot_PHASE_RUNNING,
		}.Build()
		if err := publishNode(h.pubsub, h.outputTopic, node); err != nil {
			return fmt.Errorf("output %s publish: %w", h.id, err)
		}
		h.checkLifecycle(vars)
	}

	// Publish a Closed:true EOF marker on exit so subscribers know the
	// stream has ended. Under the unified suspend design this code path
	// is reached ONLY on real termination - input drained (AnyEOF), or
	// ctx cancelled by Stop/Terminate. Suspend never reaches here; it's
	// handled at the top of the loop via waitForResume.
	phase := flowv1beta2.RunSnapshot_PHASE_SUCCEEDED
	if !eofSeen && ctx.Err() != nil {
		phase = flowv1beta2.RunSnapshot_PHASE_CANCELLED
	}
	if err := publishNode(h.pubsub, h.outputTopic, flowv1beta2.RunSnapshot_OutputNode_builder{
		Id:     h.id,
		Closed: true,
		Phase:  phase,
	}.Build()); err != nil {
		return err
	}
	if !eofSeen {
		return ctx.Err()
	}
	return nil
}

func (h *outputHandler) runWithTransforms(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	var lastErr error
	sink := func(ctx context.Context, val *expr.Value, eof bool) error {
		if eof {
			return nil
		}
		node := flowv1beta2.RunSnapshot_OutputNode_builder{
			Id:    h.id,
			Value: val,
			Phase: flowv1beta2.RunSnapshot_PHASE_RUNNING,
		}.Build()
		if err := publishNode(h.pubsub, h.outputTopic, node); err != nil {
			lastErr = err
			return err
		}
		return nil
	}

	inputTopic, err := h.transforms.Start(ctx, g, h.transformPS, h.outputTopic, sink, nil)
	if err != nil {
		return err
	}

	var eofSeen bool
	g.Go(func() error {
		var iterCount int
	loop:
		for {
			if h.throttle > 0 && iterCount > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-h.StopChan():
					break loop // graceful stop; post-loop EOF publish fires
				case <-time.After(h.throttle):
				}
			}

			act := h.cache.newActivation(ctx, h.inputs, h.env, h.SuspendChan(), h.StopChan())
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
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return fmt.Errorf("output %s resolve: %w", h.id, err)
			}
			if act.AnyEOF() {
				eofSeen = true
				break
			}
			// See comment in Run(): consumed messages must always be
			// processed; suspend is honored only at the top of the loop.
			result, err := evalCEL(h.program, vars)
			if err != nil {
				return fmt.Errorf("output %s eval: %w", h.id, err)
			}
			val, err := refValToExpr(result)
			if err != nil {
				return fmt.Errorf("output %s: converting result to cel value: %w", h.id, err)
			}
			iterCount++
			if err := h.transformPS.Publish(inputTopic, pubsub.NewMessage(val)); err != nil {
				return err
			}
		}
		return h.transformPS.Publish(inputTopic, pubsub.NewMessage(newEOFValue()))
	})

	if err := g.Wait(); err != nil {
		return err
	}
	if lastErr != nil {
		return lastErr
	}
	// See Run() above. Under the unified suspend design, ctx cancellation
	// only happens on Stop/Terminate (not suspend), so always publish a
	// Closed:true marker so subscribers know the stream ended.
	phase := flowv1beta2.RunSnapshot_PHASE_SUCCEEDED
	if !eofSeen && ctx.Err() != nil {
		phase = flowv1beta2.RunSnapshot_PHASE_CANCELLED
	}
	if err := publishNode(h.pubsub, h.outputTopic, flowv1beta2.RunSnapshot_OutputNode_builder{
		Id:     h.id,
		Closed: true,
		Phase:  phase,
	}.Build()); err != nil {
		return err
	}
	if !eofSeen {
		return ctx.Err()
	}
	return nil
}
