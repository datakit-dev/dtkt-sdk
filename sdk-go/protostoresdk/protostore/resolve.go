package protostore

import (
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
)

var resolver api.Resolver
var resolverMut sync.Mutex

func SetResolver(res api.Resolver) {
	resolverMut.Lock()
	defer resolverMut.Unlock()
	resolver = res
}

func GetResolver() api.Resolver {
	resolverMut.Lock()
	defer resolverMut.Unlock()
	if resolver != nil {
		return resolver
	}
	return api.GlobalResolver()
}
