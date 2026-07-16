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

func (f *OneOfField) Parent() *Message {
	return f.parent
}

func (f *OneOfField) Validate(env Env) error {
	return f.WhichOneOf().Validate(env)
}

func (f *OneOfField) OneOfDescriptor() protoreflect.OneofDescriptor {
	return f.descriptor
}

func (f *OneOfField) Descriptor() protoreflect.FieldDescriptor {
	return f.WhichOneOf().Descriptor()
}

func (f *OneOfField) Element() *Element {
	return f.WhichOneOf().Element()
}

func (f *OneOfField) GetFieldGroup(field Field) (*FieldGroup, bool) {
	idx := slices.Index(f.fields, field)
	if idx == -1 {
		return nil, false
	}

	field = f.fields[idx]
	if msg, ok := field.IsMessage(); ok {
		return msg.FieldGroup(), true
	}

	return NewFieldGroup(field), true
}

func (f *OneOfField) WhichOneOf() Field {
	field := f.parent.Get().WhichOneof(f.descriptor)
	idx := max(0, slices.IndexFunc(f.fields, func(f Field) bool {
		return f.Descriptor() == field
	}))
	return f.fields[idx]
}

func (f *OneOfField) SetOneOf(field Field) error {
	idx := max(slices.IndexFunc(f.fields, func(f Field) bool {
		return f.Descriptor() == field.Descriptor()
	}), 0)
	if idx == -1 {
		return fmt.Errorf("invalid oneOf field: %s", field.Descriptor().FullName())
	}

	if msg, ok := field.IsMessage(); ok {
		f.parent.SetField(msg.Descriptor(), protoreflect.ValueOfMessage(msg.Get()))
	} else if scalar, ok := field.IsScalar(); ok {
		f.parent.SetField(scalar.Descriptor(), protoreflect.ValueOf(scalar.GetAny()))
	}

	return nil
}

func (f *OneOfField) isFieldType() {}
