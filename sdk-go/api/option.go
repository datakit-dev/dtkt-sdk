package api

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
)

func WithSchemaOpts[T SpecType](opts ...common.JSONSchemaOpt) SpecLoaderOpt[T] {
	return func(l *SpecLoader[T]) {
		l.schemaOpts = opts
	}
}
