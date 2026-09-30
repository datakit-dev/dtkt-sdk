package v1beta1

import (
	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
)

var _ ElementType = (*InputElement)(nil)

type InputElement struct {
	proto *protoformv1beta1.InputElement
}

func NewInputElement(elem *Element, opts ...ElementTypeOpt[*InputElement]) *InputElement {
	proto := elem.proto.GetInput()
	if proto == nil {
		proto = &protoformv1beta1.InputElement{}
	}

	input := &InputElement{
		proto: proto,
	}

	elem.proto.Type = &protoformv1beta1.FieldElement_Input{
		Input: proto,
	}
	elem.Type = input

	for _, opt := range opts {
		if opt != nil {
			opt(input)
		}
	}

	return input
}

func (*InputElement) isElementType() {}

func (i *InputElement) AsMultiline(m bool) *InputElement {
	i.proto.MultilineText = new(m)
	return i
}

func (i *InputElement) IsMultiline() bool {
	return i.proto.GetMultilineText()
}
