package protoform

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ Binding[[]bool] = (*ListField[bool])(nil)
var _ ListFieldType = (*ListField[string])(nil)
var _ Field = (*ListField[string])(nil)

type (
	ListField[T ListValueType] struct {
		Getter[[]T]
		Setter[[]T]
		Parser[[]T]
		Stringer[[]T]
		parent     *Message
		descriptor protoreflect.FieldDescriptor
		mut        sync.Mutex
	}
	ListFieldType interface {
		Field
		AppendAny(any) error
		NewItemAny() any
		GetItemAny(int) (any, error)
		RemoveItem(int) error
		GetItemsAny() []any
		SetItemsAny([]any) error
		ParseItemsAny(string) ([]any, error)
		Len() int
		String() string
		isList()
	}
	ListValueType interface {
		ScalarValueType | *Message
	}
)

func NewListField(parent *Message, field protoreflect.FieldDescriptor) (ListFieldType, bool) {
	if parent == nil || field == nil || !field.IsList() {
		return nil, false
	}

	var list ListFieldType
	if field.Message() != nil {
		list = &ListField[*Message]{
			Getter: listGetter(parent, field, func(v protoreflect.Value) *Message {
				return newListMessage(parent, v.Message())
			}),
			Setter: listSetter(parent, field, func(m *Message) protoreflect.Value { return protoreflect.ValueOfMessage(m.Get()) }),
			Parser: ParserFunc[[]*Message](func(s string) ([]*Message, error) {
				var rawMsgs []json.RawMessage
				err := encoding.FromJSON([]byte(s), &rawMsgs)
				if err != nil {
					return nil, err
				}

				list := parent.GetField(field).List()
				items := make([]*Message, len(rawMsgs))
				for idx, rawMsg := range rawMsgs {
					msg := list.NewElement().Message().New()
					err = encoding.FromJSON(rawMsg, msg.Interface())
					if err != nil {
						return nil, err
					}

					items[idx] = newListMessage(parent, msg)
				}

				return items, nil
			}),
			Stringer: StringerFunc[[]*Message](func(v []*Message) string {
				msgs := util.SliceOfFunc(v, func(m *Message) proto.Message {
					return m.Get().Interface()
				})
				b, _ := encoding.ToJSON(msgs)
				return string(b)
			}),
			parent:     parent,
			descriptor: field,
		}
	} else {
		switch field.Kind() {
		case protoreflect.BoolKind:
			list = &ListField[bool]{
				Getter:     listGetter(parent, field, func(v protoreflect.Value) bool { return v.Bool() }),
				Setter:     listSetter(parent, field, protoreflect.ValueOfBool),
				parent:     parent,
				descriptor: field,
			}
		case protoreflect.BytesKind:
			list = &ListField[[]byte]{
				Getter: listGetter(parent, field, func(v protoreflect.Value) []byte { return v.Bytes() }),
				Setter: listSetter(parent, field, protoreflect.ValueOfBytes),
				Parser: ParserFunc[[][]byte](func(s string) ([][]byte, error) {
					var strings []string
					err := encoding.FromJSON([]byte(s), &strings)
					if err != nil {
						return nil, err
					}
					return util.SliceOfFunc(strings, func(s string) []byte { return []byte(s) }), nil
				}),
				Stringer: StringerFunc[[][]byte](func(bytes [][]byte) string {
					strings := util.SliceOfFunc(bytes, func(b []byte) string {
						return string(b)
					})
					b, _ := encoding.ToJSON(strings)
					return string(b)
				}),
				parent:     parent,
				descriptor: field,
			}
		case protoreflect.EnumKind:
			list = &ListField[protoreflect.EnumNumber]{
				Getter:     listGetter(parent, field, func(v protoreflect.Value) protoreflect.EnumNumber { return v.Enum() }),
				Setter:     listSetter(parent, field, protoreflect.ValueOfEnum),
				parent:     parent,
				descriptor: field,
			}
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			list = &ListField[int32]{
				Getter:     listGetter(parent, field, func(v protoreflect.Value) int32 { return int32(v.Int()) }),
				Setter:     listSetter(parent, field, protoreflect.ValueOfInt32),
				parent:     parent,
				descriptor: field,
			}
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			list = &ListField[int64]{
				Getter:     listGetter(parent, field, func(v protoreflect.Value) int64 { return v.Int() }),
				Setter:     listSetter(parent, field, protoreflect.ValueOfInt64),
				parent:     parent,
				descriptor: field,
			}
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			list = &ListField[uint32]{
				Getter:     listGetter(parent, field, func(v protoreflect.Value) uint32 { return uint32(v.Uint()) }),
				Setter:     listSetter(parent, field, protoreflect.ValueOfUint32),
				parent:     parent,
				descriptor: field,
			}
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			list = &ListField[uint64]{
				Getter:     listGetter(parent, field, func(v protoreflect.Value) uint64 { return uint64(v.Uint()) }),
				Setter:     listSetter(parent, field, protoreflect.ValueOfUint64),
				parent:     parent,
				descriptor: field,
			}
		case protoreflect.FloatKind:
			list = &ListField[float32]{
				Getter:     listGetter(parent, field, func(v protoreflect.Value) float32 { return float32(v.Float()) }),
				Setter:     listSetter(parent, field, protoreflect.ValueOfFloat32),
				parent:     parent,
				descriptor: field,
			}
		case protoreflect.DoubleKind:
			list = &ListField[float64]{
				Getter:     listGetter(parent, field, func(v protoreflect.Value) float64 { return v.Float() }),
				Setter:     listSetter(parent, field, protoreflect.ValueOfFloat64),
				parent:     parent,
				descriptor: field,
			}
		case protoreflect.StringKind:
			list = &ListField[string]{
				Getter:     listGetter(parent, field, func(v protoreflect.Value) string { return v.String() }),
				Setter:     listSetter(parent, field, protoreflect.ValueOfString),
				parent:     parent,
				descriptor: field,
			}
		}
	}

	if list == nil {
		return nil, false
	}

	return list, true
}

