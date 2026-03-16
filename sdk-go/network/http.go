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
		address    Address
		dialer     Dialer
		protocols  *http.Protocols
		transport  *http.Transport
		dialerOpts []ConnectorOption
	}
	HTTPClientOption     func(*HTTPClient)
	HTTPRoundTripperFunc func(*http.Request) (*http.Response, error)
)

func NewHTTPClient(address Address, opts ...HTTPClientOption) (*HTTPClient, string) {
	client := &HTTPClient{
		address: address,
	}

	client.applyOptions(opts...)

	return client, address.HTTP().String()
}

func NewHTTPClientWithDialer(dialer Dialer, opts ...HTTPClientOption) (*HTTPClient, string) {
	client, baseURL := NewHTTPClient(Address{dialer.Address().Network(), dialer.Address().String()}, opts...)
	client.dialer = dialer
	return client, baseURL
}

func WithHTTPProtocols(protocols *http.Protocols) HTTPClientOption {
	return func(c *HTTPClient) {
		c.protocols = protocols
	}
}

func WithHTTPTransport(transport *http.Transport) HTTPClientOption {
	return func(c *HTTPClient) {
		c.transport = transport
	}
}

func WithHTTPDialerOptions(opts ...ConnectorOption) HTTPClientOption {
	return func(c *HTTPClient) {
		c.dialerOpts = append(c.dialerOpts, opts...)
	}
}

func (c *HTTPClient) DialContext(ctx context.Context, _ string, _ string) (net.Conn, error) {
	if c.dialer == nil {
		dialer, err := NewConnector(c.address, c.dialerOpts...)
		if err != nil {
			return nil, err
		}
		c.dialer = dialer
	}
	return c.dialer.DialContext(ctx)
}

func (c *HTTPClient) BaseURL() string {
	return c.address.HTTP().String()
}

func (c *HTTPClient) applyOptions(opts ...HTTPClientOption) {
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	if c.Client == nil {
		if c.transport == nil {
			if c.protocols == nil {
				c.protocols = new(http.Protocols)
				c.protocols.SetUnencryptedHTTP2(true)
			}

			c.transport = &http.Transport{
				DialContext:           c.DialContext,
				ForceAttemptHTTP2:     true,
				Protocols:             c.protocols,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			}
		}

		c.Client = &http.Client{
			Transport: c.transport,
		}
	}
}

func (f HTTPRoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
