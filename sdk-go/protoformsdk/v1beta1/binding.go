package v1beta1

import (
	"buf.build/go/protovalidate"
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

func ValidateMessage(env Env, msg protoreflect.Message) error {
	if msg == nil {
		return nil
	}

	validator, err := env.Resolver().GetValidator()
	if err != nil {
		return err
	}

	return validator.Validate(msg.Interface())
}

func ValidateField(env Env, msg protoreflect.Message, field protoreflect.FieldDescriptor) error {
	if msg == nil || field == nil {
		return nil
	}

	validator, err := env.Resolver().GetValidator()
	if err != nil {
		return err
	}

	return validator.Validate(msg.Interface(), protovalidate.WithFilter(
		protovalidate.FilterFunc(func(msg protoreflect.Message, desc protoreflect.Descriptor) bool {
			return field == desc
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
