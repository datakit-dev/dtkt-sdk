package impl

import (
	"context"
	"fmt"

	"cel.dev/cel-go/cel"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

type Control struct {
	startWhenProg,
	stopWhenProg shared.Program
}

var noopVisitor = func(expr *cel.Ast) {}

func NewControl(env shared.Env, proto *flowv1beta1.Control) (*Control, error) {
	var startWhenProg shared.Program
	if proto.GetStartWhen() != "" {
		expr, err := env.ParseExpr("startWhen", proto.GetStartWhen(), noopVisitor)
		if err != nil {
			return nil, fmt.Errorf("control.startWhen parse error: %w", err)
		}

		startWhenProg, err = env.CompileExpr("startWhen", expr)
		if err != nil {
			return nil, fmt.Errorf("control.startWhen: %w", err)
		}
	}

	var stopWhenProg shared.Program
	if proto.GetStopWhen() != "" {
		expr, err := env.ParseExpr("stopWhen", proto.GetStopWhen(), noopVisitor)
		if err != nil {
			return nil, fmt.Errorf("control.stopWhen parse error: %w", err)
		}

		stopWhenProg, err = env.CompileExpr("stopWhen", expr)
		if err != nil {
			return nil, fmt.Errorf("control.startWhen: %w", err)
		}
	}

	return &Control{
		startWhenProg: startWhenProg,
		stopWhenProg:  stopWhenProg,
	}, nil
}

func (c *Control) ShouldStart(ctx context.Context, env shared.Env) (bool, error) {
	if c.startWhenProg == nil {
		return true, nil
	}

	value, err := env.Types().ValueToNative(c.startWhenProg.Eval(ctx, env))
	if err != nil {
		return false, err
	}

	shouldStart, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("control.startWhen: expected bool, got %T", value)
	}

	return shouldStart, nil
}

func (c *Control) ShouldStop(ctx context.Context, env shared.Env) (bool, error) {
	if c.stopWhenProg == nil {
		return false, nil
	}

	value, err := env.Types().ValueToNative(c.stopWhenProg.Eval(ctx, env))
	if err != nil {
		return false, err
	}

	shouldStop, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("control.stopWhen: expected bool, got %T", value)
	}

	return shouldStop, nil
}
