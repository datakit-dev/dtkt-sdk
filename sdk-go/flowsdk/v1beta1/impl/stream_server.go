package impl

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

var _ shared.ParsedNode = (*ServerStream)(nil)

// ServerStream implements a server-streaming RPC: one request, N responses.
// It is self-starting (no Recv trigger needed) — it opens the stream on first
// Send invocation and emits each response as an independent event.
type ServerStream struct {
	ident  string
	method *Method

	shouldStart,
	shouldStop func(context.Context, shared.Env) (bool, error)

	getStream func() (common.DynamicServerStream, error)
}

// NewServerStream is called by NewCaller for server-streaming RPCs.
func NewServerStream(node *flowv1beta1.Stream, stream *Stream, method *Method) *ServerStream {
	return &ServerStream{
		ident:  shared.GetNodeIdent(node),
		method: method,

		shouldStart: stream.ShouldStart,
		shouldStop:  stream.ShouldStop,
	}
}

func (s *ServerStream) Ident() string {
	return s.ident
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
	var (
		logger = log.FromCtx(ctx).With(
			slog.String("method", string(s.method.FullName())),
		)
		started, closed bool
	)

	logger.Info("ServerStream running...")
	defer logger.Info("ServerStream stopped...")

	return pipe.Run(log.NewCtx(ctx, logger), func(ctx context.Context, value any) (any, error) {
		if closed {
			return &flowv1beta1.EOF{}, nil
		} else if !started {
			shouldStart, err := s.shouldStart(ctx, env)
			if err != nil {
				return nil, err
			} else if !shouldStart {
				return nil, nil
			}

			started = true
		}

		stream, err := s.getStream()
		if err != nil {
			logger.Error("getStream", log.Err(err))
			return nil, err
		}

		res, err := stream.RecvMsg()
		if err != nil {
			logger.Error("RecvMsg", log.Err(err))

			closed = true

			if errors.Is(err, io.EOF) {
				return &flowv1beta1.EOF{}, nil
			} else {
				return nil, err
			}
		}

		return res, nil
	})
}
