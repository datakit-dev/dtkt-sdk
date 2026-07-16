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

func ParseSwitch(env shared.Env, swtch *flowv1beta1.Var_Switch, visitor shared.ExprVisitor) error {
	if _, err := shared.ParseExpr(env, swtch.GetValue(), visitor); err != nil {
		return fmt.Errorf("%s parse error: %w", switchExprId, err)
	}

	for idx, c := range swtch.Case {
		if _, err := shared.ParseExpr(env, c.GetValue(), visitor); err != nil {
			return fmt.Errorf("%s parse error: %w", switchCaseId(idx), err)
		} else if _, err := shared.ParseExpr(env, c.GetReturn(), visitor); err != nil {
			return fmt.Errorf("%s parse error: %w", switchReturnId(idx), err)
		}
	}

	if _, err := shared.ParseExpr(env, swtch.GetDefault(), visitor); err != nil {
		return fmt.Errorf("%s parse error: %w", switchDefaultId, err)
	}

	return nil
}

func CompileSwitch(env shared.Env, swtch *flowv1beta1.Var_Switch) (shared.Program, error) {
	progMap := map[string]cel.Program{}
	prg, err := shared.CompileExpr(env, swtch.GetValue())
	if err != nil {
		return nil, fmt.Errorf("%s compile error: %w", switchExprId, err)
	}
	progMap[switchExprId] = prg

	for idx, c := range swtch.Case {
		caseId := switchCaseId(idx)
		prg, err := shared.CompileExpr(env, c.GetValue())
		if err != nil {
			return nil, fmt.Errorf("%s compile error: %w", caseId, err)
		}
		progMap[caseId] = prg

		returnId := switchReturnId(idx)
		prg, err = shared.CompileExpr(env, c.GetReturn())
		if err != nil {
			return nil, fmt.Errorf("%s compile error: %w", returnId, err)
		}
		progMap[returnId] = prg
	}

	prg, err = shared.CompileExpr(env, swtch.GetDefault())
	if err != nil {
		return nil, fmt.Errorf("%s compile error: %w", switchDefaultId, err)
	}
	progMap[switchDefaultId] = prg

	return EvalSwitch(env, swtch, progMap)
}

func EvalSwitch(env shared.Env, swtch *flowv1beta1.Var_Switch, progMap map[string]cel.Program) (shared.Program, error) {
	return shared.ProgramFunc(func(ctx context.Context, _ shared.Env) ref.Val {
		// Use the env captured at Compile time so node-scoped bindings such
		// as "self" remain available (see spec.Var.Compile).
		vars, err := env.Vars(ctx)
		if err != nil {
			return types.WrapErr(err)
		}

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

func switchCaseId(idx int) string {
	return fmt.Sprintf("switch.case[%d].value", idx)
}

func switchReturnId(idx int) string {
	return fmt.Sprintf("switch.case[%d].return", idx)
}
