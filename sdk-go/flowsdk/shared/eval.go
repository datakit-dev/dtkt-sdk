package shared

import (
	"cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type (
	Program interface {
		Eval(Runtime) ref.Val
	}
	EvalFunc func(Runtime) ref.Val
	RecvFunc func(Runtime, <-chan any) error
	SendFunc func(Runtime, chan<- ref.Val) error
)

func ExprValueToNative(env Env, exprVal *expr.Value) (any, error) {
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
			Resolver:       env.Resolver(),
		})
	case *expr.Value_MapValue:
		mapVal := exprVal.GetMapValue()
		entries := make(map[any]any)
		for _, entry := range mapVal.Entries {
			key, err := ExprValueToNative(env, entry.Key)
			if err != nil {
				return nil, err
			}
			val, err := ExprValueToNative(env, entry.Value)
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
			val, err := ExprValueToNative(env, e)
			if err != nil {
				return nil, err
			}
			slice[i] = val
		}
		return slice, nil
	}

	refVal, err := cel.ProtoAsValue(env.TypeAdapter(), exprVal)
	if err != nil {
		return nil, err
	}

	return refVal.Value(), nil
}

func (f EvalFunc) Eval(run Runtime) ref.Val {
	return f(run)
}
