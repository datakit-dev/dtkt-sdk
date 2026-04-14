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
	visitor shared.NodeVisitFunc,
) (_ *ActionCaller, err error) {
	call := node.GetCall()
	caller := &ActionCaller{
		node: node,
	}

	if call.GetInvokeWhen() != "" {
		caller.invokeWhenExpr, err = shared.ParseExpr(env, call.GetInvokeWhen(), visitor.ExprVisitor(GetID(node)))
		if err != nil {
			return nil, err
		}
	}

	if call.GetOnError() != "" {
		caller.onErrorExpr, err = shared.ParseExpr(env, call.GetOnError(), visitor.ExprVisitor(GetID(node)))
		if err != nil {
			return nil, err
		}
	}

	caller.method, err = NewMethod(env, node, visitor)
	if err != nil {
		return nil, err
	}

	return caller, nil
}

func (m *ActionCaller) Compile(run shared.Runtime) error {
	env, err := run.Env()
	if err != nil {
		return err
	}

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

	err = m.method.Compile(run)
	if err != nil {
		return err
	}

	conn, err := m.method.GetConnector(run)
	if err != nil {
		return err
	}

	method, err := m.method.GetDescriptor(run)
	if err != nil {
		return err
	}

	ctx, client, err := conn.GetClient(run.Context())
	if err != nil {
		return err
	}

	m.cache = NewEvalCache(m.node, shared.EvalFunc(func(run shared.Runtime) ref.Val {
		env, err := run.Env()
		if err != nil {
			return types.WrapErr(err)
		}

		shouldInvoke, err := m.ShouldInvoke(run.Context(), env)
		if err != nil {
			return types.WrapErr(err)
		} else if !shouldInvoke {
			return types.NullValue
		}

		req, err := m.method.EvalRequest(run)
		if err != nil {
			return types.WrapErr(err)
		}

		res, err := client.CallUnary(ctx, method.FullName(), req)
		if err != nil {
			return types.WrapErr(err)
		}

		return env.TypeAdapter().NativeToValue(res)
	}))

	return nil
}

func (m *ActionCaller) ShouldInvoke(ctx context.Context, env shared.Env) (bool, error) {
	if m.invokeWhenProg != nil {
		val, _, err := m.invokeWhenProg.ContextEval(ctx, env.Vars())
		if err != nil {
			return false, err
		}

		shouldInvoke, err := val.ConvertToNative(reflect.TypeFor[bool]())
		if err != nil {
			return false, err
		}

		return shouldInvoke.(bool), nil
	}

	return true, nil
}

func (m *ActionCaller) Eval() (shared.EvalFunc, bool) {
	return m.cache.Eval()
}

func (m *ActionCaller) HasCached() (ref.Val, bool) {
	return m.cache.HasCached()
}

func (m *ActionCaller) Close() error                  { return nil }
func (m *ActionCaller) Recv() (shared.RecvFunc, bool) { return nil, false }
func (m *ActionCaller) Send() (shared.SendFunc, bool) { return nil, false }
