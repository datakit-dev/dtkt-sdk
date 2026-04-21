package v1beta1

import (
	"context"
	"log/slog"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
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
		api.Resolver
		InvokeMethod(context.Context, protoreflect.FullName, proto.Message) (proto.Message, error)
	}
	GroupCallbackFunc func(*FieldGroup) error
)
