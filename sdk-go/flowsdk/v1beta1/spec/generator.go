package spec

import (
	"context"
	"fmt"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var _ shared.ExecNode = (*Generator)(nil)

type Generator struct {
	node   *flowv1beta1.Stream
	stream *Stream

	initial,
	every time.Duration

	getValueExpr *cel.Ast
	getValueProg cel.Program
}

func NewGenerator(env shared.Env, node *flowv1beta1.Stream, stream *Stream, visitor shared.NodeVisitFunc) (_ *Generator, err error) {
	var (
		id           = GetID(node)
		generate     = node.GetGenerate()
		getValueExpr *cel.Ast
	)

	if generate.GetValue() != "" {
		getValueExpr, err = shared.ParseExpr(env, generate.GetValue(), visitor.ExprVisitor(GetID(node)))
		if err != nil {
			return nil, err
		}
	}

	if generate.Every == nil || generate.Every.AsDuration() <= 0 {
		return nil, fmt.Errorf("%s.generate.every invalid: must be > 0", id)
	}

	if generate.Initial == nil {
		generate.Initial = generate.Every
	}

	return &Generator{
		node:   node,
		stream: stream,

		initial: generate.Initial.AsDuration(),
		every:   generate.Every.AsDuration(),

		getValueExpr: getValueExpr,
	}, nil
}

func (t *Generator) Compile(run shared.Runtime) (err error) {
	env, err := run.Env()
	if err != nil {
		return err
	}

	if t.getValueExpr != nil {
		t.getValueProg, err = env.Program(t.getValueExpr)
		if err != nil {
			return fmt.Errorf("%s.generate.value compile error: %w", GetID(t.node), err)
		}
	}

	return nil
}

func (t *Generator) Send() (shared.SendFunc, bool) {
	return func(run shared.Runtime, sendCh chan<- ref.Val) error {
		env, err := run.Env()
		if err != nil {
			return err
		}

		var initialSent bool
		// loop:
		for {
			if !initialSent {
				select {
				case <-run.Context().Done():
					return context.Cause(run.Context())
				case tick := <-time.After(t.initial):
					// shouldStart, err := t.stream.ShouldStart(run)
					// if err != nil {
					// 	return err
					// } else if shouldStart {
					select {
					case <-run.Context().Done():
						return context.Cause(run.Context())
					case sendCh <- t.getValue(run.Context(), env, tick):
						initialSent = true
					}
					// } else {
					// 	continue loop
					// }
				}
			}

			// shouldStop, err := t.stream.ShouldStop(run)
			// if err != nil {
			// 	return err
			// } else if shouldStop {
			// 	select {
			// 	case <-run.Context().Done():
			// 		return context.Cause(run.Context())
			// 	case sendCh <- env.TypeAdapter().NativeToValue(&flowv1beta1.Runtime_Done{
			// 		Id:     GetID(t.node),
			// 		Reason: "stream stopped",
			// 	}):
			// 		return nil
			// 	}
			// }

			select {
			case <-run.Context().Done():
				return context.Cause(run.Context())
			case tick := <-time.After(t.every):
				select {
				case <-run.Context().Done():
					return context.Cause(run.Context())
				case sendCh <- t.getValue(run.Context(), env, tick):
				}
			}
		}
	}, true
}

func (t *Generator) getValue(ctx context.Context, env shared.Env, tick time.Time) ref.Val {
	if t.getValueProg != nil {
		value, _, err := t.getValueProg.ContextEval(ctx, env.Vars())
		if err != nil {
			return types.WrapErr(err)
		}
		return value
	}

	return env.TypeAdapter().NativeToValue(tick)
}

func (t *Generator) Recv() (shared.RecvFunc, bool) { return nil, false }
func (t *Generator) Eval() (shared.EvalFunc, bool) { return nil, false }
func (t *Generator) HasCached() (ref.Val, bool)    { return nil, false }
func (t *Generator) Close() error                  { return nil }
