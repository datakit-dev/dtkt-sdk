package impl

import (
	"context"
	"errors"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
)

var _ shared.ParsedNode = (*Var)(nil)

type Var struct {
	ident     string
	node      *flowv1beta1.Var
	switch_   *Switch
	valueExpr *cel.Ast
	cache     *EvalCache
}

func NewVar(env shared.Env, node *flowv1beta1.Var, visitor shared.NodeVisitor) (*Var, error) {
	v := &Var{
		ident: shared.GetNodeIdent(node),
		node:  node,
	}

	switch {
	case node.GetSwitch() != nil:
		switch_, err := NewSwitch(env, node.GetSwitch(), visitor.ExprVisitor(v.ident))
		if err != nil {
			return nil, err
		}

		v.switch_ = switch_
	case node.GetValue() != "":
		expr, err := env.ParseExpr("value", node.GetValue(), visitor.ExprVisitor(v.ident))
		if err != nil {
			return nil, err
		}

		v.valueExpr = expr
	default:
		return nil, errors.New("switch or value required")
	}

	return v, nil
}

func (v *Var) Ident() string {
	return v.ident
}

func (v *Var) Compile(ctx context.Context, env shared.Env) error {
	switch {
	case v.switch_ != nil:
		err := v.switch_.Compile(ctx, env)
		if err != nil {
			return err
		}

		v.cache = NewEvalCache(v.node, v.switch_)
	case v.valueExpr != nil:
		prog, err := env.CompileExpr("value", v.valueExpr)
		if err != nil {
			return err
		}

		v.cache = NewEvalCache(v.node, prog)
	default:
		return errors.New("switch or value required")
	}

	return nil
}

func (v *Var) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) error {
	return pipe.Run(ctx, func(ctx context.Context, value any) (any, error) {
		return v.cache.Eval(ctx, env), nil
	})
}
