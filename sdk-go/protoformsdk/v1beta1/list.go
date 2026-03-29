package v1beta1

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ Binding[[]bool] = (*List[bool])(nil)

var _ FieldType = (*ListField)(nil)

type (
	ListField struct {
		ListType
		element *Element
	}
	List[T ListValue] struct {
		Getter[[]T]
		Setter[[]T]
		Parser[[]T]
		Stringer[[]T]
		parent     *Message
		descriptor protoreflect.FieldDescriptor
		newItem    func() T
	}
	ListType interface {
		Parent() *Message
		Descriptor() protoreflect.FieldDescriptor
		AppendAny(any) error
		NewItemAny() any
		GetItemAny(int) (any, error)
		RemoveItem(int) error
		GetItems() []any
		SetItems([]any) error
		Len() int
		ParseItems(string) ([]any, error)
		String() string
	}
	ListValue interface {
		ScalarValue | *Message
	}
)

func NewListField(parent *Message, field protoreflect.FieldDescriptor) (*ListField, bool) {
	if parent == nil || field == nil || !field.IsList() {
		return nil, false
	}

	list, ok := NewList(parent, field)
	if !ok {
		return nil, false
	}

	element := NewElement(field)
	if !element.IsValid() {
		if field.Enum() != nil {
			element.AsMultiSelect(func(selec *MultiSelectElement) {
				rules := element.rules.GetEnum()
				if element.rules.GetRepeated() != nil && element.rules.GetRepeated().GetItems() != nil {
					element.rules.GetRepeated().HasUnique()
					rules = element.rules.GetRepeated().GetItems().GetEnum()
				}

				for idx := range field.Enum().Values().Len() {
					desc := field.Enum().Values().Get(idx)
					if rules != nil && len(rules.In) > 0 && !slices.Contains(rules.In, int32(desc.Number())) {
						continue
					}

					if rules != nil && len(rules.NotIn) > 0 && slices.Contains(rules.NotIn, int32(desc.Number())) {
						continue
					}

					selec.options = append(selec.options, util.NewMapPair(string(desc.Name()), any(protoreflect.ValueOfEnum(desc.Number()))))

				}
			})
		}
	}

	return &ListField{
		ListType: list,
		element:  element,
	}, true
}

