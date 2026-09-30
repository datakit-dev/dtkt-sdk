package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"connectrpc.com/connect"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"

	// "github.com/openziti/sdk-golang/ziti"
	"google.golang.org/grpc"
)

const (
	AuthHeader       = "authorization"
	CloudOrgHeader   = "dtkt-cloud-org"
	CloudSpaceHeader = "dtkt-cloud-space"
)

var _ Connector = (*CloudConnector)(nil)

type (
	CloudConnector struct {
		addr       Address
		resource   *corev1.Resource
		httpClient *HTTPClient

		close func() error
		mu    sync.Mutex // protects close
	}
)

func NewCloudConnector(addr Address, opts ...Option) (*CloudConnector, error) {
	conn := &CloudConnector{
		addr: addr,
	}
	conn.applyOptions(opts...)
	return conn, nil
}

func CloudAuthInterceptor(auth *corev1.Auth) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Add(AuthHeader, fmt.Sprintf("Bearer %s", auth.GetAccessToken()))
			if auth.GetOrg() != "" {
				req.Header().Add(CloudOrgHeader, auth.GetOrg())
			}
			if auth.GetSpace() != "" {
				req.Header().Add(CloudSpaceHeader, auth.GetSpace())
			}
			return next(ctx, req)
		}
	}
}

func (c *CloudConnector) Address() Address {
	return c.addr
}

func (c *CloudConnector) DialContext(ctx context.Context) (_ net.Conn, err error) {
	if c.resource == nil {
		dialer := &net.Dialer{
			Timeout: 30 * time.Second,
		}

		network := c.addr.Network()
		address := c.addr.String()
		if c.addr.Network() == "" || network == Cloud.String() {
			network = "tcp"
		}

		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		if host == "" {
			host = DefaultCloudHost
		}

		if port != "443" {
			port = "443"
		}

		return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
	}

	// NOTE: This can only be done by cloud as client dial config must never be
	// exposed to end-users.
	// ztx, err := c.getZitiContext(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	// return ztx.DialWithOptions(c.resource.GetName(), &ziti.DialOptions{
	// 	ConnectTimeout: 30 * time.Second,
	// })

	return
}

func (c *CloudConnector) HTTPClient() (*HTTPClient, string) {
	if c.httpClient == nil {
		client := NewHTTPClient(c.addr)
		client.dialer = c
		c.httpClient = client
	}
	return c.httpClient, c.httpClient.BaseUrl()
}

func (c *CloudConnector) GRPCClient(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	var target string

	// NOTE: For dialing an integration outside of cloud, c.resource is always nil
	// and requests are made through cloud proxy with the same resource name
	// as bind (passthrough identifier).
	if c.resource != nil {
		opts = append(opts,
			grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
				return c.DialContext(ctx)
			}),
		)
		target = fmt.Sprintf("passthrough:%s", c.resource.GetName())
	} else {
		target = GRPCTarget(c.addr)
	}

	return grpc.NewClient(target, opts...)
}

func (c *CloudConnector) Listener(ctx context.Context) (_ net.Listener, _ error) {
	// NOTE: For binding an integration to cloud network, c.resource must be
	// non-nil in order to bind listener on edge network.
	if c.resource == nil {
		return nil, fmt.Errorf("cloud config cannot be nil")
	}

	// ztx, err := c.getZitiContext(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	// lis, err := ztx.Listen(c.resource.GetName())
	// if err != nil {
	// 	return nil, err
	// }

	// c.mu.Lock()
	// c.close = lis.Close
	// c.mu.Unlock()

	// return lis, nil

	return
}

func (c *CloudConnector) Close() error {
	c.mu.Lock()
	close := c.close
	c.mu.Unlock()

	if close != nil {
		return close()
	}

	return nil
}

// func (c *CloudConnector) getZitiContext(ctx context.Context) (ziti.Context, error) {
// 	config, err := c.getZitiConfig(ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("get network config: %w", err)
// 	}

// 	ztx, err := ziti.NewContext(config)
// 	if err != nil {
// 		return nil, fmt.Errorf("create network context: %w", err)
// 	}

// 	err = ztx.Authenticate()
// 	if err != nil {
// 		return nil, fmt.Errorf("authenticate network context: %w", err)
// 	}

// 	return ztx, nil
// }

// func (c *CloudConnector) getZitiConfig(ctx context.Context) (*ziti.Config, error) {
// 	resource := c.resource
// 	if resource == nil || resource.GetData() == nil {
// 		return nil, fmt.Errorf("cloud config missing data")
// 	}

// 	decrypted := &wrapperspb.BytesValue{}
// 	if resource.GetDecrypted() != nil {
// 		err := resource.GetDecrypted().UnmarshalTo(decrypted)
// 		if err != nil {
// 			return nil, fmt.Errorf("unmarshal cloud config: %w", err)
// 		}
// 	} else if resource.GetContext() == nil {
// 		return nil, fmt.Errorf("cloud config context cannot be nil")
// 	} else if resource.GetContext().GetAuth() == nil {
// 		return nil, fmt.Errorf("cloud config auth cannot be nil")
// 	} else {
// 		var (
// 			auth = resource.GetContext().GetAuth()
// 			addr = AddrFromProto(resource.GetContext().GetAddress())
// 		)

// 		client := NewHTTPClient(addr)
// 		resp, err := corev1connect.NewEncryptionServiceClient(client, client.BaseUrl(),
// 			connect.WithInterceptors(CloudAuthInterceptor(auth)),
// 		).Decrypt(ctx, connect.NewRequest(&corev1.DecryptRequest{Resource: resource}))
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to decrypt cloud config: %w", err)
// 		} else if resp.Msg.GetResource() == nil || resp.Msg.GetResource().GetDecrypted() == nil {
// 			return nil, fmt.Errorf("failed to decrypt cloud config")
// 		}

// 		err = resp.Msg.GetResource().GetDecrypted().UnmarshalTo(decrypted)
// 		if err != nil {
// 			return nil, fmt.Errorf("decrypt network config: %w", err)
// 		}
// 	}

// 	var config ziti.Config
// 	err := encoding.FromJSON(decrypted.Value, &config)
// 	if err != nil {
// 		return nil, fmt.Errorf("unmarshal network config: %w", err)
// 	}

// 	return &config, nil
// }

func (c *CloudConnector) applyOptions(opts ...Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
}
