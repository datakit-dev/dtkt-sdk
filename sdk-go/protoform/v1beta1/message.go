package v1beta1

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ Binding[protoreflect.Message] = (*MessageField)(nil)
var _ FieldType = (*MessageField)(nil)

type (
	Message struct {
		Parser[protoreflect.Message]
		Stringer[protoreflect.Message]
		value      protoreflect.Message
		group      *FieldGroup
		onSet      func(protoreflect.Message)
		onFieldSet func()
		validate   func(Env) error

		depth int
		mut   sync.Mutex
	}
	MessageField struct {
		*Message
		element    *Element
		parent     *Message
		descriptor protoreflect.FieldDescriptor
	}
)

func NewMessage(value protoreflect.Message) *Message {
	if !value.IsValid() {
		value = value.New()
	}

	return &Message{
		value: value,
	}
}

func NewListMessage(parent *Message, field protoreflect.FieldDescriptor, value protoreflect.Message) *Message {
	return &Message{
		value: value,
		validate: func(env Env) error {
			return ValidateMessage(env, value)
		},
		depth: parent.depth + 1,
	}
}

func NewMapMessage(parent *Message, field protoreflect.FieldDescriptor, value protoreflect.Message) *Message {
	return &Message{
		value: value,
		validate: func(env Env) error {
			return ValidateMessage(env, value)
		},
		depth: parent.depth + 1,
	}
}

func NewMessageField(parent *Message, field protoreflect.FieldDescriptor) (*MessageField, bool) {
	if parent == nil || field == nil || field.IsMap() || field.Message() == nil {
		return nil, false
	}

	var value protoreflect.Message
	if parent.value.Has(field) {
		value = parent.value.Mutable(field).Message()
	} else {
		value = parent.value.NewField(field).Message()
	}

	msg := NewMessage(value)
	msg.onSet = func(src protoreflect.Message) {
		if src != nil {
			MergeMessage(value, src)
			parent.SetField(field, protoreflect.ValueOfMessage(value))
		}
	}
	msg.onFieldSet = func() {
		if !parent.HasField(field) {
			parent.SetField(field, protoreflect.ValueOfMessage(value))
		}
	}
	msg.validate = func(env Env) error {
		return parent.ValidateField(env, field, protoreflect.ValueOfMessage(value))
	}
	msg.depth = parent.depth + 1

	return &MessageField{
		Message:    msg,
		element:    NewElement(field),
		parent:     parent,
		descriptor: field,
	}, true
}

func MergeMessage(dst, src protoreflect.Message) {
	if dst != nil && src != nil {
		if dst.Descriptor() == src.Descriptor() {
			for idx := range dst.Descriptor().Fields().Len() {
				field := dst.Descriptor().Fields().Get(idx)
				if !src.Has(field) {
					dst.Clear(field)
				} else {
					dst.Set(field, src.Get(field))
				}
			}
		} else if dst.Descriptor().FullName() == src.Descriptor().FullName() {
			b, err := proto.Marshal(src.Interface())
			if err == nil {
				proto.UnmarshalOptions{
					Merge: true,
				}.Unmarshal(b, dst.Interface())
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

	desc := util.GetProtoDescription(m.Descriptor())
	if desc == "" {
		desc = fmt.Sprint(m.Descriptor().FullName())
	}

	m.group = NewFieldGroup(fields...).
		WithTitle(fmt.Sprint(m.Descriptor().Name())).
		WithDescription(desc)
	m.group.parent = m

	return m.group
}

func (m *Message) ValidateField(env Env, field protoreflect.FieldDescriptor, value protoreflect.Value) error {
	env.Logger().Info("Message.ValidateField",
		slog.String("msg", string(m.value.Descriptor().FullName())),
		slog.String("field", string(field.FullName())),
		slog.Int("depth", m.depth),
	)

	if m.validate != nil {
		return m.validate(env)
	}

	msg := proto.Clone(m.Get().Interface())
	msg.ProtoReflect().Set(field, value)

	return ValidateField(env, msg.ProtoReflect(), field)
}

func (f *MessageField) Validate(env Env) error {
	return f.parent.ValidateField(env, f.descriptor, protoreflect.ValueOfMessage(f.value))
}

func (f *MessageField) Descriptor() protoreflect.FieldDescriptor {
	return f.descriptor
}

func (f *MessageField) Element() *Element {
	return f.element
}

func (f *MessageField) Parent() *Message {
	return f.parent
}

func (f *MessageField) Clear() {
	f.mut.Lock()
	defer f.mut.Unlock()

	proto.Reset(f.value.Interface())
	f.parent.ClearField(f.descriptor)
}

func (f *MessageField) IsSet() bool {
	return f.parent.HasField(f.descriptor)
}

func (m *Message) IsEmpty() (empty bool) {
	empty = true
	m.Get().Range(func(_ protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		empty = false
		return false
	})
	return
}

func (m *Message) Get() protoreflect.Message {
	return m.value
}

func (m *Message) Set(msg protoreflect.Message) {
	m.mut.Lock()
	defer m.mut.Unlock()

	if m.onSet != nil {
		m.onSet(msg)
	} else if msg != nil {
		MergeMessage(m.value, msg)
	}
}

func (m *Message) HasField(desc protoreflect.FieldDescriptor) bool {
	if desc == nil {
		return false
	}

	m.mut.Lock()
	defer m.mut.Unlock()

	return m.value.Has(desc)
}

func (m *Message) ClearField(desc protoreflect.FieldDescriptor) {
	if desc == nil {
		return
	}

	m.mut.Lock()
	defer m.mut.Unlock()

	m.value.Clear(desc)
}

func (m *Message) GetField(desc protoreflect.FieldDescriptor) (value protoreflect.Value) {
	if desc == nil {
		return
	}

	hasField := m.HasField(desc)

	m.mut.Lock()
	defer m.mut.Unlock()

	if hasField && (desc.IsList() || desc.IsMap() || desc.Message() != nil) {
		return m.value.Mutable(desc)
	} else if hasField {
		return m.value.Get(desc)
	}

	return m.value.NewField(desc)
}

func (m *Message) SetField(desc protoreflect.FieldDescriptor, value protoreflect.Value) {
	if desc == nil || !isDescriptorValue(desc, value) || !value.IsValid() {
		return
	}

	var (
		fields = m.Descriptor().Fields()
		found  bool
	)
	for idx := range fields.Len() {
		found = fields.Get(idx) == desc
		if found {
			break
		}
	}

	if found {
		m.mut.Lock()
		defer m.mut.Unlock()

		m.value.Set(desc, value)

		if m.onFieldSet != nil {
			m.onFieldSet()
		}
	}
}

func (m *Message) Parse(str string) (protoreflect.Message, error) {
	if m.Parser != nil {
		return m.Parser.Parse(str)
	}

	m.mut.Lock()
	msg := m.value.New()
	m.mut.Unlock()

	if str != "" {
		err := encoding.FromJSON([]byte(str), msg.Interface())
		if err != nil {
			return nil, err
		}
	}

	return msg, nil
}

func (m *Message) StringOf(msg protoreflect.Message) string {
	if m.Stringer != nil {
		return m.Stringer.StringOf(msg)
	}

	if msg == nil {
		return ""
	}

	b, _ := encoding.ToJSON(msg.Interface())
	return string(b)
}

func (m *Message) String() string {
	return m.StringOf(m.Get())
}

func (m *Message) Descriptor() protoreflect.MessageDescriptor {
	m.mut.Lock()
	defer m.mut.Unlock()

	return m.value.Descriptor()
}

func (f *MessageField) isFieldType() {}
