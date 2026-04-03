package runtime

import (
	"context"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

var _ shared.ConnectorProvider = (Connectors)(nil)
var _ shared.Connector = (*Connector)(nil)

type (
	Connector struct {
		ctx      context.Context
		resolver shared.Resolver
		client   common.DynamicClient
		node     *flowv1beta1.Connection
	}
	Connectors map[string]*Connector
)

func NewConnectors(conns ...*Connector) Connectors {
	connMap := Connectors{}
	for _, conn := range conns {
		connMap[conn.node.Id] = conn
	}
	return connMap
}

func NewConnector(node *flowv1beta1.Connection) *Connector {
	return &Connector{
		node: node,
	}
}

func (conns Connectors) GetConnector(ctx context.Context, id string, pkg shared.Package, svcs []string) (shared.Connector, error) {
	if conn, ok := conns[id]; ok {
		return conn, nil
	}
	return nil, fmt.Errorf("connections.%s: not found", id)
}

func (c *Connector) SpecNode() shared.SpecNode {
	return c.node
}

func (c *Connector) GetResolver(context.Context) (shared.Resolver, error) {
	if c.resolver == nil {
		return nil, fmt.Errorf("connections.%s: resolver not found", c.node.Id)
	}
	return c.resolver, nil
}

func (c *Connector) GetClient(ctx context.Context) (context.Context, common.DynamicClient, error) {
	if c.ctx == nil {
		return nil, nil, fmt.Errorf("connections.%s missing context", c.node.Id)
	} else if c.ctx == nil || c.client == nil {
		return nil, nil, fmt.Errorf("connections.%s missing client", c.node.Id)
	}
	return c.ctx, c.client, nil
}

func (c *Connector) SetResolver(resolver shared.Resolver) *Connector {
	c.resolver = resolver
	return c
}

func (c *Connector) SetClient(ctx context.Context, client common.DynamicClient) *Connector {
	c.ctx = ctx
	c.client = client
	return c
}
