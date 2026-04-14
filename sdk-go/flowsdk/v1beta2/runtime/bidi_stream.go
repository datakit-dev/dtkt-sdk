package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// bidiStreamHandler streams messages in both directions concurrently.
type bidiStreamHandler struct {
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

func (h *bidiStreamHandler) Run(ctx context.Context) error {
	var stream rpc.BidiStream
	err := executeWithRetry(ctx, h.retry, nil, func(ctx context.Context) error {
		var openErr error
		stream, openErr = h.client.CallBidiStream(ctx, h.method)
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
		streamState.SetResponseClosed(true)
		streamState.SetPhase(flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
		return publish(cloneStreamNode(streamState))
	}
	if err != nil {
		return fmt.Errorf("bidi-stream %s open %q: %w", h.id, h.method, err)
	}

	streamState := &flowv1beta2.RunSnapshot_StreamNode{}
	streamState.SetId(h.id)
	streamState.SetPhase(flowv1beta2.RunSnapshot_PHASE_RUNNING)

	var mu sync.Mutex // protects streamState shared between send/receive goroutines

	// Feed input messages into the stream, applying when/close_request_when guards.
	go func() {
		defer stream.CloseSend() //nolint:errcheck
		var iterCount int
		for {
			if h.throttle > 0 && iterCount > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(h.throttle):
				}
			}

			act := newActivationFromChannels(ctx, h.inputs, h.env.TypeAdapter())
			vars, err := act.Resolve()
			if err != nil || act.AnyEOF() {
				return
			}

			if h.closeRequestWhenProg != nil {
				result, err := evalCEL(h.closeRequestWhenProg, vars)
				if err == nil && result.Value() == true {
					return
				}
			}

			if h.whenProg != nil {
				result, err := evalCEL(h.whenProg, vars)
				if err != nil || result.Value() != true {
					continue
				}
			}

			iterCount++
			mu.Lock()
			streamState.SetRequestCount(streamState.GetRequestCount() + 1)
			mu.Unlock()
			inputValue, err := resolveRequestValue(h.request, act, vars)
			if err != nil {
				return
			}
			req, err := exprToMessage(h.env, h.method, inputValue)
			if err != nil {
				return
			}
			if err := stream.SendMsg(req); err != nil {
				return
			}
		}
	}()

	for {
		resp, err := stream.RecvMsg()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("bidi-stream %s call %q: %w", h.id, h.method, err)
		}
		respExpr, err := transformResponse(h.responseProg, resp, h.env.TypeAdapter())
		if err != nil {
			return fmt.Errorf("bidi-stream %s response: %w", h.id, err)
		}
		mu.Lock()
		streamState.SetResponseCount(streamState.GetResponseCount() + 1)
		streamState.SetValue(respExpr)
		cloned := cloneStreamNode(streamState)
		mu.Unlock()
		if err := publishNode(h.pubsub, h.topic, cloned); err != nil {
			return err
		}
	}

	mu.Lock()
	streamState.SetRequestClosed(true)
	streamState.SetResponseClosed(true)
	streamState.SetValue(newEOFValue())
	streamState.SetPhase(flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
	cloned := cloneStreamNode(streamState)
	mu.Unlock()
	return publishNode(h.pubsub, h.topic, cloned)
}
