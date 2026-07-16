package spec

import (
	"context"
	"fmt"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	Method struct {
		callIdent,
		connIdent string
		node shared.SpecNode
		name protoreflect.FullName

		reqExpr *shared.ExprOrVal
		reqProg shared.Program

		ctx      context.Context
		conn     shared.Connector
		client   common.DynamicClient
		resolver shared.Resolver
		desc     protoreflect.MethodDescriptor
		reqType,
		resType protoreflect.MessageType
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

func NewMethod[T Call](env shared.Env, node CallNode[T], visitor shared.NodeVisitor) (_ *Method, err error) {
	var (
		call    = node.GetCall()
		reqExpr *shared.ExprOrVal
	)

	if call.GetRequest() != nil && call.GetRequest().AsInterface() != nil {
		reqExpr, err = shared.ParseExprOrValue(env, call.GetRequest(), visitor.ExprVisitor(GetNodeIdent(node)), "call.request")
		if err != nil {
			return nil, err
		}
	}

	connIdent := call.GetConnection()
	if id, ok := strings.CutPrefix(connIdent, shared.ConnectionPrefix+"."); !ok {
		connIdent = shared.ConnectionPrefix + "." + id
	}

	return &Method{
		callIdent: GetNodeIdent(node),
		connIdent: connIdent,
		node:      node,
		name:      protoreflect.FullName(call.GetMethod()),

		reqExpr: reqExpr,
	}, nil
}

func RangeCallNodeMethods(resolver shared.Resolver, node shared.SpecNode, f func(protoreflect.MethodDescriptor) bool) {
	resolver.RangeMethods(func(md protoreflect.MethodDescriptor) bool {
		if !api.VersionContainsName(api.CoreV1, md.FullName()) && ValidCallNodeMethod(node, md) {
			return f(md)
		}
		return true
	})
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

func (m *Method) FullName() protoreflect.FullName {
	return m.name
}

func (m *Method) Compile(ctx context.Context, env shared.Env) error {
	desc, err := m.GetDescriptor(ctx, env)
	if err != nil {
		return err
	}

	if !ValidCallNodeMethod(m.node, desc) {
		return fmt.Errorf(`%s.call.method: %s: invalid method`, m.callIdent, desc.FullName())
	}

	if m.reqExpr != nil {
		reqProg, err := m.reqExpr.Compile(env, fmt.Sprintf("%s.call.request", m.callIdent),
			cel.InterruptCheckFrequency(1),
		)
		if err != nil {
			return err
		}

		m.reqProg = reqProg
	}

	return nil
}

func (m *Method) EvalRequest(ctx context.Context, env shared.Env) (proto.Message, error) {
	resolver, err := m.GetResolver(ctx, env)
	if err != nil {
		return nil, err
	}

	reqType, err := m.GetRequestType(ctx, env)
	if err != nil {
		return nil, err
	}

	req := reqType.New().Interface()
	if m.reqProg != nil {
		refVal := m.reqProg.Eval(ctx, env)
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

func (m *Method) GetConnector(env shared.Env) (_ shared.Connector, err error) {
	if m.conn != nil {
		return m.conn, nil
	}

	conn, err := env.Connector(m.connIdent)
	if err != nil {
		return nil, fmt.Errorf(`%s.call.connection: %s: %s`, m.callIdent, m.connIdent, err)
	}

	m.conn = conn

	return conn, nil
}

func (m *Method) GetClient(ctx context.Context, env shared.Env) (context.Context, common.DynamicClient, error) {
	if m.ctx != nil && m.client != nil {
		return m.ctx, m.client, nil
	}

	conn, err := m.GetConnector(env)
	if err != nil {
		return nil, nil, err
	}

	ctx, client, err := conn.GetClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	m.ctx = ctx
	m.client = client

	return ctx, client, nil
}

func (m *Method) GetResolver(ctx context.Context, env shared.Env) (shared.Resolver, error) {
	if m.resolver != nil {
		return m.resolver, nil
	}

	conn, err := m.GetConnector(env)
	if err != nil {
		return nil, err
	}

	resolver, err := conn.GetResolver(ctx)
	if err != nil {
		return nil, err
	}

	m.resolver = resolver

	return resolver, nil
}

func (m *Method) GetDescriptor(ctx context.Context, env shared.Env) (protoreflect.MethodDescriptor, error) {
	if m.desc != nil {
		return m.desc, nil
	}

	resolver, err := m.GetResolver(ctx, env)
	if err != nil {
		return nil, err
	}

	desc, err := resolver.FindMethodByName(m.name)
	if err != nil {
		return nil, fmt.Errorf(`%s.call.method: %s: %w`, m.callIdent, m.name, err)
	}

	m.desc = desc

	return desc, nil
}

func (m *Method) GetRequestType(ctx context.Context, env shared.Env) (protoreflect.MessageType, error) {
	if m.reqType != nil {
		return m.reqType, nil
	}

	method, err := m.GetDescriptor(ctx, env)
	if err != nil {
		return nil, err
	}

	resolver, err := m.GetResolver(ctx, env)
	if err != nil {
		return nil, err
	}

	reqType, err := resolver.FindMessageByName(method.Input().FullName())
	if err != nil {
		return nil, fmt.Errorf(`%s.call: resolve request type: %s: %s`, m.callIdent, method.Input().FullName(), err)
	}

	m.reqType = reqType

	return reqType, nil
}

func (m *Method) GetResponseType(ctx context.Context, env shared.Env) (protoreflect.MessageType, error) {
	if m.resType != nil {
		return m.resType, nil
	}

	method, err := m.GetDescriptor(ctx, env)
	if err != nil {
		return nil, err
	}

	resolver, err := m.GetResolver(ctx, env)
	if err != nil {
		return nil, err
	}

	resType, err := resolver.FindMessageByName(method.Output().FullName())
	if err != nil {
		return nil, fmt.Errorf(`%s.call: resolve response type: %s: %s`, m.callIdent, method.Output().FullName(), err)
	}

	m.resType = resType

	return resType, nil
}
