package spec

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

func ParseTicker(run shared.Runtime, ticker *flowv1beta1.Ticker, visitor shared.ParseExprFunc) error {
	if ticker.GetValue() != "" {
		_, err := shared.ParseExpr(run, ticker.GetValue(), visitor)
		return err
	}
	return nil
}

func CompileTicker(run shared.Runtime, ticker *flowv1beta1.Ticker) (_ shared.EvalExpr, err error) {
	if ticker.Every == nil || ticker.Every.AsDuration() <= 0 {
		return nil, errors.New("ticker.every invalid: must be > 0")
	}

	if ticker.Initial == nil {
		ticker.Initial = ticker.Every
	}

	var getValue cel.Program
	if ticker.GetValue() != "" {
		getValue, err = shared.CompileExpr(run, ticker.GetValue())
		if err != nil {
			return nil, fmt.Errorf("ticker.value compile error: %w", err)
		}
	}

	env, err := run.Env()
	if err != nil {
		return nil, err
	}

	vars, err := run.Vars()
	if err != nil {
		return nil, err
	}

	var (
		initialDone    bool
		getValueOrTime = func(ctx context.Context, t time.Time) ref.Val {
			if getValue != nil {
				val, _, err := getValue.ContextEval(ctx, vars)
				if err != nil {
					return types.WrapErr(err)
				}
				return val
			}
			return env.CELTypeAdapter().NativeToValue(t)
		}
	)

	return shared.EvalExprFunc(func(ctx context.Context) ref.Val {
		if !initialDone {
			select {
			case <-ctx.Done():
				return types.WrapErr(ctx.Err())
			case now := <-time.After(ticker.Initial.AsDuration()):
				initialDone = true
				return getValueOrTime(ctx, now)
			}
		}

		select {
		case <-ctx.Done():
			return types.WrapErr(ctx.Err())
		case now := <-time.After(ticker.Every.AsDuration()):
			return getValueOrTime(ctx, now)
		}
	}), nil
}
