package form

import (
	"fmt"
	"slices"

	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ FieldType = (*OneOfField)(nil)

type (
	OneOfField struct {
		descriptor protoreflect.OneofDescriptor
		parent     *Message
		element    *Element
		current    *Field
		group      *FieldGroup
	}
)

func NewOneOfField(message *Message, field protoreflect.FieldDescriptor) (*OneOfField, bool) {
	if message == nil || field == nil || field.ContainingOneof() == nil {
		return nil, false
	}

	var (
		oneOfDesc  = field.ContainingOneof()
		oneOfField = &OneOfField{
			parent:     message,
			descriptor: oneOfDesc,
			element: NewElement(nil).
				WithTitle(string(oneOfDesc.Name())),
		}
		whichOne protoreflect.FieldDescriptor
		fields   []*Field
	)

	// Check which oneof field is set (if message is not nil)
	if msg := message.Get(); msg != nil {
		whichOne = msg.WhichOneof(field.ContainingOneof())
	}

	for idx := range oneOfDesc.Fields().Len() {
		fieldDesc := oneOfDesc.Fields().Get(idx)

		if fieldDesc.Message() != nil {
			msg, ok := NewMessageField(message, fieldDesc)
			if ok {
				field := NewField(msg)
				fields = append(fields, field)
				if fieldDesc == whichOne {
					oneOfField.current = field
				}
			}
		} else if scalar, ok := NewScalarField(message, fieldDesc); ok {
			field := NewField(scalar)
			fields = append(fields, field)
			if fieldDesc == whichOne {
				oneOfField.current = field
			}
		}
	}

	oneOfField.group = NewFieldGroup(message, fields...)

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
	if field, ok := o.WhichOneOf(); ok {
		return field.Type.Descriptor()
	}
	return nil
}

func (o *OneOfField) Element() *Element {
	if field, ok := o.WhichOneOf(); ok {
		return field.Type.Element()
	}
	return o.element
}

func (o *OneOfField) GetFieldGroup(field *Field) (*FieldGroup, bool) {
	idx := slices.IndexFunc(o.group.fields, func(f *Field) bool {
		return field.Type.Descriptor() == f.Type.Descriptor()
	})
	if idx == -1 {
		return nil, false
	}

	field = o.group.fields[idx]
	if msg, ok := field.IsMessage(); ok {
		return msg.FieldGroup(), true
	}

	return NewFieldGroup(o.parent, field), true
}

func (o *OneOfField) GetFields() []*Field {
	return o.group.fields
}

func (o *OneOfField) WhichOneOf() (*Field, bool) {
	if o.current == nil {
		return nil, false
	}
	return o.current, true
}

func (o *OneOfField) SetOneOf(field *Field) error {
	if field == nil {
		return nil
	} else if !slices.Contains(o.group.fields, field) {
		return fmt.Errorf("invalid oneOf field: %s", field.Type.Descriptor().FullName())
	}

	o.current = field

	if msgField := o.parent.Get(); msgField != nil {
		if msg, ok := field.IsMessage(); ok {
			msgField.Set(msg.descriptor, protoreflect.ValueOfMessage(msg.Get()))
		} else if scalar, ok := field.IsScalar(); ok {
			msgField.Set(scalar.Descriptor(), protoreflect.ValueOf(scalar.GetAny()))
		}
	}

	o.parent.Set(o.parent.Get())

	return nil
}
