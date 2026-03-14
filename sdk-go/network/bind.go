package network

import (
	"context"
	"net"
)

type Binder interface {
	Address() net.Addr
	Bind(context.Context) (net.Listener, error)
	Close() error
}

func Bind(ctx context.Context, addr Address, opts ...ConnectorOption) (net.Listener, error) {
	binder, err := NewConnector(addr, opts...)
	if err != nil {
		return nil, err
	}
	return binder.Bind(ctx)
}

func BindUsing(ctx context.Context, binder Binder) (net.Listener, error) {
	return binder.Bind(ctx)
}
