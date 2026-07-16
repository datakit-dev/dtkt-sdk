package spec

import (
	"context"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types"
)

var _ shared.ExecNode = (*Input)(nil)

type Input struct {
	id   string
	node *flowv1beta1.Input
	typ  InputType

	defaultVal,
	cachedVal any

	mut sync.Mutex
}

func NewInput(env shared.Env, input *flowv1beta1.Input) *Input {
	return &Input{
		id:   GetNodeIdent(input),
		node: input,
	}
}

func (i *Input) Compile(ctx context.Context, env shared.Env) (err error) {
	i.typ, err = NewInputTypeWithResolver(i.node, env.Resolver())
	if err != nil {
		return err
	}

	if i.typ.HasDefault() {
		i.defaultVal, err = i.typ.GetDefault()
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Input) Run(ctx context.Context, env shared.Env, pipe shared.Pipe) (err error) {
	return pipe.Run(ctx, func(ctx context.Context, value any) (any, error) {
		cached, hasCached := i.HasCached()
		if hasCached {
			value = cached
		} else {
			value, err = i.typ.Validate(value)
			if err != nil {
				return nil, err
			} else if i.typ.GetCache() {
				i.setCached(value)
			}
		}
		return value, nil
	})
}

func (i *Input) Type() InputType {
	return i.typ
}

func (i *Input) HasDefault() (any, bool) {
	return i.defaultVal, i.typ.HasDefault()
}

func (i *Input) IsCacheable() bool {
	return i.typ.GetCache()
}

func (i *Input) IsRequired() bool {
	_, hasCached := i.HasCached()
	if hasCached {
		return false
	}
	return i.typ.IsRequired()
}

func (i *Input) HasCached() (any, bool) {
	if !i.typ.GetCache() {
		return nil, false
	}

	i.mut.Lock()
	defer i.mut.Unlock()

	return i.cachedVal, i.cachedVal != nil
}

func (i *Input) setCached(value any) {
	if value != nil && value != types.NullValue {
		i.mut.Lock()
		if i.cachedVal == nil {
			i.cachedVal = value
		}
		i.mut.Unlock()
	}
}
