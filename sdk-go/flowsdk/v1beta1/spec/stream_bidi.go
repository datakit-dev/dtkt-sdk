package spec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types/ref"
)

var _ ExecNodeCloser = (*BidiStream)(nil)

// BidiStream implements a bidirectional-streaming RPC.
// Requests and responses are fully decoupled: each executor trigger sends one
// request, while responses arrive independently and are emitted as events.
// When the response loop terminates (EOF or error), the request loop also exits.
type BidiStream struct {
	id     string
	stream *Stream
	method *Method

	getStream func() (common.DynamicBidiStream, error)

	// sendDone is closed by Send() when the response loop exits so that
	// Recv() can exit cleanly instead of blocking on recvCh forever.
	sendDone chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
}

// NewBidiStream is called by NewCaller for bidirectional-streaming RPCs.
func NewBidiStream(node *flowv1beta1.Stream, stream *Stream, method *Method) *BidiStream {
	return &BidiStream{
		id:     GetID(node),
		stream: stream,
		method: method,

		sendDone: make(chan struct{}),
	}
}

func (s *BidiStream) Compile(run shared.Runtime) error {
	conn, err := s.method.GetConnector(run)
	if err != nil {
		return err
	}

	method, err := s.method.GetDescriptor(run)
	if err != nil {
		return err
	}

	ctx, client, err := conn.GetClient(run.Context())
	if err != nil {
		return err
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.getStream = sync.OnceValues(func() (common.DynamicBidiStream, error) {
		stream, err := client.CallBidiStream(s.ctx, method.FullName())
		if err != nil {
			s.cancel()
			return nil, err
		}
		return stream, nil
	})

	return nil
}

// Recv accepts one executor trigger per cycle, evaluates and sends one request.
// It exits when Send() terminates (sendDone is closed).
func (s *BidiStream) Recv() (shared.RecvFunc, bool) {
	return func(run shared.Runtime, recvCh <-chan any) error {
		log.FromCtx(s.ctx).Info("BidiStream recv running...", slog.String("id", s.id))
		defer log.FromCtx(s.ctx).Info("BidiStream recv done.", slog.String("id", s.id))

		// loop:
		for {
			select {
			case <-s.ctx.Done():
				return context.Cause(s.ctx)
			case <-s.sendDone:
				s.cancel()
				return nil
			case <-recvCh:
				// shouldStart, err := s.stream.ShouldStart(run)
				// if err != nil {
				// 	return err
				// } else if !shouldStart {
				// 	continue loop
				// }

				stream, err := s.getStream()
				if err != nil {
					return err
				}

				req, err := s.method.EvalRequest(run)
				if err != nil {
					return fmt.Errorf("%s: eval request: %w", s.id, err)
				}

				err = stream.SendMsg(req)
				if err != nil {
					if errors.Is(err, io.EOF) {
						return nil
					}
					return fmt.Errorf("%s: send: %w", s.id, err)
				}
			}
		}
	}, true
}

// Send independently receives responses from the gRPC stream and emits each
// as an event. It is completely decoupled from Recv. Closing sendDone signals
// Recv to exit when this goroutine terminates.
func (s *BidiStream) Send() (shared.SendFunc, bool) {
	return func(run shared.Runtime, sendCh chan<- ref.Val) error {
		log.FromCtx(s.ctx).Info("BidiStream send running...", slog.String("id", s.id))
		defer func() {
			close(s.sendDone)
			log.FromCtx(s.ctx).Info("BidiStream send done.", slog.String("id", s.id))
		}()

		env, err := run.Env()
		if err != nil {
			return err
		}

		for {
			// shouldStop, err := s.stream.ShouldStop(run)
			// if err != nil {
			// 	return err
			// } else if shouldStop {
			// 	return nil
			// }

			stream, err := s.getStream()
			if err != nil {
				return err
			}

			msg, err := stream.RecvMsg()
			if err != nil {
				if errors.Is(err, io.EOF) {
					select {
					case <-s.ctx.Done():
						return context.Cause(s.ctx)
					case sendCh <- env.TypeAdapter().NativeToValue(&flowv1beta1.Runtime_EOF{}):
					}
					return nil
				}
				return fmt.Errorf("%s: recv: %w", s.id, err)
			}

			select {
			case <-s.ctx.Done():
				return context.Cause(s.ctx)
			case sendCh <- env.TypeAdapter().NativeToValue(msg):
			}
		}
	}, true
}

func (s *BidiStream) Close() error {
	s.cancel()
	return context.Cause(s.ctx)
}

func (s *BidiStream) Eval() (shared.EvalFunc, bool) { return nil, false }
func (s *BidiStream) HasCached() (ref.Val, bool)    { return nil, false }
