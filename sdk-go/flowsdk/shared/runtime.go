package shared

import (
	"context"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
)

type (
	Runtime interface {
		Context() context.Context
		Connectors() ConnectorProvider
		Env() (Env, error)
		GetNode(string) (SpecNode, bool)
		GetValue(string) (any, error)
	}
	Env interface {
		Check(*cel.Ast) (*cel.Ast, *cel.Issues)
		Compile(string) (*cel.Ast, *cel.Issues)
		Parse(string) (*cel.Ast, *cel.Issues)
		Program(*cel.Ast, ...cel.ProgramOption) (cel.Program, error)
		Resolver() Resolver
		TypeAdapter() types.Adapter
		TypeProvider() types.Provider
		Vars() cel.Activation
	}
)
