package runtime

import (
	"context"
	"time"

	expr "cel.dev/expr"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

type rangeHandler struct {
	stoppableMixin
	id        string
	pubsub    executor.PubSub
	topic     string
	start     int64
	end       int64
	step      int64
	rate      *flowv1beta2.Rate
	evalCount uint64 // persists across suspend/resume
	suspendCh chan struct{}
	resumeCh  chan struct{}
}

func (h *rangeHandler) suspend() {
	select {
	case h.suspendCh <- struct{}{}:
	default:
	}
}

func (h *rangeHandler) resume() {
	select {
	case h.resumeCh <- struct{}{}:
	default:
	}
}

func (h *rangeHandler) Run(ctx context.Context) error {
	step := h.step
	if step == 0 {
		step = 1
	}

	// Rate-limited emission: wait between values when rate is configured.
	var ticker *time.Ticker
	if h.rate != nil && h.rate.GetInterval().IsValid() && h.rate.GetCount() > 0 {
		ticker = time.NewTicker(h.rate.GetInterval().AsDuration() / time.Duration(h.rate.GetCount()))
		defer ticker.Stop()
	}

	// publishSucceeded emits the terminal SUCCEEDED state with EOF.
	// Used both for natural completion (end of range) and operator stop.
	publishSucceeded := func() error {
		return publishNode(h.pubsub, h.topic, flowv1beta2.RunSnapshot_GeneratorNode_builder{
			Id:        h.id,
			Value:     newEOFValue(),
			Done:      true,
			EvalCount: h.evalCount,
			Phase:     flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
		}.Build())
	}

	// parkSuspend returns suspendStopped/suspendCancelled/suspendResumed
	// depending on which signal wakes the handler.
	parkSuspend := func() suspendResult {
		if ticker != nil {
			ticker.Stop()
		}
		_ = publishStateEvent(h.pubsub, h.topic, flowv1beta2.RunSnapshot_GeneratorNode_builder{
			Id:    h.id,
			Phase: flowv1beta2.RunSnapshot_PHASE_SUSPENDED,
		}.Build())
		select {
		case <-ctx.Done():
			return suspendCancelled
		case <-h.StopChan():
			return suspendStopped
		case <-h.resumeCh:
			_ = publishStateEvent(h.pubsub, h.topic, flowv1beta2.RunSnapshot_GeneratorNode_builder{
				Id:    h.id,
				Phase: flowv1beta2.RunSnapshot_PHASE_PENDING,
			}.Build())
			if ticker != nil {
				ticker.Reset(h.rate.GetInterval().AsDuration() / time.Duration(h.rate.GetCount()))
			}
			return suspendResumed
		}
	}

	for i := h.start; i <= h.end; i += step {
		if ticker != nil && h.evalCount > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-h.StopChan():
				return publishSucceeded()
			case <-h.suspendCh:
				switch parkSuspend() {
				case suspendCancelled:
					return ctx.Err()
				case suspendStopped:
					return publishSucceeded()
				}
				i-- // re-emit current value after resume
				continue
			case <-ticker.C:
			}
		} else {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-h.StopChan():
				return publishSucceeded()
			case <-h.suspendCh:
				switch parkSuspend() {
				case suspendCancelled:
					return ctx.Err()
				case suspendStopped:
					return publishSucceeded()
				}
				i-- // re-emit current value after resume
				continue
			default:
			}
		}

		val := &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: i}}
		h.evalCount++
		if err := publishNode(h.pubsub, h.topic, flowv1beta2.RunSnapshot_GeneratorNode_builder{
			Id:        h.id,
			Value:     val,
			EvalCount: h.evalCount,
			Phase:     flowv1beta2.RunSnapshot_PHASE_RUNNING,
		}.Build()); err != nil {
			return err
		}
	}

	return publishSucceeded()
}
