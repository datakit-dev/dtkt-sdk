package network

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var dialGRPCInsecure = grpc.WithTransportCredentials(insecure.NewCredentials())

type DialGRPCFunc[Client any] func(grpc.ClientConnInterface) Client

func NewGRPCClient(addr Address, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append([]grpc.DialOption{dialGRPCInsecure}, opts...)

	dialer, err := NewConnector(addr)
	if err != nil {
		return nil, err
	}

	return dialer.GRPCClient(opts...)
}

func NewGRPCClientFor[Client any](addr Address, newClient DialGRPCFunc[Client], opts ...grpc.DialOption) (c Client, err error) {
	opts = append([]grpc.DialOption{dialGRPCInsecure}, opts...)

	conn, err := NewGRPCClient(addr, opts...)
	if err != nil {
		return
	}

	return newClient(conn), nil
}

func NewDialerGRPCClientFor[Client any](dialer Dialer, newClient DialGRPCFunc[Client], opts ...grpc.DialOption) (c Client, err error) {
	opts = append([]grpc.DialOption{dialGRPCInsecure}, opts...)

	conn, err := NewDialerGRPCClient(dialer, opts...)
	if err != nil {
		return
	}

	return newClient(conn), nil
}

func NewDialerGRPCClient(dialer Dialer, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append([]grpc.DialOption{dialGRPCInsecure}, opts...)

	return dialer.GRPCClient(opts...)
}

func NewGRPCClientTarget(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append([]grpc.DialOption{dialGRPCInsecure}, opts...)

	return grpc.NewClient(target, opts...)
}

func GRPCTarget(addr net.Addr) string {
	switch addr.Network() {
	case Cloud.String(), TCP.String():
		return fmt.Sprintf("dns:%s", addr)
	case Socket.String():
		return fmt.Sprintf("unix:%s", addr)
	case SSH_Socket.String():
		// For SSH+Unix, use passthrough with the remote socket path
		if uri, ok := addr.(*SSHRemoteURI); ok {
			return fmt.Sprintf("passthrough://%s", uri.RemoteAddr)
		}
		return fmt.Sprintf("passthrough://%s", addr)
	case SSH_TCP.String():
		// For SSH+TCP, use passthrough with the remote host:port
		if uri, ok := addr.(*SSHRemoteURI); ok {
			return fmt.Sprintf("passthrough://%s", uri.RemoteAddr)
		}
		return fmt.Sprintf("passthrough://%s", addr)
	}
	return addr.String()
}
