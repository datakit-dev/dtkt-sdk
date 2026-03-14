package network

import (
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
	"google.golang.org/grpc"
)

type (
	ConnectorOption  func(*connectorOptions)
	connectorOptions struct {
		cloud *corev1.Resource
		grpc  []grpc.DialOption
		close func() error
	}
)

func WithGRPCDialOpts(grpcOpts ...grpc.DialOption) ConnectorOption {
	return func(opts *connectorOptions) {
		opts.grpc = append(opts.grpc, grpcOpts...)
	}
}

func WithCloser(close func() error) ConnectorOption {
	return func(opts *connectorOptions) {
		opts.close = close
	}
}

func WithCloudConfig(cloud *corev1.Resource) ConnectorOption {
	return func(opts *connectorOptions) {
		opts.cloud = cloud
	}
}

func (o *connectorOptions) apply(opts ...ConnectorOption) {
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	o.grpc = append([]grpc.DialOption{grpcInsecure}, o.grpc...)
}

func (o *connectorOptions) Close() error {
	if o.close != nil {
		return o.close()
	}
	return nil
}
