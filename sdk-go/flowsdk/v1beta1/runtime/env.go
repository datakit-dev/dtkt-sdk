package runtime

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/decls"
	celtypes "github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/proto"
)

var _ shared.Env = (*Env)(nil)

type (
	Env struct {
		*cel.Env
		proto    *flowv1beta1.Runtime
		resolver shared.Resolver
		types    *common.CELTypes
		nodes    NodeMap
		conns    ConnectorProvider

		mut sync.Mutex
	}
	EnvOption func(*Env)
)

func NewEnv(proto *flowv1beta1.Runtime, opts ...Option) (_ *Env, err error) {
	env := &Env{
		proto: proto,
	}
	env.applyOptions(opts...)

	if env.resolver == nil {
		env.resolver = api.GlobalResolver()
	}

	if env.conns == nil {
		env.conns = shared.GetConnectorFunc[*sharedv1beta1.Package_Identity](func(node shared.Connection[*sharedv1beta1.Package_Identity]) (shared.Connector, error) {
			return nil, fmt.Errorf("connections.%s: runtime missing connector provider", node.GetId())
		})
	}

	if env.nodes == nil {
		env.nodes = NewNodeMap(
			proto.Actions,
			proto.Connections,
			proto.Inputs,
			proto.Outputs,
			proto.Streams,
			proto.Vars,
		)
	}

	env.types, err = common.NewCELTypes(env.resolver)
	if err != nil {
		return nil, err
	}

	env.Env, err = newCELEnv(env.types, env.nodes.Values()...)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func ProtoFromSpec(spec *flowv1beta1.Flow) *flowv1beta1.Runtime {
	return &flowv1beta1.Runtime{
		Actions:     specNodesMap(spec.Actions),
		Connections: specNodesMap(spec.Connections),
		Inputs:      specNodesMap(spec.Inputs),
		Outputs:     specNodesMap(spec.Outputs),
		Streams:     specNodesMap(spec.Streams),
		Vars:        specNodesMap(spec.Vars),
	}
}

func newCELEnv(types shared.Types, nodes ...*Node) (*cel.Env, error) {
	return common.NewCELEnv(
		cel.CustomTypeProvider(types),
		cel.CustomTypeAdapter(types),
		cel.Abbrevs(
			string(new(flowv1beta1.Runtime_Done).ProtoReflect().Descriptor().FullName()),
			string(new(flowv1beta1.Runtime_EOF).ProtoReflect().Descriptor().FullName()),
		),
		cel.VariableDecls(util.SliceReduce(nodes, func(node *Node) (*decls.VariableDecl, bool) {
			return decls.NewVariable(spec.GetNodeIdent(node.Spec()), shared.NodeVarsType), true
		})...),
		cel.Function("now",
			cel.SingletonFunctionBinding(
				func(...ref.Val) ref.Val {
					return celtypes.Timestamp{
						Time: time.Now(),
					}
				},
			),
			cel.Overload("now", nil, cel.TimestampType),
		),
		cel.Function("isEOF",
			cel.Overload("isEOF_dyn", []*cel.Type{cel.DynType}, cel.BoolType),
			cel.SingletonFunctionBinding(
				func(args ...ref.Val) ref.Val {
					switch args[0].Value().(type) {
					case *flowv1beta1.Runtime_EOF:
						return celtypes.True
					}
					return celtypes.False
				},
			),
		))
}

func (e *Env) Proto() *flowv1beta1.Runtime {
	e.mut.Lock()
	defer e.mut.Unlock()
	return proto.CloneOf(e.proto)
}

func (e *Env) Extend(opts ...cel.EnvOption) (shared.Env, error) {
	types := e.types.Copy()
	opts = append([]cel.EnvOption{
		cel.CustomTypeProvider(types),
		cel.CustomTypeAdapter(types),
	}, opts...)

	env, err := e.Env.Extend(opts...)
	if err != nil {
		return nil, err
	}

	return &Env{
		Env:      env,
		proto:    e.proto,
		resolver: e.resolver,
		types:    e.types,
		nodes:    e.nodes,
		conns:    e.conns,
	}, nil
}

func (e *Env) Vars(context.Context) (cel.Activation, error) {
	return cel.NewActivation(e.nodes.Vars(e.types))
}

func (e *Env) Resolver() shared.Resolver {
	return e.resolver
}

func (e *Env) Types() shared.Types {
	return e.types
}

func (e *Env) Connector(id string) (shared.Connector, error) {
	node, ok := e.nodes.Load(id)
	if !ok {
		return nil, fmt.Errorf("node not found: %s", id)
	}

	conn := node.proto.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("connection not found: %s", id)
	}

	return e.conns.GetConnector(conn)
}

func (r *Env) Inputs() (inputs map[string]*Node) {
	inputs = make(map[string]*Node)
	r.nodes.Range(func(id string, node *Node) bool {
		if strings.HasPrefix(id, shared.InputPrefix+".") {
			inputs[id] = node
		}
		return true
	})
	return
}

func (r *Env) Outputs() (outputs map[string]*Node) {
	outputs = make(map[string]*Node)
	r.nodes.Range(func(id string, node *Node) bool {
		if strings.HasPrefix(id, shared.OutputPrefix+".") {
			outputs[id] = node
		}
		return true
	})
	return
}

func (r *Env) GetNode(id string) (*Node, bool) {
	return r.nodes.Load(id)
}

func (e *Env) IsDone() (done *Done, ok bool) {
	e.nodes.Range(func(_ string, node *Node) bool {
		switch value := node.currValue.(type) {
		case *flowv1beta1.Runtime_Done:
			// Prefer error Dones: keep searching if we already have a non-error.
			if !ok || value.GetIsError() {
				done = NewDone(value)
				ok = true
			}
			if value.GetIsError() {
				return false // error found — stop immediately
			}
		}
		return true
	})
	return
}

func (e *Env) applyOptions(opts ...Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
}
