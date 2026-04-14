package spec

import (
	"context"
	"fmt"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types/ref"
)

var _ shared.ExecNode = (*Input)(nil)

type Input struct {
	node *flowv1beta1.Input
	typ  InputType

	defaultVal any

	cachedMu  sync.Mutex
	cachedVal ref.Val

	valueCh chan ref.Val
}

func NewInput(env shared.Env, input *flowv1beta1.Input) *Input {
	return &Input{
		node:    input,
		valueCh: make(chan ref.Val, 1),
	}
}

func (i *Input) Compile(run shared.Runtime) error {
	env, err := run.Env()
	if err != nil {
		return err
	}

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

func (i *Input) Recv() (shared.RecvFunc, bool) {
	return func(run shared.Runtime, recvCh <-chan any) error {
		env, err := run.Env()
		if err != nil {
			return err
		}

	loop:
		for {
			select {
			case <-run.Context().Done():
				return context.Cause(run.Context())
			case value, ok := <-recvCh:
				if !ok {
					return fmt.Errorf("receive channel closed")
				}

				var refVal ref.Val
				if value == nil {
					i.cachedMu.Lock()
					cv := i.cachedVal
					i.cachedMu.Unlock()

					if i.node.GetCache() && cv != nil {
						refVal = cv
					} else if i.typ.HasDefault() {
						refVal = env.TypeAdapter().NativeToValue(i.defaultVal)
					} else if i.typ.IsRequired() {
						continue loop
					}
				} else {
					value, err := i.typ.Validate(value)
					if err != nil {
						return err
					}

					refVal = env.TypeAdapter().NativeToValue(value)

					if i.node.GetCache() {
						i.cachedMu.Lock()
						if i.cachedVal == nil {
							i.cachedVal = refVal
						}
						i.cachedMu.Unlock()
					}
				}

				select {
				case <-run.Context().Done():
					return context.Cause(run.Context())
				case i.valueCh <- refVal:
				}
			}
		}
	}, true
}

// Send reads the validated value produced by Recv and emits it as an event.
// This makes Input a proper emitter so the executor's readiness check covers it.
func (i *Input) Send() (shared.SendFunc, bool) {
	return func(run shared.Runtime, sendCh chan<- ref.Val) error {
		for {
			select {
			case <-run.Context().Done():
				return context.Cause(run.Context())
			case value := <-i.valueCh:
				select {
				case <-run.Context().Done():
					return context.Cause(run.Context())
				case sendCh <- value:
				}
			}
		}
	}, true
}

// IsRequired returns true when the input has no default and is not nullable,
// meaning it must receive an external value before it can emit.
func (i *Input) IsRequired() bool {
	return i.typ.IsRequired()
}

// HasCached returns the cached value if caching is enabled and a value has
// already been captured from an external event. The executor uses this to
// re-emit the same value each cycle without waiting on a new sendCh event.
func (i *Input) HasCached() (ref.Val, bool) {
	i.cachedMu.Lock()
	cv := i.cachedVal
	i.cachedMu.Unlock()
	if i.node.GetCache() && cv != nil {
		return cv, true
	}
	return nil, false
}

func (i *Input) Eval() (shared.EvalFunc, bool) { return nil, false }
