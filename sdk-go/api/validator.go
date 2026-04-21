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

var getGlobalValidator = sync.OnceValues(func() (protovalidate.Validator, error) {
	return protovalidate.New(protovalidate.WithExtensionTypeResolver(getGlobalResolver()))
})

type (
	RequestValidator struct {
		resolver Resolver
	}
	RequestFieldValidator struct {
		request proto.Message
	}
)

func GlobalValidator() (protovalidate.Validator, error) {
	return getGlobalValidator()
}

func NewRequestValidator(resolver Resolver) *RequestValidator {
	return &RequestValidator{
		resolver: resolver,
	}
}

func (f *RequestValidator) Validate(request proto.Message, opts ...protovalidate.ValidationOption) error {
	validator, err := f.resolver.GetValidator()
	if err != nil {
		return err
	}

	opts = append([]protovalidate.ValidationOption{
		protovalidate.WithFilter(&RequestFieldValidator{request: request}),
	}, opts...)

	return validator.Validate(request, opts...)
}

func (f *RequestFieldValidator) ShouldValidate(msg protoreflect.Message, desc protoreflect.Descriptor) bool {
	if opts, ok := desc.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
		if proto.HasExtension(opts, annotations.E_FieldBehavior) {
			if ext, ok := proto.GetExtension(opts, annotations.E_FieldBehavior).([]annotations.FieldBehavior); ok && len(ext) > 0 {
				if (strings.HasPrefix(string(f.request.ProtoReflect().Descriptor().Name()), "Create") &&
					slices.Contains(ext, annotations.FieldBehavior_IDENTIFIER)) ||
					(strings.HasSuffix(string(f.request.ProtoReflect().Descriptor().Name()), "Request") &&
						slices.Contains(ext, annotations.FieldBehavior_OUTPUT_ONLY)) {
					return false
				}
			}
		}
	}
	return true
}
