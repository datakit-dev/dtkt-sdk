package funcs

import (
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/functions"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

const GetCountFunc = "getCount"

func MakeGetCountFunc(env shared.Env) cel.EnvOption {
	return cel.Function(GetCountFunc,
		cel.MemberOverload(
			fmt.Sprintf("%s_%s", nodeType, GetCountFunc),
			[]*cel.Type{cel.ObjectType(nodeType)},
			cel.UintType,
			cel.FunctionBinding(EvalGetCountFunc(env)),
		),
	)
}

func EvalGetCountFunc(env shared.Env) functions.FunctionOp {
	return func(args ...ref.Val) ref.Val {
		node, ok := args[0].Value().(*flowv1beta1.Node)
		if ok && node != nil {
			return env.TypeAdapter().NativeToValue(node.GetCallCount())
		}

		return types.WrapErr(fmt.Errorf("%s failed to resolve for: %#v", GetCountFunc, args[0].Value()))
	}
}
