package spec

import (
	"fmt"
	"slices"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	Method struct {
		node shared.SpecNode
		call Call

		reqExpr *shared.ExprOrVal
		reqProg shared.Program
	}
	CallNode[T Call] interface {
		shared.SpecNode
		GetCall() T
	}
	Call interface {
		proto.Message
		GetConnection() string
		GetMethod() string
		GetRequest() *structpb.Value
	}
)

func NewMethod[T Call](env shared.Env, node CallNode[T], visitor shared.NodeVisitFunc) (_ *Method, err error) {
	method := &Method{
		node: node,
		call: node.GetCall(),
	}

	if method.call.GetRequest() != nil {
		method.reqExpr, err = shared.ParseExprOrValue(env, method.call.GetRequest(), visitor.ExprVisitor(GetID(node)), "call.request")
		if err != nil {
			return nil, err
		}
	}

	return method, nil
}

func ValidCallNodeMethods(resolver shared.Resolver, node shared.SpecNode) (names []string) {
	resolver.RangeMethods(func(md protoreflect.MethodDescriptor) bool {
		if !api.VersionContainsDescriptor(api.CoreV1, md) {
			name := string(md.FullName())
			if ValidCallNodeMethod(node, md) && !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
		return true
	})
	return
}

func ValidCallNodeMethod(node shared.SpecNode, method protoreflect.MethodDescriptor) bool {
	isUnary := !method.IsStreamingClient() && !method.IsStreamingServer()
	switch node.(type) {
	case *flowv1beta1.Action:
		return isUnary
	case *flowv1beta1.Stream:
		return !isUnary
	}
	return false
}

func (c *Method) Compile(run shared.Runtime) error {
	if c.reqExpr != nil {
		env, err := run.Env()
		if err != nil {
			return err
		}

		opts := []cel.ProgramOption{cel.InterruptCheckFrequency(1)}

		c.reqProg, err = c.reqExpr.Compile(env, fmt.Sprintf("%s.call.request", c.GetNodeId()), opts...)
		if err != nil {
			return err
		}
	}

	method, err := c.GetDescriptor(run)
	if err != nil {
		return err
	}

	if !ValidCallNodeMethod(c.node, method) {
		return fmt.Errorf(`%s.call.method: %s: invalid method`, c.GetNodeId(), method.FullName())
	}

	return nil
}

func (c *Method) GetNodeId() string {
	return GetID(c.node)
}

func (c *Method) EvalRequest(run shared.Runtime) (proto.Message, error) {
	resolver, err := c.GetResolver(run)
	if err != nil {
		return nil, err
	}

	reqType, err := c.GetRequestType(run)
	if err != nil {
		return nil, err
	}

	req := reqType.New().Interface()
	if c.reqProg != nil {
		refVal := c.reqProg.Eval(run)
		if refVal != nil && refVal.Value() != nil {
			switch val := refVal.Value().(type) {
			case error:
				return nil, val
			default:
				b, err := encoding.ToJSONV2(val,
					encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
						Resolver: resolver,
					}),
				)
				if err != nil {
					return nil, fmt.Errorf("failed to encode request value to JSON: %s", err)
				}

				err = encoding.FromJSONV2(b, req,
					encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
						Resolver: resolver,
					}),
				)
				if err != nil {
					return nil, fmt.Errorf("failed to decode request JSON to %s: %s", reqType.Descriptor().FullName(), err)
				}
			}
		}
	}
	return req, nil
}

func (c *Method) GetConnector(run shared.Runtime) (_ shared.Connector, err error) {
	var connNode *flowv1beta1.Connection
	if conn, ok := run.GetNode(fmt.Sprintf("%s.%s", shared.ConnectionPrefix, c.call.GetConnection())); !ok {
		return nil, fmt.Errorf(`%s.call.connection: %s: not found`, c.GetNodeId(), c.call.GetConnection())
	} else if connNode, ok = conn.(*flowv1beta1.Connection); !ok {
		return nil, fmt.Errorf(`%s.call.connection: %s, expected: Connection, got: %s`,
			c.GetNodeId(),
			c.call.GetConnection(), conn.ProtoReflect().Descriptor().Name(),
		)
	}

	pkg := connNode.GetPackage()
	if pkg == nil {
		pkg = &sharedv1beta1.Package_Identity{}
	}

	conn, err := run.Connectors().GetConnector(run.Context(), connNode.GetId(), pkg, connNode.GetServices())
	if err != nil {
		return nil, fmt.Errorf(`%s.call.connection: %s: %s`, c.GetNodeId(), c.call.GetConnection(), err)
	}

	return conn, nil
}

func (c *Method) GetResolver(run shared.Runtime) (shared.Resolver, error) {
	conn, err := c.GetConnector(run)
	if err != nil {
		return nil, err
	}

	return conn.GetResolver(run.Context())
}

func (c *Method) GetDescriptor(run shared.Runtime) (protoreflect.MethodDescriptor, error) {
	resolver, err := c.GetResolver(run)
	if err != nil {
		return nil, err
	}

	method, err := resolver.FindMethodByName(protoreflect.FullName(c.call.GetMethod()))
	if err != nil {
		method, err = resolver.FindMethodByName(protoreflect.FullName("dtkt." + c.call.GetMethod()))
		if err != nil {
			return nil, fmt.Errorf(`%s.call.method: %s: not found`, c.GetNodeId(), c.call.GetMethod())
		}
	}

	return method, nil
}

func (c *Method) GetRequestType(run shared.Runtime) (protoreflect.MessageType, error) {
	resolver, err := c.GetResolver(run)
	if err != nil {
		return nil, err
	}

	method, err := c.GetDescriptor(run)
	if err != nil {
		return nil, err
	}

	reqType, err := resolver.FindMessageByName(method.Input().FullName())
	if err != nil {
		return nil, fmt.Errorf(`%s.call: resolve request type: %s: %s`, c.GetNodeId(), method.Input().FullName(), err)
	}

	return reqType, nil
}

func (c *Method) GetResponseType(run shared.Runtime) (protoreflect.MessageType, error) {
	resolver, err := c.GetResolver(run)
	if err != nil {
		return nil, err
	}

	method, err := c.GetDescriptor(run)
	if err != nil {
		return nil, err
	}

	resType, err := resolver.FindMessageByName(method.Output().FullName())
	if err != nil {
		return nil, fmt.Errorf(`%s.call: resolve response type: %s: %s`, c.GetNodeId(), method.Output().FullName(), err)
	}

	return resType, nil
}
