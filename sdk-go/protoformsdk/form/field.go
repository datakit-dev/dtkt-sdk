package form

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	FieldGroup struct {
		message            *Message
		fields             []*Field
		hidden             bool
		title, description string
	}
	Field struct {
		Type FieldType
	}
	FieldType interface {
		Parent() *Message
		Descriptor() protoreflect.FieldDescriptor
		Element() *Element
		isFieldType()
	}
	fd struct {
		protoreflect.FieldDescriptor
	}
)

func NewFieldGroup(message *Message, fields ...*Field) *FieldGroup {
	var title, desc string
	if message != nil && message.Descriptor() != nil {
		title = fmt.Sprint(message.Descriptor().Name())
		desc = GetProtoDescription(message.Descriptor())
		if desc == "" {
			desc = fmt.Sprint(message.Descriptor().FullName())
		}
	}

	return &FieldGroup{
		message:     message,
		fields:      fields,
		title:       title,
		description: desc,
	}
}

func NewField(fieldType FieldType) *Field {
	return &Field{
		Type: fieldType,
	}
}

func (f *Field) IsList() (*ListField, bool) {
	l, ok := f.Type.(*ListField)
	return l, ok
}

func (f *Field) IsMap() (*MapField, bool) {
	m, ok := f.Type.(*MapField)
	return m, ok
}

func (f *Field) IsMessage() (*MessageField, bool) {
	m, ok := f.Type.(*MessageField)
	return m, ok
}

func (f *Field) IsOneOf() (*OneOfField, bool) {
	m, ok := f.Type.(*OneOfField)
	return m, ok
}

func (f *Field) IsScalar() (*ScalarField, bool) {
	s, ok := f.Type.(*ScalarField)
	return s, ok
}

func (g *FieldGroup) Message() *Message {
	return g.message
}

func (g *FieldGroup) Len() int {
	return len(g.fields)
}

func (g *FieldGroup) GetFields() []*Field {
	return g.fields
}

func (g *FieldGroup) WithFields(fields []*Field) *FieldGroup {
	g.fields = fields
	return g
}

func (g *FieldGroup) WithTitle(t string) *FieldGroup {
	g.title = t
	return g
}

func (g *FieldGroup) WithDescription(d string) *FieldGroup {
	g.description = d
	return g
}

func (g *FieldGroup) WithHidden(b bool) *FieldGroup {
	g.hidden = b
	return g
}

func (g *FieldGroup) GetTitle() string {
	return g.title
}

func (g *FieldGroup) GetDescription() string {
	return g.description
}

func (g *FieldGroup) GetHidden() bool {
	return g.hidden
}

func (d fd) IsEnum() bool {
	return d.Enum() != nil
}

func (d fd) IsMessage() bool {
	return d.Message() != nil && !d.IsMap()
}

func (d fd) IsOneOf() bool {
	return d.ContainingOneof() != nil && !d.ContainingOneof().IsSynthetic()
}

func (d fd) IsMapKey() bool {
	return false
}

func (d fd) GetTitle() string {
	return d.JSONName()
}

func (d fd) GetDescription() (desc string) {
	if d.IsMap() {
		mapKey, mapVal := d.MapKey(), d.MapValue()
		return fmt.Sprintf("Map<%s: %s>", mapKey.Kind(), fd{mapVal}.GetDescription())
	} else if d.IsMessage() {
		desc = fmt.Sprintf("Message<%s>", d.Message().FullName())
	} else if d.IsEnum() {
		desc = fmt.Sprintf("Enum<%s>", d.Enum().FullName())
	} else {
		desc = fmt.Sprint(d.Kind())
	}

	if d.IsList() {
		desc = fmt.Sprintf("List<%s>", desc)
	}

	return
}
