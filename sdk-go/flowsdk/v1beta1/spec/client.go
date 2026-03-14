package spec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
)

const MaxMsgSize = 1024 * 1024 * 10 // 10 MiB

var _ Stream = (*BidiStream)(nil)
var _ Stream = (*ClientStream)(nil)
var _ Stream = (*ServerStream)(nil)

type (
	Stream interface {
		Context() context.Context
		Send(proto.Message) error
		Recv() (proto.Message, error)
		Close() error
	}
	BidiStream struct {
		stream common.DynamicBidiStream
		cancel context.CancelFunc
		isEOF  bool
		resCh  chan proto.Message
		errCh  chan error

		mut sync.Mutex
	}
	ClientStream struct {
		stream   common.DynamicClientStream
		cancel   context.CancelFunc
		recvOnce func() (proto.Message, error)
	}
	ServerStream struct {
		stream   common.DynamicServerStream
		cancel   context.CancelFunc
		sendOnce func(proto.Message) error
		isEOF    bool

		mut sync.Mutex
	}
)

func NewBidiStream(ctx context.Context, client common.DynamicClient, method protoreflect.MethodDescriptor) (*BidiStream, error) {
	ctx, cancel := context.WithCancel(ctx)
	stream, err := client.CallBidiStream(ctx, method.FullName())
	if err != nil {
		cancel()
		return nil, err
	}

	s := &BidiStream{
		stream: stream,
		cancel: cancel,
		resCh:  make(chan proto.Message, 100), // buffered to prevent blocking
		errCh:  make(chan error, 100),         // buffered to prevent blocking
	}

	// Single goroutine handles all reads
	go func() {
		for {
			msg, err := stream.RecvMsg()
			if err != nil {
				select {
				case s.errCh <- err:
				case <-ctx.Done():
				}
				return
			}
			select {
			case s.resCh <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	return s, nil
}

func NewClientStream(ctx context.Context, client common.DynamicClient, method protoreflect.MethodDescriptor) (*ClientStream, error) {
	ctx, cancel := context.WithCancel(ctx)
	stream, err := client.CallClientStream(ctx, method.FullName())
	if err != nil {
		cancel()
		return nil, err
	}

	return &ClientStream{
		stream: stream,
		cancel: cancel,
		recvOnce: sync.OnceValues(func() (proto.Message, error) {
			return stream.CloseAndReceive()
		}),
	}, nil
}

func NewServerStream(ctx context.Context, client common.DynamicClient, method protoreflect.MethodDescriptor) (*ServerStream, error) {
	ctx, cancel := context.WithCancel(ctx)
	ss := &ServerStream{
		cancel: cancel,
	}

	ss.sendOnce = func(msg proto.Message) error {
		ss.mut.Lock()
		stream := ss.stream
		ss.mut.Unlock()

		if stream == nil {
			stream, err := client.CallServerStream(ctx, method.FullName(), msg)
			if err != nil {
				cancel()
				return err
			}

			ss.mut.Lock()
			ss.stream = stream
			ss.mut.Unlock()
		}

		return nil
	}

	return ss, nil
}

func (s *BidiStream) Context() context.Context {
	return s.stream.Context()
}

func (s *BidiStream) Send(msg proto.Message) error {
	return s.stream.SendMsg(msg)
}

func (s *BidiStream) Recv() (proto.Message, error) {
	s.mut.Lock()
	isEOF := s.isEOF
	s.mut.Unlock()

	if isEOF {
		return nil, io.EOF
	}

	select {
	case <-s.Context().Done():
		return nil, s.Context().Err()
	case err := <-s.errCh:
		if errors.Is(err, io.EOF) {
			s.mut.Lock()
			s.isEOF = true
			s.mut.Unlock()
		}
		return nil, err
	case res := <-s.resCh:
		return res, nil
	case <-time.After(100 * time.Millisecond):
		// TODO: Should we support custom timeout on streams?
		return structpb.NewNullValue(), nil
	}
}

func (s *BidiStream) Close() error {
	s.cancel()
	<-s.stream.Context().Done()
	return nil
}

func (s *ClientStream) Context() context.Context {
	return s.stream.Context()
}

func (s *ClientStream) Send(msg proto.Message) error {
	return s.stream.SendMsg(msg)
}

func (s *ClientStream) Recv() (proto.Message, error) {
	return s.recvOnce()
}

func (s *ClientStream) Close() error {
	s.cancel()
	<-s.stream.Context().Done()
	return nil
}

func (s *ServerStream) Context() context.Context {
	return s.stream.Context()
}

func (s *ServerStream) Send(msg proto.Message) error {
	return s.sendOnce(msg)
}

func (s *ServerStream) Recv() (proto.Message, error) {
	if s.stream == nil {
		return nil, fmt.Errorf("must call send first")
	} else if s.isEOF {
		return nil, io.EOF
	}

	res, err := s.stream.RecvMsg()
	if err != nil {
		s.isEOF = errors.Is(err, io.EOF)
		return nil, err
	}
	return res, nil
}

func (s *ServerStream) Close() error {
	s.cancel()
	<-s.stream.Context().Done()
	return nil
}
