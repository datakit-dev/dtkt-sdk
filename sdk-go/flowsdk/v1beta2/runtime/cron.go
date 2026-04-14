package runtime

import (
	"context"
	"fmt"
	"time"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/robfig/cron/v3"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

type cronHandler struct {
	id           string
	pubsub       executor.PubSub
	topic        string
	schedule     cron.Schedule
	valueProgram cel.Program // nil = emit tick count
	count        int64       // persists across suspend/resume
	suspendCh    chan struct{}
	resumeCh     chan struct{}
}

func (h *cronHandler) suspend() {
	select {
	case h.suspendCh <- struct{}{}:
	default:
	}
}

func (h *cronHandler) resume() {
	select {
	case h.resumeCh <- struct{}{}:
	default:
	}
}

func (h *cronHandler) Run(ctx context.Context) error {
	for {
		next := h.schedule.Next(time.Now())
		wait := time.Until(next)

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
			}
		case <-time.After(wait):
			h.count++
			val, err := h.evalValue(h.count)
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

func (h *cronHandler) evalValue(count int64) (*expr.Value, error) {
	if h.valueProgram != nil {
		result, err := evalCEL(h.valueProgram, map[string]any{"this": map[string]any{"count": count, "time": time.Now()}})
		if err != nil {
			return nil, fmt.Errorf("cron %s eval: %w", h.id, err)
		}
		val, err := refValToExpr(result)
		if err != nil {
			return nil, fmt.Errorf("cron %s convert: %w", h.id, err)
		}
		return val, nil
	}
	return &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: count}}, nil
}
