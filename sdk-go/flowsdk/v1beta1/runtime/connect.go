package runtime

import (
	"context"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
)

var _ ConnectorProvider = Connectors{}
var _ shared.Connector = (*Connector)(nil)

type (
	Connectors map[string]*Connector
	Connector  struct {
		ctx  context.Context
		node *flowv1beta1.Connection

		client   shared.Client
		resolver shared.Resolver
	}

	Connection        = shared.Connection[*sharedv1beta1.Package_Identity]
	ConnectorProvider = shared.ConnectorProvider[*sharedv1beta1.Package_Identity]
)

func NewConnector(ctx context.Context, node *flowv1beta1.Connection, opts ...Option) *Connector {
	conn := &Connector{
		ctx:  ctx,
		node: node,
	}
	conn.applyOptions(opts...)
	return conn
}

func (c Connectors) GetConnector(node Connection) (shared.Connector, error) {
	conn, ok := c[node.GetId()]
	if !ok {
		return nil, fmt.Errorf("connections.%s: not found", node.GetId())
	}
	return conn, nil
}

func (c *Connector) GetResolver(context.Context) (shared.Resolver, error) {
	if c.resolver == nil {
		return nil, fmt.Errorf("connections.%s: resolver not found", c.node.Id)
	}
	return c.resolver, nil
}

func (c *Connector) GetClient(context.Context) (context.Context, shared.Client, error) {
	if c.ctx == nil || c.client == nil {
		return nil, nil, fmt.Errorf("connections.%s: client not found", c.node.Id)
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
