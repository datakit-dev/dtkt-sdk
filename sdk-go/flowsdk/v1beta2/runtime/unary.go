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
	cache        cache.Cache      // non-nil when memoize is enabled
	request      *compiledRequest // nil = use FirstInputValue
	responseProg cel.Program      // nil = use raw response
	retry        *compiledRetryStrategy
}

func (h *unaryHandler) Run(ctx context.Context) error {
	var evalCount uint64
loop:
	for {
		// Pause point: between iterations, BEFORE starting the next call.
		// An in-flight call from the previous iteration always completes
		// naturally. Suspend never aborts an RPC mid-flight (we can't
		// guarantee that's safe / idempotent on the server side).
		act := newActivationFromChannelsInterruptible(ctx, h.inputs, h.env.TypeAdapter(), h.SuspendChan(), h.StopChan())
		if h.throttle > 0 && evalCount > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-h.StopChan():
				break loop // graceful stop; post-loop SUCCEEDED publish fires
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
			// Lifecycle pipeline: retry's SKIP outcome still goes through
			// checkLifecycle. NC/FC may decide to stop/terminate based on
			// the iteration's vars (e.g. NC.stop_when after N skips).
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
			node := flowv1beta2.RunSnapshot_ActionNode_builder{
				Id:        h.id,
				Value:     contErr.Value,
				EvalCount: evalCount,
				Phase:     flowv1beta2.RunSnapshot_PHASE_RUNNING,
			}.Build()
			if err := publishNode(h.pubsub, h.topic, node); err != nil {
				return err
			}
			// Lifecycle pipeline: retry's CONTINUE emits an error-derived
			// value; NC/FC should still react to the iteration as if a
			// normal value had been published.
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
			// Lifecycle pipeline: retry's SUSPEND outcome is the iteration's
			// "starting intent". NC and FC may promote it (terminate beats
			// suspend; stop also wins) or no-op. checkLifecycle returns the
			// action each control selected; we branch on it.
			nc, fc := h.checkLifecycle(vars)
			// Terminate cancels ctx; we exit on ctx.Err.
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Stop fired (NC or FC): exit the loop instead of suspending.
			// (recv() can't see this -- we're between iterations.)
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
		h.checkLifecycle(vars)
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
