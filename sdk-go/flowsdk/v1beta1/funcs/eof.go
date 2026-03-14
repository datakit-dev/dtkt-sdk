package funcs

import (
	"errors"
	"fmt"
	"io"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

const IsEOFFunc = "isEOF"

func MakeIsEOFFunc(ctx shared.Runtime) cel.EnvOption {
	return cel.Function(IsEOFFunc,
		cel.SingletonFunctionBinding(
			func(args ...ref.Val) ref.Val {
				switch val := args[0].Value().(type) {
				case *flowv1beta1.Runtime_EOF:
					return types.True
				case error:
					if errors.Is(val, io.EOF) {
						return types.True
					}
				}
				return types.False
			},
		),
		cel.Overload(fmt.Sprintf("%s_%s", IsEOFFunc, "any"), []*cel.Type{cel.DynType}, cel.BoolType),
	)
}
