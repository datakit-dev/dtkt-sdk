package funcs

import (
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/functions"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

const GetValueFunc = "getValue"

func MakeGetValueFunc(run shared.Runtime) cel.EnvOption {
	var opts []cel.FunctionOpt
	run.RangeNodes(func(id string, node shared.Node) bool {
		typeName := string(node.GetRuntimeNode().ProtoReflect().Descriptor().FullName())
		opts = append(opts,
			cel.MemberOverload(
				fmt.Sprintf("%s_%s", typeName, GetValueFunc),
				[]*cel.Type{cel.ObjectType(typeName)},
				cel.DynType,
				cel.FunctionBinding(EvalGetValueFunc(run)),
			),
		)
		return true
	})

	return cel.Function(GetValueFunc, opts...)
}

func EvalGetValueFunc(run shared.Runtime) functions.FunctionOp {
	return func(args ...ref.Val) ref.Val {
		node, ok := args[0].Value().(shared.EvalNode)
		if ok && node != nil {
			env, err := run.Env()
			if err != nil {
				return types.WrapErr(err)
			}

			val, err := run.GetNodeValue(node.GetId())
			if err != nil {
				return types.WrapErr(err)
			}

			return env.CELTypeAdapter().NativeToValue(val)
		}

		return types.WrapErr(fmt.Errorf("%s failed to resolve for: %#v", GetValueFunc, args[0].Value()))
	}
}
