package runtime

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/proto"
)

var _ shared.Runtime = (*Runtime)(nil)

type Runtime struct {
	proto *flowv1beta1.Runtime

	nodes NodeMap
	conns shared.ConnectorProvider
	env   func() (*Env, error)

	recvChs map[string]chan any
	sendChs map[string]chan ref.Val

	ctx    context.Context
	cancel context.CancelCauseFunc
	mut    sync.Mutex
}

func NewFromSpec(ctx context.Context, cancel context.CancelCauseFunc, spec *flowv1beta1.Flow, opts ...Option) *Runtime {
	return NewFromProto(ctx, cancel, ProtoFromSpec(spec), opts...)
}

func NewFromProto(ctx context.Context, cancel context.CancelCauseFunc, proto *flowv1beta1.Runtime, opts ...Option) *Runtime {
	nodes := RuntimeNodeMap(
		proto.Actions,
		proto.Connections,
		proto.Inputs,
		proto.Outputs,
		proto.Streams,
		proto.Vars,
	)

	run := &Runtime{
		proto: proto,

		nodes: nodes,
		env: sync.OnceValues(func() (*Env, error) {
			return NewEnv(proto, append(opts, WithNodes(nodes))...)
		}),

		recvChs: map[string]chan any{},
		sendChs: map[string]chan ref.Val{},

		ctx:    ctx,
		cancel: cancel,
	}

	run.applyOptions(opts...)

	if run.conns == nil {
		run.conns = NewConnectors(
			util.SliceMap(
				slices.Collect(maps.Values(run.proto.GetConnections())),
				func(n *flowv1beta1.Node) *Connector {
					return NewConnector(n.GetConnection())
				},
			)...,
		)
	}

	return run
}

func ProtoFromSpec(spec *flowv1beta1.Flow) *flowv1beta1.Runtime {
	return &flowv1beta1.Runtime{
		Actions:     SpecNodeMap(spec.Actions),
		Connections: SpecNodeMap(spec.Connections),
		Inputs:      SpecNodeMap(spec.Inputs),
		Outputs:     SpecNodeMap(spec.Outputs),
		Streams:     SpecNodeMap(spec.Streams),
		Vars:        SpecNodeMap(spec.Vars),
	}
}

func (r *Runtime) applyOptions(opts ...Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
}

func (r *Runtime) Context() context.Context {
	return r.ctx
}

func (r *Runtime) Cancel(cause error) {
	r.cancel(cause)
}

func (r *Runtime) Connectors() shared.ConnectorProvider {
	return r.conns
}

func (r *Runtime) Inputs() (inputs []*flowv1beta1.Input) {
	nodeMap := r.Proto().GetInputs()
	inputs = make([]*flowv1beta1.Input, len(nodeMap))
	var idx int
	for _, node := range nodeMap {
		inputs[idx] = node.GetInput()
		idx++
	}
	return
}

func (r *Runtime) Outputs() (outputs []*flowv1beta1.Output) {
	nodeMap := r.Proto().GetOutputs()
	outputs = make([]*flowv1beta1.Output, len(nodeMap))
	var idx int
	for _, node := range nodeMap {
		outputs[idx] = node.GetOutput()
		idx++
	}
	return
}

func (r *Runtime) Proto() *flowv1beta1.Runtime {
	r.mut.Lock()
	defer r.mut.Unlock()
	return proto.CloneOf(r.proto)
}

func (r *Runtime) Env() (shared.Env, error) {
	return r.env()
}

func (r *Runtime) GetNode(id string) (shared.SpecNode, bool) {
	node, ok := r.nodes.Load(id)
	if ok {
		return GetSpecNode(node.proto), true
	}
	return nil, false
}

func (r *Runtime) GetValue(id string) (any, error) {
	node, ok := r.nodes.Load(id)
	if !ok {
		return nil, fmt.Errorf("method GetValue: node not found: %s", id)
	}
	return node.GetValue(r)
}

func (r *Runtime) IsCompleted(id string) bool {
	node, ok := r.nodes.Load(id)
	if ok {
		return node.Completed()
	}
	return false
}

