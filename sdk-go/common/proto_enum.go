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

func AddProtoEnumPrefixFor[T protoreflect.Enum](name string) string {
	return ProtoEnumPrefixFor[T]() + name
}

func TrimProtoEnumPrefixFor[T protoreflect.Enum](name string) string {
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

func ProtoEnumPrefixFor[T protoreflect.Enum]() string {
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

func ProtoEnumStringsFor[T protoreflect.Enum]() (values []string) {
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

func TrimProtoEnumStringsFor[T protoreflect.Enum]() (values []string) {
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
