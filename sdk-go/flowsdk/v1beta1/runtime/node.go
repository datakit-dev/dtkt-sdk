package runtime

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"cel.dev/expr"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ shared.Node = (*Node)(nil)

type Node struct {
	proto *flowv1beta1.Node
	expr  shared.EvalExpr
	mut   sync.Mutex
}

func NewNode[T spec.Node](specNode T) *Node {
	return &Node{
		proto: setNodeType(&flowv1beta1.Node{
			Id: spec.GetID(specNode),
		}, specNode),
	}
}

func NewNodesFromMaps(nodeMaps ...map[string]*flowv1beta1.Node) (nodes []*Node) {
	for _, protos := range nodeMaps {
		nodes = append(nodes, util.SliceMap(slices.Collect(maps.Values(protos)), func(proto *flowv1beta1.Node) *Node {
			return &Node{
				proto: proto,
			}
		})...)
	}
	return
}

func NewNodeMap[T spec.Node](specNodes ...T) map[string]*flowv1beta1.Node {
	nodeMap := make(map[string]*flowv1beta1.Node)
	for _, specNode := range specNodes {
		nodeMap[specNode.GetId()] = setNodeType(&flowv1beta1.Node{
			Id: spec.GetID(specNode),
		}, specNode)
	}
	return nodeMap
}

func GetNodeID(node *flowv1beta1.Node) string {
	return node.GetId()
}

func setNodeType(evalNode *flowv1beta1.Node, specNode shared.SpecNode) *flowv1beta1.Node {
	switch typeNode := specNode.(type) {
	case *flowv1beta1.Connection:
		evalNode.Type = &flowv1beta1.Node_Connection{
			Connection: typeNode,
		}
	case *flowv1beta1.Input:
		evalNode.Type = &flowv1beta1.Node_Input{
			Input: typeNode,
		}
	case *flowv1beta1.Var:
		evalNode.Type = &flowv1beta1.Node_Var{
			Var: typeNode,
		}
	case *flowv1beta1.Action:
		evalNode.Type = &flowv1beta1.Node_Action{
			Action: typeNode,
		}
	case *flowv1beta1.Stream:
		evalNode.Type = &flowv1beta1.Node_Stream{
			Stream: typeNode,
		}
	case *flowv1beta1.Output:
		evalNode.Type = &flowv1beta1.Node_Output{
			Output: typeNode,
		}
	}
	return evalNode
}

func (n *Node) GetRuntimeNode() shared.EvalNode {
	return n.proto
}

func (n *Node) GetTypeNode() shared.SpecNode {
	switch n.proto.Type.(type) {
	case *flowv1beta1.Node_Connection:
		return n.proto.GetConnection()
	case *flowv1beta1.Node_Input:
		return n.proto.GetInput()
	case *flowv1beta1.Node_Var:
		return n.proto.GetVar()
	case *flowv1beta1.Node_Action:
		return n.proto.GetAction()
	case *flowv1beta1.Node_Stream:
		return n.proto.GetStream()
	case *flowv1beta1.Node_Output:
		return n.proto.GetOutput()
	}
	return nil
}

func (n *Node) Compile(run *Runtime) error {
	expr, err := spec.Compile(run, n.GetTypeNode())
	if err != nil {
		return err
	}
	n.expr = expr
	return nil
}

func (n *Node) Reset() {
	n.mut.Lock()
	defer n.mut.Unlock()

	n.proto.State = flowv1beta1.Node_STATE_UNSPECIFIED
	if n.proto.GetCurrValue() != nil {
		n.proto.PrevValue = n.proto.GetCurrValue()
		n.proto.CurrValue = nil
	}
}

func (n *Node) Eval(run *Runtime) ref.Val {
	if n.expr == nil {
		err := n.Compile(run)
		if err != nil {
			return types.WrapErr(err)
		}
	}
	return n.expr.Eval(run.Context())
}

func (n *Node) GetValue(run *Runtime) (any, error) {
	// Atomically check and transition state to prevent double-evaluation
	n.mut.Lock()

	// If already in SUCCESS or ERROR state, just return the cached value
	state := n.proto.GetState()
	if state == flowv1beta1.Node_STATE_SUCCESS || state == flowv1beta1.Node_STATE_ERROR {
		val, err := shared.ExprValueToNative(run, n.proto.GetCurrValue())
		n.mut.Unlock()

		if err != nil {
			return nil, err
		} else if val == nil {
			return nil, nil
		}

		switch v := val.(type) {
		case *flowv1beta1.Runtime_Done:
			if v != nil && v.GetId() == "" {
				v.Id = n.proto.GetId()
			}
		}

		return val, nil
	}

	// If PENDING, another goroutine is evaluating - this shouldn't happen
	// within the same group since nodes in a group don't depend on each other,
	// but we handle it defensively
	if state == flowv1beta1.Node_STATE_PENDING {
		n.mut.Unlock()
		return nil, fmt.Errorf("node %s is being evaluated by another goroutine", n.proto.GetId())
	}

	// State is UNSPECIFIED - transition to PENDING and evaluate
	n.proto.State = flowv1beta1.Node_STATE_PENDING
	n.mut.Unlock()

	// Evaluate WITHOUT holding the lock to prevent deadlocks
	// (evaluation may trigger lookups of other nodes)
	if n.expr == nil {
		if err := n.Compile(run); err != nil {
			return nil, n.setError(err)
		}
	}

	refVal := n.Eval(run)

	// Process the result
	if refVal == nil || refVal.Value() == nil || refVal == types.NullValue || refVal.Value() == structpb.NullValue_NULL_VALUE {
		n.setSuccess(&expr.Value{
			Kind: &expr.Value_NullValue{},
		})
	} else if err, ok := refVal.Value().(error); ok {
		return nil, n.setError(err)
	} else {
		exprVal, err := cel.ValueAsProto(refVal)
		if err != nil {
			return nil, n.setError(err)
		}
		n.setSuccess(exprVal)
	}

	// Now read the final value
	n.mut.Lock()
	val, err := shared.ExprValueToNative(run, n.proto.GetCurrValue())
	n.mut.Unlock()

	if err != nil {
		return nil, err
	} else if val == nil {
		return nil, nil
	}

	switch v := val.(type) {
	case *flowv1beta1.Runtime_Done:
		if v != nil && v.GetId() == "" {
			v.Id = n.proto.GetId()
		}
	}

	return val, nil
}

func (n *Node) setSuccess(value *expr.Value) {
	n.mut.Lock()
	defer n.mut.Unlock()

	n.proto.State = flowv1beta1.Node_STATE_SUCCESS
	n.proto.CurrValue = value
	n.proto.CallCount++
}

func (n *Node) setError(err error) error {
	if err != nil {
		n.mut.Lock()
		defer n.mut.Unlock()

		n.proto.State = flowv1beta1.Node_STATE_ERROR

		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("%s: evaluation timed out: %w", n.proto.GetId(), err)
		} else {
			err = fmt.Errorf("%s: %w", n.proto.GetId(), err)
		}

		wrapVal, wrapErr := anypb.New(&flowv1beta1.Runtime_Done{
			Id:      n.proto.GetId(),
			Reason:  err.Error(),
			IsError: true,
		})
		if wrapErr != nil {
			return errors.Join(err, wrapErr)
		}

		n.proto.CurrValue = &expr.Value{
			Kind: &expr.Value_ObjectValue{
				ObjectValue: wrapVal,
			},
		}
	}

	return err
}
