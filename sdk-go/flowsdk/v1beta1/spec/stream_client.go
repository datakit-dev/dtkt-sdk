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

var _ shared.ExecNode = (*ClientStream)(nil)

// ClientStream implements a client-streaming RPC: N requests, 1 response.
// Each executor trigger sends one request. When the server signals EOF,
// the stream is closed and the final response is emitted as a single event.
type ClientStream struct {
	id     string
	method *Method
	stream common.DynamicClientStream

	shouldStart,
	shouldStop func(ctx context.Context, env shared.Env) (bool, error)

	getResp func() (any, error)
}

// NewClientStream is called by NewCaller for client-streaming RPCs.
func NewClientStream(node *flowv1beta1.Stream, stream *Stream, method *Method) *ClientStream {
	return &ClientStream{
		id:     GetNodeIdent(node),
		method: method,

		shouldStart: stream.ShouldStart,
		shouldStop:  stream.ShouldStart,
	}
}

func (s *ClientStream) Compile(ctx context.Context, env shared.Env) error {
	ctx, client, err := s.method.GetClient(ctx, env)
	if err != nil {
		return err
	}

	stream, err := client.CallClientStream(ctx, s.method.FullName())
	if err != nil {
		return err
	}

	s.stream = stream
	s.getResp = sync.OnceValues(func() (any, error) {
		return stream.CloseAndReceive()
	})

	return nil
}

func (s *ClientStream) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) error {
	defer log.FromCtx(ctx).Info("ClientStream stopped...", slog.String("id", s.id))

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
			log.FromCtx(ctx).Info("ClientStream started...", slog.String("id", s.id))
		}

		req, err := s.method.EvalRequest(ctx, env)
		if err != nil {
			return nil, err
		}

		err = s.stream.SendMsg(req)
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.FromCtx(ctx).Info("ClientStream EOF.", slog.String("id", s.id))

				return s.getResp()
			}
			return nil, err
		}

		return nil, nil
	})
}
