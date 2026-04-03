package spec

import (
	"errors"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

var _ shared.ExecNode = (*StreamCall)(nil)

// StreamCall is a lazy wrapper for call-based streams. The actual sub-type
// (ServerStream, ClientStream, or BidiStream) is determined at Compile time
// when the method descriptor is available via the runtime's resolver.
type StreamCall struct {
	node  *flowv1beta1.Stream
	inner shared.ExecNode
}

func newStreamCall(env shared.Env, node *flowv1beta1.Stream, visitor shared.ExprVisitFunc) (*StreamCall, error) {
	if err := ParseMethodCall(env, node.GetCall(), visitor); err != nil {
		return nil, err
	}
	return &StreamCall{node: node}, nil
}

func (s *StreamCall) Compile(run shared.Runtime) error {
	caller, err := NewCaller(run, s.node)
	if err != nil {
		return err
	}
	rn, ok := caller.(shared.ExecNode)
	if !ok {
		return fmt.Errorf("stream caller %T does not implement RuntimeNode", caller)
	}
	s.inner = rn
	return nil
}

func (s *StreamCall) Recv() (shared.RecvFunc, bool) { return s.inner.Recv() }
func (s *StreamCall) Send() (shared.SendFunc, bool) { return s.inner.Send() }
func (s *StreamCall) Eval() (shared.EvalFunc, bool) { return s.inner.Eval() }

func NewStream(env shared.Env, node *flowv1beta1.Stream, visitor shared.ExprVisitFunc) (shared.ExecNode, error) {
	switch {
	case node.GetCall() != nil:
		return newStreamCall(env, node, visitor)
	case node.GetGenerate() != nil:
		return NewTicker(env, node, visitor)
	}
	return nil, errors.New("call or generate required")
}

// func ParseStream(env shared.Env, node *flowv1beta1.Stream, visitor shared.ExprVisitFunc) error {
// 	switch {
// 	case node.GetCall() != nil:
// 		if err := ParseMethodCall(env, node.GetCall(), visitor); err != nil {
// 			return err
// 		}
// 	case node.GetGenerate() != nil:
// 		if err := ParseTicker(env, node.GetGenerate(), visitor); err != nil {
// 			return err
// 		}
// 	default:
// 		return errors.New("call or generate required")
// 	}

// 	if node.GetStartIf() != "" {
// 		if _, err := shared.ParseExpr(env, node.GetStartIf(), visitor); err != nil {
// 			return err
// 		}
// 	}

// 	if node.GetStopIf() != "" {
// 		if _, err := shared.ParseExpr(env, node.GetStopIf(), visitor); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// func CompileStream(run shared.Runtime, node *flowv1beta1.Stream) (_ shared.Eval, err error) {
// 	var stream CallCloser
// 	switch {
// 	case node.GetCall() != nil:
// 		stream, err = NewCaller(run, node)
// 		if err != nil {
// 			return nil, err
// 		}
// 	case node.GetGenerate() != nil:
// 		stream, err = NewTicker(run, node)
// 		if err != nil {
// 			return nil, err
// 		}
// 	default:
// 		return nil, errors.New("call or generate required")
// 	}

// 	var (
// 		startIf   cel.Program
// 		isStarted bool
// 	)
// 	if node.GetStartIf() != "" {
// 		startIf, err = shared.CompileExpr(run, node.GetStartIf())
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	var (
// 		stopIf    cel.Program
// 		isStopped bool
// 	)
// 	if node.GetStopIf() != "" {
// 		stopIf, err = shared.CompileExpr(run, node.GetStopIf())
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	env, err := run.Env()
// 	if err != nil {
// 		return nil, err
// 	}

// 	return shared.EvalFunc(func(run shared.Runtime) ref.Val {
// 		if isStopped {
// 			return env.TypeAdapter().NativeToValue(&flowv1beta1.Runtime_EOF{})
// 		}

// 		if stopIf != nil {
// 			val, _, err := stopIf.ContextEval(run.Context(), env.Vars())
// 			if err != nil {
// 				return types.WrapErr(err)
// 			}
// 			if val == types.True {
// 				if err := stream.Close(); err != nil {
// 					return types.WrapErr(err)
// 				}

// 				isStopped = true
// 				return env.TypeAdapter().NativeToValue(&flowv1beta1.Runtime_EOF{})
// 			}
// 		}

// 		if !isStarted {
// 			if startIf != nil {
// 				val, _, err := startIf.ContextEval(run.Context(), env.Vars())
// 				if err != nil {
// 					return types.WrapErr(err)
// 				}
// 				if val != types.True {
// 					return types.NullValue
// 				}
// 			}
// 			isStarted = true
// 		}

// 		return stream.Eval(run)
// 	}), nil
// }
