package spec

import (
	"context"
	"reflect"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var boolType = reflect.TypeFor[bool]()

type ActionCaller struct {
	node   *flowv1beta1.Action
	method *Method

	invokeWhenExpr,
	onErrorExpr *cel.Ast

	invokeWhenProg,
	onErrorProg cel.Program

	cache *EvalCache
}

func NewActionCaller(
	env shared.Env,
	node *flowv1beta1.Action,
	visitor shared.NodeVisitor,
) (_ *ActionCaller, err error) {
	call := node.GetCall()
	caller := &ActionCaller{
		node: node,
	}

	if call.GetInvokeWhen() != "" {
		caller.invokeWhenExpr, err = shared.ParseExpr(env, call.GetInvokeWhen(), visitor.ExprVisitor(GetNodeIdent(node)))
		if err != nil {
			return nil, err
		}
	}

	if call.GetOnError() != "" {
		caller.onErrorExpr, err = shared.ParseExpr(env, call.GetOnError(), visitor.ExprVisitor(GetNodeIdent(node)))
		if err != nil {
			return nil, err
		}
	}

	method, err := NewMethod(env, node, visitor)
	if err != nil {
		return nil, err
	}

	caller.method = method
	caller.cache = NewEvalCache(node, caller)

	return caller, nil
}

func (m *ActionCaller) Compile(ctx context.Context, env shared.Env) (err error) {
	opts := []cel.ProgramOption{cel.InterruptCheckFrequency(1)}

	if m.invokeWhenExpr != nil {
		m.invokeWhenProg, err = env.Program(m.invokeWhenExpr, opts...)
		if err != nil {
			return err
		}
	}

	if m.onErrorExpr != nil {
		m.onErrorProg, err = env.Program(m.onErrorExpr, opts...)
		if err != nil {
			return err
		}
	}

	err = m.method.Compile(ctx, env)
	if err != nil {
		return err
	}

	return nil
}

func (m *ActionCaller) ShouldInvoke(ctx context.Context, env shared.Env) (bool, error) {
	if m.invokeWhenProg != nil {
		vars, err := env.Vars(ctx)
		if err != nil {
			return false, err
		}

		val, _, err := m.invokeWhenProg.ContextEval(ctx, vars)
		if err != nil {
			return false, err
		}

		shouldInvoke, err := val.ConvertToNative(boolType)
		if err != nil {
			return false, err
		}

		return shouldInvoke.(bool), nil
	}

	return true, nil
}

func (m *ActionCaller) Eval(ctx context.Context, env shared.Env) ref.Val {
	ctx, client, err := m.method.GetClient(ctx, env)
	if err != nil {
		return types.WrapErr(err)
	}

	shouldInvoke, err := m.ShouldInvoke(ctx, env)
	if err != nil {
		return types.WrapErr(err)
	} else if !shouldInvoke {
		return types.NullValue
	}

	req, err := m.method.EvalRequest(ctx, env)
	if err != nil {
		return types.WrapErr(err)
	}

	res, err := client.CallUnary(ctx, m.method.FullName(), req)
	if err != nil {
		return types.WrapErr(err)
	}

	return env.Types().NativeToValue(res)
}

func (m *ActionCaller) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) error {
	return pipe.Run(ctx, func(ctx context.Context, value any) (any, error) {
		return m.cache.Eval(ctx, env), nil
	})
}
