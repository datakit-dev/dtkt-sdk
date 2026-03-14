package common

import (
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

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

func ProtoEnumStringOptionsFor[T protoreflect.Enum]() (values []string) {
	var e T
	for idx := range e.Descriptor().Values().Len() {
		if idx > 0 {
			values = append(values, string(e.Descriptor().Values().Get(idx).Name()))
		}
	}
	return
}

func TrimProtoEnumStringOptionsFor[T protoreflect.Enum]() (values []string) {
	var e T
	for idx := range e.Descriptor().Values().Len() {
		if idx > 0 {
			values = append(values, TrimProtoEnumPrefixFor[T](string(e.Descriptor().Values().Get(idx).Name())))
		}
	}
	return
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

func ProtoEnumStringOptions(e protoreflect.Enum) (values []string) {
	for idx := range e.Descriptor().Values().Len() {
		if idx > 0 {
			values = append(values, string(e.Descriptor().Values().Get(idx).Name()))
		}
	}
	return
}

func TrimProtoEnumStringOptions(e protoreflect.Enum) (values []string) {
	for idx := range e.Descriptor().Values().Len() {
		if idx > 0 {
			values = append(values, TrimProtoEnumPrefix(e.Descriptor(), string(e.Descriptor().Values().Get(idx).Name())))
		}
	}
	return
}
