package v1beta2

import (
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
	form "github.com/datakit-dev/dtkt-sdk/sdk-go/protoformsdk/v1beta1"
	"google.golang.org/protobuf/proto"
)

// GetInteractionBinding returns the protoform FieldElement and binding proto
// for an interaction input. The binding's "value" field is used to render a
// form widget and collect user input via protoformsdk.
//
// Returns (nil, nil, false) if the input's element type is unrecognised.
func GetInteractionBinding(input *flowv1beta2.Interaction_Input) (*protoformv1beta1.FieldElement, proto.Message, bool) {
	var binding proto.Message
	switch input.GetElement().(type) {
	case *flowv1beta2.Interaction_Input_Confirm:
		binding = &flowv1beta2.Interaction_ConfirmBinding{}
	case *flowv1beta2.Interaction_Input_Input:
		binding = &flowv1beta2.Interaction_InputBinding{}
	case *flowv1beta2.Interaction_Input_File:
		binding = &flowv1beta2.Interaction_FileBinding{}
	case *flowv1beta2.Interaction_Input_Select:
		binding = &flowv1beta2.Interaction_SelectBinding{}
	case *flowv1beta2.Interaction_Input_MultiSelect:
		binding = &flowv1beta2.Interaction_MultiSelectBinding{}
	}

	if binding == nil {
		return nil, nil, false
	}

	desc := binding.ProtoReflect().Descriptor().Fields().ByName("value")
	elem, ok := form.GetFieldElement(desc)
	if !ok {
		return nil, nil, false
	}

	title := input.GetTitle()
	elem.Title = &title
	if input.HasDescription() {
		d := input.GetDescription()
		elem.Description = &d
	}

	// Merge element-specific options from the spec into the form element.
	switch elemType := input.GetElement().(type) {
	case *flowv1beta2.Interaction_Input_Confirm:
		if elemType.Confirm != nil && elem.GetConfirm() != nil {
			proto.Merge(elem.GetConfirm(), elemType.Confirm)
		} else if elemType.Confirm != nil {
			elem.Type = &protoformv1beta1.FieldElement_Confirm{Confirm: elemType.Confirm}
		}
	case *flowv1beta2.Interaction_Input_Input:
		if elemType.Input != nil {
			elem.Type = &protoformv1beta1.FieldElement_Input{Input: elemType.Input}
		}
	case *flowv1beta2.Interaction_Input_Select:
		if elemType.Select != nil {
			elem.Type = &protoformv1beta1.FieldElement_Select{Select: elemType.Select}
		}
	case *flowv1beta2.Interaction_Input_MultiSelect:
		if elemType.MultiSelect != nil {
			elem.Type = &protoformv1beta1.FieldElement_MultiSelect{MultiSelect: elemType.MultiSelect}
		}
	}

	return elem, binding, true
}
