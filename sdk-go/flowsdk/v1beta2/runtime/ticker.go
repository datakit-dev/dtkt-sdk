package runtime

import (
	"context"
	"fmt"
	"time"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// tickerHandler generates messages on a time interval.
type tickerHandler struct {
	id           string
	pubsub       executor.PubSub
	topic        string
	interval     time.Duration
	delay        time.Duration
	valueProgram cel.Program // nil = emit tick count
	count        int64       // persists across suspend/resume
	suspendCh    chan struct{}
	resumeCh     chan struct{}
}

func (h *tickerHandler) suspend() {
	select {
	case h.suspendCh <- struct{}{}:
	default:
	}
}

func (h *tickerHandler) resume() {
	select {
	case h.resumeCh <- struct{}{}:
	default:
	}
}

func (h *tickerHandler) Run(ctx context.Context) error {
	if h.delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(h.delay):
		}
	}

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = publishNode(h.pubsub, h.topic, flowv1beta2.RunSnapshot_GeneratorNode_builder{
				Id:        h.id,
				Value:     newEOFValue(),
				Done:      true,
				EvalCount: uint64(h.count),
				Phase:     flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
			}.Build())
			return nil
		case <-h.suspendCh:
			ticker.Stop()
			_ = publishStateEvent(h.pubsub, h.topic, flowv1beta2.RunSnapshot_GeneratorNode_builder{
				Id:    h.id,
				Phase: flowv1beta2.RunSnapshot_PHASE_SUSPENDED,
			}.Build())
			select {
			case <-ctx.Done():
				return nil
			case <-h.resumeCh:
				_ = publishStateEvent(h.pubsub, h.topic, flowv1beta2.RunSnapshot_GeneratorNode_builder{
					Id:    h.id,
					Phase: flowv1beta2.RunSnapshot_PHASE_PENDING,
				}.Build())
				ticker.Reset(h.interval)
			}
		case <-ticker.C:
			h.count++
			val, err := h.evalTickValue(h.count)
			if err != nil {
				return err
			}
			if err := publishNode(h.pubsub, h.topic, flowv1beta2.RunSnapshot_GeneratorNode_builder{
				Id:        h.id,
				Value:     val,
				EvalCount: uint64(h.count),
				Phase:     flowv1beta2.RunSnapshot_PHASE_RUNNING,
			}.Build()); err != nil {
				return err
			}
		}
	}
}

func (h *tickerHandler) evalTickValue(count int64) (*expr.Value, error) {
	if h.valueProgram != nil {
		result, err := evalCEL(h.valueProgram, map[string]any{"this": map[string]any{"count": count, "time": time.Now()}})
		if err != nil {
			return nil, fmt.Errorf("ticker %s eval: %w", h.id, err)
		}
		val, err := refValToExpr(result)
		if err != nil {
			return nil, fmt.Errorf("ticker %s convert: %w", h.id, err)
		}
		return val, nil
	}
	return &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: count}}, nil
}
