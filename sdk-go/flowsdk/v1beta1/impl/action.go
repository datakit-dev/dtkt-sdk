package impl

import (
	"errors"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

var _ shared.ParsedNode = (*Action)(nil)

type Action struct {
	shared.ParsedNode
	node *flowv1beta1.Action
}

func NewAction(env shared.Env, node *flowv1beta1.Action, visitor shared.NodeVisitor) (*Action, error) {
	action := &Action{
		node: node,
	}

	switch {
	case node.GetCall() != nil:
		caller, err := NewActionCaller(env, node, visitor)
		if err != nil {
			return nil, err
		}

		action.ParsedNode = caller

	// case node.GetUser() != nil:
	// err := ParseUserAction(env, node.GetUser(), visitor)
	// if err != nil {
	// 	return nil, err
	// }
	default:
		return nil, errors.New("call or user required")
	}

	return action, nil
}
func (a *Action) Caller() (*ActionCaller, bool) {
	if a.ParsedNode == nil {
		return nil, false
	} else if caller, ok := a.ParsedNode.(*ActionCaller); !ok {
		return nil, false
	} else {
		return caller, true
	}
}
