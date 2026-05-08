package runtime

import (
	"fmt"

	"github.com/google/cel-go/common/types"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
)

// Compile-time interface assertions.
var (
	_ executor.NodeHandler = (*varHandler)(nil)
	_ executor.NodeHandler = (*switchHandler)(nil)
	_ executor.NodeHandler = (*inputHandler)(nil)
	_ executor.NodeHandler = (*rangeHandler)(nil)
	_ executor.NodeHandler = (*cronHandler)(nil)
	_ executor.NodeHandler = (*interactionHandler)(nil)
	_ executor.NodeHandler = (*tickerHandler)(nil)
	_ executor.NodeHandler = (*outputHandler)(nil)
	_ executor.NodeHandler = (*unaryHandler)(nil)
	_ executor.NodeHandler = (*serverStreamHandler)(nil)
	_ executor.NodeHandler = (*clientStreamHandler)(nil)
	_ executor.NodeHandler = (*bidiStreamHandler)(nil)

	// Every handler that runs as a long-lived goroutine in launchHandlers
	// implements selfSuspendable. Suspend pauses the handler's loop at a
	// safe point (between iterations) without exiting its goroutine, and
	// without interrupting any in-flight external operation. For streams
	// this means the connection stays open and the receive side keeps
	// flowing - only the publish side pauses. For unary actions the
	// current call completes naturally; only the next iteration pauses.
	//
	// This design is intentional: closing a stream or aborting a call
	// mid-flight cannot be guaranteed idempotent (server-side state,
	// side effects). Suspend must NOT cause those.
	_ selfSuspendable = (*tickerHandler)(nil)
	_ selfSuspendable = (*cronHandler)(nil)
	_ selfSuspendable = (*rangeHandler)(nil)
	_ selfSuspendable = (*varHandler)(nil)
	_ selfSuspendable = (*switchHandler)(nil)
	_ selfSuspendable = (*outputHandler)(nil)
	_ selfSuspendable = (*unaryHandler)(nil)
	_ selfSuspendable = (*serverStreamHandler)(nil)
	_ selfSuspendable = (*clientStreamHandler)(nil)
	_ selfSuspendable = (*bidiStreamHandler)(nil)
	_ selfSuspendable = (*interactionHandler)(nil)

	// Every handler that runs in launchHandlers also implements
	// selfStoppable. Stop signals stopCh; the handler exits with
	// PHASE_SUCCEEDED at its next safe point. This is distinct from
	// TerminateNode which cancels ctx and exits with PHASE_CANCELLED.
	// Generators (ticker/cron/range) used to fall through to ctx-cancel
	// for stop -- conflating stop with terminate at the wire level. They
	// now implement selfStoppable directly so the two paths are distinct.
	_ selfStoppable = (*tickerHandler)(nil)
	_ selfStoppable = (*cronHandler)(nil)
	_ selfStoppable = (*rangeHandler)(nil)
	_ selfStoppable = (*varHandler)(nil)
	_ selfStoppable = (*switchHandler)(nil)
	_ selfStoppable = (*outputHandler)(nil)
	_ selfStoppable = (*unaryHandler)(nil)
	_ selfStoppable = (*serverStreamHandler)(nil)
	_ selfStoppable = (*clientStreamHandler)(nil)
	_ selfStoppable = (*bidiStreamHandler)(nil)
	_ selfStoppable = (*interactionHandler)(nil)
)

