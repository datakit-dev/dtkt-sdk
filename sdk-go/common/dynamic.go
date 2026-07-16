package common

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	DynamicClient interface {
		CallUnary(context.Context, protoreflect.FullName, proto.Message) (proto.Message, error)
		CallBidiStream(context.Context, protoreflect.FullName) (DynamicBidiStream, error)
		CallClientStream(context.Context, protoreflect.FullName) (DynamicClientStream, error)
		CallServerStream(context.Context, protoreflect.FullName, proto.Message) (DynamicServerStream, error)
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
