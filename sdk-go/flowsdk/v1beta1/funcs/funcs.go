package funcs

import (
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var evalNode *flowv1beta1.Node
var nodeType = string(evalNode.ProtoReflect().Descriptor().FullName())

func EnvOptions(env shared.Env) []cel.EnvOption {
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
		MakeGetCountFunc(env),
		MakeGetPrevFunc(env),
		MakeGetValueFunc(env),
		MakeIsEOFFunc(),
	}
}