// newHandler creates a NodeHandler from a compiled node and its wired
// channels/pubsub. The compiled argument is one of the compiled* types from
// compile.go. transformPS is the direct pubsub used for transform pipeline
// internal communication. adapter is the CEL type adapter from the graph-level
// shared.Env. cb is the cache backend, used by handlers that may be cache:true
// producers (Var, Action); other handlers ignore it.
//
// id is the bare spec id (Format A, e.g. "x") used in every protobuf
// event/snapshot field whose validator is the bare-id pattern. The
// fully-qualified id (Format B) is supplied separately as `topic` for
// pubsub routing; the handler does not see it directly.
func newHandler(compiled any, id string, inputs map[string]<-chan *pubsub.Message, ps executor.PubSub, topic string, transformPS executor.PubSub, adapter types.Adapter, cb *cacheBackend) (executor.NodeHandler, error) {
	switch c := compiled.(type) {
	case *compiledVarSwitch:
		h := &switchHandler{
			id:          id,
			inputs:      inputs,
			pubsub:      ps,
			topic:       topic,
			valueProg:   c.valueProg,
			cases:       c.cases,
			defaultProg: c.defaultProg,
			transforms:  c.transforms,
			transformPS: transformPS,
			adapter:     adapter,
			cache:       cb,
		}
		h.initSuspendable()
		h.initStoppable()
		return h, nil

	case *compiledVarValue:
		h := &varHandler{
			id:          id,
			inputs:      inputs,
			pubsub:      ps,
			topic:       topic,
			program:     c.program,
			transforms:  c.transforms,
			transformPS: transformPS,
			adapter:     adapter,
			cache:       cb,
		}
		h.initSuspendable()
		h.initStoppable()
		return h, nil

	case *compiledTicker:
		h := &tickerHandler{
			id:           id,
			pubsub:       ps,
			topic:        topic,
			interval:     c.interval,
			delay:        c.delay,
			valueProgram: c.valueProgram,
			suspendCh:    make(chan struct{}, 1),
			resumeCh:     make(chan struct{}, 1),
		}
		h.initStoppable()
		return h, nil

	case *compiledRange:
		h := &rangeHandler{
			id:        id,
			pubsub:    ps,
			topic:     topic,
			start:     c.start,
			end:       c.end,
			step:      c.step,
			rate:      c.rate,
			suspendCh: make(chan struct{}, 1),
			resumeCh:  make(chan struct{}, 1),
		}
		h.initStoppable()
		return h, nil

	case *compiledCron:
		h := &cronHandler{
			id:           id,
			pubsub:       ps,
			topic:        topic,
			schedule:     c.schedule,
			valueProgram: c.valueProgram,
			suspendCh:    make(chan struct{}, 1),
			resumeCh:     make(chan struct{}, 1),
		}
		h.initStoppable()
		return h, nil

	case *compiledCall:
		switch c.kind {
		case rpc.MethodUnary:
			h := &unaryHandler{
				id:           id,
				method:       c.method,
				inputs:       inputs,
				pubsub:       ps,
				topic:        topic,
				client:       c.client,
				env:          c.env,
				whenProg:     c.whenProg,
				throttle:     c.throttle,
				memoize:      c.memoize,
				cache:        cb,
				request:      c.request,
				responseProg: c.responseProg,
				retry:        c.retry,
			}
			h.initSuspendable()
			h.initStoppable()
			return h, nil
		case rpc.MethodServerStream:
			h := &serverStreamHandler{
				id:           id,
				method:       c.method,
				inputs:       inputs,
				pubsub:       ps,
				topic:        topic,
				client:       c.client,
				env:          c.env,
				whenProg:     c.whenProg,
				throttle:     c.throttle,
				request:      c.request,
				responseProg: c.responseProg,
				retry:        c.retry,
				cache:        cb,
			}
			h.initSuspendable()
			h.initStoppable()
			return h, nil
		case rpc.MethodClientStream:
			h := &clientStreamHandler{
				id:           id,
				method:       c.method,
				inputs:       inputs,
				pubsub:       ps,
				topic:        topic,
				client:       c.client,
				env:          c.env,
				whenProg:     c.whenProg,
				throttle:     c.throttle,
				request:      c.request,
				responseProg: c.responseProg,
				retry:        c.retry,
				cache:        cb,
			}
			h.initSuspendable()
			h.initStoppable()
			return h, nil
		case rpc.MethodBidiStream:
			h := &bidiStreamHandler{
				id:           id,
				method:       c.method,
				inputs:       inputs,
				pubsub:       ps,
				topic:        topic,
				client:       c.client,
				env:          c.env,
				whenProg:     c.whenProg,
				throttle:     c.throttle,
				request:      c.request,
				responseProg: c.responseProg,
				retry:        c.retry,
				cache:        cb,
			}
			h.initSuspendable()
			h.initStoppable()
			return h, nil
		default:
			return nil, fmt.Errorf("unsupported method kind for %q on node %s", c.method, id)
		}

	case *compiledOutput:
		h := &outputHandler{
			id:          id,
			inputs:      inputs,
			program:     c.program,
			transforms:  c.transforms,
			transformPS: transformPS,
			pubsub:      ps,
			outputTopic: topic,
			throttle:    c.throttle,
			adapter:     adapter,
			cache:       cb,
		}
		h.initSuspendable()
		h.initStoppable()
		return h, nil

	default:
		return nil, fmt.Errorf("unsupported compiled node type %T for node %s", compiled, id)
	}
}
