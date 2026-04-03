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

var _ shared.ExecNode = (*Ticker)(nil)

// var _ CallCloser = (*Ticker)(nil)

type Ticker struct {
	id string

	initial,
	every time.Duration

	getValueExpr,
	startIfExpr,
	stopIfExpr *cel.Ast

	getValue,
	startIf,
	stopIf cel.Program
}

func NewTicker(env shared.Env, node *flowv1beta1.Stream, visitor shared.ExprVisitFunc) (_ *Ticker, err error) {
	var startIfExpr *cel.Ast
	if node.GetStartIf() != "" {
		startIfExpr, err = shared.ParseExpr(env, node.GetStartIf(), visitor)
		if err != nil {
			return nil, err
		}
	}

	var stopIfExpr *cel.Ast
	if node.GetStopIf() != "" {
		stopIfExpr, err = shared.ParseExpr(env, node.GetStopIf(), visitor)
		if err != nil {
			return nil, err
		}
	}

	ticker := node.GetGenerate()

	var getValueExpr *cel.Ast
	if ticker.GetValue() != "" {
		getValueExpr, err = shared.ParseExpr(env, ticker.GetValue(), visitor)
		if err != nil {
			return nil, err
		}
	}

	id := GetID(node)
	if ticker.Every == nil || ticker.Every.AsDuration() <= 0 {
		return nil, fmt.Errorf("%s.generate.every invalid: must be > 0", id)
	}

	if ticker.Initial == nil {
		ticker.Initial = ticker.Every
	}

	return &Ticker{
		id: id,

		initial: ticker.Initial.AsDuration(),
		every:   ticker.Every.AsDuration(),

		getValueExpr: getValueExpr,
		startIfExpr:  startIfExpr,
		stopIfExpr:   stopIfExpr,
	}, nil
}

func (t *Ticker) Compile(run shared.Runtime) (err error) {
	env, err := run.Env()
	if err != nil {
		return err
	}

	if t.getValueExpr != nil {
		t.getValue, err = env.Program(t.getValueExpr)
		if err != nil {
			return fmt.Errorf("%s.generate.value compile error: %w", t.id, err)
		}
	}

	if t.startIfExpr != nil {
		t.startIf, err = env.Program(t.startIfExpr)
		if err != nil {
			return fmt.Errorf("%s.startIf compile error: %w", t.id, err)
		}
	}

	if t.stopIfExpr != nil {
		t.stopIf, err = env.Program(t.stopIfExpr)
		if err != nil {
			return fmt.Errorf("%s.stopIf compile error: %w", t.id, err)
		}
	}

	return nil
}

func (t *Ticker) Send() (shared.SendFunc, bool) {
	return func(run shared.Runtime, sendCh chan<- ref.Val) error {
		env, err := run.Env()
		if err != nil {
			return err
		}

		select {
		case <-run.Context().Done():
			return context.Cause(run.Context())
		case tick := <-time.After(t.initial):
			select {
			case <-run.Context().Done():
				return context.Cause(run.Context())
			case sendCh <- t.GetValue(run.Context(), env, tick):
			}
		}

		for {
			select {
			case <-run.Context().Done():
				return context.Cause(run.Context())
			case tick := <-time.After(t.every):
				select {
				case <-run.Context().Done():
					return context.Cause(run.Context())
				case sendCh <- t.GetValue(run.Context(), env, tick):
				}
			}
		}
	}, true
}

func (t *Ticker) GetValue(ctx context.Context, env shared.Env, tick time.Time) ref.Val {
	if t.getValue != nil {
		value, _, err := t.getValue.ContextEval(ctx, env.Vars())
		if err != nil {
			return types.WrapErr(err)
		}
		return value
	}

	return env.TypeAdapter().NativeToValue(tick)
}

func (t *Ticker) Eval() (shared.EvalFunc, bool) {
	return nil, false
}

func (t *Ticker) Recv() (shared.RecvFunc, bool) {
	return nil, false
}

// func (t *Ticker) Run(env shared.Env) {
// 	select {
// 	case <-run.Context().Done():
// 		return
// 	case tick := <-time.After(t.initial):
// 		select {
// 		case <-run.Context().Done():
// 			return
// 		case t.eventCh <- t.GetValue(run.Context(), env, tick):
// 		}
// 	}

// 	ticker := time.NewTicker(t.every)
// 	for {
// 		select {
// 		case <-run.Context().Done():
// 			return
// 		case tick := <-ticker.C:
// 			select {
// 			case <-run.Context().Done():
// 				return
// 			case t.eventCh <- t.GetValue(run.Context(), env, tick):
// 			}
// 		}
// 	}
// }

func (t *Ticker) Close() error {
	return nil
}

// func ParseTicker(env shared.Env, ticker *flowv1beta1.Ticker, visitor shared.ExprVisitFunc) error {
// 	if ticker.GetValue() != "" {
// 		_, err := shared.ParseExpr(env, ticker.GetValue(), visitor)
// 		return err
// 	}
// 	return nil
// }
