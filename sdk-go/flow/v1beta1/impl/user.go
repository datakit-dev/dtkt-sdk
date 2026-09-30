package impl

import (
	"context"
	"fmt"

	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
	form "github.com/datakit-dev/dtkt-sdk/sdk-go/protoform/v1beta1"
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
	return fmt.Sprintf("%s.inputs.%s", shared.GetNodeIdent(action), input.GetId())
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

func ParseUserAction(env shared.Env, action *flowv1beta1.UserAction, visitor shared.ExprVisitor) error {
	for idx, input := range action.GetInputs() {
		id := fmt.Sprintf("inputs[%d]", idx)
		switch elemType := input.GetElement().(type) {
		case *flowv1beta1.UserAction_Input_Confirm:
			if elemType.Confirm != nil {
				_, err := env.ParseExpr(id+".confirm.approve", elemType.Confirm.GetApprove(), visitor)
				if err != nil && !shared.IsInvalidExprError(err) {
					return err
				}

				_, err = env.ParseExpr(id+".confirm.decline", elemType.Confirm.GetDecline(), visitor)
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

func CompileUserAction(env shared.Env, action *flowv1beta1.Action) (_ shared.Program, err error) {
	ident := shared.GetNodeIdent(action)
	if action.GetUser() == nil || len(action.GetUser().GetInputs()) == 0 {
		return nil, fmt.Errorf("%s invalid: inputs required", ident)
	}

	inputsEval := UserActionInputsEvalMap{}
	for _, input := range action.GetUser().GetInputs() {
		inputID := UserActionInputID(action, input)
		if input.GetElement() == nil {
			return nil, fmt.Errorf("%s invalid: element required", inputID)
		}

		inputEval, err := CompileUserActionInput(env, action, input)
		if err != nil {
			return nil, fmt.Errorf("%s invalid: %w", inputID, err)
		}

		inputsEval[inputID] = inputEval
	}

	return shared.ProgramFunc(func(ctx context.Context, env shared.Env) ref.Val {
		if err := inputsEval.Eval(ctx); err != nil {
			return types.WrapErr(err)
		}

		// values, err := r.GetUserValues(actionID)
		// values, err := run.GetValue(actionID)
		// if err != nil {
		// 	return types.WrapErr(context.Cause(ctx))
		// }

		// return env.Types().NativeToValue(values)
		return nil
	}), nil
}

func CompileUserActionInput(env shared.Env, action *flowv1beta1.Action, input *flowv1beta1.UserAction_Input) (_ UserActionInputEvalFunc, err error) {
	// var elemEval UserActionInputEvalFunc
	// switch elemType := input.GetElement().(type) {
	// case *flowv1beta1.UserAction_Input_Confirm:
	// 	elemEval, err = CompileUserActionConfirm(env, action, elemType.Confirm)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// case *flowv1beta1.UserAction_Input_Input:
	// 	elemEval = func(ctx context.Context) error {
	// 		return nil
	// 	}
	// case *flowv1beta1.UserAction_Input_File:
	// case *flowv1beta1.UserAction_Input_Select:
	// default:
	// 	return nil, fmt.Errorf("%s: invalid input", UserActionInputID(action, input))
	// }

	// titleExpr, _ := shared.CompileExpr(env, input.GetTitle())
	// descExpr, _ := shared.CompileExpr(env, input.GetDescription())

	return func(ctx context.Context) error {
		// vars, err := env.Vars()
		// if err != nil {
		// 	return err
		// }

		// if titleExpr != nil {
		// 	val, _, err := titleExpr.ContextEval(ctx, vars)
		// 	if err != nil {
		// 		return err
		// 	} else if titleVal, ok := val.Value().(string); ok {
		// 		input.Title = titleVal
		// 	} else {
		// 		return fmt.Errorf("title: invalid expression, expected: string, got: %T", val.Value())
		// 	}
		// }

		// if descExpr != nil {
		// 	val, _, err := descExpr.ContextEval(ctx, vars)
		// 	if err != nil {
		// 		return err
		// 	} else if descVal, ok := val.Value().(string); ok {
		// 		input.Description = new(descVal)
		// 	} else {
		// 		return fmt.Errorf("description: invalid expression, expected: string, got: %T", val.Value())
		// 	}
		// }

		// if elemEval != nil {
		// 	return elemEval(ctx)
		// }

		return nil
	}, nil
}

func CompileUserActionConfirm(env shared.Env, action *flowv1beta1.Action, confirm *protoformv1beta1.ConfirmElement) (UserActionInputEvalFunc, error) {
	// approveExpr, _ := shared.CompileExpr(env, confirm.GetApprove())
	// declineExpr, _ := shared.CompileExpr(env, confirm.GetDecline())

	return func(ctx context.Context) error {
		// vars, err := run.Vars()
		// if err != nil {
		// 	return err
		// }

		// if approveExpr != nil {
		// 	val, _, err := approveExpr.ContextEval(ctx, vars)
		// 	if err != nil {
		// 		return err
		// 	} else if approveVal, ok := val.Value().(string); ok {
		// 		confirm.Approve = approveVal
		// 	} else {
		// 		return fmt.Errorf("confirm.approve: invalid expression, expected: string, got: %T", val.Value())
		// 	}
		// }

		// if declineExpr != nil {
		// 	val, _, err := declineExpr.ContextEval(ctx, vars)
		// 	if err != nil {
		// 		return err
		// 	} else if declineVal, ok := val.Value().(string); ok {
		// 		confirm.Decline = declineVal
		// 	} else {
		// 		return fmt.Errorf("confirm.decline: invalid expression, expected: string, got: %T", val.Value())
		// 	}
		// }

		return nil
	}, nil
}
