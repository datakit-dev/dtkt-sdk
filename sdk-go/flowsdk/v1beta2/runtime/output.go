package runtime

import (
	"context"
	"fmt"
	"time"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"golang.org/x/sync/errgroup"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// outputHandler receives messages, evaluates a CEL expression, and publishes the result to the output PubSub topic.
type outputHandler struct {
	flowControlMixin
	id          string
	inputs      map[string]<-chan *pubsub.Message
	program     cel.Program
	transforms  *transformPipeline
	transformPS executor.PubSub
	pubsub      executor.PubSub
	outputTopic string
	throttle    time.Duration
	adapter     types.Adapter
}

func (h *outputHandler) Run(ctx context.Context) error {
	if h.transforms != nil {
		return h.runWithTransforms(ctx)
	}

	var iterCount int
	for {
		if h.throttle > 0 && iterCount > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(h.throttle):
			}
		}

		act := newActivationFromChannels(ctx, h.inputs, h.adapter)
		vars, err := act.Resolve()
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			return fmt.Errorf("output %s resolve: %w", h.id, err)
		}
		if act.AnyEOF() || ctx.Err() != nil {
			break
		}
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
		h.checkFC(vars)
	}

	// Publish EOF marker so consumers know the stream ended.
	return publishNode(h.pubsub, h.outputTopic, flowv1beta2.RunSnapshot_OutputNode_builder{
		Id:     h.id,
		Closed: true,
		Phase:  flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
	}.Build())
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

	g.Go(func() error {
		var iterCount int
		for {
			if h.throttle > 0 && iterCount > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(h.throttle):
				}
			}

			act := newActivationFromChannels(ctx, h.inputs, h.adapter)
			vars, err := act.Resolve()
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return fmt.Errorf("output %s resolve: %w", h.id, err)
			}
			if act.AnyEOF() || ctx.Err() != nil {
				break
			}
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
	// Publish EOF marker so consumers know the stream ended.
	return publishNode(h.pubsub, h.outputTopic, flowv1beta2.RunSnapshot_OutputNode_builder{
		Id:     h.id,
		Closed: true,
		Phase:  flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
	}.Build())
}
