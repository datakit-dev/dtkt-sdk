package spec

import (
	"context"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
)

var _ shared.ExecNode = (*Output)(nil)

type Output struct {
	id   string
	node *flowv1beta1.Output
	expr *cel.Ast
	prog cel.Program
}

func NewOutput(env shared.Env, node *flowv1beta1.Output, visitor shared.NodeVisitor) (*Output, error) {
	expr, err := shared.ParseExpr(env, node.GetValue(), visitor.ExprVisitor(GetNodeIdent(node)))
	if err != nil {
		return nil, err
	}

	return &Output{
		id:   GetNodeIdent(node),
		node: node,
		expr: expr,
	}, nil
}

func (o *Output) Compile(ctx context.Context, env shared.Env) (err error) {
	prog, err := env.Program(o.expr)
	if err != nil {
		return err
	}

	o.prog = prog

	return nil
}

func (o *Output) Eval(ctx context.Context, env shared.Env) (any, error) {
	vars, err := env.Vars(ctx)
	if err != nil {
		return nil, err
	}

	value, _, err := o.prog.ContextEval(ctx, vars)
	if err != nil {
		return nil, err
	}

	return value.Value(), nil
}

func (o *Output) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) error {
	return pipe.Run(ctx, func(ctx context.Context, value any) (any, error) {
		return o.Eval(ctx, env)
	})
}
