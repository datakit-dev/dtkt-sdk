package spec

import (
	"context"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

const (
	switchExprId    = "switch.expr"
	switchDefaultId = "switch.default"
)

func switchCaseId(idx int) string {
	return fmt.Sprintf("switch.case[%d].value", idx)
}

func switchReturnId(idx int) string {
	return fmt.Sprintf("switch.case[%d].return", idx)
}

func ParseSwitch(ctx shared.Runtime, swtch *flowv1beta1.Switch, visitor shared.ParseExprFunc) error {
	if _, err := shared.ParseExpr(ctx, swtch.GetValue(), visitor); err != nil {
		return fmt.Errorf("%s parse error: %w", switchExprId, err)
	}

	for idx, c := range swtch.Case {
		if _, err := shared.ParseExpr(ctx, c.GetValue(), visitor); err != nil {
			return fmt.Errorf("%s parse error: %w", switchCaseId(idx), err)
		} else if _, err := shared.ParseExpr(ctx, c.GetReturn(), visitor); err != nil {
			return fmt.Errorf("%s parse error: %w", switchReturnId(idx), err)
		}
	}

	if _, err := shared.ParseExpr(ctx, swtch.GetDefault(), visitor); err != nil {
		return fmt.Errorf("%s parse error: %w", switchDefaultId, err)
	}

	return nil
}

func CompileSwitch(ctx shared.Runtime, swtch *flowv1beta1.Switch) (shared.EvalExpr, error) {
	progMap := map[string]cel.Program{}
	prg, err := shared.CompileExpr(ctx, swtch.GetValue())
	if err != nil {
		return nil, fmt.Errorf("%s compile error: %w", switchExprId, err)
	}
	progMap[switchExprId] = prg

	for idx, c := range swtch.Case {
		caseId := switchCaseId(idx)
		prg, err := shared.CompileExpr(ctx, c.GetValue())
		if err != nil {
			return nil, fmt.Errorf("%s compile error: %w", caseId, err)
		}
		progMap[caseId] = prg

		returnId := switchReturnId(idx)
		prg, err = shared.CompileExpr(ctx, c.GetReturn())
		if err != nil {
			return nil, fmt.Errorf("%s compile error: %w", returnId, err)
		}
		progMap[returnId] = prg
	}

	prg, err = shared.CompileExpr(ctx, swtch.GetDefault())
	if err != nil {
		return nil, fmt.Errorf("%s compile error: %w", switchDefaultId, err)
	}
	progMap[switchDefaultId] = prg

	return EvalSwitch(ctx, swtch, progMap)
}

func EvalSwitch(ctx shared.Runtime, swtch *flowv1beta1.Switch, progMap map[string]cel.Program) (shared.EvalExpr, error) {
	vars, err := ctx.Vars()
	if err != nil {
		return nil, err
	}

	return shared.EvalExprFunc(func(ctx context.Context) ref.Val {
		exprVal, _, err := progMap[switchExprId].ContextEval(ctx, vars)
		if err != nil {
			return types.WrapErr(fmt.Errorf("%s eval error: %w", switchExprId, err))
		}

		for idx := range swtch.Case {
			caseId := switchCaseId(idx)
			caseVal, _, err := progMap[caseId].ContextEval(ctx, vars)
			if err != nil {
				return types.WrapErr(fmt.Errorf("%s eval error: %w", caseId, err))
			}

			if exprVal.Equal(caseVal) == types.True {
				returnId := switchReturnId(idx)
				retVal, _, err := progMap[returnId].ContextEval(ctx, vars)
				if err != nil {
					return types.WrapErr(fmt.Errorf("%s eval error: %w", returnId, err))
				}

				return retVal
			}
		}

		if defExpr, ok := progMap[switchDefaultId]; ok {
			defVal, _, err := defExpr.ContextEval(ctx, vars)
			if err != nil {
				return types.WrapErr(fmt.Errorf("%s eval error: %w", switchDefaultId, err))
			}

			return defVal
		}

		return nil
	}), nil
}