func newListMessage(parent *Message, value protoreflect.Message) *Message {
	return &Message{
		value: value,
		depth: parent.depth + 1,
	}
}

func listGetter[T ListValueType](parent *Message, field protoreflect.FieldDescriptor, fromValue func(protoreflect.Value) T) Getter[[]T] {
	return GetterFunc[[]T](func() (v []T) {
		list := parent.GetField(field).List()

		parent.mut.Lock()
		size := list.Len()
		if size > 0 {
			for idx := range size {
				v = append(v, fromValue(list.Get(idx)))
			}
		}
		parent.mut.Unlock()

		return
	})
}

func listSetter[T ListValueType](parent *Message, field protoreflect.FieldDescriptor, toValue func(T) protoreflect.Value) Setter[[]T] {
	return SetterFunc[[]T](func(s []T) {
		list := parent.GetField(field).List()

		parent.mut.Lock()
		list.Truncate(0)
		for _, v := range s {
			list.Append(toValue(v))
		}
		parent.mut.Unlock()

		parent.SetField(field, protoreflect.ValueOfList(list))
	})
}

func (f *ListField[T]) IsMessage() (*ListField[*Message], bool) {
	m, ok := any(f).(*ListField[*Message])
	return m, ok
}

func (f *ListField[T]) IsEnum() (*ListField[protoreflect.EnumNumber], bool) {
	m, ok := any(f).(*ListField[protoreflect.EnumNumber])
	return m, ok
}

func (f *ListField[T]) IsScalar[S ScalarValueType]() (*ListField[S], bool) {
	m, ok := any(f).(*ListField[S])
	return m, ok
}

