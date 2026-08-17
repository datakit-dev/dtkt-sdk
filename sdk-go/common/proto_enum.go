package common

import (
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type ProtoEnum interface {
	~int32
	protoreflect.Enum
	String() string
}

func ProtoEnumNumberFromString(desc protoreflect.EnumDescriptor, name string) protoreflect.EnumNumber {
	var value protoreflect.EnumValueDescriptor
	for idx := range desc.Values().Len() {
		value = desc.Values().Get(idx)
		if value.Name() == protoreflect.Name(name) {
			return value.Number()
		}
	}
	return 0
}

func ProtoEnumNumberToString(desc protoreflect.EnumDescriptor, number protoreflect.EnumNumber) string {
	var value protoreflect.EnumValueDescriptor
	for idx := range desc.Values().Len() {
		value = desc.Values().Get(idx)
		if number == value.Number() {
			return string(value.Name())
		}
	}
	return ""
}

func ProtoEnumToString[T ProtoEnum](enum T, trimPrefix bool) string {
	if trimPrefix {
		return TrimProtoEnumPrefixFor[T](enum.String())
	}
	return enum.String()
}

func AddProtoEnumPrefixFor[T ProtoEnum](name string) string {
	return ProtoEnumPrefixFor[T]() + name
}

func TrimProtoEnumPrefixFor[T ProtoEnum](name string) string {
	var e T

	desc := e.Descriptor()
	if desc.Values().Len() == 0 {
		return ""
	}

	prefix := ProtoEnumPrefixFor[T]()
	for idx := range e.Descriptor().Values().Len() {
		if strings.EqualFold(string(e.Descriptor().Values().Get(idx).Name()), name) {
			return strings.TrimPrefix(string(e.Descriptor().Values().Get(idx).Name()), prefix)
		}
	}

	return ""
}

func ProtoEnumPrefixFor[T ProtoEnum]() string {
	var e T
	desc := e.Descriptor()
	if desc.Values().Len() == 0 {
		return ""
	}
	prefix, ok := strings.CutSuffix(string(desc.Values().ByNumber(0).Name()), "UNSPECIFIED")
	if ok {
		return prefix
	}
	return ""
}

func AddProtoEnumPrefix(desc protoreflect.EnumDescriptor, name string) string {
	return ProtoEnumPrefix(desc) + name
}

func TrimProtoEnumPrefix(desc protoreflect.EnumDescriptor, name string) string {
	return strings.TrimPrefix(name, ProtoEnumPrefix(desc))
}

func ProtoEnumPrefix(desc protoreflect.EnumDescriptor) string {
	return strings.TrimSuffix(string(desc.Values().ByNumber(0).Name()), "UNSPECIFIED")
}

func ProtoEnumStringsFor[T ProtoEnum]() (values []string) {
	var e T
	for idx := range e.Descriptor().Values().Len() {
		if idx > 0 {
			values = append(values, string(e.Descriptor().Values().Get(idx).Name()))
		}
	}
	return
}

func ProtoEnumFromStringFor[T ProtoEnum](val string) (enum T) {
	for idx := range enum.Descriptor().Values().Len() {
		enumVal := enum.Descriptor().Values().Get(idx)
		if string(enumVal.Name()) == val || TrimProtoEnumPrefixFor[T](string(enumVal.Name())) == val {
			return T(idx)
		}
	}
	return
}

func TrimProtoEnumStringsFor[T ProtoEnum]() (values []string) {
	var e T
	for idx := range e.Descriptor().Values().Len() {
		if idx > 0 {
			values = append(values, TrimProtoEnumPrefixFor[T](string(e.Descriptor().Values().Get(idx).Name())))
		}
	}
	return
}

func ProtoEnumStrings(e protoreflect.Enum) (values []string) {
	for idx := range e.Descriptor().Values().Len() {
		if idx > 0 {
			values = append(values, string(e.Descriptor().Values().Get(idx).Name()))
		}
	}
	return
}

func TrimProtoEnumStrings(e protoreflect.Enum) (values []string) {
	for idx := range e.Descriptor().Values().Len() {
		if idx > 0 {
			values = append(values, TrimProtoEnumPrefix(e.Descriptor(), string(e.Descriptor().Values().Get(idx).Name())))
		}
	}
	return
}
