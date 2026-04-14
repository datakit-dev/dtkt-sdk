package spec

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types/ref"
)

type StreamCaller struct {
	ExecNodeCloser
	node   *flowv1beta1.Stream
	stream *Stream
	method *Method
}

func NewStreamCaller(env shared.Env, node *flowv1beta1.Stream, stream *Stream, visitor shared.NodeVisitFunc) (*StreamCaller, error) {
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

func (s *StreamCaller) Compile(run shared.Runtime) error {
	err := s.method.Compile(run)
	if err != nil {
		return err
	}

	method, err := s.method.GetDescriptor(run)
	if err != nil {
		return err
	}

	if method.IsStreamingClient() && method.IsStreamingServer() {
		s.ExecNodeCloser = NewBidiStream(s.node, s.stream, s.method)
	} else if method.IsStreamingClient() {
		s.ExecNodeCloser = NewClientStream(s.node, s.stream, s.method)
	} else if method.IsStreamingServer() {
		s.ExecNodeCloser = NewServerStream(s.node, s.stream, s.method)
	} else {
		return nil
	}

	return s.ExecNodeCloser.Compile(run)
}

func (s *StreamCaller) Recv() (shared.RecvFunc, bool) {
	if s.ExecNodeCloser != nil {
		return s.ExecNodeCloser.Recv()
	}
	return nil, false
}

func (s *StreamCaller) Send() (shared.SendFunc, bool) {
	if s.ExecNodeCloser != nil {
		return s.ExecNodeCloser.Send()
	}
	return nil, false
}

func (s *StreamCaller) Close() error {
	if s.ExecNodeCloser != nil {
		return s.ExecNodeCloser.Close()
	}
	return nil
}

func (s *StreamCaller) Eval() (shared.EvalFunc, bool) { return nil, false }
func (s *StreamCaller) HasCached() (ref.Val, bool)    { return nil, false }
