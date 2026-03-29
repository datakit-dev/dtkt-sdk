package spec

import (
	"log/slog"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var _ shared.RuntimeNode = (*Output)(nil)

type Output struct {
	node *flowv1beta1.Output
	expr *cel.Ast
	eval shared.EvalFunc
}

func NewOutput(env shared.Env, node *flowv1beta1.Output, visitor shared.ExprVisitFunc) (*Output, error) {
	expr, err := shared.ParseExpr(env, node.GetValue(), visitor)
	if err != nil {
		return nil, err
	}

	return &Output{
		node: node,
		expr: expr,
	}, nil
}

func (o *Output) Compile(run shared.Runtime) error {
	env, err := run.Env()
	if err != nil {
		return err
	}

	prog, err := env.Program(o.expr)
	if err != nil {
		return err
	}

	o.eval = func(run shared.Runtime) ref.Val {
		value, _, err := prog.ContextEval(run.Context(), env.Vars())
		if err != nil {
			return types.WrapErr(err)
		}

		log.FromCtx(run.Context()).Info(GetID(o.node), slog.Any("value", util.StringFormatAny(value.Value())))

		return value
	}

	return nil
}

func (o *Output) Eval() (shared.EvalFunc, bool) {
	return o.eval, true
}

func (o *Output) Recv() (shared.RecvFunc, bool) {
	return nil, false
}

func (o *Output) Send() (shared.SendFunc, bool) {
	return nil, false
}
