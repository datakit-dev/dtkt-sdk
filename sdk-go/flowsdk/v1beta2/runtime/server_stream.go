package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// serverStreamHandler sends each incoming message as a request and forwards the stream of responses.
type serverStreamHandler struct {
	lifecycleMixin
	suspendableMixin
	stoppableMixin
	id           string
	method       protoreflect.FullName
	inputs       map[string]<-chan *pubsub.Message
	pubsub       executor.PubSub
	topic        string
	client       rpc.Client
	env          shared.Env
	whenProg     cel.Program
	throttle     time.Duration
	request      *compiledRequest // nil = use FirstInputValue
	responseProg cel.Program      // nil = use raw response
	retry        *compiledRetryStrategy
	cache        *cacheBackend
}

func (h *serverStreamHandler) Run(ctx context.Context) error {
	streamState := &flowv1beta2.RunSnapshot_StreamNode{}
	streamState.SetId(h.id)
	streamState.SetPhase(flowv1beta2.RunSnapshot_PHASE_RUNNING)
	publish := func(node executor.StateNode) error {
		return publishNode(h.pubsub, h.topic, node)
	}

	var iterCount int
loop:
	for {
		// Pause point: between iterations only. An in-flight stream call
		// completes naturally; suspend never aborts the connection.
		if h.throttle > 0 && iterCount > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-h.StopChan():
				break loop // exit outer loop; post-loop SUCCEEDED publish fires
			case <-h.SuspendChan():
				res := h.waitForResume(ctx, h.StopChan())
				if res == suspendCancelled {
					return ctx.Err()
				}
				if res == suspendStopped {
					break loop
				}
				continue
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
				break loop
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("server-stream %s resolve: %w", h.id, err)
		}
		if act.AnyEOF() {
			break
		}

		if h.whenProg != nil {
			result, err := evalCEL(h.whenProg, vars)
			if err != nil {
				return fmt.Errorf("server-stream %s when: %w", h.id, err)
			}
			if result.Value() != true {
				continue
			}
		}

		inputValue, err := resolveRequestValue(h.request, act, vars)
		if err != nil {
			return fmt.Errorf("server-stream %s request: %w", h.id, err)
		}
		iterCount++
		streamState.SetRequestCount(streamState.GetRequestCount() + 1)

		err = executeWithRetry(ctx, h.retry, vars, func(ctx context.Context) error {
			req, convErr := exprToMessage(h.env, h.method, inputValue)
			if convErr != nil {
				return convErr
			}
			stream, callErr := h.client.CallServerStream(ctx, h.method, req)
			if callErr != nil {
				return callErr
			}
			for {
				resp, recvErr := stream.RecvMsg()
				if recvErr == io.EOF {
					break
				}
				if recvErr != nil {
					return recvErr
				}
				respExpr, respErr := transformResponse(h.responseProg, resp, h.env)
				if respErr != nil {
					return fmt.Errorf("server-stream %s response: %w", h.id, respErr)
				}
				streamState.SetResponseCount(streamState.GetResponseCount() + 1)
				streamState.SetValue(respExpr)
				if pubErr := publish(cloneStreamNode(streamState)); pubErr != nil {
					return pubErr
				}
			}
			return nil
		})
		if err == errSkipped {
			// Lifecycle pipeline: SKIP still goes through checkLifecycle
			// so NC/FC can react.
			nc, fc := h.checkLifecycle(vars)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if nc == LifecycleStop || fc == LifecycleStop {
				h.requestStop()
				break loop
			}
			continue
		}
		var contErr *ContinueError
		if errors.As(err, &contErr) {
			streamState.SetValue(contErr.Value)
			if pubErr := publish(cloneStreamNode(streamState)); pubErr != nil {
				return pubErr
			}
			// Lifecycle pipeline: CONTINUE emitted an error-derived value;
			// NC/FC still react to the iteration.
			nc, fc := h.checkLifecycle(vars)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if nc == LifecycleStop || fc == LifecycleStop {
				h.requestStop()
				break loop
			}
			continue
		}
		var suspendErr *SuspendError
		if errors.As(err, &suspendErr) {
			// Lifecycle pipeline: retry → NC → FC. NC/FC may promote to
			// terminate (cancelling ctx) or stop. checkLifecycle returns
			// the action each control selected.
			nc, fc := h.checkLifecycle(vars)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if nc == LifecycleStop || fc == LifecycleStop {
				h.requestStop()
				break loop
			}
			h.selfSuspend(err)
			res := h.waitForResume(ctx, h.StopChan())
			if res == suspendCancelled {
				return ctx.Err()
			}
			if res == suspendStopped {
				break loop
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("server-stream %s call %q: %w", h.id, h.method, err)
		}
		h.checkLifecycle(vars)
	}

	streamState.SetRequestClosed(true)
	streamState.SetResponseClosed(true)
	streamState.SetValue(newEOFValue())
	streamState.SetPhase(flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
	return publish(cloneStreamNode(streamState))
}

func cloneStreamNode(s *flowv1beta2.RunSnapshot_StreamNode) *flowv1beta2.RunSnapshot_StreamNode {
	return flowv1beta2.RunSnapshot_StreamNode_builder{
		Id:             s.GetId(),
		Value:          s.GetValue(),
		RequestClosed:  s.GetRequestClosed(),
		ResponseClosed: s.GetResponseClosed(),
		RequestCount:   s.GetRequestCount(),
		ResponseCount:  s.GetResponseCount(),
		Phase:          s.GetPhase(),
	}.Build()
}
