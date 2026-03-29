// Package spectest provides test helpers and mock implementations for
// the spec and common interfaces used in flow execution.
package spectest

import (
	"context"
	"io"
	"sync"

	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// ---- DynamicClient -----------------------------------------------------

var _ common.DynamicClient = (*MockDynamicClient)(nil)

// MockDynamicClient records calls and returns pre-configured responses.
type MockDynamicClient struct {
	mu sync.Mutex

	// UnaryResponses is a FIFO queue of responses for CallUnary.
	UnaryResponses []proto.Message
	UnaryErrors    []error

	// StreamFactory returns a new bidi stream for each CallBidiStream call.
	BidiStreamFactory   func(ctx context.Context, method protoreflect.FullName) (common.DynamicBidiStream, error)
	ClientStreamFactory func(ctx context.Context, method protoreflect.FullName) (common.DynamicClientStream, error)
	ServerStreamFactory func(ctx context.Context, method protoreflect.FullName, req proto.Message) (common.DynamicServerStream, error)
}

func (m *MockDynamicClient) CallUnary(_ context.Context, _ protoreflect.FullName, _ proto.Message) (proto.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.UnaryResponses) == 0 {
		return nil, io.EOF
	}

	res := m.UnaryResponses[0]
	var err error
	if len(m.UnaryErrors) > 0 {
		err = m.UnaryErrors[0]
		m.UnaryErrors = m.UnaryErrors[1:]
	}
	m.UnaryResponses = m.UnaryResponses[1:]
	return res, err
}

func (m *MockDynamicClient) CallBidiStream(ctx context.Context, method protoreflect.FullName) (common.DynamicBidiStream, error) {
	if m.BidiStreamFactory != nil {
		return m.BidiStreamFactory(ctx, method)
	}
	return NewMockBidiStream(ctx), nil
}

func (m *MockDynamicClient) CallClientStream(ctx context.Context, method protoreflect.FullName) (common.DynamicClientStream, error) {
	if m.ClientStreamFactory != nil {
		return m.ClientStreamFactory(ctx, method)
	}
	return NewMockClientStream(ctx), nil
}

func (m *MockDynamicClient) CallServerStream(ctx context.Context, method protoreflect.FullName, req proto.Message) (common.DynamicServerStream, error) {
	if m.ServerStreamFactory != nil {
		return m.ServerStreamFactory(ctx, method, req)
	}
	return NewMockServerStream(ctx, nil), nil
}

// ---- DynamicBidiStream -------------------------------------------------

var _ common.DynamicBidiStream = (*MockBidiStream)(nil)

// MockBidiStream simulates a bidirectional gRPC stream.
// Requests sent via SendMsg are queued in Sent.
// Responses are read from the Responses channel; closing it yields io.EOF.
type MockBidiStream struct {
	ctx context.Context

	mu   sync.Mutex
	Sent []proto.Message

	// Responses is a channel of (message, error) pairs to return from RecvMsg.
	// Close the channel to signal EOF.
	Responses chan RecvItem
}

type RecvItem struct {
	Msg proto.Message
	Err error
}

func NewMockBidiStream(ctx context.Context) *MockBidiStream {
	return &MockBidiStream{
		ctx:       ctx,
		Responses: make(chan RecvItem, 16),
	}
}

func (m *MockBidiStream) Context() context.Context { return m.ctx }

func (m *MockBidiStream) SendMsg(msg proto.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sent = append(m.Sent, proto.Clone(msg))
	return nil
}

func (m *MockBidiStream) RecvMsg() (proto.Message, error) {
	item, ok := <-m.Responses
	if !ok {
		return nil, io.EOF
	}
	return item.Msg, item.Err
}

func (m *MockBidiStream) CloseSend() error { return nil }

// SentCount returns the number of recorded requests.
func (m *MockBidiStream) SentCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Sent)
}

// ---- DynamicClientStream -----------------------------------------------

