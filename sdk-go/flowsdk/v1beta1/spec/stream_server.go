package spec

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

var _ shared.ExecNode = (*ServerStream)(nil)

// ServerStream implements a server-streaming RPC: one request, N responses.
// It is self-starting (no Recv trigger needed) — it opens the stream on first
// Send invocation and emits each response as an independent event.
type ServerStream struct {
	id     string
	method *Method
	stream common.DynamicServerStream

	shouldStart,
	shouldStop func(context.Context, shared.Env) (bool, error)

	getStream func() (common.DynamicServerStream, error)
}

// NewServerStream is called by NewCaller for server-streaming RPCs.
func NewServerStream(node *flowv1beta1.Stream, stream *Stream, method *Method) *ServerStream {
	return &ServerStream{
		id:     GetNodeIdent(node),
		method: method,

		shouldStart: stream.ShouldStart,
		shouldStop:  stream.ShouldStart,
	}
}

func (s *ServerStream) Compile(ctx context.Context, env shared.Env) error {
	ctx, client, err := s.method.GetClient(ctx, env)
	if err != nil {
		return err
	}

	s.getStream = sync.OnceValues(func() (common.DynamicServerStream, error) {
		req, err := s.method.EvalRequest(ctx, env)
		if err != nil {
			return nil, err
		}

		return client.CallServerStream(ctx, s.method.FullName(), req)
	})

	return nil
}

func (s *ServerStream) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) (err error) {
	var started, closed bool

	return pipe.Run(ctx, func(ctx context.Context, value any) (any, error) {
		if closed {
			return &flowv1beta1.Runtime_EOF{}, nil
		} else if !started {
			shouldStart, err := s.shouldStart(ctx, env)
			if err != nil {
				return nil, err
			} else if !shouldStart {
				return nil, nil
			}

			started = true

			log.FromCtx(ctx).Info("ServerStream started...", slog.String("id", s.id))
		}

		stream, err := s.getStream()
		if err != nil {
			return nil, err
		}

		res, err := stream.RecvMsg()
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.FromCtx(ctx).Info("ServerStream EOF.", slog.String("id", s.id))
				return &flowv1beta1.Runtime_EOF{}, nil
			} else {
				return nil, err
			}
		}

		return res, nil
	})
}
