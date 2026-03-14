package spec

import (
	"context"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

type CacheNode interface {
	shared.SpecNode
	GetCache() bool
}

func CacheableEval(node CacheNode, eval shared.EvalExpr) shared.EvalExpr {
	if !node.GetCache() {
		return eval
	}

	var cached ref.Val
	return shared.EvalExprFunc(func(ctx context.Context) ref.Val {
		if cached != nil {
			return cached
		}

		refVal := eval.Eval(ctx)
		if refVal == types.NullValue {
			return refVal
		} else if _, ok := refVal.Value().(error); ok {
			return refVal
		}

		cached = refVal

		return refVal
	})
}
