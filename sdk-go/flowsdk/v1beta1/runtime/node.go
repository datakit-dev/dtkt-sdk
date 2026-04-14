package runtime

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"sync"

	"cel.dev/expr"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	Node struct {
		id    string
		proto *flowv1beta1.Node

		parse   func() (shared.ExecNode, error)
		compile func() error

		hasRecv,
		hasSend,
		hasEval bool
		isRequired func() bool
		hasCached  func() (ref.Val, bool)

		recv shared.RecvFunc
		send shared.SendFunc
		eval shared.EvalFunc

		recvCh chan any
		sendCh chan ref.Val
		// signals that Reset() has run; gates executor's ack goroutine to one event
		// per cycle
		resetCh chan struct{}

		// Cached prev, curr, and err to avoid duplicate type conversions
		prev,
		curr any
		err error

		mut sync.Mutex
	}
	NodeMap map[string]*Node
)

func NewNode(proto *flowv1beta1.Node) *Node {
	return &Node{
		id:      proto.Id,
		proto:   proto,
		resetCh: make(chan struct{}, 1),
	}
}

func NewNodeFromSpec(specNode shared.SpecNode) *flowv1beta1.Node {
	node := new(flowv1beta1.Node{
		Id: spec.GetID(specNode),
	})
	switch specNode := specNode.(type) {
	case *flowv1beta1.Connection:
		node.Type = &flowv1beta1.Node_Connection{
			Connection: specNode,
		}
	case *flowv1beta1.Input:
		node.Type = &flowv1beta1.Node_Input{
			Input: specNode,
		}
	case *flowv1beta1.Var:
		node.Type = &flowv1beta1.Node_Var{
			Var: specNode,
		}
	case *flowv1beta1.Action:
		node.Type = &flowv1beta1.Node_Action{
			Action: specNode,
		}
	case *flowv1beta1.Stream:
		node.Type = &flowv1beta1.Node_Stream{
			Stream: specNode,
		}
	case *flowv1beta1.Output:
		node.Type = &flowv1beta1.Node_Output{
			Output: specNode,
		}
	}
	return node
}

