package spec

import (
	"context"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	EvalMessageFunc func(shared.Runtime) (proto.Message, error)
	UnaryCaller     struct {
		ctx     context.Context
		client  common.DynamicClient
		evalRes EvalMessageFunc
	}
)

func NewCaller(run shared.Runtime, node CallNode) (CallCloser, error) {
	var connNode *flowv1beta1.Connection
	if conn, ok := run.GetNode(fmt.Sprintf("%s.%s", shared.ConnectionPrefix, node.GetCall().GetConnection())); !ok {
		return nil, fmt.Errorf(`%s.call.connection: %s: not found`, node.GetId(), node.GetCall().GetConnection())
	} else if connNode, ok = conn.(*flowv1beta1.Connection); !ok {
		return nil, fmt.Errorf(`%s.call.connection: %s, expected: Connection, got: %s`,
			node.GetId(),
			node.GetCall().GetConnection(), conn.ProtoReflect().Descriptor().Name(),
		)
	}

	id := GetID(connNode)
	pkg := connNode.GetPackage()
	if pkg == nil {
		pkg = &sharedv1beta1.Package_Identity{}
	}

	conn, err := run.Connectors().GetConnector(run.Context(), connNode.GetId(), pkg, connNode.GetServices())
	if err != nil {
		return nil, fmt.Errorf(`%s.call.connection: %s: %s`, id, node.GetCall().GetConnection(), err)
	}

	resolver, err := conn.GetResolver(run.Context())
	if err != nil {
		return nil, fmt.Errorf(`%s.call.connection: %s: %s`, id, node.GetCall().GetConnection(), err)
	}

	method, err := resolver.FindMethodByName(protoreflect.FullName(node.GetCall().GetMethod()))
	if err != nil {
		method, err = resolver.FindMethodByName(protoreflect.FullName("dtkt." + node.GetCall().GetMethod()))
		if err != nil {
			return nil, fmt.Errorf(`%s.call.method: %s: not found`, id, node.GetCall().GetMethod())
		}
	}

	if !ValidCallNodeMethod(node, method) {
		return nil, fmt.Errorf(`%s.call.method: %s: invalid method`, id, node.GetCall().GetMethod())
	}

	reqType, err := resolver.FindMessageByName(method.Input().FullName())
	if err != nil {
		return nil, fmt.Errorf(`%s.call request type: %s: %s`, id, method.Input().FullName(), err)
	}

	var reqEval shared.Eval
	if node.GetCall().GetRequest() != nil {
		reqEval, err = shared.CompileExprOrValue(run, node.GetCall().GetRequest(), fmt.Sprintf("%s.call.request", id))
		if err != nil {
			return nil, err
		}
	}

	if method.IsStreamingClient() && method.IsStreamingServer() {
		return NewBidiStream(run, conn, node, method, func(run shared.Runtime) (proto.Message, error) {
			return evalMessageType(run, resolver, reqType, reqEval)
		})
	} else if method.IsStreamingClient() {
		return NewClientStream(run, conn, node, method, func(run shared.Runtime) (proto.Message, error) {
			return evalMessageType(run, resolver, reqType, reqEval)
		})
	} else if method.IsStreamingServer() {
		return NewServerStream(run, conn, node, method, func(run shared.Runtime) (proto.Message, error) {
			return evalMessageType(run, resolver, reqType, reqEval)
		})
	}

	ctx, client, err := conn.GetClient(run.Context())
	if err != nil {
		return nil, err
	}

	return &UnaryCaller{
		evalRes: func(run shared.Runtime) (proto.Message, error) {
			req, err := evalMessageType(run, resolver, reqType, reqEval)
			if err != nil {
				return nil, err
			}

			return client.CallUnary(ctx, method.FullName(), req)
		},
	}, nil
}

func evalMessageType(run shared.Runtime, resolver shared.Resolver, msgType protoreflect.MessageType, msgEval shared.Eval) (proto.Message, error) {
	req := msgType.New().Interface()
	if msgEval != nil {
		refVal := msgEval.Eval(run)
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
					return nil, fmt.Errorf("failed to decode request JSON to %s: %s", msgType.Descriptor().FullName(), err)
				}
			}
		}
	}
	return req, nil
}

func (m *UnaryCaller) Compile(shared.Runtime) error  { return nil }
func (m *UnaryCaller) Recv() (shared.RecvFunc, bool) { return nil, false }
func (m *UnaryCaller) Send() (shared.SendFunc, bool) { return nil, false }

func (m *UnaryCaller) Eval() (shared.EvalFunc, bool) {
	return func(run shared.Runtime) ref.Val {
		env, err := run.Env()
		if err != nil {
			return types.WrapErr(err)
		}

		res, err := m.evalRes(run)
		if err != nil {
			return types.WrapErr(err)
		}

		return env.TypeAdapter().NativeToValue(res)
	}, true
}

func (m *UnaryCaller) Close() error {
	return nil
}
