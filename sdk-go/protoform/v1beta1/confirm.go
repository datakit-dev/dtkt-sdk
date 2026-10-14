package v1beta1

import (
	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
)

var _ ElementType = (*ConfirmElement)(nil)

type (
	ConfirmElement struct {
		element *protoformv1beta1.ConfirmElement
	}
)

func NewConfirmElement(elem *Element, opts ...ElementTypeOpt[*ConfirmElement]) *ConfirmElement {
	proto := elem.proto.GetConfirm()
	if proto == nil {
		proto = &protoformv1beta1.ConfirmElement{}
	}

	confirm := &ConfirmElement{
		element: proto,
	}

	elem.proto.Type = &protoformv1beta1.FieldElement_Confirm{
		Confirm: proto,
	}
	elem.Type = confirm

	for _, opt := range opts {
		if opt != nil {
			opt(confirm)
		}
	}

	return confirm
}

func (*ConfirmElement) isElementType() {}

func (c *ConfirmElement) GetApprove() string {
	return c.element.GetApprove()
}

func (c *ConfirmElement) GetDecline() string {
	return c.element.GetDecline()
}

func (c *ConfirmElement) GetDefaultSelected() bool {
	return c.element.GetDefaultSelected()
}
