package spec

import (
	"context"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protoformsdk/form"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/proto"
)

type (
	UserActionInputEvalFunc func(context.Context) error
	UserActionInputsEvalMap map[string]UserActionInputEvalFunc
	UserActionBinding       interface {
		proto.Message
		GetValue() any
	}
	userActionBinding[V any, T userActionBindingMessage[V]] struct {
		proto.Message
	}
	userActionBindingMessage[V any] interface {
		proto.Message
		GetValue() V
	}
)

func NewUserActionBinding[V any, T userActionBindingMessage[V]](msg T) *userActionBinding[V, T] {
	return &userActionBinding[V, T]{
		Message: msg,
	}
}

func (b *userActionBinding[V, T]) GetValue() any {
	return b.Message.(T).GetValue()
}

func (m UserActionInputsEvalMap) Eval(ctx context.Context) (err error) {
	for id, eval := range m {
		err = eval(ctx)
		if err != nil {
			return fmt.Errorf("%s error: %w", id, err)
		}
	}
	return
}

func UserActionInputID(action *flowv1beta1.Action, input *flowv1beta1.UserAction_Input) string {
	return fmt.Sprintf("%s.inputs.%s", GetID(action), input.GetId())
}

func GetUserActionBinding(input *flowv1beta1.UserAction_Input) (*protoformv1beta1.FieldElement, UserActionBinding, bool) {
	var binding UserActionBinding
	switch input.GetElement().(type) {
	case *flowv1beta1.UserAction_Input_Confirm:
		binding = NewUserActionBinding(&flowv1beta1.UserAction_ConfirmBinding{})
	case *flowv1beta1.UserAction_Input_Input:
		binding = NewUserActionBinding(&flowv1beta1.UserAction_InputBinding{})
	}

	if binding == nil {
		return nil, nil, false
	}

	desc := binding.ProtoReflect().Descriptor().Fields().ByName("value")
	elem, ok := form.GetFieldElement(desc)
	if ok {
		elem.Title = new(input.Title)
		elem.Description = input.Description

		switch elemType := input.GetElement().(type) {
		case *flowv1beta1.UserAction_Input_Confirm:
			if elem.GetConfirm() != nil {
				proto.Merge(elem.GetConfirm(), elemType.Confirm)
			}
			elem.Type = &protoformv1beta1.FieldElement_Confirm{
				Confirm: elem.GetConfirm(),
			}
		case *flowv1beta1.UserAction_Input_Input:
			elem.Type = &protoformv1beta1.FieldElement_Input{
				Input: elemType.Input,
			}
		}
	}

	return elem, binding, ok
}

func ParseUserAction(ctx shared.Runtime, action *flowv1beta1.UserAction, visitor shared.ParseExprFunc) error {
	for _, input := range action.GetInputs() {
		switch elemType := input.GetElement().(type) {
		case *flowv1beta1.UserAction_Input_Confirm:
			if elemType.Confirm != nil {
				_, err := shared.ParseExpr(ctx, elemType.Confirm.GetApprove(), visitor)
				if err != nil && !shared.IsInvalidExprError(err) {
					return err
				}

				_, err = shared.ParseExpr(ctx, elemType.Confirm.GetDecline(), visitor)
				if err != nil && !shared.IsInvalidExprError(err) {
					return err
				}
			}
		case *flowv1beta1.UserAction_Input_Input:
		case *flowv1beta1.UserAction_Input_File:
		case *flowv1beta1.UserAction_Input_Select:
		}
	}
	return nil
}

