package spec

import (
	"context"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	MethodCaller struct {
		ctx     context.Context
		client  common.DynamicClient
		method  protoreflect.MethodDescriptor
		reqType protoreflect.MessageType
		stream  Stream
	}
)

func NewMethodCaller(run shared.Runtime, call CallNode) (*MethodCaller, error) {
	node, ok := run.GetNode(fmt.Sprintf("%s.%s", shared.ConnectionPrefix, call.GetCall().GetConnection()))
	if !ok {
		return nil, fmt.Errorf(`%s.call.connection: %s: not found`, call.GetId(), call.GetCall().GetConnection())
	}

	conn, ok := node.GetTypeNode().(*flowv1beta1.Connection)
	if !ok {
		return nil, fmt.Errorf(`%s.call.connection: %s, expected: Connection, got: %s`,
			call.GetId(),
			call.GetCall().GetConnection(), node.GetTypeNode().ProtoReflect().Descriptor().Name(),
		)
	}

	pkg := conn.GetPackage()
	if pkg == nil {
		pkg = &sharedv1beta1.Package_Identity{}
	}

	connector, err := run.Connectors().GetConnector(run.Context(), conn.GetId(), pkg, conn.GetServices())
	if err != nil {
		return nil, fmt.Errorf(`%s.call.connection: %s: %s`, call.GetId(), call.GetCall().GetConnection(), err)
	}

	resolver, err := connector.GetResolver(run.Context())
	if err != nil {
		return nil, fmt.Errorf(`%s.call.connection: %s: %s`, call.GetId(), call.GetCall().GetConnection(), err)
	}

	method, err := resolver.FindMethodByName(protoreflect.FullName(call.GetCall().GetMethod()))
	if err != nil {
		method, err = resolver.FindMethodByName(protoreflect.FullName("dtkt." + call.GetCall().GetMethod()))
		if err != nil {
			return nil, fmt.Errorf(`%s.call.method: %s: not found`, call.GetId(), call.GetCall().GetMethod())
		}
	}

	if !ValidCallNodeMethod(call, method) {
		return nil, fmt.Errorf(`%s.call.method: %s: invalid method`, call.GetId(), call.GetCall().GetMethod())
	}

	reqType, err := resolver.FindMessageByName(method.Input().FullName())
	if err != nil {
		return nil, fmt.Errorf(`%s.call request type: %s: %s`, call.GetId(), method.Input().FullName(), err)
	}

	ctx, client, err := connector.GetClient(run.Context())
	if err != nil {
		return nil, err
	}

	var stream Stream
	if method.IsStreamingClient() && method.IsStreamingServer() {
		stream, err = NewBidiStream(ctx, client, method)
		if err != nil {
			return nil, err
		}
	} else if method.IsStreamingClient() {
		stream, err = NewClientStream(ctx, client, method)
		if err != nil {
			return nil, err
		}
	} else if method.IsStreamingServer() {
		stream, err = NewServerStream(ctx, client, method)
		if err != nil {
			return nil, err
		}
	}

	return &MethodCaller{
		ctx:     ctx,
		client:  client,
		stream:  stream,
		method:  method,
		reqType: reqType,
	}, nil
}

func (m *MethodCaller) GetResponse(req proto.Message) (proto.Message, error) {
	if m.stream != nil {
		err := m.stream.Send(req)
		if err != nil {
			return nil, err
		}
		return m.stream.Recv()
	}
	return m.client.CallUnary(m.ctx, m.method.FullName(), req)
}

func (m *MethodCaller) Close() error {
	if m.stream != nil {
		return m.stream.Close()
	}
	return nil
}
