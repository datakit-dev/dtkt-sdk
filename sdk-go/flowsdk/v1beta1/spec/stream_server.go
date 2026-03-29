package spec

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types/ref"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ shared.RuntimeNode = (*ServerStream)(nil)
var _ CallCloser = (*ServerStream)(nil)

// ServerStream implements a server-streaming RPC: one request, N responses.
// It is self-starting (no Recv trigger needed) — it opens the stream on first
// Send invocation and emits each response as an independent event.
type ServerStream struct {
	id         string
	conn       shared.Connector
	methodName protoreflect.FullName
	evalReq    EvalMessageFunc

	ctx    context.Context
	cancel context.CancelFunc
	group  errgroup.Group
}

func newServerStream(
	run shared.Runtime,
	conn shared.Connector,
	node CallNode,
	method protoreflect.MethodDescriptor,
	evalReq EvalMessageFunc,
) (*ServerStream, error) {
	if _, ok := node.(*flowv1beta1.Stream); !ok {
		return nil, fmt.Errorf("unexpected node type: %T, expected *flowv1beta1.Stream", node)
	}

	ctx, cancel := context.WithCancel(run.Context())
	return &ServerStream{
		id:         GetID(node.(*flowv1beta1.Stream)),
		conn:       conn,
		methodName: method.FullName(),
		evalReq:    evalReq,
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

func (s *ServerStream) Compile(shared.Runtime) error  { return nil }
func (s *ServerStream) Recv() (shared.RecvFunc, bool) { return nil, false }
func (s *ServerStream) Eval() (shared.EvalFunc, bool) { return nil, false }

func (s *ServerStream) Send() (shared.SendFunc, bool) {
	return func(run shared.Runtime, sendCh chan<- ref.Val) error {
		log.FromCtx(s.ctx).Info("ServerStream send running...")
		defer log.FromCtx(s.ctx).Info("ServerStream send done.")

		env, err := run.Env()
		if err != nil {
			return err
		}

		ctx, client, err := s.conn.GetClient(s.ctx)
		if err != nil {
			return fmt.Errorf("%s: get client: %w", s.id, err)
		}

		req, err := s.evalReq(run)
		if err != nil {
			return fmt.Errorf("%s: eval request: %w", s.id, err)
		}

		stream, err := client.CallServerStream(ctx, s.methodName, req)
		if err != nil {
			return fmt.Errorf("%s: call server stream: %w", s.id, err)
		}

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
	return s.group.Wait()
}

// NewServerStream is called by NewCaller for server-streaming RPCs.
func NewServerStream(
	run shared.Runtime,
	conn shared.Connector,
	node CallNode,
	method protoreflect.MethodDescriptor,
	evalReq EvalMessageFunc,
) (*ServerStream, error) {
	return newServerStream(run, conn, node, method, evalReq)
}
