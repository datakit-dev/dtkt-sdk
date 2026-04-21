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
		desc *ent.Descriptor
		a    = Annotation{
			ID: uuid.NewString(),
		}
	)
	if field.Descriptor().IsList() {
		a.IsList = true

		if field.Descriptor().Enum() != nil {
			a.ProtoName = string(field.Descriptor().Enum().FullName())
			a.IsEnum = true
		} else if field.Descriptor().Message() != nil {
			a.ProtoName = string(field.Descriptor().Message().FullName())
			a.IsMessage = true
		}

		desc = ent.
			JSON(field.Proto().GetName(), reflect.Zero(field.ReflectType()).Interface()).
			Annotations(a).
			Descriptor()
	} else if field.Descriptor().IsMap() {
		a.IsMap = true

		if field.Descriptor().MapValue().Enum() != nil {
			a.ProtoName = string(field.Descriptor().MapValue().Enum().FullName())
			a.IsEnum = true
		} else if field.Descriptor().MapValue().Message() != nil {
			a.ProtoName = string(field.Descriptor().MapValue().Message().FullName())
			a.IsMessage = true
		}

		desc = ent.
			JSON(field.Proto().GetName(), reflect.Zero(field.ReflectType()).Interface()).
			Annotations(a).
			Descriptor()
	} else if field.Descriptor().Message() != nil {
		a.ProtoName = string(field.Descriptor().Message().FullName())
		a.IsMessage = true

		desc = ent.
			JSON(field.Proto().GetName(), reflect.Zero(field.ReflectType()).Interface()).
			Annotations(a).
			Descriptor()
	} else if field.Descriptor().Enum() != nil {
		desc = ent.
			Int32(field.Proto().GetName()).
			GoType(reflect.Zero(field.ReflectType()).Interface()).
			Annotations(Annotation{
				ProtoName: string(field.Descriptor().Enum().FullName()),
				IsEnum:    true,
			}).
			Descriptor()
	} else {
		switch field.Descriptor().Kind() {
		case protoreflect.BoolKind:
			desc = ent.Bool(field.Proto().GetName()).Descriptor()
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			desc = ent.Int32(field.Proto().GetName()).Descriptor()
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			desc = ent.Int64(field.Proto().GetName()).Descriptor()
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			desc = ent.Uint32(field.Proto().GetName()).Descriptor()
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			desc = ent.Uint64(field.Proto().GetName()).Descriptor()
		case protoreflect.FloatKind:
			desc = ent.Float32(field.Proto().GetName()).Descriptor()
		case protoreflect.DoubleKind:
			desc = ent.Float(field.Proto().GetName()).Descriptor()
		case protoreflect.StringKind:
			desc = ent.String(field.Proto().GetName()).Descriptor()
		case protoreflect.BytesKind:
			desc = ent.Bytes(field.Proto().GetName()).Descriptor()
		}
	}

	f := &Field{
		field: field,
		desc:  desc,
	}

	f.applyOptions(opts...)

	fieldCache.Store(a.ID, f)

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
