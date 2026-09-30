package network

import (
	"context"
	"net"
	"net/http"
	"time"
)

type (
	HTTPClient struct {
		*http.Client
		transport *http.Transport

		addr    Address
		dialer  Dialer
		dialTLS bool
	}
	HTTPRoundTripperFunc func(*http.Request) (*http.Response, error)
)

func NewHTTPClient(address Address, opts ...Option) *HTTPClient {
	c := &HTTPClient{
		addr: address,
	}
	c.applyOptions(opts...)

	if c.transport == nil {
		c.transport = &http.Transport{
			ForceAttemptHTTP2: true,
			MaxIdleConns:      100,
			IdleConnTimeout:   90 * time.Second,
		}

		if c.transport.Protocols == nil {
			c.transport.Protocols = new(http.Protocols)
		}

		dialContext := func(ctx context.Context, _, _ string) (net.Conn, error) {
			if c.dialer != nil {
				return c.dialer.DialContext(ctx)
			}

			dialer := &net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}

			return dialer.DialContext(ctx, c.addr.Network(), c.addr.String())
		}

		if c.dialTLS {
			c.transport.Protocols.SetHTTP2(true)
			c.transport.DialTLSContext = dialContext
		} else {
			// HTTP1 must stay unset: Transport only uses prior-knowledge h2c
			// for http:// URLs when UnencryptedHTTP2 is set without HTTP1.
			c.transport.Protocols.SetUnencryptedHTTP2(true)
			c.transport.DialContext = dialContext
		}
	}

	if c.Client == nil {
		c.Client = &http.Client{}
	}

	if c.Client.Transport == nil {
		c.Client.Transport = c.transport
	}

	return c
}

func (c *HTTPClient) Address() Address {
	return c.addr
}

func (c *HTTPClient) BaseUrl() string {
	if c.dialTLS {
		return c.addr.HTTPS().String()
	}
	return c.addr.HTTP().String()
}

func (c *HTTPClient) applyOptions(opts ...Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
}

func (f HTTPRoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
