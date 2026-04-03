package spec

import (
	"slices"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	CallNode interface {
		shared.SpecNode
		GetCall() *flowv1beta1.MethodCall
	}
	CallCloser interface {
		shared.ExecNode
		Close() error
	}
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

func ParseMethodCall(env shared.Env, call *flowv1beta1.MethodCall, visitor shared.ExprVisitFunc) error {
	if call.GetRequest() != nil {
		_, err := shared.ParseExprOrValue(env, call.GetRequest(), visitor, "call.request")
		if err != nil {
			return err
		}
	}
	return nil
}
