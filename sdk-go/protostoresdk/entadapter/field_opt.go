package entadapter

import (
	"buf.build/go/protovalidate"
	"entgo.io/ent/schema"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/v1beta1"
)

type FieldOption func(FieldType)

func applyFieldOptions(f FieldType, opts ...FieldOption) {
	for _, opt := range opts {
		if opt != nil {
			opt(f)
		}
	}

	if f.Field().Proto().GetName() != "" && f.Field().Proto().GetName() != f.Descriptor().Name {
		f.Descriptor().Name = f.Field().Proto().GetName()
	}

	if f.Field().Proto().Optional != nil {
		f.Descriptor().Optional = f.Field().Proto().GetOptional()
	} else {
		if f, ok := f.(*Field); ok {
			f.Descriptor().Optional = f.field.Descriptor().IsList() || f.field.Descriptor().IsMap() || (f.field.Descriptor().ContainingOneof() != nil && !f.field.Descriptor().ContainingOneof().IsSynthetic())
		}
	}

	if f.Field().Proto().Nillable != nil {
		f.Descriptor().Nillable = f.Field().Proto().GetNillable()
	}

	if f.Field().Proto().Unique != nil {
		f.Descriptor().Unique = f.Field().Proto().GetUnique()
	}

	if f.Field().Proto().Sensitive != nil {
		f.Descriptor().Sensitive = f.Field().Proto().GetSensitive()
	}

	if f.Field().Proto().Immutable != nil {
		f.Descriptor().Immutable = f.Field().Proto().GetImmutable()
	}
}

func WithAnnotations(annotations ...schema.Annotation) FieldOption {
	return func(f FieldType) {
		desc := f.Descriptor()
		if desc != nil {
			desc.Annotations = annotations
		}
	}
}

func WithDefault(value any) FieldOption {
	return func(f FieldType) {
		desc := f.Descriptor()
		if desc != nil {
			desc.Default = value
		}
	}
}

func WithName(name string) FieldOption {
	return func(f FieldType) {
		opt := v1beta1.WithFieldName(name)
		if opt != nil {
			opt(f.Field())
		}
	}
}

func WithOptional(optional bool) FieldOption {
	return func(f FieldType) {
		opt := v1beta1.WithFieldOptional(optional)
		if opt != nil {
			opt(f.Field())
		}
	}
}

func WithUnique(unique bool) FieldOption {
	return func(f FieldType) {
		opt := v1beta1.WithFieldUnique(unique)
		if opt != nil {
			opt(f.Field())
		}
	}
}

func WithSensitive(sensitive bool) FieldOption {
	return func(f FieldType) {
		opt := v1beta1.WithFieldSensitive(sensitive)
		if opt != nil {
			opt(f.Field())
		}
	}
}

func WithImmutable(immutable bool) FieldOption {
	return func(f FieldType) {
		opt := v1beta1.WithFieldImmutable(immutable)
		if opt != nil {
			opt(f.Field())
		}
	}
}

func WithNillable(nillable bool) FieldOption {
	return func(f FieldType) {
		opt := v1beta1.WithFieldNillable(nillable)
		if opt != nil {
			opt(f.Field())
		}
	}
}

func WithSkip(skip bool) FieldOption {
	return func(f FieldType) {
		opt := v1beta1.WithFieldSkip(skip)
		if opt != nil {
			opt(f.Field())
		}
	}
}

func WithValidation(opts ...protovalidate.ValidationOption) FieldOption {
	return func(f FieldType) {
		opt := v1beta1.WithFieldValidation(opts...)
		if opt != nil {
			opt(f.Field())
		}
	}
}
