package runtime

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

type (
	Option         func(OptionReceiver)
	OptionReceiver interface {
		applyOptions(...Option)
	}
	GraphOption    = shared.GraphOption[*Node]
	ExecutorOption = shared.ExecutorOption[*Node]
)

func WithEnv(proto *flowv1beta1.Env) Option {
	return func(recv OptionReceiver) {
		if env, ok := recv.(*Env); ok {
			env.proto = proto
		}
	}
}

func WithVars(vars EnvVars) Option {
	return func(recv OptionReceiver) {
		if env, ok := recv.(*Env); ok {
			env.vars = vars
		}
	}
}

func WithConnectors(conns ConnectorProvider) Option {
	return func(recv OptionReceiver) {
		if env, ok := recv.(*Env); ok {
			env.conns = conns
		}
	}
}

func WithClient(client shared.Client) Option {
	return func(recv OptionReceiver) {
		if conn, ok := recv.(*Connector); ok {
			conn.client = client
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
