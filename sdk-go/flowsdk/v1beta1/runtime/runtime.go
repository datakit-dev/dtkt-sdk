package runtime

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/funcs"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/uuid"
)

var _ shared.Runtime = (*Runtime)(nil)

type (
	Runtime struct {
		ctx        context.Context
		cancel     context.CancelFunc
		connectors shared.ConnectorProvider
		resolver   shared.Resolver
		types      shared.Types
		env        *cel.Env
		vars       cel.Activation
		proto      *flowv1beta1.Runtime
		inputs     []*flowv1beta1.Input
		outputs    []*flowv1beta1.Output
		nodes      *util.SyncMap[string, *Node]
		inputQueue []map[string]any
		userQueue  *util.OrderedMap[uuid.UUID, *PendingUserAction]
		userCh     chan *PendingUserAction
		pendingCh  chan *PendingUserAction
		mut        sync.Mutex
	}
	RuntimeOption func(*Runtime)
)

func New(ctx context.Context, spec *flowv1beta1.Flow, opts ...RuntimeOption) *Runtime {
	return NewFromProto(ctx, ProtoFromSpec(spec), opts...)
}

func NewFromProto(ctx context.Context, proto *flowv1beta1.Runtime, opts ...RuntimeOption) *Runtime {
	r := &Runtime{
		proto: proto,
		nodes: util.NewSyncMap(
			util.SliceMap(NewNodesFromMaps(
				proto.Connections,
				proto.Inputs,
				proto.Vars,
				proto.Actions,
				proto.Streams,
				proto.Outputs,
			), func(node *Node) util.MapPair[string, *Node] {
				return util.NewMapPair(node.proto.GetId(), node)
			})...,
		),
		inputs: util.SliceMap(
			slices.Collect(maps.Values(proto.GetInputs())),
			func(n *flowv1beta1.Node) *flowv1beta1.Input {
				return n.GetInput()
			},
		),
		outputs: util.SliceMap(
			slices.Collect(maps.Values(proto.GetOutputs())),
			func(n *flowv1beta1.Node) *flowv1beta1.Output {
				return n.GetOutput()
			},
		),
	}

	ctx, cancel := context.WithCancel(ctx)
	r.ctx = ctx
	r.cancel = cancel
	r.userQueue = util.NewOrderedMap[uuid.UUID, *PendingUserAction]()
	r.userCh = make(chan *PendingUserAction)
	r.pendingCh = make(chan *PendingUserAction)

	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}

	if r.connectors == nil {
		r.connectors = NewConnectors(
			util.SliceMap(
				slices.Collect(maps.Values(proto.GetConnections())),
				func(n *flowv1beta1.Node) *Connector {
					return NewConnector(n.GetConnection().GetId())
				},
			)...,
		)
	}

	return r
}

func ProtoFromSpec(spec *flowv1beta1.Flow) *flowv1beta1.Runtime {
	return &flowv1beta1.Runtime{
		Connections: NewNodeMap(spec.Connections...),
		Inputs:      NewNodeMap(spec.Inputs...),
		Vars:        NewNodeMap(spec.Vars...),
		Actions:     NewNodeMap(spec.Actions...),
		Streams:     NewNodeMap(spec.Streams...),
		Outputs:     NewNodeMap(spec.Outputs...),
	}
}

func WithConnectors(provider shared.ConnectorProvider) RuntimeOption {
	return func(r *Runtime) {
		r.connectors = provider
	}
}

func WithResolver(resolver shared.Resolver) RuntimeOption {
	return func(r *Runtime) {
		r.resolver = resolver
	}
}

func WithTypes(types shared.Types) RuntimeOption {
	return func(r *Runtime) {
		r.types = types
	}
}

func (r *Runtime) Resolver() shared.Resolver {
	r.mut.Lock()
	resolver := r.resolver
	r.mut.Unlock()

	if resolver == nil {
		r.mut.Lock()
		resolver = api.V1Beta1
		r.resolver = resolver
		r.mut.Unlock()
	}

	return resolver
}

