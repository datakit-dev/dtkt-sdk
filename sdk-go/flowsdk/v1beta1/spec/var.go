package spec

import (
	"context"
	"errors"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var _ shared.ExecNode = (*Var)(nil)

type Var struct {
	id    string
	node  *flowv1beta1.Var
	cache *EvalCache
}

func NewVar(env shared.Env, node *flowv1beta1.Var, visitor shared.NodeVisitor) (*Var, error) {
	switch {
	case node.GetSwitch() != nil:
		err := ParseSwitch(env, node.GetSwitch(), visitor.ExprVisitor(GetNodeIdent(node)))
		if err != nil {
			return nil, err
		}
	case node.GetValue() != "":
		_, err := shared.ParseExpr(env, node.GetValue(), visitor.ExprVisitor(GetNodeIdent(node)))
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("switch or value required")
	}

	return &Var{
		id:   GetNodeIdent(node),
		node: node,
	}, nil
}

func (v *Var) Compile(ctx context.Context, env shared.Env) error {
	switch {
	case v.node.GetSwitch() != nil:
		prog, err := CompileSwitch(env, v.node.GetSwitch())
		if err != nil {
			return err
		}

		v.cache = NewEvalCache(v.node, prog)
	case v.node.GetValue() != "":
		prog, err := shared.CompileExpr(env, v.node.GetValue())
		if err != nil {
			return err
		}

		v.cache = NewEvalCache(v.node, shared.ProgramFunc(func(ctx context.Context, _ shared.Env) ref.Val {
			// Use the env captured at Compile time (rather than the env
			// argument, which reflects the shared runtime env used to drive
			// execution) so that node-scoped bindings such as "self" remain
			// available.
			vars, err := env.Vars(ctx)
			if err != nil {
				return types.WrapErr(err)
			}

			val, _, err := prog.ContextEval(ctx, vars)
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

func (v *Var) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) error {
	return pipe.Run(ctx, func(ctx context.Context, value any) (any, error) {
		return v.cache.Eval(ctx, env), nil
	})
}
