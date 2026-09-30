package entadapter

import (
	"entgo.io/ent"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostore/v1beta1"
)

var _ FieldType = (*Field[any])(nil)

type (
	Field[T any] struct {
		ent.Field
		v1beta1 *v1beta1.Field
	}
	FieldType interface {
		ent.Field
		V1beta1() *v1beta1.Field
		applyOptions(...FieldOption)
	}
)

func NewFieldOf[T any](field ent.Field, opts ...FieldOption) *Field[T] {
	f := &Field[T]{
		Field:   field,
		v1beta1: v1beta1.NewField(nil),
	}
	f.applyOptions(opts...)
	return f
}

func (f *Field[T]) V1beta1() *v1beta1.Field {
	return f.v1beta1
}

func (f *Field[T]) applyOptions(opts ...FieldOption) {
	for _, opt := range opts {
		if opt != nil {
			opt(f)
		}
	}

	if f.V1beta1() != nil {
		if f.V1beta1().Proto().GetName() != "" {
			f.Descriptor().Name = f.V1beta1().Proto().GetName()
		}

		f.Descriptor().Optional = f.V1beta1().Proto().GetNullable()
		f.Descriptor().Nillable = f.V1beta1().Proto().GetNullable()
		f.Descriptor().Unique = f.V1beta1().Proto().GetUnique()
		f.Descriptor().Sensitive = f.V1beta1().Proto().GetSensitive()
		f.Descriptor().Immutable = f.V1beta1().Proto().GetImmutable()
	}
}
