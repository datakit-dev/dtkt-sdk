package funcs

import (
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/functions"
	celtypes "github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

const GetPrevFunc = "getPrev"

func MakeGetPrevFunc(ctx shared.Runtime) cel.EnvOption {
	var opts []cel.FunctionOpt
	ctx.RangeNodes(func(id string, node shared.Node) bool {
		typeName := string(node.GetRuntimeNode().ProtoReflect().Descriptor().FullName())
		opts = append(opts, cel.MemberOverload(
			fmt.Sprintf("%s_%s", typeName, GetPrevFunc),
			[]*cel.Type{cel.ObjectType(typeName)},
			cel.DynType,
			cel.FunctionBinding(EvalGetPrevFunc(ctx)),
		))
		return true
	})

	return cel.Function(GetPrevFunc, opts...)
}

func EvalGetPrevFunc(ctx shared.Runtime) functions.FunctionOp {
	return func(args ...ref.Val) ref.Val {
		node, ok := args[0].Value().(shared.EvalNode)
		if ok && node != nil {
			types, err := ctx.Types()
			if err != nil {
				return celtypes.WrapErr(err)
			}

			val, err := shared.ExprValueToNative(ctx, node.GetPrevValue())
			if err != nil {
				return celtypes.WrapErr(err)
			}

			return types.NativeToValue(val)
		}

		return celtypes.WrapErr(fmt.Errorf("%s failed to resolve for: %#v", GetPrevFunc, args[0].Value()))
	}
}
