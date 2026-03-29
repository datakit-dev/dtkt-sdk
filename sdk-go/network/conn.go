package network

import (
	"context"
	"errors"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
)

var _ Connector = (*connector)(nil)

type (
	Connector interface {
		Binder
		Dialer
	}
	connector struct {
		addr Address
		opts connectorOptions
	}
)

func NewConnector(addr Address, opts ...ConnectorOption) (Connector, error) {
	switch addr.Network() {
	case Cloud.String():
		return NewCloudConnector(addr, opts...)
	case SSHSocket.String(), SSHTCP.String():
		return NewSSHConnector(addr)
	}

	conn := &connector{
		addr: addr,
	}

	conn.opts.apply(opts...)

	return conn, nil
}

func (c *connector) Address() net.Addr {
	return c.addr
}

func (c *connector) Bind(context.Context) (net.Listener, error) {
	lis, err := net.Listen(c.addr.Network(), c.addr.String())
	if err != nil {
		return nil, err
	}

	if c.addr.Network() == Socket.String() && c.opts.close == nil {
		c.opts.close = func() error {
			return errors.Join(lis.Close(), os.Remove(c.Address().String()))
		}
	} else {
		c.opts.close = lis.Close
	}

	return lis, nil
}

func (c *connector) DialContext(ctx context.Context) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: 30 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, c.addr.Network(), c.addr.String())
	if err != nil {
		return nil, err
	}
	c.opts.close = conn.Close

	return conn, nil
}

func (c *connector) DialGRPC(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(c.opts.grpc, opts...)

	conn, err := grpc.NewClient(c.GRPCTarget(), opts...)
	if err != nil {
		return nil, err
	}

	c.opts.close = conn.Close

	return conn, nil
}

func (c *connector) GRPCTarget() string {
	return GRPCTarget(c.addr)
}

func (c *connector) Close() error {
	return c.opts.Close()
}
