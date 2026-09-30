package network

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"os"
	"slices"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var _ Connector = (*defaultConnector)(nil)

type (
	Connector interface {
		Binder
		Dialer
	}
	Binder interface {
		Address() Address
		Listener(context.Context) (net.Listener, error)
		Close() error
	}
	Dialer interface {
		Address() Address
		DialContext(context.Context) (net.Conn, error)
		GRPCClient(...grpc.DialOption) (*grpc.ClientConn, error)
		HTTPClient() (*HTTPClient, string)
	}
	defaultConnector struct {
		addr       Address
		tlsConfig  *tls.Config
		httpClient *HTTPClient

		close func() error
		mu    sync.Mutex // protects close
	}
)

func NewConnector(addr Address, opts ...Option) (Connector, error) {
	switch addr.Type() {
	case Cloud:
		return NewCloudConnector(addr, opts...)
	case SSH_Socket, SSH_TCP:
		return NewSSHConnector(addr, opts...)
	}

	c := &defaultConnector{
		addr: addr,
	}
	c.applyOptions(opts...)

	if c.tlsConfig != nil && !slices.Contains(c.tlsConfig.NextProtos, "h2") {
		c.tlsConfig = c.tlsConfig.Clone()
		c.tlsConfig.NextProtos = append(c.tlsConfig.NextProtos, "h2")
	}

	return c, nil
}

func (c *defaultConnector) Address() Address {
	return c.addr
}

func (c *defaultConnector) Listener(context.Context) (net.Listener, error) {
	lis, err := net.Listen(c.addr.Network(), c.addr.String())
	if err != nil {
		return nil, err
	}

	if c.tlsConfig != nil {
		// Terminate TLS (and enforce tlsConfig.ClientAuth, e.g. mTLS) at the
		// listener level, since gRPC is served via ServeHTTP below rather
		// than grpc.Server's own accept loop (grpc.Creds would be a no-op).
		lis = tls.NewListener(lis, c.tlsConfig)
	}

	c.mu.Lock()
	if c.addr.Network() == Socket.String() && c.close == nil {
		c.close = func() error {
			return errors.Join(lis.Close(), os.Remove(c.Address().String()))
		}
	} else {
		c.close = lis.Close
	}
	c.mu.Unlock()

	return lis, nil
}

func (c *defaultConnector) DialContext(ctx context.Context) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if c.tlsConfig != nil {
		tlsDialer := &tls.Dialer{
			NetDialer: dialer,
			Config:    c.tlsConfig,
		}

		return tlsDialer.DialContext(ctx, c.addr.Network(), c.addr.String())
	}

	return dialer.DialContext(ctx, c.addr.Network(), c.addr.String())
}

func (c *defaultConnector) HTTPClient() (*HTTPClient, string) {
	if c.httpClient == nil {
		client := NewHTTPClient(c.addr,
			WithTLSConfig(c.tlsConfig),
		)
		client.dialer = c
		c.httpClient = client
	}
	return c.httpClient, c.httpClient.BaseUrl()
}

func (c *defaultConnector) GRPCClient(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	if c.tlsConfig != nil {
		opts = append(opts,
			grpc.WithTransportCredentials(credentials.NewTLS(c.tlsConfig)),
		)
	} else {
		opts = append(opts, dialGRPCInsecure)
	}

	return grpc.NewClient(GRPCTarget(c.addr), opts...)
}

func (c *defaultConnector) Close() error {
	c.mu.Lock()
	close := c.close
	c.mu.Unlock()

	if close != nil {
		return close()
	}

	return nil
}

func (c *defaultConnector) applyOptions(opts ...Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
}
