package spec

import (
	"context"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

func CompileConnection(run shared.Runtime, conn *flowv1beta1.Connection) (shared.EvalExpr, error) {
	return shared.EvalExprFunc(func(ctx context.Context) ref.Val {
		env, err := run.Env()
		if err != nil {
			return types.WrapErr(err)
		}
		return env.CELTypeAdapter().NativeToValue(conn)
	}), nil
}
