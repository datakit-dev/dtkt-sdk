package funcs

import (
	"fmt"

	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

const IsEOFFunc = "isEOF"

func MakeIsEOFFunc() cel.EnvOption {
	return cel.Function(IsEOFFunc,
		cel.SingletonFunctionBinding(
			func(args ...ref.Val) ref.Val {
				switch args[0].Value().(type) {
				case *flowv1beta1.Runtime_EOF:
					return types.True
				}
				return types.False
			},
		),
		cel.Overload(fmt.Sprintf("%s_%s", IsEOFFunc, "any"), []*cel.Type{cel.DynType}, cel.BoolType),
	)
}
