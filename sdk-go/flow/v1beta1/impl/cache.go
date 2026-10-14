package impl

import (
	"context"

	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/shared"
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
