package spec

import (
	"context"
	"errors"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

func ParseAction(ctx shared.Runtime, action *flowv1beta1.Action, visitor shared.ParseExprFunc) error {
	switch {
	case action.GetCall() != nil:
		if err := ParseCall(ctx, action.GetCall(), visitor); err != nil {
			return err
		}
	case action.GetUser() != nil:
		if err := ParseUserAction(ctx, action.GetUser(), visitor); err != nil {
			return err
		}
	default:
		return errors.New("call or user required")
	}

	if action.GetRunIf() != "" {
		if _, err := shared.ParseExpr(ctx, action.GetRunIf(), visitor); err != nil {
			return err
		}
	}

	if action.GetOnError() != "" {
		if _, err := shared.ParseExpr(ctx, action.GetOnError(), visitor); err != nil {
			return err
		}
	}

	return nil
}

func CompileAction(ctx shared.Runtime, action *flowv1beta1.Action) (_ shared.EvalExpr, err error) {
	var main shared.EvalExpr
	switch {
	case action.GetCall() != nil:
		main, err = CompileCall(ctx, action)
		if err != nil {
			return nil, err
		}
	case action.GetUser() != nil:
		main, err = CompileUserAction(ctx, action)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("call or user required")
	}

	var runIf cel.Program
	if action.GetRunIf() != "" {
		runIf, err = shared.CompileExpr(ctx, action.GetRunIf(),
			cel.EvalOptions(cel.EvalOption(cel.BoolKind)),
		)
		if err != nil {
			return nil, err
		}
	}

	var onError cel.Program
	if action.GetOnError() != "" {
		onError, err = shared.CompileExpr(ctx, action.GetOnError())
		if err != nil {
			return nil, err
		}
	}

	vars, err := ctx.Vars()
	if err != nil {
		return nil, err
	}

	return CacheableEval(action, shared.EvalExprFunc(func(ctx context.Context) ref.Val {
		shouldRun := true
		if runIf != nil {
			val, _, err := runIf.ContextEval(ctx, vars)
			if err != nil {
				return types.WrapErr(err)
			}
			shouldRun = val == types.True
		}

		if shouldRun {
			refVal := main.Eval(ctx)
			if valErr, ok := refVal.Value().(error); ok {
				if onError != nil {
					refVal, _, err = onError.ContextEval(ctx, vars)
					if err != nil {
						return types.WrapErr(errors.Join(valErr, err))
					}
					return refVal
				} else {
					return types.WrapErr(valErr)
				}
			}

			return refVal
		}

		return types.NullValue
	})), nil
}
