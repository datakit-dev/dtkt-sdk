package v1beta1

import (
	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/proto"
)

func WithFieldName(name string) FieldOption {
	return func(f *Field) {
		f.proto.Name = new(name)
	}
}

func WithFieldOptional(optional bool) FieldOption {
	return func(f *Field) {
		f.proto.Optional = proto.Bool(optional)
	}
}

func WithFieldUnique(unique bool) FieldOption {
	return func(f *Field) {
		f.proto.Unique = proto.Bool(unique)
	}
}

func WithFieldSensitive(sensitive bool) FieldOption {
	return func(f *Field) {
		f.proto.Sensitive = proto.Bool(sensitive)
	}
}

func WithFieldImmutable(immutable bool) FieldOption {
	return func(f *Field) {
		f.proto.Immutable = proto.Bool(immutable)
	}
}

func WithFieldNillable(nillable bool) FieldOption {
	return func(f *Field) {
		f.proto.Nillable = proto.Bool(nillable)
	}
}

func WithFieldSkip(skip bool) FieldOption {
	return func(f *Field) {
		f.proto.Skip = proto.Bool(skip)
	}
}

func WithFieldValidation(opts ...protovalidate.ValidationOption) FieldOption {
	return func(f *Field) {
		f.opts = opts
	}
}
