package network

import (
	"context"
	"fmt"
	"net"
	"time"

	"connectrpc.com/connect"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1/corev1connect"
	"github.com/openziti/sdk-golang/ziti"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	AuthHeader       = "authorization"
	CloudOrgHeader   = "dtkt-cloud-org"
	CloudSpaceHeader = "dtkt-cloud-space"
)

var _ Connector = (*CloudConnector)(nil)

type (
	CloudConnector struct {
		addr net.Addr
		opts *connectorOptions
	}
)

func NewCloudConnector(addr net.Addr, opts ...ConnectorOption) (*CloudConnector, error) {
	o := &connectorOptions{}
	o.apply(opts...)

	return &CloudConnector{
		addr: addr,
		opts: o,
	}, nil
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

func (c *CloudConnector) Address() net.Addr {
	return c.addr
}

func (c *CloudConnector) GRPCTarget() string {
	// NOTE: For dialing an integration outside of cloud, c.config is always nil
	// and c.addr must be a valid cloud proxy endpoint.
	if c.opts.cloud == nil {
		return GRPCTarget(c.addr)
	}
	return fmt.Sprintf("passthrough:%s", c.opts.cloud.GetName())
}

func (c *CloudConnector) DialContext(ctx context.Context, network, address string) (_ net.Conn, err error) {
	if c.opts.cloud == nil {
		dialer := &net.Dialer{
			Timeout: 30 * time.Second,
		}

		if network == "" || network == "cloud" {
			network = "tcp"
		}

		if address == "" {
			address = c.addr.String()
		}

		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		if port != "443" {
			port = "443"
		}

		address = net.JoinHostPort(host, port)

		conn, err := dialer.DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}
		c.opts.close = conn.Close

		return conn, nil
	}

	// NOTE: This can only be done by cloud as client dial config must never be
	// exposed to end-users.
	ztx, err := c.getZitiContext(ctx)
	if err != nil {
		return nil, err
	}

	conn, err := ztx.DialWithOptions(c.opts.cloud.GetName(), &ziti.DialOptions{
		ConnectTimeout: 30 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	c.opts.close = conn.Close

	return conn, nil
}

func (c *CloudConnector) DialGRPC(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(c.opts.grpc, opts...)

	// NOTE: For dialing an integration outside of cloud, c.config is always nil
	// and requests are made through cloud proxy with the same connection name
	// as bind (passthrough identifier)
	if c.opts.cloud != nil {
		opts = append([]grpc.DialOption{
			grpc.WithContextDialer(func(ctx context.Context, target string) (net.Conn, error) {
				return c.DialContext(ctx, "", "")
			}),
		}, opts...)
	}

	conn, err := grpc.NewClient(c.GRPCTarget(), opts...)
	if err != nil {
		return nil, err
	}
	c.opts.close = conn.Close

	return conn, nil
}

func (c *CloudConnector) Bind(ctx context.Context) (net.Listener, error) {
	// NOTE: For binding an integration to cloud network, c.config must be non-nil
	// in order to bind listener on edge network.
	if c.opts.cloud == nil {
		return nil, fmt.Errorf("cloud config cannot be nil")
	}

	ztx, err := c.getZitiContext(ctx)
	if err != nil {
		return nil, err
	}

	lis, err := ztx.Listen(c.opts.cloud.GetName())
	if err != nil {
		return nil, err
	}

	c.opts.close = lis.Close

	return lis, nil
}

func (c *CloudConnector) Close() error {
	return c.opts.Close()
}

func (c *CloudConnector) getZitiContext(ctx context.Context) (ziti.Context, error) {
	config, err := c.getZitiConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("get network config: %w", err)
	}

	ztx, err := ziti.NewContext(config)
	if err != nil {
		return nil, fmt.Errorf("create network context: %w", err)
	}

	err = ztx.Authenticate()
	if err != nil {
		return nil, fmt.Errorf("authenticate network context: %w", err)
	}

	return ztx, nil
}

func (c *CloudConnector) getZitiConfig(ctx context.Context) (*ziti.Config, error) {
	resource := c.opts.cloud
	if resource == nil || resource.GetData() == nil {
		return nil, fmt.Errorf("cloud config missing data")
	}

	decrypted := &wrapperspb.BytesValue{}
	if resource.GetDecrypted() != nil {
		err := resource.GetDecrypted().UnmarshalTo(decrypted)
		if err != nil {
			return nil, fmt.Errorf("unmarshal cloud config: %w", err)
		}
	} else if resource.GetContext() == nil {
		return nil, fmt.Errorf("cloud config context cannot be nil")
	} else if resource.GetContext().GetAuth() == nil {
		return nil, fmt.Errorf("cloud config auth cannot be nil")
	} else {
		var (
			auth           = resource.GetContext().GetAuth()
			addr           = AddrFromProto(resource.GetContext().GetAddress())
			client, apiURL = NewHTTPClient(addr)
		)

		resp, err := corev1connect.NewEncryptionServiceClient(client, apiURL,
			connect.WithInterceptors(CloudAuthInterceptor(auth)),
		).Decrypt(ctx, connect.NewRequest(&corev1.DecryptRequest{Resource: resource}))
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt cloud config: %w", err)
		} else if resp.Msg.GetResource() == nil || resp.Msg.GetResource().GetDecrypted() == nil {
			return nil, fmt.Errorf("failed to decrypt cloud config")
		}

		err = resp.Msg.GetResource().GetDecrypted().UnmarshalTo(decrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypt network config: %w", err)
		}
	}

	var config ziti.Config
	err := encoding.FromJSONV2(decrypted.Value, &config)
	if err != nil {
		return nil, fmt.Errorf("unmarshal network config: %w", err)
	}

	return &config, nil
}
