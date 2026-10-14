package v1beta1

import (
	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protoform"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	Element struct {
		Type  ElementType
		proto *protoformv1beta1.FieldElement
		rules *validate.FieldRules
	}
	ElementType interface {
		isElementType()
	}
	LoadElement interface {
		Load(protoform.Env) error
		isLoadElement()
	}
	ElementTypeOpt[T ElementType] func(T)
)

func NewElement(field protoreflect.FieldDescriptor) *Element {
	elem := &Element{}

	if field != nil {
		if fieldElem, ok := GetFieldElement(field); ok {
			elem.proto = fieldElem
		}

		if fieldRules, ok := GetFieldRules(field); ok {
			elem.rules = fieldRules
		}
	}

	if elem.proto == nil {
		elem.proto = &protoformv1beta1.FieldElement{}
	}

	if elem.rules == nil {
		elem.rules = &validate.FieldRules{}
	}

	if field != nil {
		if elem.proto.GetTitle() == "" {
			elem.proto.Title = new(protoform.NewFieldDescriptor(field).GetTitle())
		}

		if elem.proto.GetDescription() == "" {
			elem.proto.Description = new(protoform.NewFieldDescriptor(field).GetDescription())
		}
	}

	if elem.proto.Hidden == nil {
		elem.proto.Hidden = new(false)
	}

	switch elem.proto.GetType().(type) {
	case *protoformv1beta1.FieldElement_Confirm:
		elem.AsConfirm()
	case *protoformv1beta1.FieldElement_Input:
		elem.AsInput()
	case *protoformv1beta1.FieldElement_MultiSelect:
		elem.AsMultiSelect()
	case *protoformv1beta1.FieldElement_Select:
		elem.AsSelect()
	}

	return elem
}

func (e *Element) Proto() *protoformv1beta1.FieldElement {
	return proto.CloneOf(e.proto)
}

func (e *Element) Rules() *validate.FieldRules {
	return proto.CloneOf(e.rules)
}

func (e *Element) IsValid() bool {
	return e.proto != nil && e.proto.GetType() != nil && e.Type != nil
}

func (e *Element) WithTitle(title string) *Element {
	e.proto.Title = new(title)
	return e
}

func (e *Element) WithDescription(desc string) *Element {
	e.proto.Description = new(desc)
	return e
}

func (e *Element) WithHidden(hidden bool) *Element {
	e.proto.Hidden = new(hidden)
	return e
}

func (e *Element) GetTitle() string {
	return e.proto.GetTitle()
}

func (e *Element) GetDescription() string {
	return e.proto.GetDescription()
}

func (e *Element) GetHidden() bool {
	return e.proto.GetHidden()
}

func (e *Element) GetType() (ElementType, bool) {
	if e.Type == nil {
		return nil, false
	}
	return e.Type, true
}

func (e *Element) AsConfirm(opts ...ElementTypeOpt[*ConfirmElement]) *ConfirmElement {
	return NewConfirmElement(e, opts...)
}

func (e *Element) AsInput(opts ...ElementTypeOpt[*InputElement]) *InputElement {
	return NewInputElement(e, opts...)
}

func (e *Element) AsMultiSelect(opts ...ElementTypeOpt[*MultiSelectElement]) *MultiSelectElement {
	return NewMultiSelectElement(e, opts...)
}

func (e *Element) AsSelect(opts ...ElementTypeOpt[*SelectElement]) *SelectElement {
	return NewSelectElement(e, opts...)
}

func (e *Element) IsConfirm() (confirm *ConfirmElement, ok bool) {
	if e.proto.GetConfirm() != nil {
		confirm, ok = e.Type.(*ConfirmElement)
		if !ok {
			confirm = NewConfirmElement(e)
		}
		return confirm, true
	}
	return
}

func (e *Element) IsInput() (input *InputElement, ok bool) {
	if e.proto.GetInput() != nil {
		input, ok = e.Type.(*InputElement)
		if !ok {
			input = NewInputElement(e)
		}
		return input, true
	}
	return
}

func (e *Element) IsSelect() (selec *SelectElement, ok bool) {
	if e.proto.GetSelect() != nil {
		selec, ok = e.Type.(*SelectElement)
		if !ok {
			selec = NewSelectElement(e)
		}
		return selec, true
	}
	return
}

func (e *Element) IsMultiSelect() (multi *MultiSelectElement, ok bool) {
	if e.proto.GetMultiSelect() != nil {
		multi, ok = e.Type.(*MultiSelectElement)
		if !ok {
			multi = NewMultiSelectElement(e)
		}
		return multi, true
	}
	return
}

// TODO
// func (e *Element) IsFile() (elem *protoformv1beta1.FileElement, ok bool) {
// 	elem = e.proto.GetFile()
// 	ok = elem != nil
// 	return
// }

func (e *Element) Load(env protoform.Env) error {
	if t, ok := e.Type.(LoadElement); ok {
		return t.Load(env)
	}
	return nil
}
