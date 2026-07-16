package v1beta1

import (
	"slices"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func isDescriptorValue(desc protoreflect.FieldDescriptor, value protoreflect.Value) bool {
	switch value.Interface().(type) {
	case protoreflect.List:
		return desc != nil && desc.IsList() && slices.Contains(valueKinds(value), desc.Kind())
	case protoreflect.Map:
		return desc != nil && desc.IsMap() && slices.Contains(valueKinds(value), desc.MapValue().Kind())
	case protoreflect.Message:
		return desc != nil && desc.Message() == value.Message().Descriptor()
	}
	return desc != nil && slices.Contains(valueKinds(value), desc.Kind())
}

func valueKinds(value protoreflect.Value) (_ []protoreflect.Kind) {
	switch value := value.Interface().(type) {
	case bool:
		return []protoreflect.Kind{protoreflect.BoolKind}
	case int32:
		return []protoreflect.Kind{protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind}
	case int64:
		return []protoreflect.Kind{protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind}
	case uint32:
		return []protoreflect.Kind{protoreflect.Uint32Kind, protoreflect.Fixed32Kind}
	case uint64:
		return []protoreflect.Kind{protoreflect.Uint64Kind, protoreflect.Fixed64Kind}
	case float32:
		return []protoreflect.Kind{protoreflect.FloatKind}
	case float64:
		return []protoreflect.Kind{protoreflect.DoubleKind}
	case string:
		return []protoreflect.Kind{protoreflect.StringKind}
	case []byte:
		return []protoreflect.Kind{protoreflect.BytesKind}
	case protoreflect.EnumNumber:
		return []protoreflect.Kind{protoreflect.EnumKind}
	case protoreflect.List:
		return valueKinds(value.NewElement())
	case protoreflect.Map:
		return valueKinds(value.NewValue())
	case protoreflect.Message:
		return []protoreflect.Kind{protoreflect.MessageKind, protoreflect.GroupKind}
	}
	return
}
