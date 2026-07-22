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

var _ shared.ParsedNode = (*ClientStream)(nil)

// ClientStream implements a client-streaming RPC: N requests, 1 response.
// Each executor trigger sends one request. When the server signals EOF,
// the stream is closed and the final response is emitted as a single event.
type ClientStream struct {
	ident  string
	method *Method

	shouldStart,
	shouldStop func(ctx context.Context, env shared.Env) (bool, error)

	getStream func() (common.DynamicClientStream, error)
}

// NewClientStream is called by NewCaller for client-streaming RPCs.
func NewClientStream(node *flowv1beta1.Stream, stream *Stream, method *Method) *ClientStream {
	return &ClientStream{
		ident:  shared.GetNodeIdent(node),
		method: method,

		shouldStart: stream.ShouldStart,
		shouldStop:  stream.ShouldStop,
	}
}

func (s *ClientStream) Ident() string {
	return s.ident
}

func (s *ClientStream) Compile(ctx context.Context, env shared.Env) error {
	ctx, client, err := s.method.GetClient(ctx, env)
	if err != nil {
		return err
	}

	s.getStream = sync.OnceValues(func() (common.DynamicClientStream, error) {
		return client.CallClientStream(ctx, s.method.FullName())
	})

	return nil
}

func (s *ClientStream) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) error {
	var (
		logger = log.FromCtx(ctx).With(
			slog.String("method", string(s.method.FullName())),
		)
		started, closed bool
	)

	logger.Info("ClientStream running...")
	defer logger.Info("ClientStream stopped...")

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

		req, err := s.method.EvalRequest(ctx, env)
		if err != nil {
			logger.Error("EvalRequest", log.Err(err))
			return nil, err
		}

		err = stream.SendMsg(req)
		if err != nil {
			logger.Error("SendMsg", log.Err(err))

			closed = true

			if errors.Is(err, io.EOF) {
				return stream.CloseAndReceive()
			}

			return nil, err
		}

		return nil, nil
	})
}
