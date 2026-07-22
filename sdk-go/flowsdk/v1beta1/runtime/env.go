package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/decls"
	"github.com/google/cel-go/common/types"
	celtypes "github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/proto"
)

var _ shared.Env = (*Env)(nil)

var doneMsg *flowv1beta1.Done
var eofMsg *flowv1beta1.EOF
var skipMsg *flowv1beta1.Skip

var nodeVarsType = cel.MapType(cel.StringType, cel.DynType)

type (
	Env struct {
		*cel.Env

		proto *flowv1beta1.Env
		vars  EnvVars

		conns    ConnectorProvider
		resolver shared.Resolver
		types    *common.CELTypes

		mut sync.RWMutex
	}
	EnvVars   func(context.Context, shared.Env) (cel.Activation, error)
	EnvOption func(*Env)
)

func NewEnv(opts ...Option) (_ *Env, err error) {
	env := &Env{}
	env.applyOptions(opts...)

	if env.proto == nil {
		env.proto = &flowv1beta1.Env{}
	}

	if env.resolver == nil {
		env.resolver = api.GlobalResolver()
	}

	if env.vars == nil {
		env.vars = func(context.Context, shared.Env) (cel.Activation, error) {
			return nil, fmt.Errorf("env vars not set")
		}
	}

	if env.conns == nil {
		env.conns = GetConnectorFunc(func(ident string) (shared.Connector, error) {
			return nil, fmt.Errorf("%s: env connectors not set", ident)
		})
	}

	celTypes, err := common.NewCELTypes(env.resolver)
	if err != nil {
		return nil, err
	}

	env.types = celTypes

	celEnv, err := common.NewCELEnv(
		cel.CustomTypeProvider(celTypes),
		cel.CustomTypeAdapter(celTypes),
		cel.Abbrevs(
			string(doneMsg.ProtoReflect().Descriptor().FullName()),
			string(eofMsg.ProtoReflect().Descriptor().FullName()),
			string(skipMsg.ProtoReflect().Descriptor().FullName()),
		),
		cel.VariableDecls(util.SliceMap(env.proto.GetNodes(), func(node *flowv1beta1.Env_Node) *decls.VariableDecl {
			return decls.NewVariable(node.GetIdent(), nodeVarsType)
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
					case *flowv1beta1.EOF:
						return celtypes.True
					}
					return celtypes.False
				},
			),
		),
	)
	if err != nil {
		return nil, err
	}

	env.Env = celEnv

	return env, nil
}

func (e *Env) Proto() *flowv1beta1.Env {
	e.mut.RLock()
	defer e.mut.RUnlock()

	return proto.CloneOf(e.proto)
}

func (e *Env) Extend(opts ...cel.EnvOption) (shared.Env, error) {
	return e.extend(opts...)
}

func (e *Env) Vars(ctx context.Context) (cel.Activation, error) {
	return e.vars(ctx, e)
}

func (e *Env) Resolver() shared.Resolver {
	return e.resolver
}

func (e *Env) Types() shared.Types {
	return e.types
}

func (e *Env) Connector(ident string) (shared.Connector, error) {
	return e.conns.GetConnector(ident)
}

func (e *Env) ParseExpr(id, expr string, visitor shared.ExprVisitor) (*cel.Ast, error) {
	expr, valid := shared.IsValidExpr(expr)
	if !valid {
		return nil, shared.NewExprError("%s: %s: %s", id, shared.InvalidExprErrPrefix, expr)
	}

	ast, iss := e.Env.Parse(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	if visitor != nil {
		visitor(ast)
	}

	return ast, nil
}

func (e *Env) CompileExpr(id string, ast *cel.Ast) (shared.Program, error) {
	ast, iss := e.Env.Check(ast)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	prog, err := e.Env.Program(ast, cel.InterruptCheckFrequency(1))
	if err != nil {
		return nil, err
	}

	return shared.ProgramFunc(func(ctx context.Context, env shared.Env) ref.Val {
		vars, err := env.Vars(ctx)
		if err != nil {
			return types.WrapErr(err)
		}

		val, _, err := prog.ContextEval(ctx, vars)
		if err != nil {
			return types.WrapErr(err)
		}

		return val
	}), nil
}

func (e *Env) extend(opts ...cel.EnvOption) (*Env, error) {
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
		Env: env,

		proto: e.proto,
		vars:  e.vars,

		conns:    e.conns,
		resolver: e.resolver,
		types:    types,
	}, nil
}

func (e *Env) applyOptions(opts ...Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
}
