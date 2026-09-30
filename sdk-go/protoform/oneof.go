package protoform

import (
	"fmt"
	"slices"

	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ Field = (*OneOfField)(nil)

type OneOfField struct {
	FieldGroup
	parent     *Message
	descriptor protoreflect.OneofDescriptor
}

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
				oneOfField.fields = append(oneOfField.fields, msg)
			}
		} else if scalar, ok := NewScalarField(parent, desc); ok {
			oneOfField.fields = append(oneOfField.fields, scalar)
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

func (f *OneOfField) GetFieldGroup(field Field) (*FieldGroup, bool) {
	idx := slices.Index(f.fields, field)
	if idx == -1 {
		return nil, false
	}

	field = f.fields[idx]
	if msg, ok := field.(*MessageField); ok {
		return msg.FieldGroup(), true
	}

	return NewFieldGroup(field), true
}

func (f *OneOfField) WhichOneOf() Field {
	desc := f.parent.Get().WhichOneof(f.descriptor)
	idx := max(0, slices.IndexFunc(f.fields, func(f Field) bool {
		return f.Descriptor() == desc
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

	if msg, ok := field.(*MessageField); ok {
		f.parent.SetField(msg.Descriptor(), protoreflect.ValueOfMessage(msg.Get()))
	} else if scalar, ok := field.(ScalarFieldType); ok {
		f.parent.SetField(scalar.Descriptor(), protoreflect.ValueOf(scalar.GetAny()))
	}

	return nil
}

func (f *OneOfField) isField() {}
