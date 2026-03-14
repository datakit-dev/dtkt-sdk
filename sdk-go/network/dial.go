package network

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var grpcInsecure = grpc.WithTransportCredentials(insecure.NewCredentials())

type (
	Dialer interface {
		Address() net.Addr
		DialContext(context.Context, string, string) (net.Conn, error)
		DialGRPC(...grpc.DialOption) (*grpc.ClientConn, error)
		GRPCTarget() string
		Close() error
	}
	DialGRPCFunc[Client any] func(grpc.ClientConnInterface) Client
)

func DialGRPCClient[Client any](addr Address, newClient DialGRPCFunc[Client], opts ...grpc.DialOption) (c Client, err error) {
	opts = append([]grpc.DialOption{grpcInsecure}, opts...)

	conn, err := DialGRPC(addr, opts...)
	if err != nil {
		return
	}

	return newClient(conn), nil
}

func DialGRPCClientUsing[Client any](dialer Dialer, newClient DialGRPCFunc[Client], opts ...grpc.DialOption) (c Client, err error) {
	opts = append([]grpc.DialOption{grpcInsecure}, opts...)

	conn, err := DialGRPCUsing(dialer, opts...)
	if err != nil {
		return
	}

	return newClient(conn), nil
}

func DialGRPC(addr Address, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append([]grpc.DialOption{grpcInsecure}, opts...)

	dialer, err := NewConnector(addr)
	if err != nil {
		return nil, err
	}
	return dialer.DialGRPC(opts...)
}

func DialGRPCUsing(dialer Dialer, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append([]grpc.DialOption{grpcInsecure}, opts...)

	return dialer.DialGRPC(opts...)
}

func DialGRPCTarget(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append([]grpc.DialOption{grpcInsecure}, opts...)

	return grpc.NewClient(target, opts...)
}

func GRPCTarget(addr net.Addr) string {
	switch addr.Network() {
	case Cloud.String(), TCP.String():
		return fmt.Sprintf("dns:%s", addr)
	case Socket.String():
		return fmt.Sprintf("unix:%s", addr)
	case SSHSocket.String():
		// For SSH+Unix, use passthrough with the remote socket path
		if uri, ok := addr.(*SSHRemoteURI); ok {
			return fmt.Sprintf("passthrough://%s", uri.RemoteAddr)
		}
		return fmt.Sprintf("passthrough://%s", addr)
	case SSHTCP.String():
		// For SSH+TCP, use passthrough with the remote host:port
		if uri, ok := addr.(*SSHRemoteURI); ok {
			return fmt.Sprintf("passthrough://%s", uri.RemoteAddr)
		}
		return fmt.Sprintf("passthrough://%s", addr)
	}
	return addr.String()
}
