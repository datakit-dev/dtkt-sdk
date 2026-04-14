package mock

import (
	"context"
	"fmt"
	"io"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
)

// Function types for registering mock RPC handlers.
type (
	UnaryFunc        func(ctx context.Context, req proto.Message) (proto.Message, error)
	ServerStreamFunc func(ctx context.Context, req proto.Message, send func(proto.Message) error) error
	ClientStreamFunc func(ctx context.Context, recv func() (proto.Message, error)) (proto.Message, error)
	BidiStreamFunc   func(ctx context.Context, recv func() (proto.Message, error), send func(proto.Message) error) error
)

// Client is an in-memory rpc.Client implementation backed by registered function handlers.
type Client struct {
	unary        map[protoreflect.FullName]UnaryFunc
	serverStream map[protoreflect.FullName]ServerStreamFunc
	clientStream map[protoreflect.FullName]ClientStreamFunc
	bidiStream   map[protoreflect.FullName]BidiStreamFunc
}

// Compile-time interface checks.
var (
	_ rpc.Client      = (*Client)(nil)
	_ shared.Resolver = (*Client)(nil)
)

func NewClient() *Client {
	return &Client{
		unary:        make(map[protoreflect.FullName]UnaryFunc),
		serverStream: make(map[protoreflect.FullName]ServerStreamFunc),
		clientStream: make(map[protoreflect.FullName]ClientStreamFunc),
		bidiStream:   make(map[protoreflect.FullName]BidiStreamFunc),
	}
}

func (c *Client) RegisterUnary(method string, fn UnaryFunc) {
	c.unary[protoreflect.FullName(method)] = fn
}

func (c *Client) RegisterServerStream(method string, fn ServerStreamFunc) {
	c.serverStream[protoreflect.FullName(method)] = fn
}

func (c *Client) RegisterClientStream(method string, fn ClientStreamFunc) {
	c.clientStream[protoreflect.FullName(method)] = fn
}

func (c *Client) RegisterBidiStream(method string, fn BidiStreamFunc) {
	c.bidiStream[protoreflect.FullName(method)] = fn
}

func (c *Client) FindMethodByName(name protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
	if _, ok := c.unary[name]; ok {
		return methodDescs.unary, nil
	}
	if _, ok := c.serverStream[name]; ok {
		return methodDescs.serverStream, nil
	}
	if _, ok := c.clientStream[name]; ok {
		return methodDescs.clientStream, nil
	}
	if _, ok := c.bidiStream[name]; ok {
		return methodDescs.bidiStream, nil
	}
	return nil, fmt.Errorf("method %q not found", name)
}

// shared.Resolver stub methods -- mock only needs FindMethodByName for dispatch.

func (c *Client) FindMessageByName(protoreflect.FullName) (protoreflect.MessageType, error) {
	return nil, protoregistry.NotFound
}

func (c *Client) FindMessageByURL(string) (protoreflect.MessageType, error) {
	return nil, protoregistry.NotFound
}

func (c *Client) FindExtensionByName(protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}

func (c *Client) FindExtensionByNumber(protoreflect.FullName, protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}

func (c *Client) RangeServices(func(protoreflect.ServiceDescriptor) bool) {}
func (c *Client) RangeMethods(func(protoreflect.MethodDescriptor) bool)   {}
func (c *Client) RangeFiles(func(protoreflect.FileDescriptor) bool)       {}

func (c *Client) GetValidator() (protovalidate.Validator, error) {
	return nil, fmt.Errorf("mock: validator not available")
}

// methodDescs holds pre-built protoreflect.MethodDescriptors for each
// streaming kind. Only the streaming flags matter for handler dispatch.
var methodDescs = func() struct {
	unary, serverStream, clientStream, bidiStream protoreflect.MethodDescriptor
} {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("mock.proto"),
		Syntax:  proto.String("proto3"),
		Package: proto.String("mock"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("Req")},
			{Name: proto.String("Res")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("Svc"),
			Method: []*descriptorpb.MethodDescriptorProto{
				{Name: proto.String("Unary"), InputType: proto.String(".mock.Req"), OutputType: proto.String(".mock.Res"), ClientStreaming: proto.Bool(false), ServerStreaming: proto.Bool(false)},
				{Name: proto.String("ServerStream"), InputType: proto.String(".mock.Req"), OutputType: proto.String(".mock.Res"), ClientStreaming: proto.Bool(false), ServerStreaming: proto.Bool(true)},
				{Name: proto.String("ClientStream"), InputType: proto.String(".mock.Req"), OutputType: proto.String(".mock.Res"), ClientStreaming: proto.Bool(true), ServerStreaming: proto.Bool(false)},
				{Name: proto.String("BidiStream"), InputType: proto.String(".mock.Req"), OutputType: proto.String(".mock.Res"), ClientStreaming: proto.Bool(true), ServerStreaming: proto.Bool(true)},
			},
		}},
	}
	file, err := protodesc.NewFile(fd, nil)
	if err != nil {
		panic(fmt.Sprintf("mock: building method descriptors: %v", err))
	}
	methods := file.Services().Get(0).Methods()
	return struct {
		unary, serverStream, clientStream, bidiStream protoreflect.MethodDescriptor
	}{
		unary:        methods.ByName("Unary"),
		serverStream: methods.ByName("ServerStream"),
		clientStream: methods.ByName("ClientStream"),
		bidiStream:   methods.ByName("BidiStream"),
	}
}()

