package spec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types/ref"
)

var _ shared.ExecNode = (*ServerStream)(nil)
var _ ExecNodeCloser = (*ServerStream)(nil)

// ServerStream implements a server-streaming RPC: one request, N responses.
// It is self-starting (no Recv trigger needed) — it opens the stream on first
// Send invocation and emits each response as an independent event.
type ServerStream struct {
	id     string
	stream *Stream
	method *Method

	client common.DynamicClient
	ctx    context.Context
	cancel context.CancelFunc
}

// NewServerStream is called by NewCaller for server-streaming RPCs.
func NewServerStream(node *flowv1beta1.Stream, stream *Stream, method *Method) *ServerStream {
	return &ServerStream{
		id:     GetID(node),
		stream: stream,
		method: method,
	}
}

func (s *ServerStream) Compile(run shared.Runtime) error {
	conn, err := s.method.GetConnector(run)
	if err != nil {
		return err
	}

	ctx, client, err := conn.GetClient(run.Context())
	if err != nil {
		return err
	}

	s.client = client
	s.ctx, s.cancel = context.WithCancel(ctx)

	return nil
}

func (s *ServerStream) Send() (shared.SendFunc, bool) {
	return func(run shared.Runtime, sendCh chan<- ref.Val) error {
		env, err := run.Env()
		if err != nil {
			return err
		}

		method, err := s.method.GetDescriptor(run)
		if err != nil {
			return err
		}

		req, err := s.method.EvalRequest(run)
		if err != nil {
			return fmt.Errorf("%s: eval request: %w", s.id, err)
		}

		stream, err := s.client.CallServerStream(s.ctx, method.FullName(), req)
		if err != nil {
			s.cancel()
			return err
		}

		log.FromCtx(s.ctx).Info("ServerStream send running...", slog.String("id", s.id))
		defer log.FromCtx(s.ctx).Info("ServerStream send done.", slog.String("id", s.id))

		for {
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

func (s *ServerStream) Close() error {
	s.cancel()
	return context.Cause(s.ctx)
}

func (s *ServerStream) Eval() (shared.EvalFunc, bool) { return nil, false }
func (s *ServerStream) HasCached() (ref.Val, bool)    { return nil, false }
func (s *ServerStream) Recv() (shared.RecvFunc, bool) { return nil, false }
