package protoform

import (
	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	Binding[T any] interface {
		Getter[T]
		Setter[T]
		Stringer[T]
		Parser[T]
		Descriptor() protoreflect.FieldDescriptor
		String() string
	}
	Getter[T any] interface {
		Get() T
	}
	Setter[T any] interface {
		Set(T)
	}
	Parser[T any] interface {
		Parse(string) (T, error)
	}
	Stringer[T any] interface {
		StringOf(T) string
	}
	GetterFunc[T any]   func() T
	SetterFunc[T any]   func(T)
	ParserFunc[T any]   func(string) (T, error)
	StringerFunc[T any] func(T) string
)

func ValidateMessage(env Env, root proto.Message) error {
	if root == nil {
		return nil
	}

	validator, err := env.Resolver().GetValidator()
	if err != nil {
		return err
	}

	return validator.Validate(root)
}

func ValidateField(env Env, root proto.Message, desc protoreflect.FieldDescriptor) error {
	if root == nil || desc == nil {
		return nil
	}

	validator, err := env.Resolver().GetValidator()
	if err != nil {
		return err
	}

	filter := api.ValidateFilter(root)
	return validator.Validate(root, protovalidate.WithFilter(
		protovalidate.FilterFunc(func(subMsg protoreflect.Message, subDesc protoreflect.Descriptor) bool {
			return filter.ShouldValidate(subMsg, subDesc) && desc == subDesc
		}),
	))
}

func (f GetterFunc[T]) Get() T {
	return f()
}

func (f SetterFunc[T]) Set(v T) {
	f(v)
}

func (f ParserFunc[T]) Parse(input string) (T, error) {
	return f(input)
}

func (f StringerFunc[T]) StringOf(v T) string {
	return f(v)
}
