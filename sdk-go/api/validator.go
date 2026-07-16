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

type validator struct {
	resolver  Resolver
	validator func() (protovalidate.Validator, error)
}

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

func ValidatorFilter(root proto.Message) protovalidate.Filter {
	return protovalidate.FilterFunc(func(msg protoreflect.Message, desc protoreflect.Descriptor) bool {
		if opts, ok := desc.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
			if proto.HasExtension(opts, annotations.E_FieldBehavior) {
				if ext, ok := proto.GetExtension(opts, annotations.E_FieldBehavior).([]annotations.FieldBehavior); ok && len(ext) > 0 {
					if (strings.HasPrefix(string(root.ProtoReflect().Descriptor().Name()), "Create") &&
						slices.Contains(ext, annotations.FieldBehavior_IDENTIFIER)) ||
						(strings.HasSuffix(string(root.ProtoReflect().Descriptor().Name()), "Request") &&
							slices.Contains(ext, annotations.FieldBehavior_OUTPUT_ONLY)) {
						return false
					}
				}
			}
		}
		return true
	})
}

func (f *validator) Validate(root proto.Message, opts ...protovalidate.ValidationOption) error {
	validator, err := f.validator()
	if err != nil {
		return err
	}

	opts = append([]protovalidate.ValidationOption{
		protovalidate.WithFilter(ValidatorFilter(root)),
	}, opts...)

	return validator.Validate(root, opts...)
}
