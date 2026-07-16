package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"sync"

	"cel.dev/expr"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ shared.Node = (*Node)(nil)

type (
	Node struct {
		id    string
		proto *flowv1beta1.Node

		spec shared.SpecNode
		exec shared.ExecNode

		prevValue,
		currValue any

		pipe *Pipe
		send,
		recv chan any

		err error
		mut sync.Mutex
	}
	NodeMap map[string]*Node
)

func NewNode(proto *flowv1beta1.Node) *Node {
	in := make(chan any)
	out := make(chan any)

	return &Node{
		id:    proto.Id,
		proto: proto,
		spec:  specNodeFromNode(proto),

		pipe: NewPipe(in, out),
		send: out,
		recv: in,
	}
}

func NewNodeMap(protoMaps ...map[string]*flowv1beta1.Node) (nodeMap NodeMap) {
	nodeMap = make(NodeMap)
	for _, protoMap := range protoMaps {
		for _, proto := range protoMap {
			node := NewNode(proto)
			nodeMap[node.id] = node
		}
	}
	return
}

func specNodesMap[T shared.SpecNode](specNodes []T) (nodeMap map[string]*flowv1beta1.Node) {
	nodeMap = make(map[string]*flowv1beta1.Node)
	for _, specNode := range specNodes {
		nodeMap[specNode.GetId()] = nodeFromSpecNode(specNode)
	}
	return
}

