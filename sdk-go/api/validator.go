package api

import (
	"slices"
	"strings"
	"sync"

	"buf.build/go/protovalidate"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

var _ protovalidate.Validator = (ValidateFunc)(nil)

type (
	Validator interface {
		protovalidate.Validator
		ValidateInput(proto.Message, ...protovalidate.ValidationOption) error
		ValidateOutput(proto.Message, ...protovalidate.ValidationOption) error
	}
	ValidateFunc func(proto.Message, ...protovalidate.ValidationOption) error
	validator    struct {
		resolver  Resolver
		validator func() (protovalidate.Validator, error)
	}
)

func NewValidator(resolver Resolver) *validator {
	return &validator{
		resolver: resolver,
		validator: sync.OnceValues(func() (protovalidate.Validator, error) {
			return protovalidate.New(
				protovalidate.WithExtensionTypeResolver(resolver),
			)
		}),
	}
}

func ValidateInputFilter() protovalidate.Filter {
	return protovalidate.FilterFunc(func(msg protoreflect.Message, desc protoreflect.Descriptor) bool {
		if opts, ok := desc.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
			if proto.HasExtension(opts, annotations.E_FieldBehavior) {
				if ext, ok := proto.GetExtension(opts, annotations.E_FieldBehavior).([]annotations.FieldBehavior); ok && len(ext) > 0 {
					if slices.Contains(ext, annotations.FieldBehavior_IDENTIFIER) || slices.Contains(ext, annotations.FieldBehavior_OUTPUT_ONLY) {
						return false
					}
				}
			}
		}
		return true
	})
}

func ValidateOutputFilter() protovalidate.Filter {
	return protovalidate.FilterFunc(func(msg protoreflect.Message, desc protoreflect.Descriptor) bool {
		if opts, ok := desc.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
			if proto.HasExtension(opts, annotations.E_FieldBehavior) {
				if ext, ok := proto.GetExtension(opts, annotations.E_FieldBehavior).([]annotations.FieldBehavior); ok && len(ext) > 0 {
					if slices.Contains(ext, annotations.FieldBehavior_INPUT_ONLY) {
						return false
					}
				}
			}
		}
		return true
	})
}

func ValidateFilter(msg proto.Message) protovalidate.Filter {
	if strings.HasSuffix(string(msg.ProtoReflect().Descriptor().Name()), "Request") {
		return ValidateInputFilter()
	} else if strings.HasSuffix(string(msg.ProtoReflect().Descriptor().Name()), "Response") {
		return ValidateOutputFilter()
	}
	return protovalidate.FilterFunc(func(protoreflect.Message, protoreflect.Descriptor) bool {
		return true
	})
}

func (f *validator) Validate(msg proto.Message, opts ...protovalidate.ValidationOption) error {
	validator, err := f.validator()
	if err != nil {
		return err
	}

	return validator.Validate(msg, append([]protovalidate.ValidationOption{
		protovalidate.WithFilter(ValidateFilter(msg)),
	}, opts...)...)
}

func (f *validator) ValidateInput(input proto.Message, opts ...protovalidate.ValidationOption) error {
	validator, err := f.validator()
	if err != nil {
		return err
	}

	return validator.Validate(input, append([]protovalidate.ValidationOption{
		protovalidate.WithFilter(ValidateInputFilter()),
	}, opts...)...)
}

func (f *validator) ValidateOutput(output proto.Message, opts ...protovalidate.ValidationOption) error {
	validator, err := f.validator()
	if err != nil {
		return err
	}

	return validator.Validate(output, append([]protovalidate.ValidationOption{
		protovalidate.WithFilter(ValidateOutputFilter()),
	}, opts...)...)
}

func (f ValidateFunc) Validate(msg proto.Message, opts ...protovalidate.ValidationOption) error {
	return f(msg, opts...)
}
