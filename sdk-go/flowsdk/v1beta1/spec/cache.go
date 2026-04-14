package spec

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

type (
	EvalCache struct {
		node   CacheNode
		prog   shared.Program
		cached ref.Val
	}
	CacheNode interface {
		shared.SpecNode
		GetCache() bool
	}
)

func NewEvalCache(node CacheNode, prog shared.Program) *EvalCache {
	return &EvalCache{
		node: node,
		prog: prog,
	}
}

func (m *EvalCache) HasCached() (ref.Val, bool) {
	return m.cached, m.node.GetCache() && m.cached != nil
}

func (m *EvalCache) Eval() (shared.EvalFunc, bool) {
	if !m.node.GetCache() {
		return m.prog.Eval, true
	}

	return shared.EvalFunc(func(run shared.Runtime) ref.Val {
		if m.cached != nil {
			return m.cached
		}

		refVal := m.prog.Eval(run)
		if refVal == types.NullValue {
			return refVal
		} else if _, ok := refVal.Value().(error); ok {
			return refVal
		}

		m.cached = refVal

		return refVal
	}), true
}