func specNodeFromNode(node *flowv1beta1.Node) shared.SpecNode {
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

func nodeFromSpecNode(specNode shared.SpecNode) *flowv1beta1.Node {
	proto := new(flowv1beta1.Node{
		Id: spec.GetNodeIdent(specNode),
	})
	switch specNode := specNode.(type) {
	case *flowv1beta1.Connection:
		proto.Type = &flowv1beta1.Node_Connection{
			Connection: specNode,
		}
	case *flowv1beta1.Input:
		proto.Type = &flowv1beta1.Node_Input{
			Input: specNode,
		}
	case *flowv1beta1.Var:
		proto.Type = &flowv1beta1.Node_Var{
			Var: specNode,
		}
	case *flowv1beta1.Action:
		proto.Type = &flowv1beta1.Node_Action{
			Action: specNode,
		}
	case *flowv1beta1.Stream:
		proto.Type = &flowv1beta1.Node_Stream{
			Stream: specNode,
		}
	case *flowv1beta1.Output:
		proto.Type = &flowv1beta1.Node_Output{
			Output: specNode,
		}
	}
	return proto
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

func (m NodeMap) Vars(types *common.CELTypes) map[string]any {
	return util.MapToMap(m, func(id string, node *Node) any {
		return func() ref.Val {
			return node.Vars(types)
		}
	})
}

func (n *Node) Vars(types shared.Types) ref.Val {
	n.mut.Lock()
	defer n.mut.Unlock()

	return types.NativeToValue(map[string]any{
		"value":       n.currValue,
		"prev":        n.prevValue,
		"count":       n.proto.GetCount(),
		"start_time":  n.proto.GetStartTime(),
		"finish_time": n.proto.GetFinishTime(),
	})
}

func (n *Node) Proto() *flowv1beta1.Node {
	n.mut.Lock()
	defer n.mut.Unlock()

	return proto.CloneOf(n.proto)
}

func (n *Node) Ident() string {
	return n.id
}

func (n *Node) Spec() shared.SpecNode {
	return n.spec
}

func (n *Node) Exec() shared.ExecNode {
	return n.exec
}

func (n *Node) Pipe() shared.Pipe {
	return n.pipe
}

func (n *Node) Send() chan<- any {
	return n.send
}

func (n *Node) Recv() <-chan any {
	return n.recv
}

func (n *Node) Parse(env shared.Env, visitor shared.NodeVisitor) (err error) {
	if n.exec != nil {
		return
	}

	env, err = n.withSelf(env)
	if err != nil {
		return err
	}

	exec, err := spec.ParseNode(env, n.spec, visitor)
	if err != nil {
		return err
	}

	n.exec = exec

	return nil
}

func (n *Node) Compile(ctx context.Context, env shared.Env) error {
	if n.exec == nil {
		return fmt.Errorf("node %q must be parsed before compilation", n.id)
	}

	env, err := n.withSelf(env)
	if err != nil {
		return err
	}

	return n.exec.Compile(ctx, env)
}

// withSelf returns env extended with a "self" variable bound to this node's
// own state (value, prev, count, start_time, finish_time), if this node's
// type supports self-reference. This enables rolling accumulator patterns
// (Reduce/Fold) on Var nodes, e.g.:
//
//	value: = self.prev == null ? streams.tick.value : self.prev + streams.tick.value
//
// "self" is deliberately not a real node ident (it doesn't match IsNodeID),
// so referencing it does not introduce a graph self-edge or cycle.
//
// Only Var nodes support this today: other node types either have no
// meaningful prior state (Connection, Input) or already expose their
// dependency data through normal node idents.
func (n *Node) withSelf(env shared.Env) (shared.Env, error) {
	if _, ok := n.spec.(*flowv1beta1.Var); !ok {
		return env, nil
	}

	extended, err := env.Extend(cel.Variable("self", shared.NodeVarsType))
	if err != nil {
		return nil, err
	}

	return &selfEnv{Env: extended, node: n}, nil
}

// selfEnv overlays a "self" binding onto an underlying env's activation,
// resolving to the owning node's own Vars() snapshot on every evaluation.
type selfEnv struct {
	shared.Env
	node *Node
}

func (e *selfEnv) Vars(ctx context.Context) (cel.Activation, error) {
	base, err := e.Env.Vars(ctx)
	if err != nil {
		return nil, err
	}

	self, err := cel.NewActivation(map[string]any{
		"self": func() ref.Val {
			return e.node.Vars(e.Env.Types())
		},
	})
	if err != nil {
		return nil, err
	}

	return interpreter.NewHierarchicalActivation(base, self), nil
}

func (n *Node) Value() any {
	n.mut.Lock()
	defer n.mut.Unlock()

	return n.currValue
}

func (m NodeMap) reset() {
	m.Range(func(_ string, n *Node) bool {
		n.reset()
		return true
	})
}

func (n *Node) reset() {
	n.mut.Lock()
	defer n.mut.Unlock()

	n.proto.State = flowv1beta1.Node_STATE_UNSPECIFIED
	if n.proto.GetValue() != nil {
		n.proto.PrevValue = n.proto.GetValue()
	}

	n.prevValue = n.currValue
	n.currValue = nil
	n.proto.Result = nil
	n.proto.StartTime = nil
	n.proto.FinishTime = nil
}

func (n *Node) setPending() {
	n.mut.Lock()
	defer n.mut.Unlock()

	n.proto.State = flowv1beta1.Node_STATE_PENDING
	n.proto.StartTime = timestamppb.Now()
}

func (n *Node) setValue(ctx context.Context, env shared.Env, value any) (_ any, err error) {
	n.mut.Lock()
	defer n.mut.Unlock()

	if value == nil {
		value = types.NullValue
	}

	if future, ok := value.(shared.Future); ok {
		value, ok, err = future.Value(ctx)
		if err != nil {
			value = types.WrapErr(err)
		} else if !ok {
			value = types.NullValue
		}
	}

	var exprVal *expr.ExprValue
	switch v := value.(type) {
	case *types.Err:
		_, err = v.ConvertToNative(nil)
		return nil, n.setError(err)
	case error:
		return nil, n.setError(v)
	case ref.Val:
		exprVal = env.Types().ValueToExprValue(v)
	default:
		exprVal = env.Types().ValueToExprValue(env.Types().NativeToValue(v))
	}

	value, err = env.Types().ExprValueToNative(exprVal)
	if err != nil {
		return nil, n.setError(err)
	}

	switch v := value.(type) {
	case *flowv1beta1.Runtime_Done:
		if v != nil && v.Id == "" {
			v.Id = n.proto.Id
		}
	}

	proto, err := common.ProtoOptions{
		Resolver: env.Resolver(),
	}.WrapAny(value)
	if err != nil {
		return nil, n.setError(err)
	}

	n.currValue = value
	n.setSuccess(proto)

	return value, nil
}

func (n *Node) setSuccess(value *anypb.Any) {
	n.proto.State = flowv1beta1.Node_STATE_SUCCESS
	n.proto.Result = &flowv1beta1.Node_Value{
		Value: value,
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
		} else {
			statusErr = status.Newf(codes.Canceled, "%s: %s", n.proto.GetId(), err.Error())
		}
	}

	if errors.Is(err, io.EOF) {
		statusErr = status.New(codes.OK, err.Error())
	}

	switch statusErr.Code() {
	case codes.OK:
		eof, _ := anypb.New(&flowv1beta1.Runtime_EOF{})
		n.setSuccess(eof)
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
