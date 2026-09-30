package protoform

import (
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var sourceInfo = util.ProtoSourceInfoOptions{
	Multiline: false,
}

type (
	Field interface {
		Parent() *Message
		Descriptor() protoreflect.FieldDescriptor
		Validate(Env) error
		isField()
	}
	FieldGroup struct {
		fields []Field
		parent *Message
	}
	FieldDescriptor struct {
		protoreflect.FieldDescriptor
	}
)

func NewFieldGroup(fields ...Field) *FieldGroup {
	return &FieldGroup{
		fields: fields,
	}
}

func NewFieldDescriptor(desc protoreflect.FieldDescriptor) FieldDescriptor {
	return FieldDescriptor{FieldDescriptor: desc}
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

func (d FieldDescriptor) IsEnum() bool {
	return d.Enum() != nil
}

func (d FieldDescriptor) IsMessage() bool {
	return !d.IsMap() && d.Message() != nil
}

func (d FieldDescriptor) IsOneOf() bool {
	return d.ContainingOneof() != nil && !d.ContainingOneof().IsSynthetic()
}

func (d FieldDescriptor) GetTitle() string {
	return d.JSONName()
}

func (d FieldDescriptor) GetDescription() string {
	if desc := sourceInfo.GetDescription(d.FieldDescriptor); desc != "" {
		return desc
	}

	if d.IsMap() {
		mapKey, mapVal := d.MapKey(), d.MapValue()
		return fmt.Sprintf("Map<%s, %s>", mapKey.Kind(), FieldDescriptor{mapVal}.GetDescription())
	} else if d.IsMessage() {
		if d.IsList() {
			return fmt.Sprintf("List<%s>", d.Message().FullName())
		}
		return string(d.Message().FullName())
	} else if d.IsEnum() {
		if d.IsList() {
			return fmt.Sprintf("List<%s>", d.Enum().FullName())
		}
		return string(d.Enum().FullName())
	}

	if d.IsList() {
		return fmt.Sprintf("List<%s>", d.Kind())
	}

	return fmt.Sprint(d.Kind())
}
