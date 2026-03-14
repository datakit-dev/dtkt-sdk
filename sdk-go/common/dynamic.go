package common

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	DynamicClient interface {
		CallUnary(ctx context.Context, methodName protoreflect.FullName, req proto.Message) (proto.Message, error)
		CallBidiStream(ctx context.Context, methodName protoreflect.FullName) (DynamicBidiStream, error)
		CallClientStream(ctx context.Context, methodName protoreflect.FullName) (DynamicClientStream, error)
		CallServerStream(ctx context.Context, methodName protoreflect.FullName, req proto.Message) (DynamicServerStream, error)
	}
	DynamicBidiStream interface {
		Context() context.Context
		SendMsg(proto.Message) error
		RecvMsg() (proto.Message, error)
		CloseSend() error
	}
	DynamicClientStream interface {
		Context() context.Context
		SendMsg(proto.Message) error
		CloseAndReceive() (proto.Message, error)
	}
	DynamicServerStream interface {
		Context() context.Context
		RecvMsg() (proto.Message, error)
	}
)
