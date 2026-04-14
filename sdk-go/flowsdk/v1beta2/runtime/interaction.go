package runtime

import (
	"context"
	"fmt"
	"sync"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// interactionHandler presents a prompt and waits for a response from an
// external source (CLI, gRPC bidi stream, etc.). The prompt is sent via the
// prompt channel; the response arrives via TryDeliver from the demux goroutine.
type interactionHandler struct {
	flowControlMixin
	id          string
	inputs      map[string]<-chan *pubsub.Message
	pubsub      executor.PubSub
	topic       string
	prompt      chan<- *flowv1beta2.InteractionRequestEvent
	deliver     chan *expr.Value // buffered(1), written by TryDeliver, read by promptAndWait
	whenProg    cel.Program
	transforms  *transformPipeline
	transformPS executor.PubSub
	adapter     types.Adapter

	tokenMu      sync.Mutex
	pendingToken string
}

// TryDeliver validates the token and delivers the value atomically.
// Returns false if the token doesn't match (stale/invalid response).
func (h *interactionHandler) TryDeliver(token string, val *expr.Value) bool {
	h.tokenMu.Lock()
	defer h.tokenMu.Unlock()
	if h.pendingToken == "" || token != h.pendingToken {
		return false
	}
	h.pendingToken = ""
	h.deliver <- val
	return true
}

// Close signals the handler that no more responses will arrive (EOF).
func (h *interactionHandler) Close() {
	close(h.deliver)
}

func (h *interactionHandler) Run(ctx context.Context) error {
	if h.transforms != nil {
		return h.runWithTransforms(ctx)
	}

	for {
		// Wait for upstream dependencies.
		act := newActivationFromChannels(ctx, h.inputs, h.adapter)
		vars, err := act.Resolve()
		if err != nil {
			return fmt.Errorf("interaction %s resolve: %w", h.id, err)
		}
		if act.AnyEOF() {
			break
		}

		if h.whenProg != nil {
			result, err := evalCEL(h.whenProg, vars)
			if err != nil {
				return fmt.Errorf("interaction %s when: %w", h.id, err)
			}
			if result.Value() != true {
				continue
			}
		}

		val, err := h.promptAndWait(ctx)
		if err != nil {
			return err
		}
		if isEOFValue(val) {
			break
		}

		node := flowv1beta2.RunSnapshot_InteractionNode_builder{
			Id:        h.id,
			Value:     val,
			Submitted: true,
			Phase:     flowv1beta2.RunSnapshot_PHASE_RUNNING,
		}.Build()
		if err := publishNode(h.pubsub, h.topic, node); err != nil {
			return err
		}
		h.checkFC(vars)
	}
	return publishNode(h.pubsub, h.topic, flowv1beta2.RunSnapshot_InteractionNode_builder{
		Id:    h.id,
		Value: newEOFValue(),
		Phase: flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
	}.Build())
}

func (h *interactionHandler) runWithTransforms(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	var sinkCount uint64
	sink := func(_ context.Context, val *expr.Value, eof bool) error {
		sinkCount++
		phase := flowv1beta2.RunSnapshot_PHASE_RUNNING
		if eof {
			phase = flowv1beta2.RunSnapshot_PHASE_SUCCEEDED
		}
		return publishNode(h.pubsub, h.topic, flowv1beta2.RunSnapshot_InteractionNode_builder{
			Id:        h.id,
			Value:     val,
			Submitted: true,
			Phase:     phase,
		}.Build())
	}
	onState := newStateCallback(h.pubsub, h.topic, len(h.transforms.steps),
		func(t []*flowv1beta2.RunSnapshot_Transform) executor.StateNode {
			return flowv1beta2.RunSnapshot_InteractionNode_builder{
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
			act := newActivationFromChannels(ctx, h.inputs, h.adapter)
			vars, err := act.Resolve()
			if err != nil {
				return fmt.Errorf("interaction %s resolve: %w", h.id, err)
			}
			if act.AnyEOF() {
				break
			}

			if h.whenProg != nil {
				result, err := evalCEL(h.whenProg, vars)
				if err != nil {
					return fmt.Errorf("interaction %s when: %w", h.id, err)
				}
				if result.Value() != true {
					continue
				}
			}

			val, err := h.promptAndWait(ctx)
			if err != nil {
				return err
			}
			if isEOFValue(val) {
				break
			}
			if err := h.transformPS.Publish(inputTopic, pubsub.NewMessage(val)); err != nil {
				return err
			}
		}
		return h.transformPS.Publish(inputTopic, pubsub.NewMessage(newEOFValue()))
	})

	return g.Wait()
}

// promptAndWait sends a prompt with a UUIDv7 token to the external source and
// blocks until a matching response arrives or the context is cancelled.
func (h *interactionHandler) promptAndWait(ctx context.Context) (*expr.Value, error) {
	token, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("interaction %s: generating token: %w", h.id, err)
	}
	tokenStr := token.String()

	// Store the pending token so the demux goroutine can validate responses.
	h.tokenMu.Lock()
	h.pendingToken = tokenStr
	h.tokenMu.Unlock()

	// Publish pending state with token on the InteractionNode snapshot.
	// Uses a state event (not output) so downstream nodes don't see it as a value.
	pending := flowv1beta2.RunSnapshot_InteractionNode_builder{
		Id:    h.id,
		Token: tokenStr,
		Phase: flowv1beta2.RunSnapshot_PHASE_RUNNING,
	}.Build()
	if err := publishStateEvent(h.pubsub, h.topic, pending); err != nil {
		return nil, err
	}

	// Send prompt with node ID and token.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case h.prompt <- func() *flowv1beta2.InteractionRequestEvent {
		return flowv1beta2.InteractionRequestEvent_builder{
			Id:    h.id,
			Token: tokenStr,
		}.Build()
	}():
	}

	// Wait for response.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case val, ok := <-h.deliver:
		if !ok {
			return newEOFValue(), nil
		}
		return val, nil
	}
}
