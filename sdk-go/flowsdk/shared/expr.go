package shared

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/types/known/structpb"
)

const InvalidExprErrPrefix = "invalid expression"

var validExpr = regexp.MustCompile(`^\s?=\s?`)

var _ error = (*ExprErr)(nil)

type (
	ExprTree struct {
		id       string
		expr     *cel.Ast
		exprList ExprList
		exprMap  ExprMap

		value any
	}
	ExprList []*ExprTree
	ExprMap  map[string]*ExprTree
	ExprErr  struct {
		error
	}
	ExprVisitor func(expr *cel.Ast)
	programID   struct {
		Program
		id string
	}
)

func NewExprError(format string, a ...any) *ExprErr {
	return &ExprErr{
		error: fmt.Errorf(format, a...),
	}
}

func IsInvalidExprError(err error) bool {
	exprErr := new(ExprErr)
	return errors.As(err, &exprErr)
}

func IsValidExpr(expr string) (string, bool) {
	matches := validExpr.FindStringSubmatch(expr)

	if len(matches) != 1 {
		return "", false
	}

	celExpr := strings.TrimSpace(expr[len(matches[0]):])

	if len(celExpr) == 0 {
		return "", false
	}

	return celExpr, true
}

func ParseExprTree(env Env, id string, val *structpb.Value, visitor ExprVisitor) (*ExprTree, error) {
	switch val := val.GetKind().(type) {
	case *structpb.Value_StringValue:
		_, ok := IsValidExpr(val.StringValue)
		if ok {
			expr, err := env.ParseExpr(id, val.StringValue, visitor)
			if err != nil {
				return nil, fmt.Errorf("%s: %s", id, err)
			}

			return &ExprTree{
				id:   id,
				expr: expr,
			}, nil
		}
	case *structpb.Value_ListValue:
		exprList := make(ExprList, len(val.ListValue.GetValues()))
		for idx, val := range val.ListValue.GetValues() {
			exprOrVal, err := ParseExprTree(env, fmt.Sprintf("%s.%d", id, idx), val, visitor)
			if err != nil {
				return nil, err
			}
			exprList[idx] = exprOrVal
		}

		return &ExprTree{
			id:       id,
			exprList: exprList,
		}, nil
	case *structpb.Value_StructValue:
		exprMap := make(ExprMap)
		for key, val := range val.StructValue.GetFields() {
			exprOrVal, err := ParseExprTree(env, fmt.Sprintf("%s.%s", id, key), val, visitor)
			if err != nil {
				return nil, err
			}
			exprMap[key] = exprOrVal
		}

		return &ExprTree{
			id:      id,
			exprMap: exprMap,
		}, nil
	}

	return &ExprTree{
		id:    id,
		value: val.AsInterface(),
	}, nil
}

func (e *ExprTree) Compile(env Env) (Program, error) {
	if e.expr != nil {
		return env.CompileExpr(e.id, e.expr)
	} else if e.exprList != nil {
		return e.exprList.compile(env)
	} else if e.exprMap != nil {
		return e.exprMap.compile(env)
	}

	return ProgramFunc(func(ctx context.Context, env Env) ref.Val {
		if e.value == nil {
			return types.NullValue
		}
		return env.Types().NativeToValue(e.value)
	}), nil
}

func (e ExprList) compile(env Env) (Program, error) {
	progList := make([]programID, len(e))
	for i, expr := range e {
		prog, err := expr.Compile(env)
		if err != nil {
			return nil, err
		}
		progList[i] = programID{
			Program: prog,
			id:      expr.id,
		}
	}

	return ProgramFunc(func(ctx context.Context, env Env) ref.Val {
		result := make([]any, len(e))
		for i, prog := range progList {
			val := prog.Eval(ctx, env)
			if err, ok := val.Value().(error); ok {
				return types.WrapErr(fmt.Errorf("%s: %s", prog.id, err))
			}

			nativeVal, err := env.Types().ValueToNative(val)
			if err != nil {
				return types.WrapErr(fmt.Errorf("%s: %s", prog.id, err))
			}

			result[i] = nativeVal
		}

		return env.Types().NativeToValue(result)
	}), nil
}

func (e ExprMap) compile(env Env) (Program, error) {
	progMap := make(map[string]programID)
	for key, expr := range e {
		prog, err := expr.Compile(env)
		if err != nil {
			return nil, err
		}
		progMap[key] = programID{
			Program: prog,
			id:      expr.id,
		}
	}

	return ProgramFunc(func(ctx context.Context, env Env) ref.Val {
		result := make(map[string]any)
		for key, prog := range progMap {
			val := prog.Eval(ctx, env)
			if err, ok := val.Value().(error); ok {
				return types.WrapErr(fmt.Errorf("%s: %s", prog.id, err))
			} else {
				nativeVal, err := env.Types().ValueToNative(val)
				if err != nil {
					return types.WrapErr(fmt.Errorf("%s: %s", prog.id, err))
				}

				result[key] = nativeVal
			}
		}

		return env.Types().NativeToValue(result)
	}), nil
}

func (e *ExprErr) Is(err error) bool {
	return IsInvalidExprError(err)
}
