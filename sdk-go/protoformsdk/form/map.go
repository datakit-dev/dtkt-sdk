package form

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ FieldType = (*MapField)(nil)

type (
	MapField struct {
		MapType
		element *Element
	}
	Map[K MapKey, V MapValue] struct {
		Getter[map[K]V]
		Setter[map[K]V]
		Parser[map[K]V]
		Stringer[map[K]V]
		parent      *Message
		descriptor  protoreflect.FieldDescriptor
		keyParser   func(string) K
		keyStringer func(K) string
		newValue    func() V
	}
	MapType interface {
		Parent() *Message
		Descriptor() protoreflect.FieldDescriptor
		NewKeyAny() any
		NewValAny() any
		GetAnyPair(string) (any, any, bool)
		SetAnyPair(any, any) error
		RemoveKey(string)
		Keys() []string
		Vals() []any
		Range(func(string, any) bool)
		Len() int
		String() string
	}
	MapKey interface {
		bool | int32 | int64 | uint32 | uint64 | string
	}
	MapValue interface {
		ScalarValue | *Message
	}
)

func NewMapField(parent *Message, field protoreflect.FieldDescriptor) (*MapField, bool) {
	if parent == nil || field == nil || !field.IsMap() {
		return nil, false
	}

	m, ok := NewMap(parent, field)
	if !ok {
		return nil, false
	}

	return &MapField{
		MapType: m,
		element: NewElement(field),
	}, true
}

func NewMap(parent *Message, field protoreflect.FieldDescriptor) (MapType, bool) {
	if field == nil {
		return nil, false
	}

	switch field.MapKey().Kind() {
	case protoreflect.BoolKind:
		return NewMapWithKey(parent, field,
			func(s string) (b bool) {
				b, _ = strconv.ParseBool(s)
				return
			},
			func(b bool) string { return strconv.FormatBool(b) },
			func(v protoreflect.MapKey) bool { return v.Bool() },
			protoreflect.ValueOfBool,
		)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return NewMapWithKey(parent, field,
			func(s string) int32 {
				i, _ := util.ParseInt32(s)
				return i
			},
			func(i int32) string { return util.FormatInt32(i) },
			func(v protoreflect.MapKey) int32 { return int32(v.Int()) },
			protoreflect.ValueOfInt32,
		)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return NewMapWithKey(parent, field,
			func(s string) int64 {
				i, _ := util.ParseInt64(s)
				return i
			},
			func(i int64) string { return util.FormatInt64(i) },
			func(v protoreflect.MapKey) int64 { return v.Int() },
			protoreflect.ValueOfInt64,
		)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return NewMapWithKey(parent, field,
			func(s string) uint32 {
				i, _ := util.ParseUInt32(s)
				return i
			},
			func(i uint32) string { return util.FormatUInt32(i) },
			func(v protoreflect.MapKey) uint32 { return uint32(v.Uint()) },
			protoreflect.ValueOfUint32,
		)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return NewMapWithKey(parent, field,
			func(s string) uint64 {
				i, _ := util.ParseUInt64(s)
				return i
			},
			func(i uint64) string { return util.FormatUInt64(i) },
			func(v protoreflect.MapKey) uint64 { return v.Uint() },
			protoreflect.ValueOfUint64,
		)
	case protoreflect.StringKind:
		return NewMapWithKey(parent, field,
			func(s string) string { return s },
			func(s string) string { return s },
			func(v protoreflect.MapKey) string { return v.String() },
			protoreflect.ValueOfString,
		)
	}

	return nil, false
}

