package v1beta1

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
		Message
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

func NewMessage(val protoreflect.Message) *Message {
	if val == nil {
		log.Fatal("message cannot be nil")
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
	}
}

func NewMessageField(parent *Message, desc protoreflect.FieldDescriptor) (*MessageField, bool) {
	if parent == nil || desc == nil || desc.Message() == nil {
		return nil, false
	}

	field := &MessageField{
		element:    NewElement(desc),
		parent:     parent,
		descriptor: desc,
	}

	field.Message.descriptor = desc.Message()
	field.Getter = GetterFunc[protoreflect.Message](func() protoreflect.Message {
		if parent.Get().Has(desc) {
			return parent.Get().Mutable(desc).Message()
		}
		return parent.Get().NewField(desc).Message()
	})
	field.Setter = SetterFunc[protoreflect.Message](func(msg protoreflect.Message) {
		if msg == nil {
			parent.Get().Clear(desc)
		} else {
			parent.Get().Set(desc, protoreflect.ValueOfMessage(msg))
		}
	})

	return field, true
}

func MergeMessage(dst, src protoreflect.Message) {
	if dst.Descriptor() == src.Descriptor() {
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
	if m.group != nil {
		return m.group
	}

	var (
		fields []Field
		oneOfs []protoreflect.OneofDescriptor
	)
	for idx := range m.Descriptor().Fields().Len() {
		desc := m.Descriptor().Fields().Get(idx)
		if desc.ContainingOneof() != nil && !desc.ContainingOneof().IsSynthetic() {
			if slices.Contains(oneOfs, desc.ContainingOneof()) {
				continue
			}

			if field, ok := NewOneOfField(m, desc); ok {
				fields = append(fields, Field{field})
				oneOfs = append(oneOfs, desc.ContainingOneof())
			}
		} else if desc.IsList() {
			if field, ok := NewListField(m, desc); ok {
				fields = append(fields, Field{field})
			}
		} else if desc.IsMap() {
			if field, ok := NewMapField(m, desc); ok {
				fields = append(fields, Field{field})
			}
		} else if desc.Message() != nil {
			if field, ok := NewMessageField(m, desc); ok {
				fields = append(fields, Field{field})
			}
		} else {
			if field, ok := NewScalarField(m, desc); ok {
				fields = append(fields, Field{field})
			}
		}
	}

	desc := GetProtoDescription(m.Descriptor())
	if desc == "" {
		desc = fmt.Sprint(m.Descriptor().FullName())
	}

	m.group = NewFieldGroup(fields...).
		WithTitle(fmt.Sprint(m.Descriptor().Name())).
		WithDescription(desc)

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
