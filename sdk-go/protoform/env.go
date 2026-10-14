package protoform

import (
	"context"
	"log/slog"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Env interface {
	Context() context.Context
	Logger() *slog.Logger
	Resolver() api.Resolver
	Encoder() (encoding.Encoder, bool)
	Decoder() (encoding.Decoder, bool)
	InvokeMethod(context.Context, protoreflect.FullName, proto.Message) (proto.Message, error)
}
