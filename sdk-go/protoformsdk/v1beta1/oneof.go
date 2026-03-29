package v1beta1

import (
	"fmt"
	"slices"

	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ FieldType = (*OneOfField)(nil)

type (
	OneOfField struct {
		FieldGroup
		parent     *Message
		descriptor protoreflect.OneofDescriptor
	}
)

func NewOneOfField(parent *Message, field protoreflect.FieldDescriptor) (*OneOfField, bool) {
	if parent == nil || field == nil || field.ContainingOneof() == nil {
		return nil, false
	}

	var (
		oneOfDesc  = field.ContainingOneof()
		oneOfField = &OneOfField{
			parent:     parent,
			descriptor: oneOfDesc,
		}
	)

	for idx := range oneOfDesc.Fields().Len() {
		desc := oneOfDesc.Fields().Get(idx)
		if desc.Message() != nil {
			msg, ok := NewMessageField(parent, desc)
			if ok {
				oneOfField.fields = append(oneOfField.fields, Field{msg})
			}
		} else if scalar, ok := NewScalarField(parent, desc); ok {
			oneOfField.fields = append(oneOfField.fields, Field{scalar})
		}
	}

	return oneOfField, true
}

func (o *OneOfField) isFieldType() {}

func (o *OneOfField) Parent() *Message {
	return o.parent
}

func (o *OneOfField) OneOfDescriptor() protoreflect.OneofDescriptor {
	return o.descriptor
}

func (o *OneOfField) Descriptor() protoreflect.FieldDescriptor {
	return o.WhichOneOf().Descriptor()
}

func (o *OneOfField) Element() *Element {
	return o.WhichOneOf().Element()
}

func (o *OneOfField) GetFieldGroup(field Field) (*FieldGroup, bool) {
	idx := slices.Index(o.fields, field)
	if idx == -1 {
		return nil, false
	}

	field = o.fields[idx]
	if msg, ok := field.IsMessage(); ok {
		return msg.FieldGroup(), true
	}

	return NewFieldGroup(field), true
}

func (o *OneOfField) WhichOneOf() Field {
	field := o.parent.Get().WhichOneof(o.descriptor)
	idx := max(slices.IndexFunc(o.fields, func(f Field) bool {
		return f.Descriptor() == field
	}), 0)

	return o.fields[idx]
}

func (o *OneOfField) SetOneOf(field Field) error {
	idx := max(slices.IndexFunc(o.fields, func(f Field) bool {
		return f.Descriptor() == field.Descriptor()
	}), 0)
	if idx == -1 {
		return fmt.Errorf("invalid oneOf field: %s", field.Descriptor().FullName())
	}

	parent := o.parent.Get()
	if msg, ok := field.IsMessage(); ok && parent != nil {
		parent.Set(msg.Descriptor(), protoreflect.ValueOfMessage(msg.Get()))
	} else if scalar, ok := field.IsScalar(); ok && parent != nil {
		parent.Set(scalar.Descriptor(), protoreflect.ValueOf(scalar.GetAny()))
	}

	o.parent.Set(parent)

	return nil
}