func NewList(parent *Message, field protoreflect.FieldDescriptor) (ListType, bool) {
	if parent == nil || field == nil || !field.IsList() {
		return nil, false
	}

	// Check if parent message is nil
	parentMsg := parent.Get()
	if parentMsg == nil {
		return nil, false
	}

	list := parentMsg.NewField(field).List()
	if field.Message() != nil {
		return &List[*Message]{
			Getter: ListGetter(parent.Get(), field, func(v protoreflect.Value) *Message {
				return NewMessage(v.Message())
			}),
			Setter: ListSetter(parent.Get(), field, func(m *Message) protoreflect.Value { return protoreflect.ValueOfMessage(m.Get()) }),
			Parser: ParserFunc[[]*Message](func(s string) ([]*Message, error) {
				var rawMsgs []json.RawMessage
				err := encoding.FromJSONV2([]byte(s), &rawMsgs)
				if err != nil {
					return nil, err
				}

				list_ := make([]*Message, len(rawMsgs))
				for idx, rawMsg := range rawMsgs {
					m := list.NewElement().Message().New().Interface()
					err = encoding.FromJSONV2(rawMsg, m)
					if err != nil {
						return nil, err
					}

					list_[idx] = NewMessage(m.ProtoReflect())
				}

				return list_, nil
			}),
			Stringer: StringerFunc[[]*Message](func(v []*Message) string {
				msgs := util.SliceMap(v, func(m *Message) proto.Message {
					return m.Get().Interface()
				})
				b, _ := encoding.ToJSONV2(msgs)
				return string(b)
			}),
			newItem: func() *Message {
				return NewMessage(list.NewElement().Message().New())
			},
			parent:     parent,
			descriptor: field,
		}, true
	} else {
		switch field.Kind() {
		case protoreflect.BoolKind:
			return &List[bool]{
				Getter:     ListGetter(parent.Get(), field, func(v protoreflect.Value) bool { return v.Bool() }),
				Setter:     ListSetter(parent.Get(), field, protoreflect.ValueOfBool),
				parent:     parent,
				descriptor: field,
			}, true
		case protoreflect.BytesKind:
			return &List[[]byte]{
				Getter: ListGetter(parent.Get(), field, func(v protoreflect.Value) []byte { return v.Bytes() }),
				Setter: ListSetter(parent.Get(), field, protoreflect.ValueOfBytes),
				Parser: ParserFunc[[][]byte](func(s string) ([][]byte, error) {
					var strings []string
					err := encoding.FromJSONV2([]byte(s), &strings)
					if err != nil {
						return nil, err
					}
					return util.SliceMap(strings, func(s string) []byte { return []byte(s) }), nil
				}),
				Stringer: StringerFunc[[][]byte](func(bytes [][]byte) string {
					strings := util.SliceMap(bytes, func(b []byte) string {
						return string(b)
					})
					b, _ := encoding.ToJSONV2(strings)
					return string(b)
				}),
				parent:     parent,
				descriptor: field,
			}, true
		case protoreflect.EnumKind:
			return &List[protoreflect.EnumNumber]{
				Getter:     ListGetter(parent.Get(), field, func(v protoreflect.Value) protoreflect.EnumNumber { return v.Enum() }),
				Setter:     ListSetter(parent.Get(), field, protoreflect.ValueOfEnum),
				parent:     parent,
				descriptor: field,
			}, true
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			return &List[int32]{
				Getter:     ListGetter(parent.Get(), field, func(v protoreflect.Value) int32 { return int32(v.Int()) }),
				Setter:     ListSetter(parent.Get(), field, protoreflect.ValueOfInt32),
				parent:     parent,
				descriptor: field,
			}, true
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			return &List[int64]{
				Getter:     ListGetter(parent.Get(), field, func(v protoreflect.Value) int64 { return v.Int() }),
				Setter:     ListSetter(parent.Get(), field, protoreflect.ValueOfInt64),
				parent:     parent,
				descriptor: field,
			}, true
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			return &List[uint32]{
				Getter:     ListGetter(parent.Get(), field, func(v protoreflect.Value) uint32 { return uint32(v.Uint()) }),
				Setter:     ListSetter(parent.Get(), field, protoreflect.ValueOfUint32),
				parent:     parent,
				descriptor: field,
			}, true
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			return &List[uint64]{
				Getter:     ListGetter(parent.Get(), field, func(v protoreflect.Value) uint64 { return uint64(v.Uint()) }),
				Setter:     ListSetter(parent.Get(), field, protoreflect.ValueOfUint64),
				parent:     parent,
				descriptor: field,
			}, true
		case protoreflect.FloatKind:
			return &List[float32]{
				Getter:     ListGetter(parent.Get(), field, func(v protoreflect.Value) float32 { return float32(v.Float()) }),
				Setter:     ListSetter(parent.Get(), field, protoreflect.ValueOfFloat32),
				parent:     parent,
				descriptor: field,
			}, true
		case protoreflect.DoubleKind:
			return &List[float64]{
				Getter:     ListGetter(parent.Get(), field, func(v protoreflect.Value) float64 { return v.Float() }),
				Setter:     ListSetter(parent.Get(), field, protoreflect.ValueOfFloat64),
				parent:     parent,
				descriptor: field,
			}, true
		case protoreflect.StringKind:
			return &List[string]{
				Getter:     ListGetter(parent.Get(), field, func(v protoreflect.Value) string { return v.String() }),
				Setter:     ListSetter(parent.Get(), field, protoreflect.ValueOfString),
				parent:     parent,
				descriptor: field,
			}, true
		}
	}

	return nil, false
}

func ListGetter[T ListValue](parent protoreflect.Message, field protoreflect.FieldDescriptor, get func(protoreflect.Value) T) Getter[[]T] {
	return GetterFunc[[]T](func() (v []T) {
		l := parent.Get(field).List()
		if l.Len() > 0 {
			for idx := range l.Len() {
				v = append(v, get(l.Get(idx)))
			}
		}
		return
	})
}

