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

var _ shared.ParsedNode = (*BidiStream)(nil)

// BidiStream implements a bidirectional-streaming RPC.
// Requests and responses are fully decoupled: each executor trigger sends one
// request, while responses arrive independently and are emitted as events.
// When the response loop terminates (EOF or error), the request loop also exits.
type BidiStream struct {
	ident  string
	method *Method
	stream common.DynamicBidiStream

	shouldStart,
	shouldStop func(ctx context.Context, env shared.Env) (bool, error)

	ctx     context.Context
	cancel  context.CancelCauseFunc
	runSend func()
}

// NewBidiStream is called by NewCaller for bidirectional-streaming RPCs.
func NewBidiStream(node *flowv1beta1.Stream, stream *Stream, method *Method) *BidiStream {
	return &BidiStream{
		ident:  shared.GetNodeIdent(node),
		method: method,

		shouldStart: stream.ShouldStart,
		shouldStop:  stream.ShouldStop,
	}
}

func (s *BidiStream) Ident() string {
	return s.ident
}

func (s *BidiStream) Compile(ctx context.Context, env shared.Env) error {
	ctx, client, err := s.method.GetClient(ctx, env)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancelCause(ctx)

	stream, err := client.CallBidiStream(ctx, s.method.FullName())
	if err != nil {
		cancel(err)
		return err
	}

	s.runSend = sync.OnceFunc(func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			req, err := s.method.EvalRequest(ctx, env)
			if err != nil {
				cancel(err)
				return
			}

			err = s.stream.SendMsg(req)
			if err != nil {
				cancel(err)
				log.FromCtx(ctx).Info("BidiStream send EOF.", slog.String("id", s.ident))
				return
			}
		}
	})

	s.stream = stream
	s.ctx = ctx
	s.cancel = cancel

	return nil
}

func (s *BidiStream) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) error {
	defer log.FromCtx(ctx).Info("BidiStream done.", slog.String("id", s.ident))
	defer s.cancel(nil)

	go s.runSend()

	var started, closed bool

	return pipe.Run(ctx, func(ctx context.Context, value any) (any, error) {
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

			log.FromCtx(ctx).Info("BidiStream send started...", slog.String("id", s.ident))
		}

		res, err := s.stream.RecvMsg()
		if err != nil {
			s.cancel(err)

			if errors.Is(err, io.EOF) {
				log.FromCtx(ctx).Info("BidiStream recv EOF.", slog.String("id", s.ident))

				closed = true
				return &flowv1beta1.EOF{}, nil
			}

			return nil, err
		}

		return res, nil
	})
}
