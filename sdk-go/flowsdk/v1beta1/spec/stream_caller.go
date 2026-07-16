package spec

import (
	"context"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

type StreamCaller struct {
	shared.ExecNode
	node   *flowv1beta1.Stream
	stream *Stream
	method *Method
}

func NewStreamCaller(env shared.Env, node *flowv1beta1.Stream, stream *Stream, visitor shared.NodeVisitor) (*StreamCaller, error) {
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

		s.ExecNode = stream
	case method.IsStreamingClient():
		stream := NewClientStream(s.node, s.stream, s.method)
		err = stream.Compile(ctx, env)
		if err != nil {
			return err
		}

		s.ExecNode = stream
	case method.IsStreamingServer():
		stream := NewServerStream(s.node, s.stream, s.method)
		err = stream.Compile(ctx, env)
		if err != nil {
			return err
		}

		s.ExecNode = stream
	default:
		return fmt.Errorf("streams.%s method invalid: %s", s.node.Id, method.FullName())
	}

	return nil
}
