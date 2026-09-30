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

func NewConfirmElement(elem *Element) *ConfirmElement {
	confirm := elem.proto.GetConfirm()
	if confirm == nil {
		confirm = &protoformv1beta1.ConfirmElement{}
	}

	elem.proto.Type = &protoformv1beta1.FieldElement_Confirm{
		Confirm: confirm,
	}

	elem.Type = &ConfirmElement{
		element: confirm,
	}

	return elem.Type.(*ConfirmElement)
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
