package spec

import (
	"errors"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

var _ shared.ExecNode = (*Action)(nil)

type Action struct {
	shared.ExecNode
	node *flowv1beta1.Action
}

func NewAction(env shared.Env, node *flowv1beta1.Action, visitor shared.NodeVisitor) (*Action, error) {
	switch {
	case node.GetCall() != nil:
		caller, err := NewActionCaller(env, node, visitor)
		if err != nil {
			return nil, err
		}

		return &Action{
			ExecNode: caller,
			node:     node,
		}, nil
	case node.GetUser() != nil:
		// err := ParseUserAction(env, node.GetUser(), visitor)
		// if err != nil {
		// 	return nil, err
		// }
	}

	return nil, errors.New("call or user required")
}
