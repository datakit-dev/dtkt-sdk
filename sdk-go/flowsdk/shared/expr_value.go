package shared

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	exprOrVal struct {
		isExpr *cel.Ast
		isList exprList
		isMap  exprMap

		isValue any
	}
	exprMap  map[string]*exprOrVal
	exprList []*exprOrVal
)

func ParseExprOrValue(env Env, val *structpb.Value, visitor ExprVisitFunc, path string) (*exprOrVal, error) {
	return parseExpOrValue(env, val, visitor, path)
}

func CompileExprOrValue(run Runtime, val *structpb.Value, path string) (Eval, error) {
	env, err := run.Env()
	if err != nil {
		return nil, err
	}

	exprOrVal, err := parseExpOrValue(env, val, nil, path)
	if err != nil {
		return nil, err
	}

	eval, err := exprOrVal.compile(env, path)
	if err != nil {
		return nil, err
	}

	return eval, nil
}

func (e *exprOrVal) compile(env Env, path string) (Eval, error) {
	if e.isExpr != nil {
		prog, err := env.Program(e.isExpr)
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
	} else if e.isList != nil {
		return e.isList.compile(env, path)
	} else if e.isMap != nil {
		return e.isMap.compile(env, path)
	}

	return EvalFunc(func(run Runtime) ref.Val {
		env, err := run.Env()
		if err != nil {
			return types.WrapErr(err)
		}

		if e.isValue != nil {
			return env.TypeAdapter().NativeToValue(e.isValue)
		}

		return types.NullValue
	}), nil
}

func (e exprList) compile(env Env, path string) (Eval, error) {
	evals := make([]Eval, len(e))
	for i, expr := range e {
		eval, err := expr.compile(env, path)
		if err != nil {
			return nil, err
		}
		evals[i] = eval
	}

	return EvalFunc(func(run Runtime) ref.Val {
		result := make([]any, len(e))
		for i, eval := range evals {
			val := eval.Eval(run)
			if err, ok := val.Value().(error); ok {
				return types.WrapErr(fmt.Errorf("%s: %s", path, err))
			}
			result[i] = val.Value()
		}

		return env.TypeAdapter().NativeToValue(result)
	}), nil
}

func (e exprMap) compile(env Env, path string) (Eval, error) {
	evals := make(map[string]Eval)
	for key, expr := range e {
		eval, err := expr.compile(env, path)
		if err != nil {
			return nil, err
		}
		evals[key] = eval
	}

	return EvalFunc(func(run Runtime) ref.Val {
		result := make(map[string]any)
		for key, eval := range evals {
			val := eval.Eval(run)
			if err, ok := val.Value().(error); ok {
				return types.WrapErr(fmt.Errorf("%s: %s", path, err))
			} else {
				result[key] = val.Value()
			}
		}

		return env.TypeAdapter().NativeToValue(result)
	}), nil
}

func parseExpOrValue(env Env, val *structpb.Value, visitor ExprVisitFunc, path string) (*exprOrVal, error) {
	switch val := val.GetKind().(type) {
	case *structpb.Value_StringValue:
		_, ok := IsValidExpr(val.StringValue)
		if ok {
			expr, err := ParseExpr(env, val.StringValue, visitor)
			if err != nil {
				return nil, fmt.Errorf("%s: %s", path, err)
			}
			return &exprOrVal{
				isExpr: expr,
			}, nil
		}
	case *structpb.Value_ListValue:
		isList, err := parseExprList(env, val.ListValue, visitor, path)
		if err != nil {
			return nil, err
		}

		return &exprOrVal{
			isList: isList,
		}, nil
	case *structpb.Value_StructValue:
		isMap, err := parseExprMap(env, val.StructValue, visitor, path)
		if err != nil {
			return nil, err
		}

		return &exprOrVal{
			isMap: isMap,
		}, nil
	}

	return &exprOrVal{
		isValue: val.AsInterface(),
	}, nil
}

func parseExprList(env Env, rawList *structpb.ListValue, visitor ExprVisitFunc, path string) (exprList, error) {
	exprList := make(exprList, len(rawList.GetValues()))
	for idx, val := range rawList.GetValues() {
		exprOrVal, err := parseExpOrValue(env, val, visitor, fmt.Sprintf("%s[%d]", path, idx))
		if err != nil {
			return nil, err
		}
		exprList[idx] = exprOrVal
	}
	return exprList, nil
}

func parseExprMap(env Env, src *structpb.Struct, visitor ExprVisitFunc, path string) (exprMap, error) {
	exprMap := make(exprMap)
	for key, val := range src.GetFields() {
		exprOrVal, err := parseExpOrValue(env, val, visitor, fmt.Sprintf("%s.%s", path, key))
		if err != nil {
			return nil, err
		}
		exprMap[key] = exprOrVal
	}
	return exprMap, nil
}
