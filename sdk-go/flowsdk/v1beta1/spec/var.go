package spec

import (
	"errors"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var _ shared.RuntimeNode = (*Var)(nil)

type Var struct {
	node *flowv1beta1.Var
	eval shared.EvalFunc
}

func NewVar(env shared.Env, node *flowv1beta1.Var, visitor shared.ExprVisitFunc) (*Var, error) {
	if err := ParseVar(env, node, visitor); err != nil {
		return nil, err
	}
	return &Var{node: node}, nil
}

func (v *Var) Compile(run shared.Runtime) error {
	eval, err := CompileVar(run, v.node)
	if err != nil {
		return err
	}
	v.eval = eval.Eval
	return nil
}

func (v *Var) Eval() (shared.EvalFunc, bool) { return v.eval, true }
func (v *Var) Recv() (shared.RecvFunc, bool) { return nil, false }
func (v *Var) Send() (shared.SendFunc, bool) { return nil, false }

func ParseVar(env shared.Env, node *flowv1beta1.Var, visitor shared.ExprVisitFunc) error {
	switch {
	case node.GetSwitch() != nil:
		return ParseSwitch(env, node.GetSwitch(), visitor)
	case node.GetValue() != "":
		_, err := shared.ParseExpr(env, node.GetValue(), visitor)
		return err
	}

	return errors.New("switch or value required")
}

func CompileVar(run shared.Runtime, node *flowv1beta1.Var) (_ shared.Eval, err error) {
	env, err := run.Env()
	if err != nil {
		return nil, err
	}

	var expr shared.Eval
	switch {
	case node.GetSwitch() != nil:
		expr, err = CompileSwitch(run, node.GetSwitch())
		if err != nil {
			return
		}
	case node.GetValue() != "":
		var prog cel.Program
		prog, err = shared.CompileExpr(run, node.GetValue())
		if err != nil {
			return
		}

		expr = shared.EvalFunc(func(run shared.Runtime) ref.Val {
			val, _, err := prog.ContextEval(run.Context(), env.Vars())
			if err != nil {
				return types.WrapErr(err)
			}

			return val
		})
	default:
		return nil, errors.New("switch or value required")
	}

	return CacheableEval(node, shared.EvalFunc(func(run shared.Runtime) ref.Val {
		return expr.Eval(run)
	})), nil
}
