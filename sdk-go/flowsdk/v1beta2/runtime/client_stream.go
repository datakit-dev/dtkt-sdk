package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// clientStreamHandler streams all incoming messages as requests and forwards the single response.
type clientStreamHandler struct {
	lifecycleMixin
	suspendableMixin
	stoppableMixin
	id                   string
	method               protoreflect.FullName
	inputs               map[string]<-chan *pubsub.Message
	pubsub               executor.PubSub
	topic                string
	client               rpc.Client
	env                  shared.Env
	whenProg             cel.Program
	throttle             time.Duration
	request              *compiledRequest // nil = use FirstInputValue
	responseProg         cel.Program      // nil = use raw response
	retry                *compiledRetryStrategy
}

func (h *clientStreamHandler) Run(ctx context.Context) error {
	var stream rpc.ClientStream
	for {
		err := executeWithRetry(ctx, h.retry, nil, func(ctx context.Context) error {
			var openErr error
			stream, openErr = h.client.CallClientStream(ctx, h.method)
			return openErr
		})
		var contErr *ContinueError
		if errors.As(err, &contErr) {
			streamState := &flowv1beta2.RunSnapshot_StreamNode{}
			streamState.SetId(h.id)
			streamState.SetPhase(flowv1beta2.RunSnapshot_PHASE_RUNNING)
			streamState.SetValue(contErr.Value)
			publish := func(node *flowv1beta2.RunSnapshot_StreamNode) error {
				return publishNode(h.pubsub, h.topic, node)
			}
			if pubErr := publish(cloneStreamNode(streamState)); pubErr != nil {
				return pubErr
			}
			streamState.SetValue(newEOFValue())
			streamState.SetRequestClosed(true)
			streamState.SetPhase(flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
			return publish(cloneStreamNode(streamState))
		}
		var suspendErr *SuspendError
		if errors.As(err, &suspendErr) {
			h.selfSuspend(err)
			res := h.waitForResume(ctx, h.StopChan())
			if res == suspendCancelled {
				return ctx.Err()
			}
			if res == suspendStopped {
				// Stopped while suspended waiting to retry the open; publish a
				// minimal terminal SUCCEEDED state and exit cleanly. The
				// stream was never opened, so there's nothing to drain.
				streamState := &flowv1beta2.RunSnapshot_StreamNode{}
				streamState.SetId(h.id)
				streamState.SetRequestClosed(true)
				streamState.SetResponseClosed(true)
				streamState.SetValue(newEOFValue())
				streamState.SetPhase(flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
				return publishNode(h.pubsub, h.topic, cloneStreamNode(streamState))
			}
			continue // retry the open after resume
		}
		if err != nil {
			return fmt.Errorf("client-stream %s open %q: %w", h.id, h.method, err)
		}
		break // stream opened successfully
	}

	streamState := &flowv1beta2.RunSnapshot_StreamNode{}
	streamState.SetId(h.id)
	streamState.SetPhase(flowv1beta2.RunSnapshot_PHASE_RUNNING)

	// Feed input messages into the stream, applying the `when` guard.
	sendDone := make(chan error, 1)
	go func() {
		var iterCount int
		for {
			// Pause point: between sends. In-flight stream stays open;
			// only the publish side pauses on suspend.
			if h.throttle > 0 && iterCount > 0 {
				select {
				case <-ctx.Done():
					sendDone <- ctx.Err()
					return
				case <-h.StopChan():
					sendDone <- nil // graceful stop; defer CloseSend, recv drains
					return
				case <-h.SuspendChan():
					res := h.waitForResume(ctx, h.StopChan())
					if res == suspendCancelled {
						sendDone <- ctx.Err()
						return
					}
					if res == suspendStopped {
						sendDone <- nil // graceful stop while suspended
						return
					}
					continue
				case <-time.After(h.throttle):
				}
			}

			act := newActivationFromChannelsInterruptible(ctx, h.inputs, h.env.TypeAdapter(), h.SuspendChan(), h.StopChan())
			vars, err := act.Resolve()
			if errors.Is(err, errOperatorStopped) {
				sendDone <- nil // graceful: close-send via defer, recv drains
				return
			}
			if errors.Is(err, errOperatorSuspended) {
				res := h.waitForResume(ctx, h.StopChan())
				if res == suspendCancelled {
					sendDone <- ctx.Err()
					return
				}
				if res == suspendStopped {
					sendDone <- nil // graceful stop while suspended
					return
				}
				continue
			}
			if err != nil || act.AnyEOF() {
				sendDone <- nil
				return
			}

			if h.whenProg != nil {
				result, err := evalCEL(h.whenProg, vars)
				if err != nil || result.Value() != true {
					continue
				}
			}

			iterCount++
			streamState.SetRequestCount(streamState.GetRequestCount() + 1)
			inputValue, err := resolveRequestValue(h.request, act, vars)
			if err != nil {
				sendDone <- err
				return
			}
			req, err := exprToMessage(h.env, h.method, inputValue)
			if err != nil {
				sendDone <- err
				return
			}
			if err := stream.SendMsg(req); err != nil {
				sendDone <- err
				return
			}
		}
	}()

	if err := <-sendDone; err != nil {
		return fmt.Errorf("client-stream %s send: %w", h.id, err)
	}

	resp, err := stream.CloseAndReceive()
	if err != nil {
		return fmt.Errorf("client-stream %s call %q: %w", h.id, h.method, err)
	}

	streamState.SetRequestClosed(true)

	respExpr, err := transformResponse(h.responseProg, resp, h.env.TypeAdapter())
	if err != nil {
		return fmt.Errorf("client-stream %s response: %w", h.id, err)
	}
	streamState.SetResponseCount(1)
	streamState.SetValue(respExpr)
	if err := publishNode(h.pubsub, h.topic, cloneStreamNode(streamState)); err != nil {
		return err
	}

	streamState.SetResponseClosed(true)
	streamState.SetValue(newEOFValue())
	streamState.SetPhase(flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
	return publishNode(h.pubsub, h.topic, cloneStreamNode(streamState))
}
