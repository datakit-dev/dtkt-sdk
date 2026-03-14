package spec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	CallNode interface {
		shared.SpecNode
		GetCall() *flowv1beta1.MethodCall
	}
	CallCloser func() error
)

func ValidCallNodeMethods(resolver shared.Resolver, node shared.SpecNode) (names []string) {
	resolver.RangeMethods(func(md protoreflect.MethodDescriptor) bool {
		name := string(md.FullName())
		if ValidCallNodeMethod(node, md) && !slices.Contains(names, name) {
			names = append(names, name)
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

func ParseCall(run shared.Runtime, call *flowv1beta1.MethodCall, visitor shared.ParseExprFunc) error {
	if call.GetRequest() != nil {
		_, err := shared.ParseExprOrValue(run, call.GetRequest(), visitor, "call.request")
		if err != nil {
			return err
		}
	}

	if call.GetResponse() != "" {
		_, err := shared.ParseExpr(run, call.GetResponse(), visitor)
		return err
	}
	return nil
}

func CompileCallCloser(run shared.Runtime, node CallNode) (shared.EvalExpr, CallCloser, error) {
	env, err := run.Env()
	if err != nil {
		return nil, nil, err
	}

	vars, err := run.Vars()
	if err != nil {
		return nil, nil, err
	}

	caller, err := NewMethodCaller(run, node)
	if err != nil {
		return nil, nil, err
	}

	var reqEval shared.EvalExpr
	if node.GetCall().GetRequest() != nil {
		reqEval, err = shared.CompileExprOrValue(run, node.GetCall().GetRequest(), "call.request")
		if err != nil {
			return nil, nil, err
		} else if reqEval == nil {
			return nil, nil, fmt.Errorf("failed to compile call request expression")
		}
	}

	var resEval shared.EvalExpr
	if node.GetCall().GetResponse() != "" {
		prog, err := shared.CompileExpr(run, node.GetCall().GetResponse())
		if err != nil {
			return nil, nil, err
		}

		resEval = shared.EvalExprFunc(func(ctx context.Context) ref.Val {
			val, _, err := prog.ContextEval(ctx, vars)
			if err != nil {
				return types.WrapErr(err)
			}
			return val
		})
	}

	return shared.EvalExprFunc(func(ctx context.Context) ref.Val {
		req := caller.reqType.New().Interface()
		if reqEval != nil {
			refVal := reqEval.Eval(ctx)
			if refVal != nil && refVal.Value() != nil {
				switch val := refVal.Value().(type) {
				case error:
					return types.WrapErr(val)
				default:
					b, err := encoding.ToJSONV2(val,
						encoding.WithEncodeProtoJSONOptions(protojson.MarshalOptions{
							Resolver: run.Resolver(),
						}),
					)
					if err != nil {
						return types.WrapErr(fmt.Errorf("failed to encode request value to JSON: %s", err))
					}

					err = encoding.FromJSONV2(b, req,
						encoding.WithDecodeProtoJSONOptions(protojson.UnmarshalOptions{
							Resolver: run.Resolver(),
						}),
					)
					if err != nil {
						return types.WrapErr(fmt.Errorf("failed to decode request JSON to %s: %s", caller.reqType.Descriptor().FullName(), err))
					}
				}
			}
		}

		res, err := caller.GetResponse(req)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return env.CELTypeAdapter().NativeToValue(&flowv1beta1.Runtime_EOF{})
			}
			return types.WrapErr(err)
		}

		if resEval != nil {
			return resEval.Eval(ctx)
		}

		return env.CELTypeAdapter().NativeToValue(res)
	}), caller.Close, nil
}

func CompileCall(run shared.Runtime, node CallNode) (shared.EvalExpr, error) {
	eval, _, err := CompileCallCloser(run, node)
	return eval, err
}
