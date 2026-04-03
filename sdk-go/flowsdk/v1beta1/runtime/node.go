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
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	Node struct {
		id    string
		proto *flowv1beta1.Node

		parse   func() (shared.ExecNode, error)
		compile func() error

		isRequiredInput,
		hasRecv,
		hasSend,
		hasEval bool

		recvCh chan any
		sendCh chan ref.Val
		recv   shared.RecvFunc
		send   shared.SendFunc
		eval   shared.EvalFunc

		valueCh chan ref.Val
		resetCh chan struct{} // signals that Reset() has run; gates the ack goroutine to one event per cycle
		value   any
		err     error

		mut sync.Mutex
	}
	NodeMap map[string]*Node
)

func NewNode(proto *flowv1beta1.Node) *Node {
	return &Node{
		id:      proto.Id,
		proto:   proto,
		valueCh: make(chan ref.Val, 1),
		resetCh: make(chan struct{}, 1),
	}
}

func NewNodeFromSpec(specNode shared.SpecNode) *flowv1beta1.Node {
	node := new(flowv1beta1.Node)
	switch specNode := specNode.(type) {
	case *flowv1beta1.Connection:
		node.Id = spec.GetID(specNode)
		node.Type = &flowv1beta1.Node_Connection{
			Connection: specNode,
		}
	case *flowv1beta1.Input:
		node.Id = spec.GetID(specNode)
		node.Type = &flowv1beta1.Node_Input{
			Input: specNode,
		}
	case *flowv1beta1.Var:
		node.Id = spec.GetID(specNode)
		node.Type = &flowv1beta1.Node_Var{
			Var: specNode,
		}
	case *flowv1beta1.Action:
		node.Id = spec.GetID(specNode)
		node.Type = &flowv1beta1.Node_Action{
			Action: specNode,
		}
	case *flowv1beta1.Stream:
		node.Id = spec.GetID(specNode)
		node.Type = &flowv1beta1.Node_Stream{
			Stream: specNode,
		}
	case *flowv1beta1.Output:
		node.Id = spec.GetID(specNode)
		node.Type = &flowv1beta1.Node_Output{
			Output: specNode,
		}
	}
	return node
}

func RuntimeNodeMap(protoMaps ...map[string]*flowv1beta1.Node) (nodeMap NodeMap) {
	nodeMap = make(NodeMap)
	for _, proto := range util.JoinMaps(protoMaps...) {
		nodeMap[spec.GetID(GetSpecNode(proto))] = NewNode(proto)
	}
	return
}

func SpecNodeMap[T shared.SpecNode](specNodes []T) (nodeMap map[string]*flowv1beta1.Node) {
	nodeMap = make(map[string]*flowv1beta1.Node)
	for _, specNode := range specNodes {
		nodeMap[specNode.GetId()] = NewNodeFromSpec(specNode)
	}
	return
}

func (m NodeMap) Protos() (protos []*flowv1beta1.Node) {
	protos = make([]*flowv1beta1.Node, len(m))
	for idx, node := range m.Values() {
		protos[idx] = node.proto
	}
	return
}

func (m NodeMap) Load(id string) (node *Node, ok bool) {
	node, ok = m[id]
	return
}

func (m NodeMap) Values() []*Node {
	return slices.Collect(maps.Values(m))
}

func (m NodeMap) Range(f func(string, *Node) bool) {
	for id, node := range m {
		if !f(id, node) {
			return
		}
	}
}

func (m NodeMap) Parse(env *Env, visitor shared.NodeVisitFunc) (err error) {
	for _, node := range m {
		_, err = node.Parse(env, visitor)
		if err != nil {
			return
		}
	}
	return
}

func (m NodeMap) Compile(run *Runtime, visitor shared.NodeVisitFunc) (err error) {
	for _, node := range m {
		err = node.Compile(run, visitor)
		if err != nil {
			return
		}
	}
	return
}

func GetNodeID(node *flowv1beta1.Node) string {
	return node.GetId()
}

func GetSpecNode(node *flowv1beta1.Node) shared.SpecNode {
	switch node.Type.(type) {
	case *flowv1beta1.Node_Action:
		return node.GetAction()
	case *flowv1beta1.Node_Connection:
		return node.GetConnection()
	case *flowv1beta1.Node_Input:
		return node.GetInput()
	case *flowv1beta1.Node_Output:
		return node.GetOutput()
	case *flowv1beta1.Node_Stream:
		return node.GetStream()
	case *flowv1beta1.Node_Var:
		return node.GetVar()
	}
	return nil
}

func (n *Node) ID() string {
	return n.id
}

func (n *Node) Parse(env *Env, visitor shared.NodeVisitFunc) (shared.ExecNode, error) {
	n.mut.Lock()
	if n.parse == nil {
		n.parse = sync.OnceValues(func() (shared.ExecNode, error) {
			return spec.ParseNode(env, GetSpecNode(n.proto), visitor)
		})
	}
	n.mut.Unlock()

	return n.parse()
}

