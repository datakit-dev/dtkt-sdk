package v1beta1

import (
	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
)

var _ ElementType = (*InputElement)(nil)

type (
	InputElement struct {
		element *protoformv1beta1.InputElement
	}
)

func NewInputElement(elem *Element) *InputElement {
	input := elem.proto.GetInput()
	if input == nil {
		input = &protoformv1beta1.InputElement{}
	}

	elem.proto.Type = &protoformv1beta1.FieldElement_Input{
		Input: input,
	}

	elem.Type = &InputElement{
		element: input,
	}

	return elem.Type.(*InputElement)
}

func (*InputElement) isElementType() {}

func (i *InputElement) AsMultiline(m bool) *InputElement {
	i.element.MultilineText = new(m)
	return i
}

func (i *InputElement) IsMultiline() bool {
	return i.element.GetMultilineText()
}
