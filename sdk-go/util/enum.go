package util

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func EnumNumberFromString(field protoreflect.FieldDescriptor, name string) (protoreflect.EnumNumber, error) {
	var value protoreflect.EnumValueDescriptor
	for idx := range field.Enum().Values().Len() {
		value = field.Enum().Values().Get(idx)
		if value.Name() == protoreflect.Name(name) {
			return value.Number(), nil
		}
	}
	return 0, fmt.Errorf("invalid enum for field %s: %s", field.FullName(), name)
}

func EnumNumberToString(field protoreflect.FieldDescriptor, number protoreflect.EnumNumber) (name string) {
	var value protoreflect.EnumValueDescriptor
	for idx := range field.Enum().Values().Len() {
		value = field.Enum().Values().Get(idx)
		if number == value.Number() {
			return string(value.Name())
		}
	}
	return
}
