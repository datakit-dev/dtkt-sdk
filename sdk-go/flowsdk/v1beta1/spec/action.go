package spec

import (
	"errors"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

var _ shared.ExecNode = (*Action)(nil)

type Action struct {
	ExecNodeCloser
	node *flowv1beta1.Action
	eval shared.Program
}

func NewAction(env shared.Env, node *flowv1beta1.Action, visitor shared.NodeVisitFunc) (*Action, error) {
	switch {
	case node.GetCall() != nil:
		caller, err := NewActionCaller(env, node, visitor)
		if err != nil {
			return nil, err
		}

		return &Action{
			ExecNodeCloser: caller,
			node:           node,
		}, nil
	case node.GetUser() != nil:
		// err := ParseUserAction(env, node.GetUser(), visitor)
		// if err != nil {
		// 	return nil, err
		// }
	}

	return nil, errors.New("call or user required")
}

// func (a *Action) Compile(run shared.Runtime) error {
// 	// switch {
// 	// case a.call != nil:
// 	// 	err := a.call.Compile(run)
// 	// 	if err != nil {
// 	// 		return err
// 	// 	}

// 	// 	caller, err := NewCaller(run, a.node, a.call)
// 	// 	if err != nil {
// 	// 		return err
// 	// 	}

// 	// 	eval, ok := caller.Eval()
// 	// 	if !ok {
// 	// 		return fmt.Errorf("expected unary method for action: %s", a.node.Id)
// 	// 	}

// 	// 	a.eval = CacheableEval(a.node, eval)

// 	// 	return nil

// 	// 	// TBD:
// 	// 	// case action.GetUser() != nil:
// 	// 	// 	userEval, err := CompileUserAction(run, action)
// 	// 	// 	if err != nil {
// 	// 	// 		return nil, err
// 	// 	// 	}
// 	// 	// 	main = userEval.Eval

// 	// 	// CacheableEval(action, shared.EvalFunc(func(run shared.Runtime) ref.Val {
// 	// 	// 	// shouldRun := true
// 	// 	// 	// if runIf != nil {
// 	// 	// 	// 	val, _, err := runIf.ContextEval(run.Context(), env.Vars())
// 	// 	// 	// 	if err != nil {
// 	// 	// 	// 		return types.WrapErr(err)
// 	// 	// 	// 	}
// 	// 	// 	// 	shouldRun = val == types.True
// 	// 	// 	// }

// 	// 	// 	// if shouldRun {
// 	// 	// 	refVal := main.Eval(run)
// 	// 	// 	if valErr, ok := refVal.Value().(error); ok {
// 	// 	// 		// if onError != nil {
// 	// 	// 		// 	refVal, _, err = onError.ContextEval(run.Context(), env.Vars())
// 	// 	// 		// 	if err != nil {
// 	// 	// 		// 		return types.WrapErr(errors.Join(valErr, err))
// 	// 	// 		// 	}
// 	// 	// 		// 	return refVal
// 	// 	// 		// } else {
// 	// 	// 		return types.WrapErr(valErr)
// 	// 	// 		// }
// 	// 	// 	}

// 	// 	// 	return refVal
// 	// 	// 	// }

// 	// 	// 	// return types.NullValue
// 	// 	// }))

// 	// }

// 	return errors.New("call or user required")
// }

// func (a *Action) Eval() (shared.EvalFunc, bool) {
// 	// return func(run shared.Runtime) ref.Val {
// 	// 	return a.eval.Eval(run)
// 	// }, true
// 	return nil, false
// }
// func (a *Action) Recv() (shared.RecvFunc, bool) { return nil, false }
// func (a *Action) Send() (shared.SendFunc, bool) { return nil, false }
// func (a *Action) HasCached() (ref.Val, bool)    { return nil, false }