func NewMapWithKey[K MapKey](parent *Message, field protoreflect.FieldDescriptor, keyParser func(string) K, keyStringer func(K) string, getKey func(protoreflect.MapKey) K, setKey func(K) protoreflect.Value) (MapType, bool) {
	if field == nil {
		return nil, false
	}

	// Check if parent message is nil
	parentMsg := parent.Get()
	if parentMsg == nil {
		return nil, false
	}

	map_ := parentMsg.NewField(field).Map()
	if field.MapValue().Message() != nil {
		return &Map[K, *Message]{
			Getter: MapGetter(parentMsg, field, getKey, func(v protoreflect.Value) *Message {
				msg, _ := NewMessage(v.Message())
				return msg
			}),
			Setter:      MapSetter(parentMsg, field, setKey, func(m *Message) protoreflect.Value { return protoreflect.ValueOfMessage(m.Get()) }),
			Parser:      MapMessageParser(keyParser, parentMsg.NewField(field).Map().NewValue().Message),
			Stringer:    MapMessageStringer(keyStringer),
			parent:      parent,
			descriptor:  field,
			keyParser:   keyParser,
			keyStringer: keyStringer,
			newValue: func() *Message {
				msg, _ := NewMessage(map_.NewValue().Message().New())
				return msg
			},
		}, true
	} else {
		switch field.MapValue().Kind() {
		case protoreflect.BoolKind:
			return &Map[K, bool]{
				Getter:      MapGetter(parent.Get(), field, getKey, func(v protoreflect.Value) bool { return v.Bool() }),
				Setter:      MapSetter(parent.Get(), field, setKey, protoreflect.ValueOfBool),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
			}, true
		case protoreflect.BytesKind:
			return &Map[K, []byte]{
				Getter:      MapGetter(parent.Get(), field, getKey, func(v protoreflect.Value) []byte { return v.Bytes() }),
				Setter:      MapSetter(parent.Get(), field, setKey, protoreflect.ValueOfBytes),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
			}, true
		case protoreflect.EnumKind:
			return &Map[K, protoreflect.EnumNumber]{
				Getter:      MapGetter(parent.Get(), field, getKey, func(v protoreflect.Value) protoreflect.EnumNumber { return v.Enum() }),
				Setter:      MapSetter(parent.Get(), field, setKey, protoreflect.ValueOfEnum),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
			}, true
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			return &Map[K, int32]{
				Getter:      MapGetter(parent.Get(), field, getKey, func(v protoreflect.Value) int32 { return int32(v.Int()) }),
				Setter:      MapSetter(parent.Get(), field, setKey, protoreflect.ValueOfInt32),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
			}, true
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			return &Map[K, int64]{
				Getter:      MapGetter(parent.Get(), field, getKey, func(v protoreflect.Value) int64 { return v.Int() }),
				Setter:      MapSetter(parent.Get(), field, setKey, protoreflect.ValueOfInt64),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
			}, true
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			return &Map[K, uint32]{
				Getter:      MapGetter(parent.Get(), field, getKey, func(v protoreflect.Value) uint32 { return uint32(v.Uint()) }),
				Setter:      MapSetter(parent.Get(), field, setKey, protoreflect.ValueOfUint32),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
			}, true
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			return &Map[K, uint64]{
				Getter:      MapGetter(parent.Get(), field, getKey, func(v protoreflect.Value) uint64 { return uint64(v.Uint()) }),
				Setter:      MapSetter(parent.Get(), field, setKey, protoreflect.ValueOfUint64),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
			}, true
		case protoreflect.FloatKind:
			return &Map[K, float32]{
				Getter:      MapGetter(parent.Get(), field, getKey, func(v protoreflect.Value) float32 { return float32(v.Float()) }),
				Setter:      MapSetter(parent.Get(), field, setKey, protoreflect.ValueOfFloat32),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
			}, true
		case protoreflect.DoubleKind:
			return &Map[K, float64]{
				Getter:      MapGetter(parent.Get(), field, getKey, func(v protoreflect.Value) float64 { return v.Float() }),
				Setter:      MapSetter(parent.Get(), field, setKey, protoreflect.ValueOfFloat64),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
			}, true
		case protoreflect.StringKind:
			return &Map[K, string]{
				Getter:      MapGetter(parent.Get(), field, getKey, func(v protoreflect.Value) string { return v.String() }),
				Setter:      MapSetter(parent.Get(), field, setKey, protoreflect.ValueOfString),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
			}, true
		}
	}

	return nil, false
}

func MapGetter[K MapKey, V MapValue](parent protoreflect.Message, field protoreflect.FieldDescriptor, getKey func(protoreflect.MapKey) K, getVal func(protoreflect.Value) V) Getter[map[K]V] {
	return GetterFunc[map[K]V](func() (nativeMap map[K]V) {
		nativeMap = map[K]V{}
		reflectMap := parent.Get(field).Map()
		if reflectMap.Len() > 0 {
			reflectMap.Range(func(key protoreflect.MapKey, val protoreflect.Value) bool {
				nativeMap[getKey(key)] = getVal(val)
				return true
			})
		}
		return
	})
}

func MapSetter[K MapKey, V MapValue](parent protoreflect.Message, field protoreflect.FieldDescriptor, setKey func(K) protoreflect.Value, setVal func(V) protoreflect.Value) Setter[map[K]V] {
	return SetterFunc[map[K]V](func(nativeMap map[K]V) {
		if len(nativeMap) > 0 {
			reflectMap := parent.Get(field).Map()
			if !reflectMap.IsValid() {
				reflectMap = parent.NewField(field).Map()
			}
			for k, v := range nativeMap {
				reflectMap.Set(setKey(k).MapKey(), setVal(v))
			}
			parent.Set(field, protoreflect.ValueOfMap(reflectMap))
		}
	})
}

