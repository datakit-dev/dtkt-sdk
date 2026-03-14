package spec

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

func Parse(ctx shared.Runtime, node shared.SpecNode, visitor shared.ParseExprFunc) error {
	switch node := node.(type) {
	case *flowv1beta1.Var:
		return ParseVar(ctx, node, visitor)
	case *flowv1beta1.Action:
		return ParseAction(ctx, node, visitor)
	case *flowv1beta1.Stream:
		return ParseStream(ctx, node, visitor)
	case *flowv1beta1.Output:
		return ParseOutput(ctx, node, visitor)
	}
	return nil
}

func Compile(ctx shared.Runtime, node shared.SpecNode) (shared.EvalExpr, error) {
	switch node := node.(type) {
	case *flowv1beta1.Connection:
		return CompileConnection(ctx, node)
	case *flowv1beta1.Input:
		return CompileInput(ctx, node)
	case *flowv1beta1.Var:
		return CompileVar(ctx, node)
	case *flowv1beta1.Action:
		return CompileAction(ctx, node)
	case *flowv1beta1.Stream:
		return CompileStream(ctx, node)
	case *flowv1beta1.Output:
		return CompileOutput(ctx, node)
	}
	return nil, nil
}
