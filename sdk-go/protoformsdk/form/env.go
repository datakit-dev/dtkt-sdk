package form

import (
	"context"
	"log/slog"

	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	Env interface {
		Context() context.Context
		Logger() *slog.Logger
		OnGroupCompleted(*FieldGroup, GroupCallbackFunc) error
		Resolver() Resolver
	}
	Resolver interface {
		common.CELResolver
		FindMethodByName(protoreflect.FullName) (protoreflect.MethodDescriptor, error)
		RangeMessages(func(protoreflect.MessageType) bool)
		InvokeMethod(context.Context, protoreflect.FullName, proto.Message) (proto.Message, error)
		GetValidator() (protovalidate.Validator, error)
	}
	GroupCallbackFunc func(*FieldGroup) error
)
