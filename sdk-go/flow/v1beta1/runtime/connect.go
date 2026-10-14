package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

var _ ConnectorProvider = Connectors{}
var _ shared.Connector = (*Connector)(nil)

type (
	Connectors map[string]*Connector
	Connector  struct {
		ctx   context.Context
		ident string

		client   shared.Client
		resolver shared.Resolver
	}

	ConnectorProvider interface {
		// GetConnector returns a Connector for a connection node with the given id.
		// It must match the given package (name & version) or implement the given
		// list of services.
		GetConnector(ident string) (shared.Connector, error)
	}
	GetConnectorFunc func(ident string) (shared.Connector, error)
)

func NewConnector(ctx context.Context, node *flowv1beta1.Connection, opts ...Option) *Connector {
	conn := &Connector{
		ctx:   ctx,
		ident: shared.GetNodeIdent(node),
	}
	conn.applyOptions(opts...)
	return conn
}

func (c Connectors) GetConnector(ident string) (shared.Connector, error) {
	conn, ok := c[ident]
	if !ok && shared.IsNodeIdent(ident) {
		id := strings.TrimPrefix(ident, shared.ConnectionPrefix+".")
		conn, ok = c[id]
		if !ok {
			return nil, fmt.Errorf("connector not found: %s", ident)
		}
	}

	return conn, nil
}

func (c *Connector) GetResolver(context.Context) (shared.Resolver, error) {
	if c.resolver == nil {
		return nil, fmt.Errorf("%s: resolver not found", c.ident)
	}
	return c.resolver, nil
}

func (c *Connector) GetClient(context.Context) (context.Context, shared.Client, error) {
	if c.ctx == nil || c.client == nil {
		return nil, nil, fmt.Errorf("%s: client not found", c.ident)
	}
	return c.ctx, c.client, nil
}

func (c *Connector) applyOptions(opts ...Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
}

func (f GetConnectorFunc) GetConnector(ident string) (shared.Connector, error) {
	return f(ident)
}
