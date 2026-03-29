package spec

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

type CacheNode interface {
	shared.SpecNode
	GetCache() bool
}

func CacheableEval(node CacheNode, eval shared.Eval) shared.Eval {
	if !node.GetCache() {
		return eval
	}

	var cached ref.Val
	return shared.EvalFunc(func(run shared.Runtime) ref.Val {
		if cached != nil {
			return cached
		}

		refVal := eval.Eval(run)
		if refVal == types.NullValue {
			return refVal
		} else if _, ok := refVal.Value().(error); ok {
			return refVal
		}

		cached = refVal

		return refVal
	})
}
