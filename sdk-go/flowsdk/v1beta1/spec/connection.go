package spec

import (
	"slices"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ shared.ExecNode = (*Connection)(nil)

type Connection struct {
	node *flowv1beta1.Connection
	eval shared.EvalFunc
}

func NewConnection(_ shared.Env, node *flowv1beta1.Connection) *Connection {
	return &Connection{node: node}
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

func (c *Connection) Compile(run shared.Runtime) error {
	eval, err := CompileConnection(run, c.node)
	if err != nil {
		return err
	}
	c.eval = eval.Eval
	return nil
}

func (c *Connection) Eval() (shared.EvalFunc, bool) { return c.eval, true }
func (c *Connection) Recv() (shared.RecvFunc, bool) { return nil, false }
func (c *Connection) Send() (shared.SendFunc, bool) { return nil, false }
func (c *Connection) HasCached() (ref.Val, bool)    { return nil, false }

func CompileConnection(run shared.Runtime, conn *flowv1beta1.Connection) (shared.Program, error) {
	return shared.EvalFunc(func(run shared.Runtime) ref.Val {
		env, err := run.Env()
		if err != nil {
			return types.WrapErr(err)
		}

		return env.TypeAdapter().NativeToValue(conn)
	}), nil
}
