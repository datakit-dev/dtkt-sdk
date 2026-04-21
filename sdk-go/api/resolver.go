package api

import (
	"buf.build/go/protovalidate"
	"github.com/jhump/protoreflect/v2/protoresolve"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Resolver interface {
	protoresolve.DependencyResolver
	protoresolve.FilePool
	protoresolve.TypeResolver
	FindMethodByName(protoreflect.FullName) (protoreflect.MethodDescriptor, error)
	FindServiceByName(protoreflect.FullName) (protoreflect.ServiceDescriptor, error)
	GetValidator() (protovalidate.Validator, error)
	RangeEnums(func(protoreflect.EnumType) bool)
	RangeExtensionsByMessage(protoreflect.FullName, func(protoreflect.ExtensionType) bool)
	RangeMessages(func(protoreflect.MessageType) bool)
	RangeMethods(func(protoreflect.MethodDescriptor) bool)
	RangeServices(func(protoreflect.ServiceDescriptor) bool)
}
