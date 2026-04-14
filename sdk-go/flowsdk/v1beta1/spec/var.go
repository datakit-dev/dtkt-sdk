package spec

import (
	"errors"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var _ shared.ExecNode = (*Var)(nil)

type Var struct {
	node  *flowv1beta1.Var
	cache *EvalCache
}

func NewVar(env shared.Env, node *flowv1beta1.Var, visitor shared.NodeVisitFunc) (*Var, error) {
	switch {
	case node.GetSwitch() != nil:
		err := ParseSwitch(env, node.GetSwitch(), visitor.ExprVisitor(GetID(node)))
		if err != nil {
			return nil, err
		}
	case node.GetValue() != "":
		_, err := shared.ParseExpr(env, node.GetValue(), visitor.ExprVisitor(GetID(node)))
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("switch or value required")
	}

	return &Var{
		node: node,
	}, nil
}

func (v *Var) Compile(run shared.Runtime) error {
	env, err := run.Env()
	if err != nil {
		return err
	}

	switch {
	case v.node.GetSwitch() != nil:
		prog, err := CompileSwitch(run, v.node.GetSwitch())
		if err != nil {
			return err
		}

		v.cache = NewEvalCache(v.node, prog)
	case v.node.GetValue() != "":
		prog, err := shared.CompileExpr(run, v.node.GetValue())
		if err != nil {
			return err
		}

		v.cache = NewEvalCache(v.node, shared.EvalFunc(func(run shared.Runtime) ref.Val {
			val, _, err := prog.ContextEval(run.Context(), env.Vars())
			if err != nil {
				return types.WrapErr(err)
			}

			return val
		}))
	default:
		return errors.New("switch or value required")
	}

	return nil
}

func (v *Var) Eval() (shared.EvalFunc, bool) {
	return v.cache.Eval()
}

func (v *Var) HasCached() (ref.Val, bool) {
	return v.cache.HasCached()
}

func (v *Var) Recv() (shared.RecvFunc, bool) { return nil, false }
func (v *Var) Send() (shared.SendFunc, bool) { return nil, false }
