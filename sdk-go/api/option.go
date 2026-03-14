package api

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
)

func WithSchemaOpts[T SpecType](opts ...common.JSONSchemaOpt) SpecLoaderOpt[T] {
	return func(l *SpecLoader[T]) {
		l.schemaOpts = opts
	}
}

func WithEncoder[T SpecType](enc encoding.Encoder) SpecLoaderOpt[T] {
	return func(r *SpecLoader[T]) {
		r.encoder = enc
	}
}

func WithDecodeFunc[T SpecType](dec encoding.Decoder) SpecLoaderOpt[T] {
	return func(r *SpecLoader[T]) {
		r.decoder = dec
	}
}
