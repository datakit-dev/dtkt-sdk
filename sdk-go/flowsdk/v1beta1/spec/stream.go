package spec

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
)

var _ ExecNodeCloser = (*Stream)(nil)

type (
	Stream struct {
		ExecNodeCloser
		node *flowv1beta1.Stream

		startWhenExpr,
		stopWhenExpr *cel.Ast

		startWhenProg,
		stopWhenProg cel.Program
	}
)

func NewStream(env shared.Env, node *flowv1beta1.Stream, visitor shared.NodeVisitFunc) (_ *Stream, err error) {
	stream := &Stream{
		node: node,
	}

	switch {
	case node.GetCall() != nil:
		stream.ExecNodeCloser, err = NewStreamCaller(env, node, stream, visitor)
		if err != nil {
			return nil, err
		}
	case node.GetGenerate() != nil:
		stream.ExecNodeCloser, err = NewGenerator(env, node, stream, visitor)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("call or generate required")
	}

	if node.GetStartWhen() != "" {
		stream.startWhenExpr, err = shared.ParseExpr(env, node.GetStartWhen(), visitor.ExprVisitor(GetID(node)))
		if err != nil {
			return nil, fmt.Errorf("%s.startWhen parse error: %w", GetID(node), err)
		}
	}

	if node.GetStopWhen() != "" {
		stream.stopWhenExpr, err = shared.ParseExpr(env, node.GetStopWhen(), visitor.ExprVisitor(GetID(node)))
		if err != nil {
			return nil, fmt.Errorf("%s.stopWhen parse error: %w", GetID(node), err)
		}
	}

	return stream, nil
}

func (s *Stream) Compile(run shared.Runtime) error {
	env, err := run.Env()
	if err != nil {
		return err
	}

	if s.startWhenExpr != nil {
		s.startWhenProg, err = env.Program(s.startWhenExpr)
		if err != nil {
			return fmt.Errorf("%s.startWhen compile error: %w", GetID(s.node), err)
		}
	}

	if s.stopWhenExpr != nil {
		s.stopWhenProg, err = env.Program(s.stopWhenExpr)
		if err != nil {
			return fmt.Errorf("%s.stopWhen compile error: %w", GetID(s.node), err)
		}
	}

	return s.ExecNodeCloser.Compile(run)
}

func (s *Stream) ShouldStart(run shared.Runtime) (bool, error) {
	if s.startWhenProg == nil {
		return true, nil
	}

	env, err := run.Env()
	if err != nil {
		return true, err
	}

	val, _, err := s.startWhenProg.ContextEval(run.Context(), env.Vars())
	if err != nil {
		return true, err
	}

	shouldStart, err := val.ConvertToNative(reflect.TypeFor[bool]())
	if err != nil {
		return true, err
	}

	return shouldStart.(bool), nil
}

func (s *Stream) ShouldStop(run shared.Runtime) (bool, error) {
	if s.stopWhenProg == nil {
		return false, nil
	}

	env, err := run.Env()
	if err != nil {
		return true, err
	}

	val, _, err := s.stopWhenProg.ContextEval(run.Context(), env.Vars())
	if err != nil {
		return true, err
	}

	shouldStop, err := val.ConvertToNative(reflect.TypeFor[bool]())
	if err != nil {
		return true, err
	}

	return shouldStop.(bool), nil
}
