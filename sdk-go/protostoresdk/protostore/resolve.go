package protostore

import (
	"sync"

	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var resolver ResolverType
var resolverMut sync.Mutex

type ResolverType interface {
	protoregistry.ExtensionTypeResolver
	protoregistry.MessageTypeResolver
	GetValidator() (protovalidate.Validator, error)
}

func SetResolver(res ResolverType) {
	resolverMut.Lock()
	defer resolverMut.Unlock()
	resolver = res
}

func GetResolver() ResolverType {
	resolverMut.Lock()
	defer resolverMut.Unlock()
	if resolver != nil {
		return resolver
	}
	return api.GlobalResolver()
}
