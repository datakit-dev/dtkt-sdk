package shared

import (
	"context"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	exprOrVal struct {
		isParsed   *cel.Ast
		isCompiled cel.Program
		isList     exprList
		isMap      exprMap
		isValue    any
	}
	exprMap  map[string]*exprOrVal
	exprList []*exprOrVal
)

func ParseExprOrValue(run Runtime, val *structpb.Value, visitor ParseExprFunc, path string) (*exprOrVal, error) {
	return parseExpOrValue(run, val, visitor, path)
}

func CompileExprOrValue(run Runtime, val *structpb.Value, path string) (EvalExpr, error) {
	exprOrVal, err := parseExpOrValue(run, val, nil, path)
	if err != nil {
		return nil, err
	}

	return exprOrVal.compile(run, path)
}

func (e *exprOrVal) compile(run Runtime, path string) (EvalExpr, error) {
	if e.isList != nil {
		return e.isList.compile(run, path)
	} else if e.isMap != nil {
		return e.isMap.compile(run, path)
	} else if e.isParsed != nil {
		env, err := run.Env()
		if err != nil {
			return nil, fmt.Errorf("%s: %s", path, err)
		}

		vars, err := run.Vars()
		if err != nil {
			return nil, fmt.Errorf("%s: %s", path, err)
		}

		prog, err := env.Program(e.isParsed)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", path, err)
		}

		e.isCompiled = prog

		return EvalExprFunc(func(ctx context.Context) ref.Val {
			val, _, err := prog.ContextEval(ctx, vars)
			if err != nil {
				return types.WrapErr(err)
			}
			return val
		}), nil
	}

	return EvalExprFunc(func(ctx context.Context) ref.Val {
		env, err := run.Env()
		if err != nil {
			return types.WrapErr(err)
		}

		if e.isValue != nil {
			return env.CELTypeAdapter().NativeToValue(e.isValue)
		}

		return types.NullValue
	}), nil
}

func (e exprList) compile(run Runtime, path string) (EvalExpr, error) {
	evals := make([]EvalExpr, len(e))
	for i, expr := range e {
		eval, err := expr.compile(run, path)
		if err != nil {
			return nil, err
		}
		evals[i] = eval
	}

	env, err := run.Env()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", path, err)
	}

	return EvalExprFunc(func(ctx context.Context) ref.Val {
		result := make([]any, len(e))
		for i, eval := range evals {
			val := eval.Eval(ctx)
			if err, ok := val.Value().(error); ok {
				return types.WrapErr(fmt.Errorf("%s: %s", path, err))
			}
			result[i] = val.Value()
		}

		return env.CELTypeAdapter().NativeToValue(result)
	}), nil
}

func (e exprMap) compile(run Runtime, path string) (EvalExpr, error) {
	evals := make(map[string]EvalExpr)
	for key, expr := range e {
		eval, err := expr.compile(run, path)
		if err != nil {
			return nil, err
		}
		evals[key] = eval
	}

	env, err := run.Env()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", path, err)
	}

	return EvalExprFunc(func(ctx context.Context) ref.Val {
		result := make(map[string]any)
		for key, eval := range evals {
			val := eval.Eval(ctx)
			if err, ok := val.Value().(error); ok {
				return types.WrapErr(fmt.Errorf("%s: %s", path, err))
			} else {
				result[key] = val.Value()
			}
		}

		return env.CELTypeAdapter().NativeToValue(result)
	}), nil
}

func parseExpOrValue(run Runtime, val *structpb.Value, visitor ParseExprFunc, path string) (*exprOrVal, error) {
	switch val := val.GetKind().(type) {
	case *structpb.Value_StringValue:
		_, ok := IsValidExpr(val.StringValue)
		if ok {
			parsed, err := ParseExpr(run, val.StringValue, visitor)
			if err != nil {
				return nil, fmt.Errorf("%s: %s", path, err)
			}
			return &exprOrVal{
				isParsed: parsed,
			}, nil
		}
	case *structpb.Value_ListValue:
		isList, err := parseExprList(run, val.ListValue, visitor, path)
		if err != nil {
			return nil, err
		}

		return &exprOrVal{
			isList: isList,
		}, nil
	case *structpb.Value_StructValue:
		isMap, err := parseExprMap(run, val.StructValue, visitor, path)
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

func parseExprList(run Runtime, rawList *structpb.ListValue, visitor ParseExprFunc, path string) (exprList, error) {
	exprList := make(exprList, len(rawList.GetValues()))
	for idx, val := range rawList.GetValues() {
		exprOrVal, err := parseExpOrValue(run, val, visitor, fmt.Sprintf("%s[%d]", path, idx))
		if err != nil {
			return nil, err
		}
		exprList[idx] = exprOrVal
	}
	return exprList, nil
}

func parseExprMap(run Runtime, src *structpb.Struct, visitor ParseExprFunc, path string) (exprMap, error) {
	exprMap := make(exprMap)
	for key, val := range src.GetFields() {
		exprOrVal, err := parseExpOrValue(run, val, visitor, fmt.Sprintf("%s.%s", path, key))
		if err != nil {
			return nil, err
		}
		exprMap[key] = exprOrVal
	}
	return exprMap, nil
}
