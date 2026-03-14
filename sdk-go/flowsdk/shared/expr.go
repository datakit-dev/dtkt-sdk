package shared

import (
	"context"
	"regexp"
	"strings"

	"cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	InvalidExprErrPrefix = "invalid expression"
)

var (
	invalidExprErr = NewExprError(InvalidExprErrPrefix)
	validExpr      = regexp.MustCompile(`^\s?=\s?`)
)

var _ error = (*ExprError)(nil)

type (
	EvalExpr interface {
		Eval(context.Context) ref.Val
	}
	EvalExprFunc func(context.Context) ref.Val
	ExprError    struct {
		msg string
	}
)

func ExprValueToNative(run Runtime, exprVal *expr.Value) (any, error) {
	if exprVal == nil {
		return nil, nil
	} else if _, val := exprVal.GetKind().(*expr.Value_NullValue); val {
		return nil, nil
	}

	switch exprVal.GetKind().(type) {
	case *expr.Value_ObjectValue:
		objVal := exprVal.GetObjectValue()
		return anypb.UnmarshalNew(objVal, proto.UnmarshalOptions{
			DiscardUnknown: true,
			Resolver:       run.Resolver(),
		})
	case *expr.Value_MapValue:
		mapVal := exprVal.GetMapValue()
		entries := make(map[any]any)
		for _, entry := range mapVal.Entries {
			key, err := ExprValueToNative(run, entry.Key)
			if err != nil {
				return nil, err
			}
			val, err := ExprValueToNative(run, entry.Value)
			if err != nil {
				return nil, err
			}
			entries[key] = val
		}
		return entries, nil
	case *expr.Value_ListValue:
		listVal := exprVal.GetListValue()
		slice := make([]any, len(listVal.Values))
		for i, e := range listVal.Values {
			rv, err := ExprValueToNative(run, e)
			if err != nil {
				return nil, err
			}
			slice[i] = rv
		}
		return slice, nil
	}

	types, err := run.Types()
	if err != nil {
		return nil, err
	}

	refVal, err := cel.ProtoAsValue(types, exprVal)
	if err != nil {
		return nil, err
	}

	return refVal.Value(), nil
}

func NewExprError(err string) *ExprError {
	return &ExprError{
		msg: err,
	}
}

func (e *ExprError) Is(err error) bool {
	return strings.HasPrefix(err.Error(), e.msg)
}

func (e *ExprError) Error() string {
	return e.msg
}

func (f EvalExprFunc) Eval(ctx context.Context) ref.Val {
	return f(ctx)
}
