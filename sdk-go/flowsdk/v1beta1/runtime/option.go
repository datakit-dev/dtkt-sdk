package runtime

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
)

type (
	Option         func(OptionReceiver)
	OptionReceiver interface {
		applyOptions(...Option)
	}
)

func WithConnectors(conns shared.ConnectorProvider) Option {
	return func(r OptionReceiver) {
		if run, ok := r.(*Runtime); ok {
			run.conns = conns
		}
	}
}

func WithResolver(resolver shared.Resolver) Option {
	return func(r OptionReceiver) {
		if env, ok := r.(*Env); ok {
			env.resolver = resolver
		}
	}
}

func WithNodes(nodes NodeMap) Option {
	return func(r OptionReceiver) {
		if env, ok := r.(*Env); ok {
			env.nodes = nodes
		}
	}
}
