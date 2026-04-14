package rpc

import (
	"context"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
)

// MethodKind describes the streaming type of an RPC method.
type MethodKind int

const (
	MethodUnary        MethodKind = iota
	MethodServerStream            // server sends a stream of responses
	MethodClientStream            // client sends a stream of requests
	MethodBidiStream              // both sides stream
)

// Package describes the identity of an integration package.
// *sharedv1beta1.Package_Identity satisfies this interface.
type Package interface {
	GetName() string
	GetVersion() string
}

// ConnectorProvider resolves a flow Connection into a Connector (client + resolver).
// The id, pkg, and services parameters correspond to the Connection spec fields.
// pkg is nil when the connection is services-based; services is nil when package-based.
type ConnectorProvider interface {
	GetConnector(ctx context.Context, id string, pkg Package, services []string) (*Connector, error)
}

// Connectors is a map-based ConnectorProvider that looks up connectors by ID.
// Package and services fields are not used for resolution. Suitable for tests
// and pre-resolved connection sets.
type Connectors map[string]*Connector

func (c Connectors) GetConnector(_ context.Context, id string, _ Package, _ []string) (*Connector, error) {
	conn, ok := c[id]
	if !ok {
		return nil, fmt.Errorf("no connector for connection %q", id)
	}
	return conn, nil
}

// Connector pairs a Client and Resolver for a single connection.
type Connector struct {
	Client   Client
	Resolver shared.Resolver
}

// Client, BidiStream, ClientStream, and ServerStream are aliases for the
// identical interfaces in sdk-go/common. This lets any common.DynamicClient
// satisfy rpc.Client without adapter wrappers.
type (
	Client       = common.DynamicClient
	BidiStream   = common.DynamicBidiStream
	ClientStream = common.DynamicClientStream
	ServerStream = common.DynamicServerStream
)
