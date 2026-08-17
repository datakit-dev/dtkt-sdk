package common

import (
	"slices"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func MessageWithFields[T proto.Message](msg T, fields ...string) T {
	msg = proto.CloneOf(msg)
	msg.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if !slices.Contains(fields, string(fd.Name())) && !slices.Contains(fields, fd.JSONName()) {
			msg.ProtoReflect().Clear(fd)
		}
		return true
	})
	return msg
}

func MessageWithoutFields[T proto.Message](msg T, fields ...string) T {
	msg = proto.CloneOf(msg)
	msg.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if slices.Contains(fields, string(fd.Name())) || slices.Contains(fields, fd.JSONName()) {
			msg.ProtoReflect().Clear(fd)
		}
		return true
	})
	return msg
}
