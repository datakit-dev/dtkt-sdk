package fivetran

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func protoFromValue[T proto.Message](src any) (dst T, err error) {
	b, err := encoding.ToJSONV2(src)
	if err != nil {
		return
	}

	dst = dst.ProtoReflect().New().Interface().(T)
	err = encoding.FromJSONV2(b, dst)
	return
}

func wrapProtoFromValue[T proto.Message](src any) (*anypb.Any, error) {
	dst, err := protoFromValue[T](src)
	if err != nil {
		return nil, err
	}

	return anypb.New(dst)
}