func (r *Runtime) Types() (shared.Types, error) {
	r.mut.Lock()
	types := r.types
	r.mut.Unlock()

	if types == nil {
		var err error
		types, err = common.NewCELTypes(r.Resolver())
		if err != nil {
			return nil, err
		}

		r.mut.Lock()
		r.types = types
		r.mut.Unlock()
	}
	return types, nil
}

func (r *Runtime) Context() context.Context {
	return r.ctx
}

func (r *Runtime) Cancel() {
	r.cancel()
}

func (r *Runtime) Connectors() shared.ConnectorProvider {
	return r.connectors
}

func (r *Runtime) Inputs() []*flowv1beta1.Input {
	return r.inputs
}

func (r *Runtime) Outputs() []*flowv1beta1.Output {
	return r.outputs
}

func (r *Runtime) Proto() *flowv1beta1.Runtime {
	return r.proto
}

func (r *Runtime) PendingUserActions() <-chan *PendingUserAction {
	return r.pendingCh
}

func (r *Runtime) Parse(visitor shared.ParseNodeFunc) (err error) {
	r.nodes.Range(func(id string, node *Node) bool {
		err = spec.Parse(r, node.GetTypeNode(), func(ast *cel.Ast) {
			if visitor != nil {
				visitor(id, ast)
			}
		})
		if err != nil {
			err = fmt.Errorf("%s parse error: %w", id, err)
			return false
		}
		return true
	})
	return
}

func (r *Runtime) Compile() (err error) {
	r.nodes.Range(func(id string, node *Node) bool {
		err = node.Compile(r)
		if err != nil {
			err = fmt.Errorf("%s compile error: %w", id, err)
			return false
		}
		return true
	})
	return
}

func (r *Runtime) Build() (*cel.Env, cel.Activation, error) {
	env, err := r.Env()
	if err != nil {
		return nil, nil, err
	}

	vars, err := r.Vars()
	if err != nil {
		return nil, nil, err
	}

	return env, vars, nil
}

func (r *Runtime) Env() (*cel.Env, error) {
	r.mut.Lock()
	env := r.env
	r.mut.Unlock()

	if env == nil {
		types, err := r.Types()
		if err != nil {
			return nil, err
		}

		env, err := common.NewCELEnv(append([]cel.EnvOption{
			cel.CustomTypeProvider(types),
			cel.CustomTypeAdapter(types),
			cel.Container("dtkt"),
			cel.Abbrevs(
				string(new(flowv1beta1.Runtime_Done).ProtoReflect().Descriptor().FullName()),
				string(new(flowv1beta1.Runtime_EOF).ProtoReflect().Descriptor().FullName()),
			),
			cel.DeclareContextProto(r.proto.ProtoReflect().Descriptor()),
		}, funcs.Make(r)...)...)
		if err != nil {
			return nil, err
		}

		r.mut.Lock()
		if r.env == nil {
			r.env = env
		}
		result := r.env
		r.mut.Unlock()
		return result, nil
	}

	return env, nil
}

func (r *Runtime) Vars() (cel.Activation, error) {
	r.mut.Lock()
	defer r.mut.Unlock()

	if r.vars == nil {
		vars, err := cel.ContextProtoVars(r.proto)
		if err != nil {
			return nil, err
		}
		r.vars = vars
	}
	return r.vars, nil
}

func (r *Runtime) SetInputValues(values map[string]any) error {
	r.mut.Lock()
	defer r.mut.Unlock()

	if len(r.inputs) == 0 || len(values) == 0 {
		return nil
	}

	inputs := maps.Clone(values)
	maps.DeleteFunc(inputs, func(id string, _ any) bool {
		return !slices.ContainsFunc(r.inputs, func(i *flowv1beta1.Input) bool {
			return id == i.GetId()
		})
	})

	if len(inputs) > 0 {
		r.inputQueue = append(r.inputQueue, inputs)
		return nil
	}

	return fmt.Errorf("invalid inputs")
}