func ListSetter[T ListValue](parent protoreflect.Message, field protoreflect.FieldDescriptor, set func(T) protoreflect.Value) Setter[[]T] {
	return SetterFunc[[]T](func(s []T) {
		l := parent.Get(field).List()
		if !l.IsValid() {
			l = parent.NewField(field).List()
		}
		l.Truncate(0)
		for _, v := range s {
			l.Append(set(v))
		}
		parent.Set(field, protoreflect.ValueOfList(l))
	})
}

func (l *ListField) isFieldType() {}

func (l *ListField) IsMessage() (*List[*Message], bool) {
	m, ok := l.ListType.(*List[*Message])
	return m, ok
}

func (l *ListField) IsEnum() (*List[protoreflect.EnumNumber], bool) {
	m, ok := l.ListType.(*List[protoreflect.EnumNumber])
	return m, ok
}

func (l *ListField) Element() *Element {
	return l.element
}

func (l *List[T]) Parent() *Message {
	return l.parent
}

func (l *List[T]) Descriptor() protoreflect.FieldDescriptor {
	return l.descriptor
}

func (l *List[T]) NewItemAny() any {
	return l.NewItem()
}

func (l *List[T]) NewItem() (v T) {
	if l.newItem != nil {
		return l.newItem()
	}
	return
}

func (l *List[T]) GetItemAny(idx int) (any, error) {
	return l.GetItem(idx)
}

func (l *List[T]) GetItem(idx int) (v T, err error) {
	if idx < 0 || idx > len(l.Get())-1 {
		err = fmt.Errorf("index out of range: %d", idx)
		return
	}
	return l.Get()[idx], nil
}

func (l *List[T]) Append(item T) {
	l.Set(append(l.Get(), item))
}

func (l *List[T]) AppendAny(item any) error {
	tItem, ok := item.(T)
	if !ok {
		return fmt.Errorf("item invalid, expected: %T, got: %T", tItem, item)
	}
	l.Append(tItem)
	return nil
}

func (l *List[T]) RemoveItem(idx int) error {
	if idx < 0 || idx > l.Len()-1 {
		return fmt.Errorf("list index out of range: %d", idx)
	}

	var items []T
	for _idx, item := range l.Get() {
		if idx != _idx {
			items = append(items, item)
		}
	}

	if len(items) > 0 {
		l.Set(items)
	} else if l.parent.Get().Get(l.descriptor).IsValid() {
		l.parent.Get().Get(l.descriptor).List().Truncate(0)
	}

	return nil
}

func (l *List[T]) Len() (v int) {
	if l.parent.Get().Get(l.descriptor).IsValid() {
		v = l.parent.Get().Get(l.descriptor).List().Len()
	}
	return
}

func (l *List[T]) GetItems() []any {
	return util.AnySlice(l.Get())
}

func (l *List[T]) Get() (v []T) {
	if l.Getter != nil {
		return l.Getter.Get()
	}
	return
}

func (l *List[T]) Set(v []T) {
	if l.Setter != nil {
		l.Setter.Set(v)
	}
	l.parent.Set(l.parent.Get())
}

func (l *List[T]) SetItems(vals []any) error {
	var tVals []T
	for idx, val := range vals {
		tVal, ok := val.(T)
		if ok {
			tVals = append(tVals, tVal)
		} else {
			return fmt.Errorf("item %d invalid, expected: %T, got: %T", idx, tVal, val)
		}
	}
	l.Set(tVals)
	return nil
}

func (l *List[T]) ParseItems(s string) ([]any, error) {
	v, err := l.Parse(s)
	if err != nil {
		return nil, err
	}

	return util.AnySlice(v), nil
}

func (l *List[T]) Parse(s string) (v []T, err error) {
	s = strings.TrimSpace(s)

	if l.Parser != nil {
		return l.Parser.Parse(s)
	}

	err = encoding.FromJSONV2([]byte(s), &v)
	return
}

func (l *List[T]) StringOf(v []T) string {
	if l.Stringer != nil {
		return l.Stringer.StringOf(v)
	}
	b, _ := encoding.ToJSONV2(v)
	return string(b)
}

func (l *List[T]) String() string {
	return l.StringOf(l.Get())
}
