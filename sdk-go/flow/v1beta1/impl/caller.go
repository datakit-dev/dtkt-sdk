package impl

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

var boolType = reflect.TypeFor[bool]()

const (
	CallRequestCount  = "request_count"
	CallResponseCount = "response_count"
)

type (
	ActionCaller struct {
		ident    string
		node     *flowv1beta1.Action
		method   *Method
		metadata CallMetadata

		invokeWhenExpr,
		onErrorExpr *cel.Ast

		invokeWhenProg,
		onErrorProg shared.Program

		cache *EvalCache
	}
	StreamCaller struct {
		shared.ParsedNode
		node   *flowv1beta1.Stream
		stream *Stream
		method *Method
	}
	CallNode[T Call] interface {
		shared.SpecNode
		GetCall() T
	}
	Call interface {
		proto.Message
		GetConnection() string
		GetMethod() string
		GetRequest() *structpb.Value
	}
	CallMetadata struct {
		reqCount, resCount atomic.Int64
	}
)

func NewActionCaller(
	env shared.Env,
	node *flowv1beta1.Action,
	visitor shared.NodeVisitor,
) (_ *ActionCaller, err error) {
	call := node.GetCall()
	caller := &ActionCaller{
		ident: shared.GetNodeIdent(node),
		node:  node,
	}

	if call.GetInvokeWhen() != "" {
		caller.invokeWhenExpr, err = env.ParseExpr("invokeWhen", call.GetInvokeWhen(), visitor.ExprVisitor(caller.ident))
		if err != nil {
			return nil, err
		}
	}

	if call.GetOnError() != "" {
		caller.onErrorExpr, err = env.ParseExpr("onError", call.GetOnError(), visitor.ExprVisitor(caller.ident))
		if err != nil {
			return nil, err
		}
	}

	method, err := NewMethod(env, node, visitor)
	if err != nil {
		return nil, err
	}

	caller.method = method
	caller.cache = NewEvalCache(node, caller)

	return caller, nil
}

func NewStreamCaller(
	env shared.Env,
	node *flowv1beta1.Stream,
	stream *Stream,
	visitor shared.NodeVisitor,
) (*StreamCaller, error) {
	method, err := NewMethod(env, node, visitor)
	if err != nil {
		return nil, err
	}

	return &StreamCaller{
		node:   node,
		stream: stream,
		method: method,
	}, nil
}

func (a *ActionCaller) Metadata() map[string]any {
	return a.metadata.AsMap()
}

func (m *ActionCaller) Compile(ctx context.Context, env shared.Env) (err error) {
	if m.invokeWhenExpr != nil {
		m.invokeWhenProg, err = env.CompileExpr("invokeWhen", m.invokeWhenExpr)
		if err != nil {
			return err
		}
	}

	if m.onErrorExpr != nil {
		m.onErrorProg, err = env.CompileExpr("onError", m.onErrorExpr)
		if err != nil {
			return err
		}
	}

	err = m.method.Compile(ctx, env)
	if err != nil {
		return err
	}

	return nil
}

func (m *ActionCaller) Ident() string {
	return m.ident
}

func (m *ActionCaller) Method() *Method {
	return m.method
}

func (m *ActionCaller) ShouldInvoke(ctx context.Context, env shared.Env) (bool, error) {
	if m.invokeWhenProg != nil {
		shouldInvoke, err := m.invokeWhenProg.Eval(ctx, env).ConvertToNative(boolType)
		if err != nil {
			return false, err
		}

		return shouldInvoke.(bool), nil
	}

	return true, nil
}

func (m *ActionCaller) Eval(ctx context.Context, env shared.Env) ref.Val {
	ctx, client, err := m.method.GetClient(ctx, env)
	if err != nil {
		return types.WrapErr(err)
	}

	shouldInvoke, err := m.ShouldInvoke(ctx, env)
	if err != nil {
		return types.WrapErr(err)
	} else if !shouldInvoke {
		return types.NullValue
	}

	req, err := m.method.EvalRequest(ctx, env)
	if err != nil {
		return types.WrapErr(err)
	}

	res, err := client.CallUnary(ctx, m.method.FullName(), req)
	if err != nil {
		return types.WrapErr(err)
	}

	m.metadata.IncrReqCount()
	m.metadata.IncrResCount()

	return env.Types().NativeToValue(res)
}

func (m *StreamCaller) Method() *Method {
	return m.method
}

func (s *StreamCaller) Compile(ctx context.Context, env shared.Env) error {
	err := s.method.Compile(ctx, env)
	if err != nil {
		return err
	}

	method, err := s.method.GetDescriptor(ctx, env)
	if err != nil {
		return err
	}

	switch {
	case method.IsStreamingClient() && method.IsStreamingServer():
		stream := NewBidiStream(s.node, s.stream, s.method)
		err = stream.Compile(ctx, env)
		if err != nil {
			return err
		}

		s.ParsedNode = stream
	case method.IsStreamingClient():
		stream := NewClientStream(s.node, s.stream, s.method)
		err = stream.Compile(ctx, env)
		if err != nil {
			return err
		}

		s.ParsedNode = stream
	case method.IsStreamingServer():
		stream := NewServerStream(s.node, s.stream, s.method)
		err = stream.Compile(ctx, env)
		if err != nil {
			return err
		}

		s.ParsedNode = stream
	default:
		return fmt.Errorf("streams.%s method invalid: %s", s.node.Id, method.FullName())
	}

	return nil
}

func (m *ActionCaller) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) error {
	return pipe.Run(ctx, func(ctx context.Context, value any) (any, error) {
		return m.cache.Eval(ctx, env), nil
	})
}

func (m *CallMetadata) IncrReqCount() {
	m.reqCount.Add(1)
}

func (m *CallMetadata) IncrResCount() {
	m.resCount.Add(1)
}

func (m *CallMetadata) AsMap() map[string]any {
	return map[string]any{
		CallRequestCount:  m.reqCount.Load(),
		CallResponseCount: m.resCount.Load(),
	}
}
