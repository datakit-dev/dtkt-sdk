package impl

import (
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

func NewNode(env shared.Env, node shared.SpecNode, visitor shared.NodeVisitor) (shared.ParsedNode, error) {
	switch node := node.(type) {
	case *flowv1beta1.Connection:
		return NewConnection(env, node), nil
	case *flowv1beta1.Input:
		return NewInput(env, node), nil
	case *flowv1beta1.Action:
		return NewAction(env, node, visitor)
	case *flowv1beta1.Stream:
		return NewStream(env, node, visitor)
	case *flowv1beta1.Output:
		return NewOutput(env, node, visitor)
	case *flowv1beta1.Var:
		return NewVar(env, node, visitor)
	}
	return nil, fmt.Errorf("unknown node type: %T: %s", node, node.GetId())
}
