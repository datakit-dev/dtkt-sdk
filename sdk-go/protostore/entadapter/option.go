package entadapter

import (
	"buf.build/go/protovalidate"
	"entgo.io/ent/schema"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostore/v1beta1"
)

type FieldOption func(FieldType)

func WithAnnotations(annotations ...schema.Annotation) FieldOption {
	return func(f FieldType) {
		desc := f.Descriptor()
		desc.Annotations = append(desc.Annotations, annotations...)
	}
}

func WithDefault(value any) FieldOption {
	return func(f FieldType) {
		desc := f.Descriptor()
		desc.Default = value
		desc.Optional = true
	}
}

func WithName(name string) FieldOption {
	return func(f FieldType) {
		if f.V1beta1() != nil {
			v1beta1.WithFieldName(name)(f.V1beta1())
		}
	}
}

func WithNullable(nullable bool) FieldOption {
	return func(f FieldType) {
		if f.V1beta1() != nil {
			v1beta1.WithFieldNullable(nullable)(f.V1beta1())
		}
	}
}

func WithUnique(unique bool) FieldOption {
	return func(f FieldType) {
		if f.V1beta1() != nil {
			v1beta1.WithFieldUnique(unique)(f.V1beta1())
		}
	}
}

func WithSensitive(sensitive bool) FieldOption {
	return func(f FieldType) {
		if f.V1beta1() != nil {
			v1beta1.WithFieldSensitive(sensitive)(f.V1beta1())
		}
	}
}

func WithImmutable(immutable bool) FieldOption {
	return func(f FieldType) {
		if f.V1beta1() != nil {
			v1beta1.WithFieldImmutable(immutable)(f.V1beta1())
		}
	}
}

func WithSkip(skip bool) FieldOption {
	return func(f FieldType) {
		if f.V1beta1() != nil {
			v1beta1.WithFieldSkip(skip)(f.V1beta1())
		}
	}
}

func WithValidation(opts ...protovalidate.ValidationOption) FieldOption {
	return func(f FieldType) {
		if f.V1beta1() != nil {
			v1beta1.WithFieldValidation(opts...)(f.V1beta1())
		}
	}
}
