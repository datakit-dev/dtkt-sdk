package protostore

import (
	"log"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type (
	EnumType interface {
		protoreflect.Enum
		~int32
		String() string
	}
	MapKeyType interface {
		bool | int32 | int64 | uint32 | uint64 | string
	}
	MessageType[T any] interface {
		proto.Message
		*T
	}
	ScalarType interface {
		bool | []byte | int32 | int64 | uint32 | uint64 | float32 | float64 | string
	}
)

func ReflectFieldType(desc protoreflect.FieldDescriptor) reflect.Type {
	if desc == nil {
		log.Fatal("field descriptor cannot be nil")
	}

	if desc.IsList() {
		if desc.Enum() != nil {
			return reflect.SliceOf(ReflectEnumType(desc.Enum()))
		} else if desc.Message() != nil {
			return reflect.SliceOf(ReflectMessageType(desc.Message()))
		}
		return reflect.SliceOf(ReflectKindType(desc.Kind()))
	} else if desc.IsMap() {
		return reflect.MapOf(ReflectFieldType(desc.MapKey()), ReflectFieldType(desc.MapValue()))
	} else if desc.Enum() != nil {
		return ReflectEnumType(desc.Enum())
	} else if desc.Message() != nil {
		return ReflectMessageType(desc.Message())
	}

	return ReflectKindType(desc.Kind())
}

func ReflectMessageType(desc protoreflect.MessageDescriptor) reflect.Type {
	msgType, err := protoregistry.GlobalTypes.FindMessageByName(desc.FullName())
	if err != nil {
		log.Fatalf("message type: %s: %s", desc.FullName(), err)
	}
	return reflect.TypeOf(msgType.New().Interface())
}

func ReflectEnumType(desc protoreflect.EnumDescriptor) reflect.Type {
	enumType, err := protoregistry.GlobalTypes.FindEnumByName(desc.FullName())
	if err != nil {
		log.Fatalf("enum type: %s: %s", desc.FullName(), err)
	}
	return reflect.TypeOf(enumType.New(0))
}

func ReflectKindType(kind protoreflect.Kind) reflect.Type {
	switch kind {
	case protoreflect.BoolKind:
		return reflect.TypeFor[bool]()
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return reflect.TypeFor[int32]()
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return reflect.TypeFor[int64]()
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return reflect.TypeFor[uint32]()
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return reflect.TypeFor[uint64]()
	case protoreflect.FloatKind:
		return reflect.TypeFor[float32]()
	case protoreflect.DoubleKind:
		return reflect.TypeFor[float64]()
	case protoreflect.StringKind:
		return reflect.TypeFor[string]()
	case protoreflect.BytesKind:
		return reflect.TypeFor[[]byte]()
	}
	log.Fatalf("unknown field kind: %s", kind)
	return nil
}
