package v1beta1

import (
	"buf.build/go/protovalidate"
)

func WithFieldName(name string) FieldOption {
	return func(f *Field) {
		f.proto.Name = new(name)
	}
}

func WithFieldUnique(unique bool) FieldOption {
	return func(f *Field) {
		f.proto.Unique = new(unique)
	}
}

func WithFieldSensitive(sensitive bool) FieldOption {
	return func(f *Field) {
		f.proto.Sensitive = new(sensitive)
	}
}

func WithFieldImmutable(immutable bool) FieldOption {
	return func(f *Field) {
		f.proto.Immutable = new(immutable)
	}
}

func WithFieldNullable(nullable bool) FieldOption {
	return func(f *Field) {
		f.proto.Nullable = new(nullable)
	}
}

func WithFieldSkip(skip bool) FieldOption {
	return func(f *Field) {
		f.proto.Skip = new(skip)
	}
}

func WithFieldValidation(opts ...protovalidate.ValidationOption) FieldOption {
	return func(f *Field) {
		f.opts = opts
	}
}