func (c *Client) CallUnary(ctx context.Context, name protoreflect.FullName, req proto.Message) (proto.Message, error) {
	fn, ok := c.unary[name]
	if !ok {
		return nil, fmt.Errorf("unary method %q not found", name)
	}
	return fn(ctx, req)
}

func (c *Client) CallServerStream(ctx context.Context, name protoreflect.FullName, req proto.Message) (rpc.ServerStream, error) {
	fn, ok := c.serverStream[name]
	if !ok {
		return nil, fmt.Errorf("server-stream method %q not found", name)
	}
	ch := make(chan proto.Message, 16)
	errCh := make(chan error, 1)
	go func() {
		defer close(ch)
		errCh <- fn(ctx, req, func(msg proto.Message) error {
			select {
			case ch <- msg:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	return &serverStream{ctx: ctx, ch: ch, errCh: errCh}, nil
}

type clientStreamResult struct {
	resp proto.Message
	err  error
}

func (c *Client) CallClientStream(ctx context.Context, name protoreflect.FullName) (rpc.ClientStream, error) {
	fn, ok := c.clientStream[name]
	if !ok {
		return nil, fmt.Errorf("client-stream method %q not found", name)
	}
	sendCh := make(chan proto.Message, 16)
	resultCh := make(chan clientStreamResult, 1)
	go func() {
		recv := func() (proto.Message, error) {
			msg, ok := <-sendCh
			if !ok {
				return nil, io.EOF
			}
			return msg, nil
		}
		resp, err := fn(ctx, recv)
		resultCh <- clientStreamResult{resp, err}
	}()
	return &clientStream{ctx: ctx, sendCh: sendCh, resultCh: resultCh}, nil
}

func (c *Client) CallBidiStream(ctx context.Context, name protoreflect.FullName) (rpc.BidiStream, error) {
	fn, ok := c.bidiStream[name]
	if !ok {
		return nil, fmt.Errorf("bidi-stream method %q not found", name)
	}
	sendCh := make(chan proto.Message, 16)
	recvCh := make(chan proto.Message, 16)
	errCh := make(chan error, 1)
	go func() {
		defer close(recvCh)
		recv := func() (proto.Message, error) {
			msg, ok := <-sendCh
			if !ok {
				return nil, io.EOF
			}
			return msg, nil
		}
		send := func(msg proto.Message) error {
			select {
			case recvCh <- msg:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		errCh <- fn(ctx, recv, send)
	}()
	return &bidiStream{ctx: ctx, sendCh: sendCh, recvCh: recvCh, errCh: errCh}, nil
}

// --- mock stream implementations ---

type serverStream struct {
	ctx   context.Context
	ch    <-chan proto.Message
	errCh <-chan error
}

func (s *serverStream) Context() context.Context { return s.ctx }

func (s *serverStream) RecvMsg() (proto.Message, error) {
	select {
	case msg, ok := <-s.ch:
		if !ok {
			select {
			case err := <-s.errCh:
				if err != nil {
					return nil, err
				}
			default:
			}
			return nil, io.EOF
		}
		return msg, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

type clientStream struct {
	ctx      context.Context
	sendCh   chan<- proto.Message
	resultCh <-chan clientStreamResult
}

func (s *clientStream) Context() context.Context { return s.ctx }

func (s *clientStream) SendMsg(msg proto.Message) error {
	select {
	case s.sendCh <- msg:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *clientStream) CloseAndReceive() (proto.Message, error) {
	close(s.sendCh)
	r := <-s.resultCh
	return r.resp, r.err
}

type bidiStream struct {
	ctx    context.Context
	sendCh chan<- proto.Message
	recvCh <-chan proto.Message
	errCh  <-chan error
}

func (s *bidiStream) Context() context.Context { return s.ctx }

func (s *bidiStream) SendMsg(msg proto.Message) error {
	select {
	case s.sendCh <- msg:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *bidiStream) RecvMsg() (proto.Message, error) {
	select {
	case msg, ok := <-s.recvCh:
		if !ok {
			select {
			case err := <-s.errCh:
				if err != nil {
					return nil, err
				}
			default:
			}
			return nil, io.EOF
		}
		return msg, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

func (s *bidiStream) CloseSend() error {
	close(s.sendCh)
	return nil
}
