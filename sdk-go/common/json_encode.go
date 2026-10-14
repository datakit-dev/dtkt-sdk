package common

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"google.golang.org/protobuf/proto"
)

func MustUnmarshalJSON[T any, D ~string | ~[]byte](data D) (value T) {
	var err error
	if msg, ok := any(value).(proto.Message); ok {
		value = msg.ProtoReflect().New().Interface().(T)
		err = encoding.FromJSON([]byte(data), value)
	} else {
		err = encoding.FromJSON([]byte(data), &value)
	}
	if err != nil {
		panic(err)
	}
	return
}
