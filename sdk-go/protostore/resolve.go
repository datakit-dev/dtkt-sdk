package protostore

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
)

var Resolver api.Resolver

func init() {
	Resolver = api.GlobalResolver()
}
