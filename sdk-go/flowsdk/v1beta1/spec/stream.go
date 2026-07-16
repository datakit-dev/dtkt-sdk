package spec

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
)

type (
	Stream struct {
		shared.ExecNode
		node *flowv1beta1.Stream

		startWhenExpr,
		stopWhenExpr *cel.Ast

		startWhenProg,
		stopWhenProg cel.Program
	}
)

func NewStream(env shared.Env, node *flowv1beta1.Stream, visitor shared.NodeVisitor) (_ *Stream, err error) {
	stream := &Stream{
		node: node,
	}

	switch {
	case node.GetCall() != nil:
		stream, err := NewStreamCaller(env, node, stream, visitor)
		if err != nil {
			return nil, err
		}

		stream.ExecNode = stream
	case node.GetGenerate() != nil:
		generator, err := NewGenerator(env, node, stream, visitor)
		if err != nil {
			return nil, err
		}

		stream.ExecNode = generator
	default:
		return nil, errors.New("call or generate required")
	}

	if node.GetStartWhen() != "" {
		stream.startWhenExpr, err = shared.ParseExpr(env, node.GetStartWhen(), visitor.ExprVisitor(GetNodeIdent(node)))
		if err != nil {
			return nil, fmt.Errorf("%s.startWhen parse error: %w", GetNodeIdent(node), err)
		}
	}

	if node.GetStopWhen() != "" {
		stream.stopWhenExpr, err = shared.ParseExpr(env, node.GetStopWhen(), visitor.ExprVisitor(GetNodeIdent(node)))
		if err != nil {
			return nil, fmt.Errorf("%s.stopWhen parse error: %w", GetNodeIdent(node), err)
		}
	}

	return stream, nil
}

func (s *Stream) Compile(ctx context.Context, env shared.Env) (err error) {
	if s.startWhenExpr != nil {
		s.startWhenProg, err = env.Program(s.startWhenExpr)
		if err != nil {
			return fmt.Errorf("%s.startWhen compile error: %w", GetNodeIdent(s.node), err)
		}
	}

	if s.stopWhenExpr != nil {
		s.stopWhenProg, err = env.Program(s.stopWhenExpr)
		if err != nil {
			return fmt.Errorf("%s.stopWhen compile error: %w", GetNodeIdent(s.node), err)
		}
	}

	return s.ExecNode.Compile(ctx, env)
}

func (s *Stream) ShouldStart(ctx context.Context, env shared.Env) (bool, error) {
	if s.startWhenProg == nil {
		return true, nil
	}

	vars, err := env.Vars(ctx)
	if err != nil {
		return false, err
	}

	val, _, err := s.startWhenProg.ContextEval(ctx, vars)
	if err != nil {
		return true, err
	}

	shouldStart, err := val.ConvertToNative(reflect.TypeFor[bool]())
	if err != nil {
		return true, err
	}

	return shouldStart.(bool), nil
}

func (s *Stream) ShouldStop(ctx context.Context, env shared.Env) (bool, error) {
	if s.stopWhenProg == nil {
		return false, nil
	}

	vars, err := env.Vars(ctx)
	if err != nil {
		return false, err
	}

	val, _, err := s.stopWhenProg.ContextEval(ctx, vars)
	if err != nil {
		return true, err
	}

	shouldStop, err := val.ConvertToNative(reflect.TypeFor[bool]())
	if err != nil {
		return true, err
	}

	return shouldStop.(bool), nil
}
