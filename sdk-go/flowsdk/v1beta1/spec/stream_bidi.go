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

var _ shared.ExecNode = (*BidiStream)(nil)
var _ CallCloser = (*BidiStream)(nil)

// BidiStream implements a bidirectional-streaming RPC.
// Requests and responses are fully decoupled: each executor trigger sends one
// request, while responses arrive independently and are emitted as events.
// When the response loop terminates (EOF or error), the request loop also exits.
type BidiStream struct {
	id      string
	stream  common.DynamicBidiStream
	evalReq EvalMessageFunc

	// sendDone is closed by Send() when the response loop exits so that
	// Recv() can exit cleanly instead of blocking on recvCh forever.
	sendDone chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
	group  errgroup.Group
}

func newBidiStream(
	run shared.Runtime,
	conn shared.Connector,
	node CallNode,
	method protoreflect.MethodDescriptor,
	evalReq EvalMessageFunc,
) (*BidiStream, error) {
	if _, ok := node.(*flowv1beta1.Stream); !ok {
		return nil, fmt.Errorf("unexpected node type: %T, expected *flowv1beta1.Stream", node)
	}

	ctx, cancel := context.WithCancel(run.Context())
	ctx, client, err := conn.GetClient(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	stream, err := client.CallBidiStream(ctx, method.FullName())
	if err != nil {
		cancel()
		return nil, err
	}

	return &BidiStream{
		id:       GetID(node.(*flowv1beta1.Stream)),
		stream:   stream,
		evalReq:  evalReq,
		sendDone: make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func (s *BidiStream) Compile(shared.Runtime) error  { return nil }
func (s *BidiStream) Eval() (shared.EvalFunc, bool) { return nil, false }

// Recv accepts one executor trigger per cycle, evaluates and sends one request.
// It exits when Send() terminates (sendDone is closed).
func (s *BidiStream) Recv() (shared.RecvFunc, bool) {
	return func(run shared.Runtime, recvCh <-chan any) error {
		log.FromCtx(s.ctx).Info("BidiStream recv running...")
		defer log.FromCtx(s.ctx).Info("BidiStream recv done.")

		for {
			select {
			case <-s.ctx.Done():
				return context.Cause(s.ctx)
			case <-s.sendDone:
				return nil
			case <-recvCh:
				req, err := s.evalReq(run)
				if err != nil {
					return fmt.Errorf("%s: eval request: %w", s.id, err)
				}

				err = s.stream.SendMsg(req)
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
		log.FromCtx(s.ctx).Info("BidiStream send running...")
		defer func() {
			close(s.sendDone)
			log.FromCtx(s.ctx).Info("BidiStream send done.")
		}()

		env, err := run.Env()
		if err != nil {
			return err
		}

		for {
			msg, err := s.stream.RecvMsg()
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
	return s.group.Wait()
}

// NewBidiStream is called by NewCaller for bidirectional-streaming RPCs.
func NewBidiStream(
	run shared.Runtime,
	conn shared.Connector,
	node CallNode,
	method protoreflect.MethodDescriptor,
	evalReq EvalMessageFunc,
) (*BidiStream, error) {
	return newBidiStream(run, conn, node, method, evalReq)
}
