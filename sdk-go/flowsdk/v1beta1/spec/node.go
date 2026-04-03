package spec

import (
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
)

func ParseNode(env shared.Env, node shared.SpecNode, visitor shared.NodeVisitFunc) (shared.ExecNode, error) {
	exprVisitor := func(ast *cel.Ast) {
		visitor(GetID(node), ast)
	}

	switch node := node.(type) {
	case *flowv1beta1.Connection:
		return NewConnection(env, node), nil
	case *flowv1beta1.Input:
		return NewInput(env, node), nil
	case *flowv1beta1.Var:
		return NewVar(env, node, exprVisitor)
	case *flowv1beta1.Action:
		return NewAction(env, node, exprVisitor)
	case *flowv1beta1.Stream:
		return NewStream(env, node, exprVisitor)
	case *flowv1beta1.Output:
		return NewOutput(env, node, exprVisitor)
	}
	return nil, nil
}

func GetID(node shared.SpecNode) string {
	return fmt.Sprintf("%s.%s", GetIDPrefix(node), node.GetId())
}

func GetIDPrefix(node shared.SpecNode) string {
	switch node.(type) {
	case *flowv1beta1.Action:
		return shared.ActionPrefix
	case *flowv1beta1.Connection:
		return shared.ConnectionPrefix
	case *flowv1beta1.Input:
		return shared.InputPrefix
	case *flowv1beta1.Output:
		return shared.OutputPrefix
	case *flowv1beta1.Stream:
		return shared.StreamPrefix
	case *flowv1beta1.Var:
		return shared.VarPrefix
	}
	return ""
}
