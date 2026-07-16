package runtime

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
)

type (
	Option         func(OptionReceiver)
	OptionReceiver interface {
		applyOptions(...Option)
	}
	GraphOption    = shared.GraphOption[*Node]
	ExecutorOption = shared.ExecutorOption[*Node]
)

func WithConnectors(conns ConnectorProvider) Option {
	return func(recv OptionReceiver) {
		if run, ok := recv.(*Env); ok {
			run.conns = conns
		}
	}
}

func WithNodes(nodes NodeMap) Option {
	return func(recv OptionReceiver) {
		if env, ok := recv.(*Env); ok {
			env.nodes = nodes
		}
	}
}

func WithClient(client shared.Client) Option {
	return func(recv OptionReceiver) {
		if c, ok := recv.(*Connector); ok {
			c.client = client
		}
	}
}

func WithResolver(resolver shared.Resolver) Option {
	return func(recv OptionReceiver) {
		switch recv := recv.(type) {
		case *Env:
			recv.resolver = resolver
		case *Connector:
			recv.resolver = resolver
		}
	}
}
