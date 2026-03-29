package spec

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types/ref"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ shared.RuntimeNode = (*ClientStream)(nil)
var _ CallCloser = (*ClientStream)(nil)

// ClientStream implements a client-streaming RPC: N requests, 1 response.
// Each executor trigger sends one request. When the server signals EOF,
// the stream is closed and the final response is emitted as a single event.
type ClientStream struct {
	id      string
	stream  common.DynamicClientStream
	evalReq EvalMessageFunc

	// doneCh is closed by Recv when the stream is ready to be finalized.
	doneCh chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
	group  errgroup.Group
}

func newClientStream(
	run shared.Runtime,
	conn shared.Connector,
	node CallNode,
	method protoreflect.MethodDescriptor,
	evalReq EvalMessageFunc,
) (*ClientStream, error) {
	if _, ok := node.(*flowv1beta1.Stream); !ok {
		return nil, fmt.Errorf("unexpected node type: %T, expected *flowv1beta1.Stream", node)
	}

	ctx, cancel := context.WithCancel(run.Context())
	ctx, client, err := conn.GetClient(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	stream, err := client.CallClientStream(ctx, method.FullName())
	if err != nil {
		cancel()
		return nil, err
	}

	return &ClientStream{
		id:      GetID(node.(*flowv1beta1.Stream)),
		stream:  stream,
		evalReq: evalReq,
		doneCh:  make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

func (s *ClientStream) Compile(shared.Runtime) error  { return nil }
func (s *ClientStream) Eval() (shared.EvalFunc, bool) { return nil, false }

// Recv accepts one executor trigger per cycle, evaluates the request, and
// sends it to the gRPC stream. When the server signals EOF, doneCh is
// closed to unblock Send.
func (s *ClientStream) Recv() (shared.RecvFunc, bool) {
	return func(run shared.Runtime, recvCh <-chan any) error {
		log.FromCtx(s.ctx).Info("ClientStream recv running...")
		defer log.FromCtx(s.ctx).Info("ClientStream recv done.")

		for {
			select {
			case <-s.ctx.Done():
				return context.Cause(s.ctx)
			case <-recvCh:
				req, err := s.evalReq(run)
				if err != nil {
					return fmt.Errorf("%s: eval request: %w", s.id, err)
				}

				err = s.stream.SendMsg(req)
				if err != nil {
					if errors.Is(err, io.EOF) {
						close(s.doneCh)
						return nil
					}
					return fmt.Errorf("%s: send: %w", s.id, err)
				}
			}
		}
	}, true
}

// Send waits for Recv to signal that the server is done, then calls
// CloseAndReceive and emits the response as a single event.
func (s *ClientStream) Send() (shared.SendFunc, bool) {
	return func(run shared.Runtime, sendCh chan<- ref.Val) error {
		log.FromCtx(s.ctx).Info("ClientStream send running...")
		defer log.FromCtx(s.ctx).Info("ClientStream send done.")

		env, err := run.Env()
		if err != nil {
			return err
		}

		select {
		case <-s.ctx.Done():
			return context.Cause(s.ctx)
		case <-s.doneCh:
		}

		res, err := s.stream.CloseAndReceive()
		if err != nil {
			return fmt.Errorf("%s: close and receive: %w", s.id, err)
		}

		select {
		case <-s.ctx.Done():
			return context.Cause(s.ctx)
		case sendCh <- env.TypeAdapter().NativeToValue(res):
		}

		return nil
	}, true
}

func (s *ClientStream) Close() error {
	s.cancel()
	return s.group.Wait()
}

// NewClientStream is called by NewCaller for client-streaming RPCs.
func NewClientStream(
	run shared.Runtime,
	conn shared.Connector,
	node CallNode,
	method protoreflect.MethodDescriptor,
	evalReq EvalMessageFunc,
) (*ClientStream, error) {
	return newClientStream(run, conn, node, method, evalReq)
}