func (n *Node) Compile(run *Runtime, visitor shared.NodeVisitFunc) error {
	env, err := run.env()
	if err != nil {
		return err
	}

	node, err := n.Parse(env, visitor)
	if err != nil {
		return err
	}

	n.mut.Lock()
	if n.compile == nil {
		n.compile = sync.OnceValue(func() error {
			return node.Compile(run)
		})
	}
	n.mut.Unlock()

	err = n.compile()
	if err != nil {
		return err
	}

	n.mut.Lock()
	defer n.mut.Unlock()

	if input, ok := node.(*spec.Input); ok && input.IsRequired() {
		n.isRequiredInput = input.IsRequired()
	}

	n.eval, n.hasEval = node.Eval()

	n.recv, n.hasRecv = node.Recv()
	if n.hasRecv {
		n.recvCh, err = run.getRecvCh(n.id)
		if err != nil {
			return err
		}
	}

	n.send, n.hasSend = node.Send()
	if n.hasSend {
		n.sendCh, err = run.getSendCh(n.id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) Eval(run *Runtime) ref.Val {
	if n.hasEval {
		return n.eval(run)
	}

	select {
	case <-run.Context().Done():
		return types.WrapErr(context.Cause(run.Context()))
	case value, ok := <-n.valueCh:
		if !ok {
			return types.WrapErr(fmt.Errorf("%s: event channel closed", n.id))
		}
		return value
	}
}

func (n *Node) Completed() bool {
	n.mut.Lock()
	defer n.mut.Unlock()

	return n.proto.State > flowv1beta1.Node_STATE_PENDING
}

func (n *Node) GetValue(run *Runtime) (any, error) {
	// Atomically check and transition state to prevent double-evaluation
	n.mut.Lock()

	// If already in SUCCESS or ERROR state, just return the cached value
	state := n.proto.GetState()
	if state == flowv1beta1.Node_STATE_SUCCESS || state == flowv1beta1.Node_STATE_ERROR {
		if n.value != nil {
			n.mut.Unlock()
			return n.value, nil
		}

		env, err := run.env()
		if err != nil {
			n.mut.Unlock()
			return nil, err
		}

		val, err := shared.ExprValueToNative(env, n.proto.GetCurrValue())
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
	n.proto.StartTime = timestamppb.Now()
	n.mut.Unlock()

	return n.applyValue(run, n.Eval(run))
}

// applyValue is the single path that serialises a ref.Val into node state and
// caches the native Go value. Both the GetValue UNSPECIFIED path and the ack
// goroutine (via valueCh → eval) converge here, ensuring state mutations are
// guarded by the same invariants as GetValue.
func (n *Node) applyValue(run *Runtime, refVal ref.Val) (any, error) {
	if refVal == nil || refVal.Value() == nil || refVal == types.NullValue || refVal.Value() == structpb.NullValue_NULL_VALUE {
		n.setSuccess(&expr.Value{Kind: &expr.Value_NullValue{}})
		return nil, nil
	}

	if err, ok := refVal.Value().(error); ok {
		return nil, n.setError(err)
	}

	exprVal, err := cel.ValueAsProto(refVal)
	if err != nil {
		return nil, n.setError(err)
	}
	n.setSuccess(exprVal)

	env, err := run.env()
	if err != nil {
		return nil, n.setError(err)
	}

	n.mut.Lock()
	val, err := shared.ExprValueToNative(env, n.proto.GetCurrValue())
	if err != nil {
		n.mut.Unlock()
		return nil, err
	}

	switch v := val.(type) {
	case *flowv1beta1.Runtime_Done:
		if v != nil && v.GetId() == "" {
			v.Id = n.proto.GetId()
		}
	}

	n.value = val
	n.mut.Unlock()

	return val, nil
}

func (n *Node) Reset() {
	n.mut.Lock()
	defer n.mut.Unlock()

	// For required inputs, preserve any value already buffered in n.valueCh.
	// That value was placed there by the ack goroutine during the previous cycle
	// and is earmarked for the next cycle's evalReq. Draining it would lose the
	// user's injection. For all other nodes the buffer is stale and must be cleared.
	if !n.isRequiredInput {
		select {
		case <-n.valueCh:
		default:
		}
	}

	n.proto.State = flowv1beta1.Node_STATE_UNSPECIFIED
	if n.proto.GetCurrValue() != nil {
		n.proto.PrevValue = n.proto.GetCurrValue()
		n.proto.CurrValue = nil
	}

	n.proto.StartTime = nil
	n.proto.FinishTime = nil

	n.value = nil
	n.err = nil

	// Unblock the ack goroutine so it can accept the next value from sendCh.
	select {
	case n.resetCh <- struct{}{}:
	default:
	}
}

func (n *Node) setSuccess(value *expr.Value) {
	n.mut.Lock()
	defer n.mut.Unlock()

	n.proto.State = flowv1beta1.Node_STATE_SUCCESS
	n.proto.CurrValue = value
	n.proto.CallCount++
	n.proto.FinishTime = timestamppb.Now()
}

func (n *Node) setError(err error) error {
	n.mut.Lock()
	defer n.mut.Unlock()

	n.proto.State = flowv1beta1.Node_STATE_ERROR
	n.proto.FinishTime = timestamppb.Now()

	if errors.Is(err, context.DeadlineExceeded) {
		n.err = fmt.Errorf("%s: evaluation timed out: %w", n.proto.GetId(), err)
	} else {
		n.err = fmt.Errorf("%s: %w", n.proto.GetId(), err)
	}

	value, _ := anypb.New(&flowv1beta1.Runtime_Done{
		Id:      n.proto.GetId(),
		Reason:  err.Error(),
		IsError: true,
	})
	n.proto.CurrValue = &expr.Value{
		Kind: &expr.Value_ObjectValue{
			ObjectValue: value,
		},
	}

	return n.err
}
