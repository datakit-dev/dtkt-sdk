package spec

import (
	"context"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types/ref"
)

var _ shared.RuntimeNode = (*Input)(nil)

type Input struct {
	node *flowv1beta1.Input
	typ  InputType

	defaultVal any
	cachedVal  ref.Val

	valueCh chan ref.Val
}

func NewInput(env shared.Env, input *flowv1beta1.Input) (*Input, error) {
	inputType, err := NewInputTypeWithResolver(input, env.Resolver())
	if err != nil {
		return nil, err
	}

	var defaultVal any
	if inputType.HasDefault() {
		defaultVal, err = inputType.GetDefault()
		if err != nil {
			return nil, err
		}
	}

	return &Input{
		node: input,

		typ:        inputType,
		defaultVal: defaultVal,

		valueCh: make(chan ref.Val, 1),
	}, nil
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
				if i.node.GetCache() && i.cachedVal != nil && i.cachedVal.Value() != nil {
					refVal = i.cachedVal
				} else if value == nil && i.node.GetCache() && i.typ.HasDefault() {
					refVal = env.TypeAdapter().NativeToValue(i.defaultVal)
				} else if value == nil && i.typ.IsRequired() {
					continue loop
				} else {
					value, err := i.typ.Validate(value)
					if err != nil {
						return err
					}

					refVal = env.TypeAdapter().NativeToValue(value)
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

func (i *Input) Eval() (shared.EvalFunc, bool) {
	return nil, false
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

func (i *Input) Compile(shared.Runtime) error {
	return nil
}

// IsRequired returns true when the input has no default and is not nullable,
// meaning it must receive an external value before it can emit.
func (i *Input) IsRequired() bool {
	return i.typ.IsRequired()
}
