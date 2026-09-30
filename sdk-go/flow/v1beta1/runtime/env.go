package runtime

import (
	"context"
	"fmt"
	"reflect"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/decls"
	"cel.dev/cel-go/common/types"
	celtypes "cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/proto"
)

var _ shared.Env = (*Env)(nil)

var (
	varsCELType = cel.MapType(cel.StringType, cel.DynType)
	varsMapType = reflect.TypeFor[map[string]any]()
)

type (
	Env struct {
		*cel.Env

		proto *flowv1beta1.Env
		vars  Vars

		conns    ConnectorProvider
		resolver shared.Resolver
		types    *common.CELTypes
	}
	Vars func(shared.Env) (cel.Activation, error)
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
		env.vars = func(shared.Env) (cel.Activation, error) {
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
			string(doneProto.ProtoReflect().Descriptor().FullName()),
			string(skipProto.ProtoReflect().Descriptor().FullName()),
		),
		cel.VariableDecls(util.SliceMap(env.proto.GetNodes(), func(node *flowv1beta1.Env_Node) *decls.VariableDecl {
			return decls.NewVariable(node.GetIdent(), varsCELType)
		})...),
		cel.Function("isEOF",
			cel.Overload("isEOF_dyn", []*cel.Type{cel.DynType}, cel.BoolType),
			cel.SingletonFunctionBinding(
				func(args ...ref.Val) ref.Val {
					if done, ok := IsDone(args[0].Value()); ok && done.IsEOF() {
						return celtypes.True
					} else if vars, err := args[0].ConvertToNative(varsMapType); err == nil {
						if vars, ok := vars.(map[string]any); ok {
							if _, ok := vars["value"].(*flowv1beta1.Done); ok {
								return celtypes.True
							}
						}
					}
					return celtypes.False
				},
			),
		),
		cel.Function("isSkip",
			cel.Overload("isSkip_dyn", []*cel.Type{cel.DynType}, cel.BoolType),
			cel.SingletonFunctionBinding(
				func(args ...ref.Val) ref.Val {
					if _, ok := IsSkip(args[0].Value()); ok {
						return celtypes.True
					} else if vars, err := args[0].ConvertToNative(varsMapType); err == nil {
						if vars, ok := vars.(map[string]any); ok {
							if _, ok := vars["value"].(*flowv1beta1.Skip); ok {
								return celtypes.True
							}
						}
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
	return proto.CloneOf(e.proto)
}

func (e *Env) Extend(opts ...cel.EnvOption) (shared.Env, error) {
	return e.extend(opts...)
}

func (e *Env) Vars() (cel.Activation, error) {
	return e.vars(e)
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
		vars, err := env.Vars()
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