func CompileUserAction(r shared.Runtime, action *flowv1beta1.Action) (_ shared.EvalExpr, err error) {
	env, err := r.Env()
	if err != nil {
		return nil, err
	}

	actionID := GetID(action)
	if action.GetUser() == nil || len(action.GetUser().GetInputs()) == 0 {
		return nil, fmt.Errorf("%s invalid: inputs required", actionID)
	}

	inputsEval := UserActionInputsEvalMap{}
	for _, input := range action.GetUser().GetInputs() {
		inputID := UserActionInputID(action, input)
		if input.GetElement() == nil {
			return nil, fmt.Errorf("%s invalid: element required", inputID)
		}

		inputEval, err := CompileUserActionInput(r, action, input)
		if err != nil {
			return nil, fmt.Errorf("%s invalid: %w", inputID, err)
		}

		inputsEval[inputID] = inputEval
	}

	return shared.EvalExprFunc(func(ctx context.Context) ref.Val {
		if err := inputsEval.Eval(ctx); err != nil {
			return types.WrapErr(err)
		}

		values, err := r.GetUserValues(actionID)
		if err != nil {
			return types.WrapErr(ctx.Err())
		}

		return env.CELTypeAdapter().NativeToValue(values)
	}), nil
}

func CompileUserActionInput(ctx shared.Runtime, action *flowv1beta1.Action, input *flowv1beta1.UserAction_Input) (_ UserActionInputEvalFunc, err error) {
	var elemEval UserActionInputEvalFunc
	switch elemType := input.GetElement().(type) {
	case *flowv1beta1.UserAction_Input_Confirm:
		elemEval, err = CompileUserActionConfirm(ctx, action, elemType.Confirm)
		if err != nil {
			return nil, err
		}
	case *flowv1beta1.UserAction_Input_Input:
		elemEval = func(ctx context.Context) error {
			return nil
		}
	case *flowv1beta1.UserAction_Input_File:
	case *flowv1beta1.UserAction_Input_Select:
	default:
		return nil, fmt.Errorf("%s: invalid input", UserActionInputID(action, input))
	}

	vars, err := ctx.Vars()
	if err != nil {
		return nil, err
	}

	titleExpr, _ := shared.CompileExpr(ctx, input.GetTitle())
	descExpr, _ := shared.CompileExpr(ctx, input.GetDescription())

	return func(ctx context.Context) error {
		if titleExpr != nil {
			val, _, err := titleExpr.ContextEval(ctx, vars)
			if err != nil {
				return err
			} else if titleVal, ok := val.Value().(string); ok {
				input.Title = titleVal
			} else {
				return fmt.Errorf("title: invalid expression, expected: string, got: %T", val.Value())
			}
		}

		if descExpr != nil {
			val, _, err := descExpr.ContextEval(ctx, vars)
			if err != nil {
				return err
			} else if descVal, ok := val.Value().(string); ok {
				input.Description = new(descVal)
			} else {
				return fmt.Errorf("description: invalid expression, expected: string, got: %T", val.Value())
			}
		}

		if elemEval != nil {
			return elemEval(ctx)
		}

		return nil
	}, nil
}

func CompileUserActionConfirm(ctx shared.Runtime, action *flowv1beta1.Action, confirm *protoformv1beta1.ConfirmElement) (UserActionInputEvalFunc, error) {
	vars, err := ctx.Vars()
	if err != nil {
		return nil, err
	}

	approveExpr, _ := shared.CompileExpr(ctx, confirm.GetApprove())
	declineExpr, _ := shared.CompileExpr(ctx, confirm.GetDecline())

	return func(ctx context.Context) error {
		if approveExpr != nil {
			val, _, err := approveExpr.ContextEval(ctx, vars)
			if err != nil {
				return err
			} else if approveVal, ok := val.Value().(string); ok {
				confirm.Approve = approveVal
			} else {
				return fmt.Errorf("confirm.approve: invalid expression, expected: string, got: %T", val.Value())
			}
		}

		if declineExpr != nil {
			val, _, err := declineExpr.ContextEval(ctx, vars)
			if err != nil {
				return err
			} else if declineVal, ok := val.Value().(string); ok {
				confirm.Decline = declineVal
			} else {
				return fmt.Errorf("confirm.decline: invalid expression, expected: string, got: %T", val.Value())
			}
		}

		return nil
	}, nil
}
