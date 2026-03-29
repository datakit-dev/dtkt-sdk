package spec

import (
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

type Node interface {
	shared.SpecNode
	*flowv1beta1.Action |
		*flowv1beta1.Connection |
		*flowv1beta1.Input |
		*flowv1beta1.Output |
		*flowv1beta1.Stream |
		*flowv1beta1.Var
}

func NewRuntimeNode(env shared.Env, node shared.SpecNode, visitor shared.ExprVisitFunc) (shared.RuntimeNode, error) {
	switch node := node.(type) {
	case *flowv1beta1.Connection:
		return NewConnection(env, node)
	case *flowv1beta1.Input:
		return NewInput(env, node)
	case *flowv1beta1.Var:
		return NewVar(env, node, visitor)
	case *flowv1beta1.Action:
		return NewAction(env, node, visitor)
	case *flowv1beta1.Stream:
		return NewStream(env, node, visitor)
	case *flowv1beta1.Output:
		return NewOutput(env, node, visitor)
	}
	return nil, nil
}

func GetID[T Node](node T) string {
	return fmt.Sprintf("%s.%s", GetIDPrefix(node), node.GetId())
}

func GetIDPrefix[T Node](node T) string {
	switch node.ProtoReflect().Interface().(type) {
	case *flowv1beta1.Connection:
		return shared.ConnectionPrefix
	case *flowv1beta1.Input:
		return shared.InputPrefix
	case *flowv1beta1.Var:
		return shared.VarPrefix
	case *flowv1beta1.Action:
		return shared.ActionPrefix
	case *flowv1beta1.Output:
		return shared.OutputPrefix
	case *flowv1beta1.Stream:
		return shared.StreamPrefix
	}
	return ""
}
