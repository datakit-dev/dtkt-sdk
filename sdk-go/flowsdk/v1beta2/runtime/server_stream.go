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
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// serverStreamHandler sends each incoming message as a request and forwards the stream of responses.
type serverStreamHandler struct {
	flowControlMixin
	id                   string
	method               protoreflect.FullName
	inputs               map[string]<-chan *pubsub.Message
	pubsub               executor.PubSub
	topic                string
	client               rpc.Client
	env                  shared.Env
	whenProg             cel.Program
	closeRequestWhenProg cel.Program
	throttle             time.Duration
	request              *compiledRequest // nil = use FirstInputValue
	responseProg         cel.Program      // nil = use raw response
	retry                *compiledRetryStrategy
}

func (h *serverStreamHandler) Run(ctx context.Context) error {
	streamState := &flowv1beta2.RunSnapshot_StreamNode{}
	streamState.SetId(h.id)
	streamState.SetPhase(flowv1beta2.RunSnapshot_PHASE_RUNNING)
	publish := func(node executor.StateNode) error {
		return publishNode(h.pubsub, h.topic, node)
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

		act := newActivationFromChannels(ctx, h.inputs, h.env.TypeAdapter())
		vars, err := act.Resolve()
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

		if h.closeRequestWhenProg != nil {
			result, err := evalCEL(h.closeRequestWhenProg, vars)
			if err != nil {
				return fmt.Errorf("server-stream %s close_request_when: %w", h.id, err)
			}
			if result.Value() == true {
				break
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
				respExpr, respErr := transformResponse(h.responseProg, resp, h.env.TypeAdapter())
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
			continue
		}
		var contErr *ContinueError
		if errors.As(err, &contErr) {
			streamState.SetValue(contErr.Value)
			if pubErr := publish(cloneStreamNode(streamState)); pubErr != nil {
				return pubErr
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("server-stream %s call %q: %w", h.id, h.method, err)
		}
		h.checkFC(vars)
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
