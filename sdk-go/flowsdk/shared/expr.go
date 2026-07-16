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
	ExprOrVal struct {
		expr     *cel.Ast
		exprList ExprList
		exprMap  ExprMap

		value any
	}
	ExprMap  map[string]*ExprOrVal
	ExprList []*ExprOrVal
	ExprErr  struct {
		msg string
	}
	ExprVisitor func(expr *cel.Ast)

	Program interface {
		Eval(context.Context, Env) ref.Val
	}
	ProgramFunc func(context.Context, Env) ref.Val
)

func ParseExprOrValue(env Env, val *structpb.Value, visitor ExprVisitor, path string) (*ExprOrVal, error) {
	switch val := val.GetKind().(type) {
	case *structpb.Value_StringValue:
		_, ok := IsValidExpr(val.StringValue)
		if ok {
			expr, err := ParseExpr(env, val.StringValue, visitor)
			if err != nil {
				return nil, fmt.Errorf("%s: %s", path, err)
			}

			return &ExprOrVal{
				expr: expr,
			}, nil
		}
	case *structpb.Value_ListValue:
		exprList := make(ExprList, len(val.ListValue.GetValues()))
		for idx, val := range val.ListValue.GetValues() {
			exprOrVal, err := ParseExprOrValue(env, val, visitor, fmt.Sprintf("%s[%d]", path, idx))
			if err != nil {
				return nil, err
			}
			exprList[idx] = exprOrVal
		}

		return &ExprOrVal{
			exprList: exprList,
		}, nil
	case *structpb.Value_StructValue:
		exprMap := make(ExprMap)
		for key, val := range val.StructValue.GetFields() {
			exprOrVal, err := ParseExprOrValue(env, val, visitor, fmt.Sprintf("%s.%s", path, key))
			if err != nil {
				return nil, err
			}
			exprMap[key] = exprOrVal
		}

		return &ExprOrVal{
			exprMap: exprMap,
		}, nil
	}

	return &ExprOrVal{
		value: val.AsInterface(),
	}, nil
}

func ParseExpr(env Env, expr string, visitor ExprVisitor) (*cel.Ast, error) {
	expr, valid := IsValidExpr(expr)
	if !valid {
		return nil, NewExprError(fmt.Sprintf("%s: %s", InvalidExprErrPrefix, expr))
	}

	ast, iss := env.Parse(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	ast, iss = env.Check(ast)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	if visitor != nil {
		visitor(ast)
	}

	return ast, nil
}

func CompileExpr(env Env, expr string, opts ...cel.ProgramOption) (cel.Program, error) {
	expr, valid := IsValidExpr(expr)
	if !valid {
		return nil, NewExprError(fmt.Sprintf("%s: %s", InvalidExprErrPrefix, expr))
	}

	opts = append(opts, cel.InterruptCheckFrequency(1))

	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	return env.Program(ast, opts...)
}

func NewExprError(msg string) *ExprErr {
	return &ExprErr{
		msg: msg,
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

func (e *ExprOrVal) Compile(env Env, path string, opts ...cel.ProgramOption) (Program, error) {
	if e.expr != nil {
		prog, err := env.Program(e.expr, opts...)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", path, err)
		}

		return ProgramFunc(func(ctx context.Context, env Env) ref.Val {
			vars, err := env.Vars(ctx)
			if err != nil {
				return types.WrapErr(err)
			}

			val, _, err := prog.ContextEval(ctx, vars)
			if err != nil {
				return types.WrapErr(err)
			}
			return val
		}), nil
	} else if e.exprList != nil {
		return e.exprList.compile(env, path, opts...)
	} else if e.exprMap != nil {
		return e.exprMap.compile(env, path, opts...)
	}

	return ProgramFunc(func(ctx context.Context, env Env) ref.Val {
		if e.value != nil {
			return env.Types().NativeToValue(e.value)
		}

		return types.NullValue
	}), nil
}

func (e ExprList) compile(env Env, path string, opts ...cel.ProgramOption) (Program, error) {
	progList := make([]Program, len(e))
	for i, expr := range e {
		prog, err := expr.Compile(env, path, opts...)
		if err != nil {
			return nil, err
		}
		progList[i] = prog
	}

	return ProgramFunc(func(ctx context.Context, env Env) ref.Val {
		result := make([]any, len(e))
		for i, prog := range progList {
			val := prog.Eval(ctx, env)
			if err, ok := val.Value().(error); ok {
				return types.WrapErr(fmt.Errorf("%s: %s", path, err))
			}

			nativeVal, err := env.Types().ValueToNative(val)
			if err != nil {
				return types.WrapErr(fmt.Errorf("%s: %s", path, err))
			}

			result[i] = nativeVal
		}

		return env.Types().NativeToValue(result)
	}), nil
}

func (e ExprMap) compile(env Env, path string, opts ...cel.ProgramOption) (Program, error) {
	progMap := make(map[string]Program)
	for key, expr := range e {
		prog, err := expr.Compile(env, path, opts...)
		if err != nil {
			return nil, err
		}
		progMap[key] = prog
	}

	return ProgramFunc(func(ctx context.Context, env Env) ref.Val {
		result := make(map[string]any)
		for key, prog := range progMap {
			val := prog.Eval(ctx, env)
			if err, ok := val.Value().(error); ok {
				return types.WrapErr(fmt.Errorf("%s: %s", path, err))
			} else {
				nativeVal, err := env.Types().ValueToNative(val)
				if err != nil {
					return types.WrapErr(fmt.Errorf("%s: %s", path, err))
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

func (e *ExprErr) Error() string {
	return e.msg
}

func (f NodeVisitor) ExprVisitor(id string) ExprVisitor {
	return func(expr *cel.Ast) {
		f(id, expr)
	}
}

func (f ProgramFunc) Eval(ctx context.Context, env Env) ref.Val {
	return f(ctx, env)
}
