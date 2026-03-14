package funcs

import (
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/functions"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

const GetCountFunc = "getCount"

func MakeGetCountFunc(ctx shared.Runtime) cel.EnvOption {
	var opts []cel.FunctionOpt
	ctx.RangeNodes(func(id string, node shared.Node) bool {
		typeName := string(node.GetRuntimeNode().ProtoReflect().Descriptor().FullName())

		opts = append(opts, cel.MemberOverload(
			fmt.Sprintf("%s_%s", typeName, GetCountFunc),
			[]*cel.Type{cel.ObjectType(typeName)},
			cel.UintType,
			cel.FunctionBinding(EvalGetCountFunc(ctx)),
		))
		return true
	})

	return cel.Function(GetCountFunc, opts...)
}

func EvalGetCountFunc(ctx shared.Runtime) functions.FunctionOp {
	return func(args ...ref.Val) ref.Val {
		node, ok := args[0].Value().(shared.EvalNode)
		if ok && node != nil {
			env, err := ctx.Env()
			if err != nil {
				return types.WrapErr(err)
			}

			return env.CELTypeAdapter().NativeToValue(node.GetCallCount())
		}

		return types.WrapErr(fmt.Errorf("%s failed to resolve for: %#v", GetCountFunc, args[0].Value()))
	}
}
