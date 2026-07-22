package impl

import (
	"context"
	"fmt"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/types/known/durationpb"
)

var _ shared.ParsedNode = (*Generator)(nil)

type (
	Generator struct {
		ident  string
		node   *flowv1beta1.Stream
		stream *Stream

		initial,
		every time.Duration

		getValueExpr *cel.Ast
		getValueProg shared.Program
	}
)

func NewGenerator(env shared.Env, node *flowv1beta1.Stream, stream *Stream, visitor shared.NodeVisitor) (_ *Generator, err error) {
	var (
		ident        = shared.GetNodeIdent(node)
		generate     = node.GetGenerate()
		getValueExpr *cel.Ast
	)

	if generate.GetValue() != "" {
		getValueExpr, err = env.ParseExpr("generate.value", generate.GetValue(), visitor.ExprVisitor(ident))
		if err != nil {
			return nil, err
		}
	}

	if generate.Every == nil || generate.Every.AsDuration() <= 0 {
		return nil, fmt.Errorf("%s.generate.every invalid: must be > 0", ident)
	}

	if generate.Initial == nil {
		generate.Initial = generate.Every
	}

	if generate.Initial.AsDuration() <= 0 {
		generate.Initial = durationpb.New(0)
	}

	return &Generator{
		ident:  ident,
		node:   node,
		stream: stream,

		initial: generate.Initial.AsDuration(),
		every:   generate.Every.AsDuration(),

		getValueExpr: getValueExpr,
	}, nil
}

func (g *Generator) Ident() string {
	return g.ident
}

func (g *Generator) Compile(ctx context.Context, env shared.Env) (err error) {
	if g.getValueExpr != nil {
		prog, err := env.CompileExpr("generate.value", g.getValueExpr)
		if err != nil {
			return fmt.Errorf("%s.generate.value compile error: %w", g.ident, err)
		}

		g.getValueProg = prog
	}

	return nil
}

func (g *Generator) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) error {
	var (
		initialSent bool
		duration    time.Duration
	)

	return pipe.Run(ctx, func(ctx context.Context, value any) (_ any, err error) {
		if !initialSent {
			duration = g.initial
			initialSent = true
		} else {
			duration = g.every
		}

		select {
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		case tick := <-time.After(duration):
			value, err = g.getValue(ctx, env, tick)
			if err != nil {
				return nil, err
			}
			return value, nil
		}
	})
}

func (g *Generator) getValue(ctx context.Context, env shared.Env, tick time.Time) (any, error) {
	if g.getValueProg != nil {
		return g.getValueProg.Eval(ctx, env), nil
	}

	return tick, nil
}

func (g *Generator) HasCached() (any, bool) { return nil, false }
func (g *Generator) Close() error           { return nil }