func (f *ListField[T]) Validate(env Env) error {
	list := f.parent.Get().NewField(f.descriptor).List()
	for _, v := range f.GetItemsAny() {
		if msg, ok := v.(*Message); ok {
			list.Append(protoreflect.ValueOfMessage(msg.value))
		} else {
			list.Append(protoreflect.ValueOf(v))
		}
	}
	return f.parent.ValidateField(env, f.Descriptor(), protoreflect.ValueOfList(list))
}

func (l *ListField[T]) Parent() *Message {
	return l.parent
}

func (l *ListField[T]) Descriptor() protoreflect.FieldDescriptor {
	return l.descriptor
}

func (l *ListField[T]) NewItemAny() any {
	return l.NewItem()
}

func (l *ListField[T]) GetItemAny(idx int) (any, error) {
	return l.GetItem(idx)
}

func (l *ListField[T]) GetItemsAny() []any {
	return util.SliceOfAny(l.Get())
}

func (l *ListField[T]) AppendAny(item any) error {
	tItem, ok := item.(T)
	if !ok {
		return fmt.Errorf("item invalid, expected: %T, got: %T", tItem, item)
	}
	l.Append(tItem)
	return nil
}

func (l *ListField[T]) SetItemsAny(vals []any) error {
	var items []T
	for idx, val := range vals {
		item, ok := val.(T)
		if !ok {
			if msg, ok := any(val).(*Message); ok {
				return fmt.Errorf("list item %d invalid, expected: %T, got: %T", idx, msg.Descriptor().FullName(), val)
			}

			return fmt.Errorf("list item %d invalid, expected: %T, got: %T", idx, item, val)
		}

		items = append(items, item)
	}
	l.Set(items)
	return nil
}

func (l *ListField[T]) ParseItemsAny(s string) ([]any, error) {
	items, err := l.Parse(s)
	if err != nil {
		return nil, err
	}
	return util.SliceOfAny(items), nil
}

func (l *ListField[T]) NewItem() (v T) {
	if msg, ok := any(v).(*Message); ok {
		msg = newListMessage(l.parent, l.parent.GetField(l.descriptor).List().NewElement().Message())
		v = any(msg).(T)
	}
	return
}

func (l *ListField[T]) GetItem(idx int) (v T, err error) {
	list := l.Get()
	if idx < 0 || idx > len(list)-1 {
		err = fmt.Errorf("index out of range: %d", idx)
		return
	}
	return list[idx], nil
}

func (l *ListField[T]) Append(item T) {
	l.Set(append(l.Get(), item))
}

func (l *ListField[T]) RemoveItem(idx int) error {
	list := l.Get()
	if idx < 0 || idx > len(list)-1 {
		return fmt.Errorf("list index out of range: %d", idx)
	}

	var items []T
	for i, item := range list {
		if idx != i {
			items = append(items, item)
		}
	}

	if len(items) == 0 && l.parent.GetField(l.descriptor).IsValid() {
		l.parent.GetField(l.descriptor).List().Truncate(0)
	} else {
		l.Set(items)
	}

	return nil
}

func (l *ListField[T]) Len() int {
	return len(l.Get())
}

func (l *ListField[T]) Get() (v []T) {
	l.mut.Lock()
	defer l.mut.Unlock()

	if l.Getter != nil {
		v = l.Getter.Get()
	}
	return
}

func (l *ListField[T]) Set(v []T) {
	l.mut.Lock()
	defer l.mut.Unlock()

	if l.Setter != nil {
		l.Setter.Set(v)
	}
}

func (l *ListField[T]) Parse(s string) (v []T, err error) {
	s = strings.TrimSpace(s)
	if l.Parser != nil {
		return l.Parser.Parse(s)
	}
	err = encoding.FromJSON([]byte(s), &v)
	return
}

func (l *ListField[T]) StringOf(v []T) string {
	if l.Stringer != nil {
		return l.Stringer.StringOf(v)
	}
	b, _ := encoding.ToJSON(v)
	return string(b)
}

func (l *ListField[T]) String() string {
	return l.StringOf(l.Get())
}

func (l *ListField[T]) isField() {}
func (l *ListField[T]) isList()  {}
