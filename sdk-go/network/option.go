package network

import (
	"crypto/tls"

	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
)

type (
	Option   func(Optionee)
	Optionee interface {
		applyOptions(...Option)
	}
)

func WithCloudResource(resource *corev1.Resource) Option {
	return func(o Optionee) {
		switch o := o.(type) {
		case *CloudConnector:
			o.resource = resource
		}
	}
}

func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(r Optionee) {
		switch c := r.(type) {
		case *defaultConnector:
			c.tlsConfig = tlsConfig
		case *HTTPClient:
			c.dialTLS = tlsConfig != nil
		}
	}
}