var _ common.DynamicClientStream = (*MockClientStream)(nil)

// MockClientStream simulates a client-streaming gRPC stream.
// Requests are queued in Sent. Response is returned from CloseAndReceive.
// Use InjectSendErr to cause the next SendMsg to return a specific error.
type MockClientStream struct {
	ctx context.Context

	mu          sync.Mutex
	Sent        []proto.Message
	sendErrOnce error // next SendMsg returns this then clears

	Response        proto.Message
	CloseAndRecvErr error
}

func NewMockClientStream(ctx context.Context) *MockClientStream {
	return &MockClientStream{ctx: ctx}
}

func (m *MockClientStream) Context() context.Context { return m.ctx }

func (m *MockClientStream) SendMsg(msg proto.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErrOnce != nil {
		err := m.sendErrOnce
		m.sendErrOnce = nil
		return err
	}
	m.Sent = append(m.Sent, proto.Clone(msg))
	return nil
}

// InjectSendErr causes the next call to SendMsg to return err.
func (m *MockClientStream) InjectSendErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendErrOnce = err
}

// SentCount returns the number of successfully recorded sends.
func (m *MockClientStream) SentCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Sent)
}

func (m *MockClientStream) CloseAndReceive() (proto.Message, error) {
	return m.Response, m.CloseAndRecvErr
}

// ---- DynamicServerStream -----------------------------------------------

var _ common.DynamicServerStream = (*MockServerStream)(nil)

// MockServerStream simulates a server-streaming gRPC stream.
// Messages are served from the Responses slice in order; after the last one
// RecvMsg returns io.EOF.
type MockServerStream struct {
	ctx context.Context

	mu        sync.Mutex
	Responses []RecvItem
	pos       int
}

func NewMockServerStream(ctx context.Context, responses []RecvItem) *MockServerStream {
	return &MockServerStream{ctx: ctx, Responses: responses}
}

func (m *MockServerStream) Context() context.Context { return m.ctx }

func (m *MockServerStream) RecvMsg() (proto.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pos >= len(m.Responses) {
		return nil, io.EOF
	}
	item := m.Responses[m.pos]
	m.pos++
	return item.Msg, item.Err
}

// ---- Connector ---------------------------------------------------------

var _ shared.Connector = (*MockConnector)(nil)

// MockConnector returns the provided DynamicClient and a NoopResolver.
type MockConnector struct {
	Client   common.DynamicClient
	Resolver shared.Resolver
}

func (m *MockConnector) GetResolver(_ context.Context) (shared.Resolver, error) {
	if m.Resolver != nil {
		return m.Resolver, nil
	}
	return &NoopResolver{}, nil
}

func (m *MockConnector) GetClient(ctx context.Context) (context.Context, common.DynamicClient, error) {
	return ctx, m.Client, nil
}

// ---- NoopResolver ------------------------------------------------------

var _ shared.Resolver = (*NoopResolver)(nil)

// NoopResolver satisfies shared.Resolver with empty/no-op implementations.
type NoopResolver struct{}

func (r *NoopResolver) FindExtensionByName(protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}
func (r *NoopResolver) FindExtensionByNumber(protoreflect.FullName, protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}
func (r *NoopResolver) FindMessageByName(protoreflect.FullName) (protoreflect.MessageType, error) {
	return nil, protoregistry.NotFound
}
func (r *NoopResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	return nil, protoregistry.NotFound
}
func (r *NoopResolver) RangeServices(func(protoreflect.ServiceDescriptor) bool) {}
func (r *NoopResolver) RangeMethods(func(protoreflect.MethodDescriptor) bool)   {}
func (r *NoopResolver) FindMethodByName(protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
	return nil, protoregistry.NotFound
}
func (r *NoopResolver) RangeFiles(func(protoreflect.FileDescriptor) bool) {}
func (r *NoopResolver) GetValidator() (protovalidate.Validator, error) {
	return protovalidate.New()
}
