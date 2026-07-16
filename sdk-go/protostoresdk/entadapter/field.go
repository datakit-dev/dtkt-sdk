package entadapter

import (
	"reflect"

	ent "entgo.io/ent/schema/field"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/uuid"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var fieldCache util.SyncMap[string, *Field]

type (
	FieldType interface {
		Field() *v1beta1.Field
		Descriptor() *ent.Descriptor
		applyOptions(...FieldOption)
	}
	Field struct {
		field *v1beta1.Field
		desc  *ent.Descriptor
	}
	fieldType[T any] struct {
		field *v1beta1.Field
		desc  *ent.Descriptor
	}
)

func NewField(field *v1beta1.Field, opts ...FieldOption) *Field {
	var (
		descriptor *ent.Descriptor
		annotation = Annotation{
			ID: uuid.NewString(),
		}
	)
	if field.Descriptor().IsList() {
		annotation.IsList = true

		if field.Descriptor().Enum() != nil {
			annotation.ProtoName = string(field.Descriptor().Enum().FullName())
			annotation.IsEnum = true
		} else if field.Descriptor().Message() != nil {
			annotation.ProtoName = string(field.Descriptor().Message().FullName())
			annotation.IsMessage = true
		}

		descriptor = ent.
			JSON(field.Proto().GetName(), reflect.Zero(field.Type()).Interface()).
			Annotations(annotation).
			Descriptor()
	} else if field.Descriptor().IsMap() {
		annotation.IsMap = true

		if field.Descriptor().MapValue().Enum() != nil {
			annotation.ProtoName = string(field.Descriptor().MapValue().Enum().FullName())
			annotation.IsEnum = true
		} else if field.Descriptor().MapValue().Message() != nil {
			annotation.ProtoName = string(field.Descriptor().MapValue().Message().FullName())
			annotation.IsMessage = true
		}

		descriptor = ent.
			JSON(field.Proto().GetName(), reflect.Zero(field.Type()).Interface()).
			Annotations(annotation).
			Descriptor()
	} else if field.Descriptor().Message() != nil {
		annotation.ProtoName = string(field.Descriptor().Message().FullName())
		annotation.IsMessage = true

		descriptor = ent.
			JSON(field.Proto().GetName(), reflect.Zero(field.Type()).Interface()).
			Annotations(annotation).
			Descriptor()
	} else if field.Descriptor().Enum() != nil {
		descriptor = ent.
			Int32(field.Proto().GetName()).
			GoType(reflect.Zero(field.Type()).Interface()).
			Annotations(Annotation{
				ProtoName: string(field.Descriptor().Enum().FullName()),
				IsEnum:    true,
			}).
			Descriptor()
	} else {
		switch field.Descriptor().Kind() {
		case protoreflect.BoolKind:
			descriptor = ent.Bool(field.Proto().GetName()).Descriptor()
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			descriptor = ent.Int32(field.Proto().GetName()).Descriptor()
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			descriptor = ent.Int64(field.Proto().GetName()).Descriptor()
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			descriptor = ent.Uint32(field.Proto().GetName()).Descriptor()
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			descriptor = ent.Uint64(field.Proto().GetName()).Descriptor()
		case protoreflect.FloatKind:
			descriptor = ent.Float32(field.Proto().GetName()).Descriptor()
		case protoreflect.DoubleKind:
			descriptor = ent.Float(field.Proto().GetName()).Descriptor()
		case protoreflect.StringKind:
			descriptor = ent.String(field.Proto().GetName()).Descriptor()
		case protoreflect.BytesKind:
			descriptor = ent.Bytes(field.Proto().GetName()).Descriptor()
		}
	}

	f := &Field{
		field: field,
		desc:  descriptor,
	}

	f.applyOptions(opts...)

	fieldCache.Store(annotation.ID, f)

	return f
}

func (f *Field) Field() *v1beta1.Field {
	return f.field
}

func (f *Field) Descriptor() *ent.Descriptor {
	return f.desc
}

func (f *fieldType[V]) Field() *v1beta1.Field {
	return f.field
}

func (f *fieldType[V]) Descriptor() *ent.Descriptor {
	return f.desc
}

func (f *Field) applyOptions(opts ...FieldOption) {
	applyFieldOptions(f, opts...)
}

func (f *fieldType[V]) applyOptions(opts ...FieldOption) {
	applyFieldOptions(f, opts...)
}