func (r *Runtime) GetInputValue(id string) (any, error) {
	r.mut.Lock()
	defer r.mut.Unlock()

	if len(r.inputQueue) > 0 {
		value, ok := r.inputQueue[len(r.inputQueue)-1][id]
		if ok {
			return value, nil
		}
	}

	return nil, spec.NewInputValueError(id)
}

func (r *Runtime) GetUserAction(id string) (*flowv1beta1.UserAction, bool) {
	r.mut.Lock()
	defer r.mut.Unlock()

	if node, ok := r.nodes.Load(id); ok {
		if action, ok := node.GetTypeNode().(*flowv1beta1.Action); ok && action.GetUser() != nil {
			return action.GetUser(), true
		}
	}
	return nil, false
}

func (r *Runtime) GetUserValues(id string) (map[string]any, error) {
	if node, ok := r.nodes.Load(id); ok {
		if action, ok := node.GetTypeNode().(*flowv1beta1.Action); ok && action.GetUser() != nil {
			pending := &PendingUserAction{
				uid:        uuid.New(),
				nodeID:     id,
				userAction: action.GetUser(),
			}

			r.mut.Lock()
			r.userQueue.Store(pending.uid, pending)
			r.mut.Unlock()

			r.pendingCh <- pending

			select {
			case <-r.ctx.Done():
				return nil, fmt.Errorf("get user action: %w", r.ctx.Err())
			case userAction := <-r.userCh:
				if userAction.uid != pending.uid {
					return nil, fmt.Errorf("invalid user action, waiting for: %s, got: %s", userAction.nodeID, id)
				}

				r.mut.Lock()
				r.userQueue.Delete(userAction.uid)
				r.mut.Unlock()

				return userAction.values, nil
			}
		}
	}

	return nil, fmt.Errorf("user action not found (GetUserValues): %s", id)
}

func (r *Runtime) SetUserValues(id string, values map[string]any) error {
	if node, ok := r.nodes.Load(id); ok {
		if action, ok := node.GetTypeNode().(*flowv1beta1.Action); ok && action.GetUser() != nil {
			_, userAction, ok := r.userQueue.Last()
			if ok && userAction.nodeID != id {
				return fmt.Errorf("invalid user action, waiting for: %s, got: %s", userAction.nodeID, id)
			} else if ok {
				userAction.values = values
				r.userCh <- userAction
				return nil
			}
		}
	}

	return fmt.Errorf("user action not found (SetUserValues): %s", id)
}

func (r *Runtime) GetOutputValues() (map[string]any, error) {
	r.mut.Lock()
	outputs := r.outputs
	r.mut.Unlock()

	var (
		values = map[string]any{}
		errs   []error
	)
	for _, output := range outputs {
		value, err := r.GetNodeValue(spec.GetID(output))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		values[output.GetId()] = value
	}
	return values, errors.Join(errs...)
}

func (r *Runtime) RangeNodes(f func(id string, node shared.Node) bool) {
	r.nodes.Range(func(id string, node *Node) bool {
		return f(id, node)
	})
}

func (r *Runtime) GetNode(id string) (shared.Node, bool) {
	node, ok := r.nodes.Load(id)
	if ok {
		return node, true
	}
	return nil, false
}

func (r *Runtime) GetNodeValue(id string) (any, error) {
	node, ok := r.nodes.Load(id)
	if !ok {
		return nil, fmt.Errorf("node not found: %s", id)
	}
	return node.GetValue(r)
}

func (r *Runtime) Eval(expr string) (ref.Val, error) {
	env, vars, err := r.Build()
	if err != nil {
		return nil, err
	}

	ast, iss := env.Parse(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	ast, iss = env.Check(ast)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, err
	}

	val, _, err := prg.ContextEval(r.Context(), vars)
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (r *Runtime) Reset() {
	r.nodes.Range(func(_ string, node *Node) bool {
		node.Reset()
		return true
	})

	r.mut.Lock()
	if len(r.inputQueue) > 0 {
		r.inputQueue = r.inputQueue[:len(r.inputQueue)-1]
	}
	r.mut.Unlock()
}
