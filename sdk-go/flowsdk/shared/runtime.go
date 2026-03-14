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
		Resolver() Resolver
		Types() (Types, error)
		Env() (*cel.Env, error)
		Vars() (cel.Activation, error)
		Parse(ParseNodeFunc) error
		Compile() error
		Reset()
		GetNode(string) (Node, bool)
		GetNodeValue(string) (any, error)
		GetInputValue(string) (any, error)
		SetInputValues(map[string]any) error
		GetUserValues(string) (map[string]any, error)
		SetUserValues(string, map[string]any) error
		GetOutputValues() (map[string]any, error)
		RangeNodes(func(string, Node) bool)
	}
	Types interface {
		types.Adapter
		types.Provider
	}
	Node interface {
		GetRuntimeNode() EvalNode
		GetTypeNode() SpecNode
	}
)
