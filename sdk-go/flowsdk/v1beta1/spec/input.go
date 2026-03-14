package spec

import (
	"context"
	"errors"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var InputValueMissingErr missingInputValueError

type missingInputValueError struct {
	id string
}

func NewInputValueError(id string) missingInputValueError {
	return missingInputValueError{id: id}
}

func (e missingInputValueError) Error() string {
	return fmt.Sprintf("inputs.%s: value not found", e.id)
}

func (e missingInputValueError) Is(target error) bool {
	_, ok := target.(missingInputValueError)
	return ok
}

func CompileInput(run shared.Runtime, input *flowv1beta1.Input) (shared.EvalExpr, error) {
	inputType, err := NewInputTypeWithResolver(input, run.Resolver())
	if err != nil {
		return nil, err
	}

	env, err := run.Env()
	if err != nil {
		return nil, err
	}

	return CacheableEval(input, shared.EvalExprFunc(func(ctx context.Context) ref.Val {
		value, err := run.GetInputValue(inputType.GetId())
		if err != nil {
			if !errors.Is(err, InputValueMissingErr) {
				return types.WrapErr(err)
			}
		}

		value, err = inputType.Validate(value)
		if err != nil {
			return types.WrapErr(err)
		}

		return env.CELTypeAdapter().NativeToValue(value)
	})), nil
}
