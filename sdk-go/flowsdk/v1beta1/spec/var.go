package spec

import (
	"context"
	"errors"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

func ParseVar(run shared.Runtime, node *flowv1beta1.Var, visitor shared.ParseExprFunc) error {
	switch {
	case node.GetSwitch() != nil:
		return ParseSwitch(run, node.GetSwitch(), visitor)
	case node.GetValue() != "":
		_, err := shared.ParseExpr(run, node.GetValue(), visitor)
		return err
	}

	return errors.New("switch or value required")
}

func CompileVar(run shared.Runtime, node *flowv1beta1.Var) (_ shared.EvalExpr, err error) {
	vars, err := run.Vars()
	if err != nil {
		return nil, err
	}

	var expr shared.EvalExpr
	switch {
	case node.GetSwitch() != nil:
		expr, err = CompileSwitch(run, node.GetSwitch())
		if err != nil {
			return
		}
	case node.GetValue() != "":
		var prog cel.Program
		prog, err = shared.CompileExpr(run, node.GetValue())
		if err != nil {
			return
		}

		expr = shared.EvalExprFunc(func(ctx context.Context) ref.Val {
			val, _, err := prog.ContextEval(ctx, vars)
			if err != nil {
				return types.WrapErr(err)
			}

			return val
		})
	default:
		return nil, errors.New("switch or value required")
	}

	return CacheableEval(node, shared.EvalExprFunc(func(ctx context.Context) ref.Val {
		return expr.Eval(ctx)
	})), nil
}
