package v1beta1

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	Field struct {
		FieldType
	}
	FieldType interface {
		Parent() *Message
		Descriptor() protoreflect.FieldDescriptor
		Element() *Element
		Validate(Env) error
		isFieldType()
	}
	FieldGroup struct {
		fields []Field
		title,
		description string
		hidden bool
		parent *Message
	}
	fd struct {
		protoreflect.FieldDescriptor
	}
)

func NewFieldGroup(fields ...Field) *FieldGroup {
	return &FieldGroup{
		fields: fields,
	}
}

func (f *Field) Get() protoreflect.Value {
	return f.Parent().GetField(f.Descriptor())
}

func (f *Field) Set(value protoreflect.Value) {
	f.Parent().SetField(f.Descriptor(), value)
}

func (f *Field) IsList() (*ListField, bool) {
	l, ok := f.FieldType.(*ListField)
	return l, ok
}

func (f *Field) IsMap() (*MapField, bool) {
	m, ok := f.FieldType.(*MapField)
	return m, ok
}

func (f *Field) IsMessage() (*MessageField, bool) {
	m, ok := f.FieldType.(*MessageField)
	return m, ok
}

func (f *Field) IsOneOf() (*OneOfField, bool) {
	m, ok := f.FieldType.(*OneOfField)
	return m, ok
}

func (f *Field) IsScalar() (*ScalarField, bool) {
	s, ok := f.FieldType.(*ScalarField)
	return s, ok
}

func (g *FieldGroup) Parent() (*Message, bool) {
	if g.parent == nil {
		return nil, false
	}
	return g.parent, true
}

func (g *FieldGroup) Len() int {
	return len(g.fields)
}

func (g *FieldGroup) GetField(idx int) (_ Field, ok bool) {
	if g.Len() == 0 || idx < 0 || idx >= g.Len() {
		return
	}
	return g.fields[idx], true
}

func (g *FieldGroup) GetFields() []Field {
	return g.fields
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
