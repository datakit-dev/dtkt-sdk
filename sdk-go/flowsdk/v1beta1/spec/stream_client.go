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
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var _ ExecNodeCloser = (*ClientStream)(nil)

// ClientStream implements a client-streaming RPC: N requests, 1 response.
// Each executor trigger sends one request. When the server signals EOF,
// the stream is closed and the final response is emitted as a single event.
type ClientStream struct {
	id     string
	stream *Stream
	method *Method

	getStream func() (common.DynamicClientStream, error)

	// doneCh is closed by Recv when the stream is ready to be finalized.
	doneCh chan bool

	ctx    context.Context
	cancel context.CancelFunc
}

// NewClientStream is called by NewCaller for client-streaming RPCs.
func NewClientStream(node *flowv1beta1.Stream, stream *Stream, method *Method) *ClientStream {
	return &ClientStream{
		id:     GetID(node),
		stream: stream,
		method: method,

		doneCh: make(chan bool),
	}
}

func (s *ClientStream) Compile(run shared.Runtime) error {
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
	s.getStream = sync.OnceValues(func() (common.DynamicClientStream, error) {
		stream, err := client.CallClientStream(ctx, method.FullName())
		if err != nil {
			s.cancel()
			return nil, err
		}

		return stream, nil
	})

	return nil
}

// Recv accepts one executor trigger per cycle, evaluates the request, and
// sends it to the gRPC stream. When the server signals EOF, doneCh is
// closed to unblock Send.
func (s *ClientStream) Recv() (shared.RecvFunc, bool) {
	return func(run shared.Runtime, recvCh <-chan any) error {
		log.FromCtx(s.ctx).Info("ClientStream recv running...", slog.String("id", s.id))
		defer log.FromCtx(s.ctx).Info("ClientStream recv done.", slog.String("id", s.id))

		// loop:
		for {
			select {
			case <-s.ctx.Done():
				return context.Cause(s.ctx)
			case <-recvCh:
				// shouldStart, err := s.stream.ShouldStart(run)
				// if err != nil {
				// 	return err
				// } else if !shouldStart {
				// 	continue loop
				// }

				req, err := s.method.EvalRequest(run)
				if err != nil {
					return fmt.Errorf("%s: eval request: %w", s.id, err)
				}

				stream, err := s.getStream()
				if err != nil {
					return err
				}

				err = stream.SendMsg(req)
				if err != nil {
					if errors.Is(err, io.EOF) {
						select {
						case <-s.ctx.Done():
							return context.Cause(s.ctx)
						case s.doneCh <- true:
						}
						return nil
					}

					return fmt.Errorf("%s: send: %w", s.id, err)
				}

				select {
				case <-s.ctx.Done():
					return context.Cause(s.ctx)
				case s.doneCh <- false:
				}
			}
		}
	}, true
}

// Send waits for Recv to signal that the server is done, then calls
// CloseAndReceive and emits the response as a single event.
func (s *ClientStream) Send() (shared.SendFunc, bool) {
	return func(run shared.Runtime, sendCh chan<- ref.Val) error {
		log.FromCtx(s.ctx).Info("ClientStream send running...", slog.String("id", s.id))
		defer log.FromCtx(s.ctx).Info("ClientStream send done.", slog.String("id", s.id))

		env, err := run.Env()
		if err != nil {
			return err
		}

		var recvDone bool
		for {
			// shouldStop, err := s.stream.ShouldStop(run)
			// if err != nil {
			// 	return err
			// } else if shouldStop {
			// 	s.cancel()
			// 	return nil
			// }

			select {
			case <-s.ctx.Done():
				return context.Cause(s.ctx)
			case recvDone = <-s.doneCh:
			}

			if recvDone {
				stream, err := s.getStream()
				if err != nil {
					return err
				}

				res, err := stream.CloseAndReceive()
				if err != nil {
					return fmt.Errorf("%s: close and receive: %w", s.id, err)
				}

				select {
				case <-s.ctx.Done():
					return context.Cause(s.ctx)
				case sendCh <- env.TypeAdapter().NativeToValue(res):
				}

				return nil
			}

			select {
			case <-s.ctx.Done():
				return context.Cause(s.ctx)
			case sendCh <- types.NullValue:
			}
		}
	}, true
}

func (s *ClientStream) Close() error {
	s.cancel()
	return context.Cause(s.ctx)
}

func (s *ClientStream) Eval() (shared.EvalFunc, bool) { return nil, false }
func (s *ClientStream) HasCached() (ref.Val, bool)    { return nil, false }
