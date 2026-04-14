package runtime

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"time"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/cache"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// unaryHandler sends each incoming message as a request and forwards the single response.
type unaryHandler struct {
	flowControlMixin
	id           string
	method       protoreflect.FullName
	inputs       map[string]<-chan *pubsub.Message
	pubsub       executor.PubSub
	topic        string
	client       rpc.Client
	env          shared.Env
	whenProg     cel.Program
	throttle     time.Duration
	cache        cache.Cache      // non-nil when memoize is enabled
	request      *compiledRequest // nil = use FirstInputValue
	responseProg cel.Program      // nil = use raw response
	retry        *compiledRetryStrategy
}

func (h *unaryHandler) Run(ctx context.Context) error {
	var evalCount uint64
	for {
		if h.throttle > 0 && evalCount > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(h.throttle):
			}
		}

		act := newActivationFromChannels(ctx, h.inputs, h.env.TypeAdapter())
		vars, err := act.Resolve()
		if err != nil {
			return fmt.Errorf("unary %s resolve: %w", h.id, err)
		}
		if act.AnyEOF() {
			break
		}

		if h.whenProg != nil {
			result, err := evalCEL(h.whenProg, vars)
			if err != nil {
				return fmt.Errorf("unary %s when: %w", h.id, err)
			}
			if result.Value() != true {
				continue
			}
		}

		inputValue, err := resolveRequestValue(h.request, act, vars)
		if err != nil {
			return fmt.Errorf("unary %s request: %w", h.id, err)
		}

		evalCount++
		var resp proto.Message
		err = executeWithRetry(ctx, h.retry, vars, func(ctx context.Context) error {
			var callErr error
			resp, callErr = h.callUnary(ctx, inputValue)
			return callErr
		})
		if err == errSkipped {
			continue
		}
		var contErr *ContinueError
		if errors.As(err, &contErr) {
			node := flowv1beta2.RunSnapshot_ActionNode_builder{
				Id:        h.id,
				Value:     contErr.Value,
				EvalCount: evalCount,
				Phase:     flowv1beta2.RunSnapshot_PHASE_RUNNING,
			}.Build()
			if err := publishNode(h.pubsub, h.topic, node); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("unary %s call %q: %w", h.id, h.method, err)
		}

		respExpr, err := transformResponse(h.responseProg, resp, h.env.TypeAdapter())
		if err != nil {
			return fmt.Errorf("unary %s response: %w", h.id, err)
		}

		node := flowv1beta2.RunSnapshot_ActionNode_builder{
			Id:        h.id,
			Value:     respExpr,
			EvalCount: evalCount,
			Phase:     flowv1beta2.RunSnapshot_PHASE_RUNNING,
		}.Build()
		if err := publishNode(h.pubsub, h.topic, node); err != nil {
			return err
		}
		h.checkFC(vars)
	}

	return publishNode(h.pubsub, h.topic, flowv1beta2.RunSnapshot_ActionNode_builder{
		Id:        h.id,
		Value:     newEOFValue(),
		EvalCount: evalCount,
		Phase:     flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
	}.Build())
}

// callUnary either returns a cached response or calls through to the RPC client.
func (h *unaryHandler) callUnary(ctx context.Context, input *expr.Value) (proto.Message, error) {
	req, err := exprToMessage(h.env, h.method, input)
	if err != nil {
		return nil, err
	}
	if h.cache == nil {
		return h.client.CallUnary(ctx, h.method, req)
	}
	hash, err := hashExprValue(input)
	if err != nil {
		// If hashing fails, fall through to the RPC.
		return h.client.CallUnary(ctx, h.method, req)
	}
	key := fmt.Sprintf("%s:%016x", h.id, hash)
	if cached, ok, err := h.cache.Get(ctx, key); err == nil && ok {
		return cached, nil
	}
	resp, err := h.client.CallUnary(ctx, h.method, req)
	if err != nil {
		return nil, err
	}
	_ = h.cache.Set(ctx, key, resp)
	return resp, nil
}

// hashExprValue produces a FNV-1a hash of a deterministically marshaled *expr.Value.
func hashExprValue(v *expr.Value) (uint64, error) {
	if v == nil {
		return 0, nil
	}
	b, err := proto.MarshalOptions{Deterministic: true}.Marshal(v)
	if err != nil {
		return 0, err
	}
	h := fnv.New64a()
	_, _ = h.Write(b)
	return h.Sum64(), nil
}
