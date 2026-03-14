package funcs

import (
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

func Make(run shared.Runtime) []cel.EnvOption {
	return []cel.EnvOption{
		cel.Function("now",
			cel.SingletonFunctionBinding(
				func(...ref.Val) ref.Val {
					return types.Timestamp{
						Time: time.Now(),
					}
				},
			),
			cel.Overload("now", nil, cel.TimestampType),
		),
		MakeGetCountFunc(run),
		MakeGetPrevFunc(run),
		MakeGetValueFunc(run),
		MakeIsEOFFunc(run),
	}
}
