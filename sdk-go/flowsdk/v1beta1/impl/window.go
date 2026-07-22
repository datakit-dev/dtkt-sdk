package impl

import (
	"context"
	"fmt"
	"slices"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
)

var _ shared.ParsedNode = (*Window)(nil)

type (
	Window struct {
		ident  string
		node   *flowv1beta1.Stream
		stream *Stream

		sliding    bool
		size, step int

		sourceExpr *cel.Ast
		sourceProg shared.Program

		buffer []any
		seen   int
	}
)

func NewWindow(env shared.Env, node *flowv1beta1.Stream, stream *Stream, visitor shared.NodeVisitor) (_ *Window, err error) {
	var (
		ident  = shared.GetNodeIdent(node)
		window = node.GetWindow()
	)

	sourceExpr, err := env.ParseExpr("window.source", window.GetSource(), visitor.ExprVisitor(ident))
	if err != nil {
		return nil, err
	}

	w := &Window{
		ident:      ident,
		node:       node,
		stream:     stream,
		sourceExpr: sourceExpr,
	}

	switch {
	case window.GetTumbling() != nil:
		if window.GetTumbling().GetSize() == 0 {
			return nil, fmt.Errorf("%s.window.tumbling.size invalid: must be > 0", ident)
		}

		w.size = int(window.GetTumbling().GetSize())
	case window.GetSliding() != nil:
		if window.GetSliding().GetSize() == 0 {
			return nil, fmt.Errorf("%s.window.sliding.size invalid: must be > 0", ident)
		} else if window.GetSliding().GetStep() == 0 {
			return nil, fmt.Errorf("%s.window.sliding.step invalid: must be > 0", ident)
		}

		w.sliding = true
		w.size = int(window.GetSliding().GetSize())
		w.step = int(window.GetSliding().GetStep())
	default:
		return nil, fmt.Errorf("%s.window: tumbling or sliding required", ident)
	}

	return w, nil
}

func (w *Window) Ident() string {
	return w.ident
}

func (w *Window) Compile(ctx context.Context, env shared.Env) error {
	prog, err := env.CompileExpr("window.source", w.sourceExpr)
	if err != nil {
		return fmt.Errorf("%s.window.source compile error: %w", w.ident, err)
	}

	w.sourceProg = prog
	return nil
}

func (w *Window) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) error {
	return pipe.Run(ctx, func(ctx context.Context, _ any) (any, error) {
		return w.push(w.sourceProg.Eval(ctx, env).Value()), nil
	})
}

// push appends value to the window's buffer and, once a full window is
// available, returns a copy of its contents; otherwise it returns nil.
func (w *Window) push(value any) any {
	if w.sliding {
		return w.pushSliding(value)
	}
	return w.pushTumbling(value)
}

func (w *Window) pushTumbling(value any) any {
	w.buffer = append(w.buffer, value)
	if len(w.buffer) < w.size {
		return nil
	}

	out := slices.Clone(w.buffer)
	w.buffer = nil
	return out
}

func (w *Window) pushSliding(value any) any {
	w.buffer = append(w.buffer, value)
	w.seen++

	if len(w.buffer) > w.size {
		w.buffer = w.buffer[len(w.buffer)-w.size:]
	}

	if len(w.buffer) < w.size || (w.seen-w.size)%w.step != 0 {
		return nil
	}

	return slices.Clone(w.buffer)
}
