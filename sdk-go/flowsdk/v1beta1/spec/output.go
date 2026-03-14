package spec

import (
	"context"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

func ParseOutput(run shared.Runtime, node *flowv1beta1.Output, visitor shared.ParseExprFunc) error {
	_, err := shared.ParseExpr(run, node.GetValue(), visitor)
	return err
}

func CompileOutput(run shared.Runtime, node *flowv1beta1.Output) (shared.EvalExpr, error) {
	vars, err := run.Vars()
	if err != nil {
		return nil, err
	}

	prog, err := shared.CompileExpr(run, node.GetValue())
	if err != nil {
		return nil, err
	}

	return shared.EvalExprFunc(func(ctx context.Context) ref.Val {
		val, _, err := prog.ContextEval(ctx, vars)
		if err != nil {
			return types.WrapErr(err)
		}

		return val
	}), nil
}
