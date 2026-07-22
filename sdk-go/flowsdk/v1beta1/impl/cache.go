package impl

import (
	"context"

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

func (m *EvalCache) Eval(ctx context.Context, env shared.Env) ref.Val {
	if !m.node.GetCache() {
		return m.prog.Eval(ctx, env)
	} else if m.cached != nil {
		return m.cached
	}

	refVal := m.prog.Eval(ctx, env)
	switch refVal.(type) {
	case types.Null, error:
		return refVal
	}

	m.cached = refVal

	return refVal
}