func MapMessageParser[K MapKey](parseKey func(string) K, newMsg func() protoreflect.Message) Parser[map[K]*Message] {
	return ParserFunc[map[K]*Message](func(s string) (map[K]*Message, error) {
		var rawMsgs map[string]json.RawMessage
		err := encoding.FromJSONV2([]byte(s), &rawMsgs)
		if err != nil {
			return nil, err
		}

		map_ := map[K]*Message{}
		for key, rawMsg := range rawMsgs {
			msg := newMsg()
			err = encoding.FromJSONV2(rawMsg, msg.Interface())
			if err != nil {
				return nil, err
			}

			m, _ := NewMessage(msg)
			map_[parseKey(key)] = m
		}

		return map_, nil
	})
}

func MapMessageStringer[K MapKey](stringKey func(K) string) Stringer[map[K]*Message] {
	return StringerFunc[map[K]*Message](func(m map[K]*Message) string {
		_m := map[string]proto.Message{}
		for k, m := range m {
			_m[stringKey(k)] = m.Get().Interface()
		}
		b, _ := encoding.ToJSONV2(_m)
		return string(b)
	})
}

func (f *MapField) isFieldType() {}

func (f *MapField) Element() *Element {
	return f.element
}

func (m *Map[K, V]) Parent() *Message {
	return m.parent
}

func (m *Map[K, V]) Descriptor() protoreflect.FieldDescriptor {
	return m.descriptor
}

func (m *Map[K, V]) NewKeyAny() any {
	return m.NewKey()
}

func (m *Map[K, V]) NewValAny() any {
	return m.NewVal()
}

func (m *Map[K, V]) NewKey() (k K) {
	return
}

func (m *Map[K, V]) NewVal() (v V) {
	if m.newValue != nil {
		return m.newValue()
	}
	return
}

func (m *Map[K, V]) GetAnyPair(k string) (any, any, bool) {
	key := m.keyParser(k)
	val, ok := m.GetValue(key)
	return key, val, ok
}

func (m *Map[K, V]) GetValue(k K) (V, bool) {
	val, ok := m.Get()[k]
	return val, ok
}

func (m *Map[K, V]) SetAnyPair(k, v any) error {
	key, ok := k.(K)
	if ok {
		if val, ok := v.(V); ok {
			m.SetPair(key, val)
			return nil
		} else {
			return fmt.Errorf("invalid map value: %v, expected: %T", v, val)
		}
	}
	return fmt.Errorf("invalid map key: %v, expected: %T", k, key)
}

func (m *Map[K, V]) SetPair(k K, v V) {
	map_ := m.Get()
	map_[k] = v
	m.Set(map_)
}

func (m *Map[K, V]) RemoveKey(k string) {
	m.Remove(m.keyParser(k))
}

func (m *Map[K, V]) Remove(k K) {
	if parentMsg := m.parent.Get(); parentMsg != nil {
		parentMsg.Get(m.descriptor).Map().Clear(protoreflect.ValueOf(k).MapKey())
	}
}

func (m *Map[K, V]) Get() (v map[K]V) {
	if m.Getter != nil {
		return m.Getter.Get()
	}
	return
}

func (m *Map[K, V]) Set(v map[K]V) {
	if m.Setter != nil {
		m.Setter.Set(v)
	}
	m.parent.Set(m.parent.Get())
}

func (m *Map[K, V]) Keys() (keys []string) {
	m.Range(func(k string, _ any) bool {
		keys = append(keys, k)
		return true
	})
	return
}

func (m *Map[K, V]) Vals() (vals []any) {
	m.Range(func(_ string, v any) bool {
		vals = append(vals, v)
		return true
	})
	return
}

func (m *Map[K, V]) Len() (l int) {
	if parentMsg := m.parent.Get(); parentMsg != nil && parentMsg.Get(m.descriptor).IsValid() {
		l = parentMsg.Get(m.descriptor).Map().Len()
	}
	return
}

func (m *Map[K, V]) Range(f func(string, any) bool) {
	for k, v := range m.Get() {
		if !f(m.keyStringer(k), v) {
			return
		}
	}
}

func (m *Map[K, V]) Parse(s string) (_m map[K]V, err error) {
	if m.Parser != nil {
		return m.Parser.Parse(s)
	}

	var rawMap map[string]json.RawMessage
	err = encoding.FromJSONV2([]byte(s), &rawMap)
	if err != nil {
		return
	}

	_m = map[K]V{}
	for key, val := range rawMap {
		var v V
		err = encoding.FromJSONV2(val, &v)
		if err != nil {
			return
		}
		_m[m.keyParser(key)] = v
	}

	return
}

func (m *Map[K, V]) StringOf(v map[K]V) string {
	if m.Stringer != nil {
		return m.Stringer.StringOf(v)
	}

	_m := map[string]V{}
	for k, v := range m.Get() {
		_m[m.keyStringer(k)] = v
	}
	b, _ := encoding.ToJSONV2(_m)
	return string(b)
}

func (m *Map[K, V]) String() string {
	return m.StringOf(m.Get())
}