func RuntimeNodeMap(protoMaps ...map[string]*flowv1beta1.Node) (nodeMap NodeMap) {
	nodeMap = make(NodeMap)
	for _, protoMap := range protoMaps {
		for _, proto := range protoMap {
			nodeMap[spec.GetID(GetSpecNode(proto))] = NewNode(proto)
		}
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

func (m NodeMap) asMap() map[string]any {
	root := map[string]any{}
	for id, node := range m {
		prefix, id, ok := shared.ParseNodePrefixAndID(id)
		if ok {
			nodes, ok := root[prefix].(map[string]any)
			if !ok {
				nodes = map[string]any{}
				root[prefix] = nodes
			}

			nodes[id] = node.asMap()
		}
	}
	return root
}

func (n *Node) asMap() map[string]any {
	n.mut.Lock()
	defer n.mut.Unlock()
	return map[string]any{
		"count":       n.proto.GetCount(),
		"prev":        n.prev,
		"value":       n.curr,
		"error":       n.err,
		"start_time":  n.proto.StartTime,
		"finish_time": n.proto.FinishTime,
	}
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

	n.isRequired = func() bool {
		if input, ok := node.(*spec.Input); ok {
			_, hasCached := input.HasCached()
			return input.IsRequired() && !hasCached
		}
		return false
	}

	n.hasCached = node.HasCached

	n.eval, n.hasEval = node.Eval()

	n.recv, n.hasRecv = node.Recv()
	if n.hasRecv {
		n.recvCh, err = run.GetRecvCh(n.id)
		if err != nil {
			return err
		}
	}

	n.send, n.hasSend = node.Send()
	if n.hasSend {
		n.sendCh, err = run.GetSendCh(n.id)
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
	return types.WrapErr(errors.New("unexpected call to eval: not supported"))
}

func (n *Node) Completed() bool {
	n.mut.Lock()
	defer n.mut.Unlock()

	return n.proto.State > flowv1beta1.Node_STATE_PENDING
}

func (n *Node) GetValue(run *Runtime) (any, error) {
	env, err := run.env()
	if err != nil {
		return nil, err
	}

	// Atomically check and transition state to prevent double-evaluation
	n.mut.Lock()

	switch n.proto.GetState() {
	case flowv1beta1.Node_STATE_SUCCESS:
		n.mut.Unlock()
		return n.curr, nil
	case flowv1beta1.Node_STATE_ERROR:
		if n.err == nil {
			n.mut.Unlock()
			return nil, fmt.Errorf("unknown error")
		}

		n.mut.Unlock()
		return nil, n.err
	case flowv1beta1.Node_STATE_PENDING:
		n.mut.Unlock()
		// If PENDING, another goroutine is evaluating - this shouldn't happen
		// within the same group since nodes in a group don't depend on each other,
		// but we handle it defensively
		return nil, fmt.Errorf("node %s is being evaluated by another goroutine", n.proto.GetId())
	}

	// State is UNSPECIFIED - transition to PENDING and evaluate
	n.proto.State = flowv1beta1.Node_STATE_PENDING
	n.proto.StartTime = timestamppb.Now()
	n.mut.Unlock()

	return n.applyValue(env, n.Eval(run))
}

// applyValue is the single path that serialises a ref.Val into node state and
// caches the native Go value. Both the GetValue UNSPECIFIED path and the ack
// goroutine (via valueCh → eval) converge here, ensuring state mutations are
// guarded by the same invariants as GetValue.
func (n *Node) applyValue(env *Env, refVal ref.Val) (any, error) {
	n.mut.Lock()
	defer n.mut.Unlock()

	if err, ok := refVal.Value().(error); ok {
		return nil, n.setError(err)
	}

	exprVal, err := cel.ValueAsProto(refVal)
	if err != nil {
		return nil, n.setError(err)
	}

	nativeVal, err := shared.ExprValueToNative(env, exprVal)
	if err != nil {
		return nil, n.setError(err)
	}

	switch v := nativeVal.(type) {
	case *flowv1beta1.Runtime_Done:
		if v != nil && v.Id == "" {
			v.Id = n.proto.Id
		}
	}

	n.curr = nativeVal
	n.setValue(exprVal)

	return nativeVal, nil
}

func (n *Node) Reset() {
	n.mut.Lock()
	defer n.mut.Unlock()

	n.proto.State = flowv1beta1.Node_STATE_UNSPECIFIED
	if n.proto.GetValue() != nil {
		n.proto.PrevValue = n.proto.GetValue()
		n.proto.Result = nil
	}

	n.proto.StartTime = nil
	n.proto.FinishTime = nil

	n.prev = n.curr
	n.curr = nil
	n.err = nil

	// Unblock the ack goroutine so it can accept the next value from sendCh.
	select {
	case n.resetCh <- struct{}{}:
	default:
	}
}

func (n *Node) setValue(exprVal *expr.Value) {
	n.proto.State = flowv1beta1.Node_STATE_SUCCESS
	n.proto.Result = &flowv1beta1.Node_Value{
		Value: exprVal,
	}
	n.proto.FinishTime = timestamppb.Now()
	n.proto.Count++
}

func (n *Node) setError(err error) error {
	statusErr, ok := status.FromError(err)
	if !ok {
		doneErr, ok := IsDoneError(err)
		if ok && doneErr.proto.GetIsError() {
			statusErr = status.New(codes.Aborted, doneErr.Error())
		} else if errors.Is(err, io.EOF) {
			statusErr = status.New(codes.OK, err.Error())
		} else {
			ctxErr := status.FromContextError(err)
			statusErr = status.Newf(ctxErr.Code(), "%s: %s", n.proto.GetId(), ctxErr.String())
		}
	}

	switch statusErr.Code() {
	case codes.OK:
		eof, _ := anypb.New(&flowv1beta1.Runtime_EOF{})
		n.setValue(&expr.Value{
			Kind: &expr.Value_ObjectValue{
				ObjectValue: eof,
			},
		})
		return nil
	}

	n.err = statusErr.Err()
	n.proto.State = flowv1beta1.Node_STATE_ERROR
	n.proto.Result = &flowv1beta1.Node_Error{
		Error: statusErr.Proto(),
	}
	n.proto.FinishTime = timestamppb.Now()

	return n.err
}
