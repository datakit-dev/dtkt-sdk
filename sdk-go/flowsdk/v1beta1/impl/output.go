package impl

import (
	"context"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
)

var _ shared.ParsedNode = (*Output)(nil)

type Output struct {
	ident string
	node  *flowv1beta1.Output
	expr  *cel.Ast
	prog  shared.Program
}

func NewOutput(env shared.Env, node *flowv1beta1.Output, visitor shared.NodeVisitor) (*Output, error) {
	ident := shared.GetNodeIdent(node)
	expr, err := env.ParseExpr("value", node.GetValue(), visitor.ExprVisitor(ident))
	if err != nil {
		return nil, err
	}

	return &Output{
		ident: ident,
		node:  node,
		expr:  expr,
	}, nil
}

func (o *Output) Ident() string {
	return o.ident
}

func (o *Output) Compile(ctx context.Context, env shared.Env) (err error) {
	prog, err := env.CompileExpr("value", o.expr)
	if err != nil {
		return err
	}

	o.prog = prog

	return nil
}

func (o *Output) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) error {
	return pipe.Run(ctx, func(ctx context.Context, value any) (any, error) {
		return o.prog.Eval(ctx, env), nil
	})
}
