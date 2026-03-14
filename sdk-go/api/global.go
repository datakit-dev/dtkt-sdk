package api

import (
	"fmt"
	"sync"

	"buf.build/go/protovalidate"
	"github.com/jhump/protoreflect/v2/protoresolve"
	"github.com/jhump/protoreflect/v2/sourceinfo"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var getGlobalResolver = sync.OnceValue(func() *globalResolver {
	return &globalResolver{
		DescriptorPool: sourceinfo.Files,
		TypePool:       sourceinfo.Types,
	}
})

var getGlobalValidator = sync.OnceValues(func() (protovalidate.Validator, error) {
	return protovalidate.New(protovalidate.WithExtensionTypeResolver(getGlobalResolver()))
})

type globalResolver struct {
	protoresolve.DescriptorPool
	protoresolve.TypePool
}

func GlobalResolver() *globalResolver {
	return getGlobalResolver()
}

func GlobalValidator() (protovalidate.Validator, error) {
	return getGlobalValidator()
}

func (r *globalResolver) GetValidator() (protovalidate.Validator, error) {
	return GlobalValidator()
}

func (r *globalResolver) FindServiceByName(name protoreflect.FullName) (protoreflect.ServiceDescriptor, error) {
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

func (r *globalResolver) FindMethodByName(name protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
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

func (r *globalResolver) RangeServices(f func(protoreflect.ServiceDescriptor) bool) {
	r.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for idx := range fd.Services().Len() {
			if !f(fd.Services().Get(idx)) {
				return false
			}
		}
		return true
	})
}

func (r *globalResolver) RangeMethods(f func(protoreflect.MethodDescriptor) bool) {
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
