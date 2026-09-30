package impl

import (
	"context"
	"fmt"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

const (
	switchExprId    = "switch.expr"
	switchDefaultId = "switch.default"
)

type Switch struct {
	node *flowv1beta1.Var_Switch

	exprMap map[string]*cel.Ast
	progMap map[string]shared.Program
}

func NewSwitch(env shared.Env, node *flowv1beta1.Var_Switch, visitor shared.ExprVisitor) (*Switch, error) {
	expr, err := env.ParseExpr("switch.value", node.GetValue(), visitor)
	if err != nil {
		return nil, fmt.Errorf("%s parse error: %w", switchExprId, err)
	}

	defaultExpr, err := env.ParseExpr("switch.default", node.GetDefault(), visitor)
	if err != nil {
		return nil, fmt.Errorf("%s parse error: %w", switchDefaultId, err)
	}

	exprMap := map[string]*cel.Ast{
		switchExprId:    expr,
		switchDefaultId: defaultExpr,
	}
	for idx, c := range node.Case {
		id := switchCaseId(idx)
		expr, err := env.ParseExpr(id, c.GetValue(), visitor)
		if err != nil {
			return nil, fmt.Errorf("%s parse error: %w", id, err)
		}

		exprMap[id] = expr

		id = switchReturnId(idx)
		expr, err = env.ParseExpr(id, c.GetReturn(), visitor)
		if err != nil {
			return nil, fmt.Errorf("%s parse error: %w", id, err)
		}

		exprMap[id] = expr
	}

	return &Switch{
		node:    node,
		exprMap: exprMap,
	}, nil
}

func (s *Switch) Compile(ctx context.Context, env shared.Env) error {
	s.progMap = map[string]shared.Program{}
	for id, expr := range s.exprMap {
		prog, err := env.CompileExpr(id, expr)
		if err != nil {
			return err
		}
		s.progMap[id] = prog
	}

	return nil
}

func (s *Switch) Eval(ctx context.Context, env shared.Env) ref.Val {
	exprVal := s.progMap[switchExprId].Eval(ctx, env)
	for idx := range s.node.Case {
		caseId := switchCaseId(idx)
		caseVal := s.progMap[caseId].Eval(ctx, env)
		if exprVal.Equal(caseVal) == types.True {
			returnId := switchReturnId(idx)
			return s.progMap[returnId].Eval(ctx, env)
		}
	}

	return s.progMap[switchDefaultId].Eval(ctx, env)
}

func switchCaseId(idx int) string {
	return fmt.Sprintf("switch.case[%d].value", idx)
}

func switchReturnId(idx int) string {
	return fmt.Sprintf("switch.case[%d].return", idx)
}
