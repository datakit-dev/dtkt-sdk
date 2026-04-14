package spec

import (
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

func ParseSwitch(env shared.Env, swtch *flowv1beta1.Var_Switch, visitor shared.ExprVisitFunc) error {
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

func CompileSwitch(run shared.Runtime, swtch *flowv1beta1.Var_Switch) (shared.Program, error) {
	progMap := map[string]cel.Program{}
	prg, err := shared.CompileExpr(run, swtch.GetValue())
	if err != nil {
		return nil, fmt.Errorf("%s compile error: %w", switchExprId, err)
	}
	progMap[switchExprId] = prg

	for idx, c := range swtch.Case {
		caseId := switchCaseId(idx)
		prg, err := shared.CompileExpr(run, c.GetValue())
		if err != nil {
			return nil, fmt.Errorf("%s compile error: %w", caseId, err)
		}
		progMap[caseId] = prg

		returnId := switchReturnId(idx)
		prg, err = shared.CompileExpr(run, c.GetReturn())
		if err != nil {
			return nil, fmt.Errorf("%s compile error: %w", returnId, err)
		}
		progMap[returnId] = prg
	}

	prg, err = shared.CompileExpr(run, swtch.GetDefault())
	if err != nil {
		return nil, fmt.Errorf("%s compile error: %w", switchDefaultId, err)
	}
	progMap[switchDefaultId] = prg

	return EvalSwitch(run, swtch, progMap)
}

func EvalSwitch(run shared.Runtime, swtch *flowv1beta1.Var_Switch, progMap map[string]cel.Program) (shared.Program, error) {
	env, err := run.Env()
	if err != nil {
		return nil, err
	}

	return shared.EvalFunc(func(run shared.Runtime) ref.Val {
		exprVal, _, err := progMap[switchExprId].ContextEval(run.Context(), env.Vars())
		if err != nil {
			return types.WrapErr(fmt.Errorf("%s eval error: %w", switchExprId, err))
		}

		for idx := range swtch.Case {
			caseId := switchCaseId(idx)
			caseVal, _, err := progMap[caseId].ContextEval(run.Context(), env.Vars())
			if err != nil {
				return types.WrapErr(fmt.Errorf("%s eval error: %w", caseId, err))
			}

			if exprVal.Equal(caseVal) == types.True {
				returnId := switchReturnId(idx)
				retVal, _, err := progMap[returnId].ContextEval(run.Context(), env.Vars())
				if err != nil {
					return types.WrapErr(fmt.Errorf("%s eval error: %w", returnId, err))
				}

				return retVal
			}
		}

		if defExpr, ok := progMap[switchDefaultId]; ok {
			defVal, _, err := defExpr.ContextEval(run.Context(), env.Vars())
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