func (r *Runtime) GetSendCh(id string) (chan ref.Val, error) {
	_, ok := r.nodes.Load(id)
	if !ok {
		return nil, fmt.Errorf("method getSendCh: node not found: %q", id)
	}

	r.mut.Lock()
	defer r.mut.Unlock()

	sendCh, ok := r.sendChs[id]
	if !ok {
		sendCh = make(chan ref.Val, 1)
		r.sendChs[id] = sendCh
	}

	return sendCh, nil
}

func (r *Runtime) GetRecvCh(id string) (chan any, error) {
	_, ok := r.nodes.Load(id)
	if !ok {
		return nil, fmt.Errorf("method getRecvCh: node not found: %q", id)
	}

	r.mut.Lock()
	defer r.mut.Unlock()

	recvCh, ok := r.recvChs[id]
	if !ok {
		recvCh = make(chan any, 1)
		r.recvChs[id] = recvCh
	}

	return recvCh, nil
}

// func (r *Runtime) GetUserValues(id string) (map[string]any, error) {
// 	if node, ok := r.nodes.Load(id); ok {
// 		if action, ok := node.GetTypeNode().(*flowv1beta1.Action); ok && action.GetUser() != nil {
// 			pending := &PendingUserAction{
// 				uid:        uuid.New(),
// 				nodeID:     id,
// 				userAction: action.GetUser(),
// 			}

// 			r.mut.Lock()
// 			r.userQueue.Store(pending.uid, pending)
// 			r.mut.Unlock()

// 			r.pendingCh <- pending

// 			select {
// 			case <-r.ctx.Done():
// 				return nil, fmt.Errorf("get user action: %w", r.ctx.Err())
// 			case userAction := <-r.userCh:
// 				if userAction.uid != pending.uid {
// 					return nil, fmt.Errorf("invalid user action, waiting for: %s, got: %s", userAction.nodeID, id)
// 				}

// 				r.mut.Lock()
// 				r.userQueue.Delete(userAction.uid)
// 				r.mut.Unlock()

// 				return userAction.values, nil
// 			}
// 		}
// 	}

// 	return nil, fmt.Errorf("user action not found (GetUserValues): %s", id)
// }

// func (r *Runtime) SetUserValues(id string, values map[string]any) error {
// 	if node, ok := r.nodes.Load(id); ok {
// 		if action, ok := node.GetTypeNode().(*flowv1beta1.Action); ok && action.GetUser() != nil {
// 			_, userAction, ok := r.userQueue.Last()
// 			if ok && userAction.nodeID != id {
// 				return fmt.Errorf("invalid user action, waiting for: %s, got: %s", userAction.nodeID, id)
// 			} else if ok {
// 				userAction.values = values
// 				r.userCh <- userAction
// 				return nil
// 			}
// 		}
// 	}

// 	return fmt.Errorf("user action not found (SetUserValues): %s", id)
// }

// func (r *Runtime) GetOutputValues() (map[string]any, error) {
// 	r.mut.Lock()
// 	outputs := r.outputs
// 	r.mut.Unlock()

// 	var (
// 		values = map[string]any{}
// 		errs   []error
// 	)
// 	for _, output := range outputs {
// 		value, err := r.GetNodeValue(spec.GetID(output))
// 		if err != nil {
// 			errs = append(errs, err)
// 			continue
// 		}
// 		values[output.GetId()] = value
// 	}
// 	return values, errors.Join(errs...)
// }

// func (r *Runtime) SetInputValues(values map[string]any) error {
// 	r.mut.Lock()
// 	defer r.mut.Unlock()

// 	if len(r.inputs) == 0 || len(values) == 0 {
// 		return nil
// 	}

// 	inputs := maps.Clone(values)
// 	maps.DeleteFunc(inputs, func(id string, _ any) bool {
// 		return !slices.ContainsFunc(r.inputs, func(i *flowv1beta1.Input) bool {
// 			return id == i.GetId()
// 		})
// 	})

// 	if len(inputs) > 0 {
// 		r.inputQueue = append(r.inputQueue, inputs)
// 		return nil
// 	}

// 	return fmt.Errorf("invalid inputs")
// }

// func (r *Runtime) PendingUserActions() <-chan *PendingUserAction {
// 	return r.pendingCh
// }
