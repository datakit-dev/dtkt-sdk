package shared

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	ExprOrVal struct {
		expr     *cel.Ast
		exprList exprList
		exprMap  exprMap

		value any
	}
	exprMap  map[string]*ExprOrVal
	exprList []*ExprOrVal
)

func ParseExprOrValue(env Env, val *structpb.Value, visitor ExprVisitFunc, path string) (*ExprOrVal, error) {
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
		exprList := make(exprList, len(val.ListValue.GetValues()))
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
		exprMap := make(exprMap)
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

func (e *ExprOrVal) Compile(env Env, path string, opts ...cel.ProgramOption) (Program, error) {
	if e.expr != nil {
		prog, err := env.Program(e.expr, opts...)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", path, err)
		}

		return EvalFunc(func(run Runtime) ref.Val {
			val, _, err := prog.ContextEval(run.Context(), env.Vars())
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

	return EvalFunc(func(run Runtime) ref.Val {
		env, err := run.Env()
		if err != nil {
			return types.WrapErr(err)
		}

		if e.value != nil {
			return env.TypeAdapter().NativeToValue(e.value)
		}

		return types.NullValue
	}), nil
}

func (e exprList) compile(env Env, path string, opts ...cel.ProgramOption) (Program, error) {
	progList := make([]Program, len(e))
	for i, expr := range e {
		prog, err := expr.Compile(env, path, opts...)
		if err != nil {
			return nil, err
		}
		progList[i] = prog
	}

	return EvalFunc(func(run Runtime) ref.Val {
		result := make([]any, len(e))
		for i, prog := range progList {
			val := prog.Eval(run)
			if err, ok := val.Value().(error); ok {
				return types.WrapErr(fmt.Errorf("%s: %s", path, err))
			}

			exprVal, err := cel.ValueAsProto(val)
			if err != nil {
				return types.WrapErr(fmt.Errorf("%s: %s", path, err))
			}

			nativeVal, err := ExprValueToNative(env, exprVal)
			if err != nil {
				return types.WrapErr(fmt.Errorf("%s: %s", path, err))
			}

			result[i] = nativeVal
		}

		return env.TypeAdapter().NativeToValue(result)
	}), nil
}

func (e exprMap) compile(env Env, path string, opts ...cel.ProgramOption) (Program, error) {
	progMap := make(map[string]Program)
	for key, expr := range e {
		prog, err := expr.Compile(env, path, opts...)
		if err != nil {
			return nil, err
		}
		progMap[key] = prog
	}

	return EvalFunc(func(run Runtime) ref.Val {
		result := make(map[string]any)
		for key, prog := range progMap {
			val := prog.Eval(run)
			if err, ok := val.Value().(error); ok {
				return types.WrapErr(fmt.Errorf("%s: %s", path, err))
			} else {
				exprVal, err := cel.ValueAsProto(val)
				if err != nil {
					return types.WrapErr(fmt.Errorf("%s: %s", path, err))
				}

				nativeVal, err := ExprValueToNative(env, exprVal)
				if err != nil {
					return types.WrapErr(fmt.Errorf("%s: %s", path, err))
				}

				result[key] = nativeVal
			}
		}

		return env.TypeAdapter().NativeToValue(result)
	}), nil
}
