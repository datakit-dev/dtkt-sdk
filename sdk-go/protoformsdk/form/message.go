package form

import (
	"fmt"
	"log"
	"slices"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ Binding[protoreflect.Message] = (*MessageField)(nil)
var _ FieldType = (*MessageField)(nil)

type (
	MessageField struct {
		*Message
		element    *Element
		parent     *Message
		descriptor protoreflect.FieldDescriptor
	}
	Message struct {
		Getter[protoreflect.Message]
		Setter[protoreflect.Message]
		Parser[protoreflect.Message]
		Stringer[protoreflect.Message]
		descriptor protoreflect.MessageDescriptor
		group      *FieldGroup
	}
)

func NewMessage(val protoreflect.Message) (*Message, bool) {
	if val == nil {
		return nil, false
	}

	return &Message{
		descriptor: val.Descriptor(),
		Getter: GetterFunc[protoreflect.Message](func() protoreflect.Message {
			return val
		}),
		Setter: SetterFunc[protoreflect.Message](func(msg protoreflect.Message) {
			if msg == nil {
				val.Range(func(field protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
					val.Clear(field)
					return true
				})
			} else {
				MergeMessage(val, msg)
			}
		}),
	}, true
}

func NewMessageField(parent *Message, field protoreflect.FieldDescriptor) (*MessageField, bool) {
	if parent == nil || field == nil || field.Message() == nil {
		return nil, false
	}

	var (
		get = func() protoreflect.Message {
			if parent.Get().Has(field) {
				return parent.Get().Mutable(field).Message()
			}
			return parent.Get().NewField(field).Message()
		}
		set = func(msg protoreflect.Message) {
			if msg == nil {
				msg = get()
				msg.Range(func(field protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
					msg.Clear(field)
					return true
				})
			} else {
				parent.Get().Set(field, protoreflect.ValueOfMessage(msg))
			}
		}
	)

	return &MessageField{
		Message: &Message{
			descriptor: field.Message(),
			Getter:     GetterFunc[protoreflect.Message](get),
			Setter:     SetterFunc[protoreflect.Message](set),
		},
		element:    NewElement(field),
		parent:     parent,
		descriptor: field,
	}, true
}

func MergeMessage(dst, src protoreflect.Message) {
	if dst.Descriptor().FullName() == src.Descriptor().FullName() {
		for idx := range dst.Descriptor().Fields().Len() {
			field := dst.Descriptor().Fields().Get(idx)
			if !src.Has(field) {
				dst.Clear(field)
			} else {
				dst.Set(field, src.Get(field))
			}
		}
	}
}

func (m *Message) FieldGroup() *FieldGroup {
	if m == nil {
		log.Fatal("field group message cannot be nil")
	} else if m.Descriptor() == nil {
		log.Fatal("field group message descriptor cannot be nil")
	} else if m.group != nil {
		return m.group
	}

	var (
		fields []*Field
		oneOfs []protoreflect.OneofDescriptor
	)
	for idx := range m.Descriptor().Fields().Len() {
		desc := m.Descriptor().Fields().Get(idx)
		if desc.ContainingOneof() != nil && !desc.ContainingOneof().IsSynthetic() {
			if slices.Contains(oneOfs, desc.ContainingOneof()) {
				continue
			}

			if field, ok := NewOneOfField(m, desc); ok {
				fields = append(fields, NewField(field))
				oneOfs = append(oneOfs, desc.ContainingOneof())
			}
		} else if desc.IsList() {
			if field, ok := NewListField(m, desc); ok {
				fields = append(fields, NewField(field))
			}
		} else if desc.IsMap() {
			if field, ok := NewMapField(m, desc); ok {
				fields = append(fields, NewField(field))
			}
		} else if desc.Message() != nil {
			if field, ok := NewMessageField(m, desc); ok {
				fields = append(fields, NewField(field))
			}
		} else {
			if field, ok := NewScalarField(m, desc); ok {
				fields = append(fields, NewField(field))
			}
		}
	}

	m.group = NewFieldGroup(m, fields...)
	return m.group
}

func (m *MessageField) isFieldType() {}

func (m *MessageField) Descriptor() protoreflect.FieldDescriptor {
	return m.descriptor
}

func (m *MessageField) Element() *Element {
	return m.element
}

func (m *MessageField) Parent() *Message {
	return m.parent
}

func (m *MessageField) Clear() {
	if m.parent != nil && m.parent.Get() != nil {
		m.parent.Get().Clear(m.descriptor)
	}
}

func (m *MessageField) IsSet() bool {
	return m.parent.Get().Has(m.descriptor)
}

func (m *Message) IsEmpty() bool {
	if m.Get() == nil {
		return true
	}

	var count int
	m.Get().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		count++
		return true
	})
	return count == 0
}

func (m *Message) WithGetter(g Getter[protoreflect.Message]) *Message {
	if g != nil {
		m.Getter = g
	}
	return m
}

func (m *Message) WithSetter(s Setter[protoreflect.Message]) *Message {
	if s != nil {
		m.Setter = s
	}
	return m
}

func (m *Message) Get() protoreflect.Message {
	if m.Getter != nil {
		return m.Getter.Get()
	}
	return nil
}

func (m *Message) Set(msg protoreflect.Message) {
	if m.Setter != nil {
		m.Setter.Set(msg)
	}
}

func (m *Message) Parse(str string) (protoreflect.Message, error) {
	if m.Parser != nil {
		return m.Parser.Parse(str)
	}

	msg := m.Get()
	if msg == nil {
		return nil, fmt.Errorf("message not set")
	}

	if str != "" {
		err := encoding.FromJSONV2([]byte(str), msg.Interface())
		if err != nil {
			return nil, err
		}

		return msg, nil
	}

	return msg.New(), nil
}

func (m *Message) StringOf(msg protoreflect.Message) string {
	if m.Stringer != nil {
		return m.Stringer.StringOf(msg)
	}
	if msg == nil {
		return ""
	}
	b, _ := encoding.ToJSONV2(msg.Interface())
	return string(b)
}

func (m *Message) String() string {
	return m.StringOf(m.Get())
}

func (m *Message) Descriptor() protoreflect.MessageDescriptor {
	return m.descriptor
}
