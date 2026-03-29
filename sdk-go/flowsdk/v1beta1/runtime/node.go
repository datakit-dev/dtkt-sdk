package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"cel.dev/expr"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Node struct {
	shared.RuntimeNode

	id      string
	proto   *flowv1beta1.Node
	compile func() error
	start   func() error

	valueCh chan ref.Val
	resetCh chan struct{} // signals that Reset() has run; gates the ack goroutine to one event per cycle
	value   any
	err     error

	mut sync.Mutex
}

func NewNode(proto *flowv1beta1.Node) *Node {
	return &Node{
		id:      GetNodeID(proto),
		proto:   proto,
		valueCh: make(chan ref.Val, 1),
		resetCh: make(chan struct{}, 1),
	}
}

func NewNodes(nodeMaps ...map[string]*flowv1beta1.Node) (nodes []*Node) {
	for _, protos := range nodeMaps {
		for _, proto := range protos {
			nodes = append(nodes, NewNode(proto))
		}
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

func setNodeType(evalNode *flowv1beta1.Node, specNode shared.SpecNode) *flowv1beta1.Node {
	switch specNode := specNode.(type) {
	case *flowv1beta1.Connection:
		evalNode.Type = &flowv1beta1.Node_Connection{
			Connection: specNode,
		}
	case *flowv1beta1.Input:
		evalNode.Type = &flowv1beta1.Node_Input{
			Input: specNode,
		}
	case *flowv1beta1.Var:
		evalNode.Type = &flowv1beta1.Node_Var{
			Var: specNode,
		}
	case *flowv1beta1.Action:
		evalNode.Type = &flowv1beta1.Node_Action{
			Action: specNode,
		}
	case *flowv1beta1.Stream:
		evalNode.Type = &flowv1beta1.Node_Stream{
			Stream: specNode,
		}
	case *flowv1beta1.Output:
		evalNode.Type = &flowv1beta1.Node_Output{
			Output: specNode,
		}
	}
	return evalNode
}

func (n *Node) SpecNode() shared.SpecNode {
	return GetSpecNode(n.proto)
}

func (n *Node) Compile(run *Runtime, graph *Graph) error {
	n.mut.Lock()
	if n.compile == nil {
		n.compile = sync.OnceValue(func() error {
			env, err := run.env()
			if err != nil {
				return err
			}

			visitor := GraphVisitor(graph)
			runNode, err := spec.NewRuntimeNode(env, GetSpecNode(n.proto), func(ast *cel.Ast) {
				visitor(n.proto.GetId(), ast)
			})
			if err != nil {
				return err
			}

			err = runNode.Compile(run)
			if err != nil {
				return err
			}

			n.mut.Lock()
			n.RuntimeNode = runNode
			n.mut.Unlock()

			return nil
		})
	}
	n.mut.Unlock()

	return n.compile()
}

// func (n *Node) Recv() {}
// func (n *Node) Send() {}

func (n *Node) startExecution(exec *Executor) error {
	n.mut.Lock()
	if n.start == nil {
		n.start = sync.OnceValue(func() error {
			err := n.Compile(exec.runtime, exec.graph)
			if err != nil {
				return err
			}

			recv, hasRecv := n.RuntimeNode.Recv()
			send, hasSend := n.RuntimeNode.Send()

			var (
				recvCh chan any
				sendCh chan ref.Val
			)

			if hasRecv {
				recvCh, err = exec.runtime.getRecvCh(n.id)
				if err != nil {
					return err
				}

				exec.grp.Go(func() error {
					return recv(exec.runtime, recvCh)
				})
			}

			if hasSend {
				sendCh, err = exec.runtime.getSendCh(n.id)
				if err != nil {
					return err
				}

				exec.grp.Go(func() error {
					return send(exec.runtime, sendCh)
				})

				exec.grp.Go(func() error {
					for {
						select {
						case <-exec.runtime.ctx.Done():
							return context.Cause(exec.runtime.ctx)
						case value, ok := <-sendCh:
							if !ok {
								return fmt.Errorf("%s: event channel closed", n.id)
							}

							log.FromCtx(exec.runtime.Context()).
								Info(n.id, slog.String("value", util.StringFormatAny(value.Value())))

							// Runtime_EOF is a transport sentinel: the stream closed cleanly.
							// Retire this node without surfacing a spurious output cycle.
							if _, isEOF := value.Value().(*flowv1beta1.Runtime_EOF); isEOF {
								n.applyValue(exec.runtime, types.NullValue) //nolint:errcheck
								select {
								case <-exec.runtime.ctx.Done():
									return context.Cause(exec.runtime.ctx)
								case exec.eofCh <- n.id:
								}
								return nil
							}

							// Apply the value immediately so that CurrValue is set on the
							// proto before any downstream stream-recv nodes (e.g. BidiStream)
							// read it via getValue(). The Eval() path checks SUCCESS state and
							// returns the cached value without re-reading n.valueCh.
							if _, err := n.applyValue(exec.runtime, value); err != nil {
								return err
							}

							select {
							case <-exec.runtime.ctx.Done():
								return context.Cause(exec.runtime.ctx)
							case exec.sendAckCh <- n.id:
							}
							// Wait for Reset() before accepting the next value. This prevents
							// a second external event from overwriting CurrValue before the
							// current cycle completes and evalReq has read the value.
							select {
							case <-exec.runtime.ctx.Done():
								return context.Cause(exec.runtime.ctx)
							case <-n.resetCh:
							}
						}
					}
				})
			}

			return nil
		})
	}
	n.mut.Unlock()

	return n.start()
}

func (n *Node) eval(run *Runtime) ref.Val {
	if n.RuntimeNode == nil {
		return types.WrapErr(fmt.Errorf("%s is not compiled", n.id))
	} else if eval, hasEval := n.RuntimeNode.Eval(); hasEval {
		return eval(run)
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

	return n.applyValue(run, n.eval(run))
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
	isRequiredInput := false
	if n.RuntimeNode != nil {
		if inp, ok := n.RuntimeNode.(*spec.Input); ok {
			isRequiredInput = inp.IsRequired()
		}
	}
	if !isRequiredInput {
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
