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

	_ selfSuspendable = (*tickerHandler)(nil)
	_ selfSuspendable = (*cronHandler)(nil)
	_ selfSuspendable = (*rangeHandler)(nil)
)

// newHandler creates a NodeHandler from a compiled node and its wired
// channels/pubsub. The compiled argument is one of the compiled* types from
// compile.go. transformPS is the direct pubsub used for transform pipeline
// internal communication. adapter is the CEL type adapter from the graph-level
// shared.Env.
func newHandler(compiled any, nodeID string, inputs map[string]<-chan *pubsub.Message, ps executor.PubSub, topic string, transformPS executor.PubSub, adapter types.Adapter) (executor.NodeHandler, error) {
	switch c := compiled.(type) {
	case *compiledVarSwitch:
		return &switchHandler{
			id:          nodeID,
			inputs:      inputs,
			pubsub:      ps,
			topic:       topic,
			valueProg:   c.valueProg,
			cases:       c.cases,
			defaultProg: c.defaultProg,
			transforms:  c.transforms,
			transformPS: transformPS,
			adapter:     adapter,
		}, nil

	case *compiledVarValue:
		return &varHandler{
			id:          nodeID,
			inputs:      inputs,
			pubsub:      ps,
			topic:       topic,
			program:     c.program,
			transforms:  c.transforms,
			transformPS: transformPS,
			adapter:     adapter,
		}, nil

	case *compiledTicker:
		return &tickerHandler{
			id:           nodeID,
			pubsub:       ps,
			topic:        topic,
			interval:     c.interval,
			delay:        c.delay,
			valueProgram: c.valueProgram,
			suspendCh:    make(chan struct{}, 1),
			resumeCh:     make(chan struct{}, 1),
		}, nil

	case *compiledRange:
		return &rangeHandler{
			id:        nodeID,
			pubsub:    ps,
			topic:     topic,
			start:     c.start,
			end:       c.end,
			step:      c.step,
			rate:      c.rate,
			suspendCh: make(chan struct{}, 1),
			resumeCh:  make(chan struct{}, 1),
		}, nil

	case *compiledCron:
		return &cronHandler{
			id:           nodeID,
			pubsub:       ps,
			topic:        topic,
			schedule:     c.schedule,
			valueProgram: c.valueProgram,
			suspendCh:    make(chan struct{}, 1),
			resumeCh:     make(chan struct{}, 1),
		}, nil

	case *compiledCall:
		switch c.kind {
		case rpc.MethodUnary:
			return &unaryHandler{
				id:           nodeID,
				method:       c.method,
				inputs:       inputs,
				pubsub:       ps,
				topic:        topic,
				client:       c.client,
				env:          c.env,
				whenProg:     c.whenProg,
				throttle:     c.throttle,
				cache:        c.cache,
				request:      c.request,
				responseProg: c.responseProg,
				retry:        c.retry,
			}, nil
		case rpc.MethodServerStream:
			return &serverStreamHandler{
				id:                   nodeID,
				method:               c.method,
				inputs:               inputs,
				pubsub:               ps,
				topic:                topic,
				client:               c.client,
				env:                  c.env,
				whenProg:             c.whenProg,
				closeRequestWhenProg: c.closeRequestWhenProg,
				throttle:             c.throttle,
				request:              c.request,
				responseProg:         c.responseProg,
				retry:                c.retry,
			}, nil
		case rpc.MethodClientStream:
			return &clientStreamHandler{
				id:                   nodeID,
				method:               c.method,
				inputs:               inputs,
				pubsub:               ps,
				topic:                topic,
				client:               c.client,
				env:                  c.env,
				whenProg:             c.whenProg,
				closeRequestWhenProg: c.closeRequestWhenProg,
				throttle:             c.throttle,
				request:              c.request,
				responseProg:         c.responseProg,
				retry:                c.retry,
			}, nil
		case rpc.MethodBidiStream:
			return &bidiStreamHandler{
				id:                   nodeID,
				method:               c.method,
				inputs:               inputs,
				pubsub:               ps,
				topic:                topic,
				client:               c.client,
				env:                  c.env,
				whenProg:             c.whenProg,
				closeRequestWhenProg: c.closeRequestWhenProg,
				throttle:             c.throttle,
				request:              c.request,
				responseProg:         c.responseProg,
				retry:                c.retry,
			}, nil
		default:
			return nil, fmt.Errorf("unsupported method kind for %q on node %s", c.method, nodeID)
		}

	case *compiledOutput:
		return &outputHandler{
			id:          nodeID,
			inputs:      inputs,
			program:     c.program,
			transforms:  c.transforms,
			transformPS: transformPS,
			pubsub:      ps,
			outputTopic: topic,
			throttle:    c.throttle,
			adapter:     adapter,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported compiled node type %T for node %s", compiled, nodeID)
	}
}
