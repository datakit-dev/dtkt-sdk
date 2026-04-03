package spec

import (
	"errors"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var _ shared.ExecNode = (*Action)(nil)

type Action struct {
	node *flowv1beta1.Action
	eval shared.EvalFunc
}

func NewAction(env shared.Env, node *flowv1beta1.Action, visitor shared.ExprVisitFunc) (*Action, error) {
	if err := ParseAction(env, node, visitor); err != nil {
		return nil, err
	}
	return &Action{node: node}, nil
}

func (a *Action) Compile(run shared.Runtime) error {
	eval, err := CompileAction(run, a.node)
	if err != nil {
		return err
	}
	a.eval = eval.Eval
	return nil
}

func (a *Action) Eval() (shared.EvalFunc, bool) { return a.eval, true }
func (a *Action) Recv() (shared.RecvFunc, bool) { return nil, false }
func (a *Action) Send() (shared.SendFunc, bool) { return nil, false }

func ParseAction(env shared.Env, action *flowv1beta1.Action, visitor shared.ExprVisitFunc) error {
	switch {
	case action.GetCall() != nil:
		if err := ParseMethodCall(env, action.GetCall(), visitor); err != nil {
			return err
		}
	case action.GetUser() != nil:
		if err := ParseUserAction(env, action.GetUser(), visitor); err != nil {
			return err
		}
	default:
		return errors.New("call or user required")
	}

	if action.GetRunIf() != "" {
		if _, err := shared.ParseExpr(env, action.GetRunIf(), visitor); err != nil {
			return err
		}
	}

	if action.GetOnError() != "" {
		if _, err := shared.ParseExpr(env, action.GetOnError(), visitor); err != nil {
			return err
		}
	}

	return nil
}

func CompileAction(run shared.Runtime, action *flowv1beta1.Action) (_ shared.Eval, err error) {
	var main shared.EvalFunc
	switch {
	case action.GetCall() != nil:
		caller, err := NewCaller(run, action)
		if err != nil {
			return nil, err
		}
		eval, _ := caller.Eval()
		main = eval
	case action.GetUser() != nil:
		userEval, err := CompileUserAction(run, action)
		if err != nil {
			return nil, err
		}
		main = userEval.Eval
	default:
		return nil, errors.New("call or user required")
	}

	var runIf cel.Program
	if action.GetRunIf() != "" {
		runIf, err = shared.CompileExpr(run, action.GetRunIf())
		if err != nil {
			return nil, err
		}
	}

	var onError cel.Program
	if action.GetOnError() != "" {
		onError, err = shared.CompileExpr(run, action.GetOnError())
		if err != nil {
			return nil, err
		}
	}

	env, err := run.Env()
	if err != nil {
		return nil, err
	}

	return CacheableEval(action, shared.EvalFunc(func(run shared.Runtime) ref.Val {
		shouldRun := true
		if runIf != nil {
			val, _, err := runIf.ContextEval(run.Context(), env.Vars())
			if err != nil {
				return types.WrapErr(err)
			}
			shouldRun = val == types.True
		}

		if shouldRun {
			refVal := main.Eval(run)
			if valErr, ok := refVal.Value().(error); ok {
				if onError != nil {
					refVal, _, err = onError.ContextEval(run.Context(), env.Vars())
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
