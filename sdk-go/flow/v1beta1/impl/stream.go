package impl

import (
	"context"
	"errors"
	"fmt"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

type (
	Stream struct {
		shared.ParsedNode
		ident string
		node  *flowv1beta1.Stream

		startWhenExpr,
		stopWhenExpr *cel.Ast

		startWhenProg,
		stopWhenProg shared.Program
	}
)

func NewStream(env shared.Env, node *flowv1beta1.Stream, visitor shared.NodeVisitor) (_ *Stream, err error) {
	stream := &Stream{
		ident: shared.GetNodeIdent(node),
		node:  node,
	}

	switch {
	case node.GetCall() != nil:
		caller, err := NewStreamCaller(env, node, stream, visitor)
		if err != nil {
			return nil, err
		}

		stream.ParsedNode = caller
	case node.GetGenerate() != nil:
		generator, err := NewGenerator(env, node, stream, visitor)
		if err != nil {
			return nil, err
		}

		stream.ParsedNode = generator
	case node.GetWindow() != nil:
		window, err := NewWindow(env, node, stream, visitor)
		if err != nil {
			return nil, err
		}

		stream.ParsedNode = window
	default:
		return nil, errors.New("call, generate, or window required")
	}

	if node.GetStartWhen() != "" {
		stream.startWhenExpr, err = env.ParseExpr("startWhen", node.GetStartWhen(), visitor.ExprVisitor(stream.ident))
		if err != nil {
			return nil, fmt.Errorf("%s.startWhen parse error: %w", stream.ident, err)
		}
	}

	if node.GetStopWhen() != "" {
		stream.stopWhenExpr, err = env.ParseExpr("stopWhen", node.GetStopWhen(), visitor.ExprVisitor(stream.ident))
		if err != nil {
			return nil, fmt.Errorf("%s.stopWhen parse error: %w", stream.ident, err)
		}
	}

	return stream, nil
}

func (a *Stream) Caller() (*StreamCaller, bool) {
	if a.ParsedNode == nil {
		return nil, false
	} else if caller, ok := a.ParsedNode.(*StreamCaller); !ok {
		return nil, false
	} else {
		return caller, true
	}
}
func (s *Stream) Compile(ctx context.Context, env shared.Env) (err error) {
	if s.ParsedNode == nil {
		return fmt.Errorf("%s: exec node is nil", s.ident)
	}

	if s.startWhenExpr != nil {
		s.startWhenProg, err = env.CompileExpr("startWhen", s.startWhenExpr)
		if err != nil {
			return fmt.Errorf("%s.startWhen compile error: %w", s.ident, err)
		}
	}

	if s.stopWhenExpr != nil {
		s.stopWhenProg, err = env.CompileExpr("stopWhen", s.stopWhenExpr)
		if err != nil {
			return fmt.Errorf("%s.stopWhen compile error: %w", s.ident, err)
		}
	}

	return s.ParsedNode.Compile(ctx, env)
}

func (s *Stream) ShouldStart(ctx context.Context, env shared.Env) (bool, error) {
	if s.startWhenProg == nil {
		return true, nil
	}

	return s.startWhenProg.Eval(ctx, env) == types.True, nil
	// vars, err := env.Vars(ctx)
	// if err != nil {
	// 	return false, err
	// }

	// val, _, err := s.startWhenProg.ContextEval(ctx, vars)
	// if err != nil {
	// 	return true, err
	// }

	// shouldStart, err := val.ConvertToNative(reflect.TypeFor[bool]())
	// if err != nil {
	// 	return true, err
	// }

	// return shouldStart.(bool), nil
}

func (s *Stream) ShouldStop(ctx context.Context, env shared.Env) (bool, error) {
	if s.stopWhenProg == nil {
		return false, nil
	}

	return s.stopWhenProg.Eval(ctx, env) == types.True, nil
	// vars, err := env.Vars(ctx)
	// if err != nil {
	// 	return false, err
	// }

	// val, _, err := s.stopWhenProg.ContextEval(ctx, vars)
	// if err != nil {
	// 	return true, err
	// }

	// shouldStop, err := val.ConvertToNative(reflect.TypeFor[bool]())
	// if err != nil {
	// 	return true, err
	// }

	// return shouldStop.(bool), nil
}
