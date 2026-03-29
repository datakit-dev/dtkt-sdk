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

const GetPrevFunc = "getPrev"

func MakeGetPrevFunc(env shared.Env) cel.EnvOption {
	return cel.Function(GetPrevFunc,
		cel.MemberOverload(
			fmt.Sprintf("%s_%s", nodeType, GetPrevFunc),
			[]*cel.Type{cel.ObjectType(nodeType)},
			cel.DynType,
			cel.FunctionBinding(EvalGetPrevFunc(env)),
		),
	)
}

func EvalGetPrevFunc(env shared.Env) functions.FunctionOp {
	return func(args ...ref.Val) ref.Val {
		node, ok := args[0].Value().(*flowv1beta1.Node)
		if ok && node != nil {
			val, err := shared.ExprValueToNative(env, node.GetPrevValue())
			if err != nil {
				return types.WrapErr(err)
			}

			return env.TypeAdapter().NativeToValue(val)
		}

		return types.WrapErr(fmt.Errorf("%s failed to resolve for: %#v", GetPrevFunc, args[0].Value()))
	}
}
