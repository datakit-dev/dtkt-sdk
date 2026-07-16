package shared

import (
	"context"

	"cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

type (
	Env interface {
		Check(*cel.Ast) (*cel.Ast, *cel.Issues)
		Compile(expr string) (*cel.Ast, *cel.Issues)
		Connector(ident string) (Connector, error)
		Parse(expr string) (*cel.Ast, *cel.Issues)
		Program(*cel.Ast, ...cel.ProgramOption) (cel.Program, error)
		Resolver() Resolver
		Types() Types
		Vars(context.Context) (cel.Activation, error)
		Extend(opts ...cel.EnvOption) (Env, error)
	}
	Types interface {
		types.Adapter
		types.Provider
		ValueToNative(ref.Val) (any, error)
		ValueToExprValue(ref.Val) *expr.ExprValue
		ExprValueToNative(*expr.ExprValue) (any, error)
	}
)
