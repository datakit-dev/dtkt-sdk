package spec

import (
	"context"
	"fmt"
	"slices"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ shared.ExecNode = (*Connection)(nil)

type Connection struct {
	ident string
	node  *flowv1beta1.Connection
	conn  shared.Connector
}

func NewConnection(env shared.Env, node *flowv1beta1.Connection) *Connection {
	return &Connection{
		ident: GetNodeIdent(node),
		node:  node,
	}
}

func ValidConnectionServices(resolver shared.Resolver) (names []string) {
	resolver.RangeServices(func(sd protoreflect.ServiceDescriptor) bool {
		if !api.VersionContainsDescriptor(api.CoreV1, sd) {
			name := string(sd.FullName())
			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
		return true
	})
	return
}

func (c *Connection) Compile(ctx context.Context, env shared.Env) error {
	conn, err := env.Connector(c.ident)
	if err != nil {
		return fmt.Errorf("%s: %s", c.ident, err)
	}
	c.conn = conn
	return nil
}

func (c *Connection) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) error {
	return pipe.Run(ctx, func(ctx context.Context, value any) (any, error) {
		return c.node, nil
	})
}
