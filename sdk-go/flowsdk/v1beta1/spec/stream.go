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

func ParseStream(run shared.Runtime, stream *flowv1beta1.Stream, visitor shared.ParseExprFunc) error {
	switch {
	case stream.GetCall() != nil:
		if err := ParseCall(run, stream.GetCall(), visitor); err != nil {
			return err
		}
	case stream.GetGenerate() != nil:
		if err := ParseTicker(run, stream.GetGenerate(), visitor); err != nil {
			return err
		}
	default:
		return errors.New("call or generate required")
	}

	if stream.GetStartIf() != "" {
		if _, err := shared.ParseExpr(run, stream.GetStartIf(), visitor); err != nil {
			return err
		}
	}

	if stream.GetStopIf() != "" {
		if _, err := shared.ParseExpr(run, stream.GetStopIf(), visitor); err != nil {
			return err
		}
	}

	return nil
}

func CompileStream(run shared.Runtime, stream *flowv1beta1.Stream) (_ shared.EvalExpr, err error) {
	var (
		main  shared.EvalExpr
		close CallCloser
	)
	switch {
	case stream.GetCall() != nil:
		main, close, err = CompileCallCloser(run, stream)
		if err != nil {
			return nil, err
		}
	case stream.GetGenerate() != nil:
		main, err = CompileTicker(run, stream.GetGenerate())
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("call or generate required")
	}

	var (
		startIf   cel.Program
		isStarted bool
	)
	if stream.GetStartIf() != "" {
		startIf, err = shared.CompileExpr(run, stream.GetStartIf(),
			cel.EvalOptions(cel.EvalOption(cel.BoolKind)),
		)
		if err != nil {
			return nil, err
		}
	}

	var (
		stopIf    cel.Program
		isStopped bool
	)
	if stream.GetStopIf() != "" {
		stopIf, err = shared.CompileExpr(run, stream.GetStopIf())
		if err != nil {
			return nil, err
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

	return shared.EvalExprFunc(func(ctx context.Context) ref.Val {
		if isStopped {
			return env.CELTypeAdapter().NativeToValue(&flowv1beta1.Runtime_EOF{})
		}

		if stopIf != nil {
			val, _, err := stopIf.ContextEval(ctx, vars)
			if err != nil {
				return types.WrapErr(err)
			}
			if val == types.True {
				if close != nil {
					if err := close(); err != nil {
						return types.WrapErr(err)
					}
				}

				isStopped = true
				return env.CELTypeAdapter().NativeToValue(&flowv1beta1.Runtime_EOF{})
			}
		}

		if !isStarted {
			if startIf != nil {
				val, _, err := startIf.ContextEval(ctx, vars)
				if err != nil {
					return types.WrapErr(err)
				}
				if val != types.True {
					return types.NullValue
				}
			}
			isStarted = true
		}

		return main.Eval(ctx)
	}), nil
}
