package api

import (
	"fmt"
	"sync"

	"github.com/jhump/protoreflect/v2/protoresolve"
	"github.com/jhump/protoreflect/v2/sourceinfo"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var getGlobalResolver = sync.OnceValue(func() *resolver {
	return &resolver{
		DescriptorPool: sourceinfo.Files,
		TypePool:       sourceinfo.Types,
	}
})

type (
	Resolver interface {
		protoresolve.DependencyResolver
		protoresolve.FilePool
		protoresolve.TypeResolver
		FindMethodByName(protoreflect.FullName) (protoreflect.MethodDescriptor, error)
		FindServiceByName(protoreflect.FullName) (protoreflect.ServiceDescriptor, error)
		GetValidator() (Validator, error)
		RangeEnums(func(protoreflect.EnumType) bool)
		RangeExtensionsByMessage(protoreflect.FullName, func(protoreflect.ExtensionType) bool)
		RangeMessages(func(protoreflect.MessageType) bool)
		RangeMethods(func(protoreflect.MethodDescriptor) bool)
		RangeServices(func(protoreflect.ServiceDescriptor) bool)
	}
	resolver struct {
		protoresolve.DescriptorPool
		protoresolve.TypePool
		validator Validator
	}
)

func NewResolver(files protoresolve.DescriptorPool, types protoresolve.TypePool) *resolver {
	r := &resolver{
		DescriptorPool: files,
		TypePool:       types,
	}
	r.validator = NewValidator(r)
	return r
}

func GlobalResolver() *resolver {
	return getGlobalResolver()
}

func (r *resolver) GetValidator() (Validator, error) {
	return r.validator, nil
}

func (r *resolver) FindServiceByName(name protoreflect.FullName) (protoreflect.ServiceDescriptor, error) {
	desc, err := r.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}

	sd, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("method not found: %s", name)
	}

	return sd, nil
}

func (r *resolver) FindMethodByName(name protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
	desc, err := r.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}

	md, ok := desc.(protoreflect.MethodDescriptor)
	if !ok {
		return nil, fmt.Errorf("method not found: %s", name)
	}

	return md, nil
}

func (r *resolver) RangeServices(f func(protoreflect.ServiceDescriptor) bool) {
	r.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for idx := range fd.Services().Len() {
			if !f(fd.Services().Get(idx)) {
				return false
			}
		}
		return true
	})
}

func (r *resolver) RangeMethods(f func(protoreflect.MethodDescriptor) bool) {
	r.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := range fd.Services().Len() {
			for j := range fd.Services().Get(i).Methods().Len() {
				if !f(fd.Services().Get(i).Methods().Get(j)) {
					return false
				}
			}
		}
		return true
	})
}
