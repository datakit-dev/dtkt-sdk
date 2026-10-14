package shared

import (
	"context"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"cel.dev/expr"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type (
	Env interface {
		Connector(ident string) (Connector, error)
		Resolver() Resolver
		Types() Types
		Vars() (cel.Activation, error)
		ParseExpr(id string, expr string, visitor ExprVisitor) (*cel.Ast, error)
		CompileExpr(id string, ast *cel.Ast) (Program, error)
	}
	Client interface {
		common.DynamicClient
		Address() network.Address
	}
	Connector interface {
		GetResolver(context.Context) (Resolver, error)
		GetClient(context.Context) (context.Context, Client, error)
	}
	Program interface {
		Eval(context.Context, Env) ref.Val
	}
	ProgramFunc func(context.Context, Env) ref.Val
	Resolver    interface {
		protoregistry.ExtensionTypeResolver
		protoregistry.MessageTypeResolver
		RangeServices(func(protoreflect.ServiceDescriptor) bool)
		RangeMethods(func(protoreflect.MethodDescriptor) bool)
		FindMethodByName(protoreflect.FullName) (protoreflect.MethodDescriptor, error)
		RangeFiles(func(protoreflect.FileDescriptor) bool)
		GetValidator() (api.Validator, error)
	}
	Types interface {
		types.Adapter
		types.Provider
		ValueToNative(ref.Val) (any, error)
		ValueToExprValue(ref.Val) *expr.ExprValue
		ExprValueToNative(*expr.ExprValue) (any, error)
	}
)

func (f ProgramFunc) Eval(ctx context.Context, env Env) ref.Val {
	return f(ctx, env)
}
