package v1beta1

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

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

		mut sync.Mutex
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
		Validate(Env) error
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
	if parent == nil || field == nil || !field.IsMap() || field.MapKey() == nil || field.MapValue() == nil {
		return nil, false
	}

	var m MapType
	switch field.MapKey().Kind() {
	case protoreflect.BoolKind:
		m = newMapOf(parent, field,
			func(s string) (b bool) {
				b, _ = strconv.ParseBool(s)
				return
			},
			func(b bool) string { return strconv.FormatBool(b) },
			func(v protoreflect.MapKey) bool { return v.Bool() },
			protoreflect.ValueOfBool,
		)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		m = newMapOf(parent, field,
			func(s string) int32 {
				i, _ := util.ParseInt32(s)
				return i
			},
			func(i int32) string { return util.FormatInt32(i) },
			func(v protoreflect.MapKey) int32 { return int32(v.Int()) },
			protoreflect.ValueOfInt32,
		)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		m = newMapOf(parent, field,
			func(s string) int64 {
				i, _ := util.ParseInt64(s)
				return i
			},
			func(i int64) string { return util.FormatInt64(i) },
			func(v protoreflect.MapKey) int64 { return v.Int() },
			protoreflect.ValueOfInt64,
		)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		m = newMapOf(parent, field,
			func(s string) uint32 {
				i, _ := util.ParseUInt32(s)
				return i
			},
			func(i uint32) string { return util.FormatUInt32(i) },
			func(v protoreflect.MapKey) uint32 { return uint32(v.Uint()) },
			protoreflect.ValueOfUint32,
		)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		m = newMapOf(parent, field,
			func(s string) uint64 {
				i, _ := util.ParseUInt64(s)
				return i
			},
			func(i uint64) string { return util.FormatUInt64(i) },
			func(v protoreflect.MapKey) uint64 { return v.Uint() },
			protoreflect.ValueOfUint64,
		)
	case protoreflect.StringKind:
		m = newMapOf(parent, field,
			func(s string) string { return s },
			func(s string) string { return s },
			func(v protoreflect.MapKey) string { return v.String() },
			protoreflect.ValueOfString,
		)
	}

	if m == nil {
		return nil, false
	}

	return &MapField{
		MapType: m,
		element: NewElement(field),
	}, true
}

func newMapOf[K MapKey](parent *Message, field protoreflect.FieldDescriptor, keyParser func(string) K, keyStringer func(K) string, getKey func(protoreflect.MapKey) K, setKey func(K) protoreflect.Value) MapType {
	if field.MapValue().Message() != nil {
		return &Map[K, *Message]{
			Getter:      mapGetter(parent, field, getKey, func(v protoreflect.Value) *Message { return NewMapMessage(parent, field, v.Message()) }),
			Setter:      mapSetter(parent, field, setKey, func(m *Message) protoreflect.Value { return protoreflect.ValueOfMessage(m.Get()) }),
			Stringer:    messageMapStringer(keyStringer),
			parent:      parent,
			descriptor:  field,
			keyParser:   keyParser,
			keyStringer: keyStringer,
			newValue: func() *Message {
				return NewMapMessage(parent, field, parent.Get().NewField(field).Map().NewValue().Message())
			},
		}
	} else {
		switch field.MapValue().Kind() {
		case protoreflect.BoolKind:
			return &Map[K, bool]{
				Getter:      mapGetter(parent, field, getKey, func(v protoreflect.Value) bool { return v.Bool() }),
				Setter:      mapSetter(parent, field, setKey, protoreflect.ValueOfBool),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v bool) { return },
			}
		case protoreflect.BytesKind:
			return &Map[K, []byte]{
				Getter:      mapGetter(parent, field, getKey, func(v protoreflect.Value) []byte { return v.Bytes() }),
				Setter:      mapSetter(parent, field, setKey, protoreflect.ValueOfBytes),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v []byte) { return },
			}
		case protoreflect.EnumKind:
			return &Map[K, protoreflect.EnumNumber]{
				Getter:      mapGetter(parent, field, getKey, func(v protoreflect.Value) protoreflect.EnumNumber { return v.Enum() }),
				Setter:      mapSetter(parent, field, setKey, protoreflect.ValueOfEnum),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v protoreflect.EnumNumber) { return },
			}
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			return &Map[K, int32]{
				Getter:      mapGetter(parent, field, getKey, func(v protoreflect.Value) int32 { return int32(v.Int()) }),
				Setter:      mapSetter(parent, field, setKey, protoreflect.ValueOfInt32),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v int32) { return },
			}
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			return &Map[K, int64]{
				Getter:      mapGetter(parent, field, getKey, func(v protoreflect.Value) int64 { return v.Int() }),
				Setter:      mapSetter(parent, field, setKey, protoreflect.ValueOfInt64),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v int64) { return },
			}
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			return &Map[K, uint32]{
				Getter:      mapGetter(parent, field, getKey, func(v protoreflect.Value) uint32 { return uint32(v.Uint()) }),
				Setter:      mapSetter(parent, field, setKey, protoreflect.ValueOfUint32),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v uint32) { return },
			}
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			return &Map[K, uint64]{
				Getter:      mapGetter(parent, field, getKey, func(v protoreflect.Value) uint64 { return uint64(v.Uint()) }),
				Setter:      mapSetter(parent, field, setKey, protoreflect.ValueOfUint64),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v uint64) { return },
			}
		case protoreflect.FloatKind:
			return &Map[K, float32]{
				Getter:      mapGetter(parent, field, getKey, func(v protoreflect.Value) float32 { return float32(v.Float()) }),
				Setter:      mapSetter(parent, field, setKey, protoreflect.ValueOfFloat32),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v float32) { return },
			}
		case protoreflect.DoubleKind:
			return &Map[K, float64]{
				Getter:      mapGetter(parent, field, getKey, func(v protoreflect.Value) float64 { return v.Float() }),
				Setter:      mapSetter(parent, field, setKey, protoreflect.ValueOfFloat64),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v float64) { return },
			}
		case protoreflect.StringKind:
			return &Map[K, string]{
				Getter:      mapGetter(parent, field, getKey, func(v protoreflect.Value) string { return v.String() }),
				Setter:      mapSetter(parent, field, setKey, protoreflect.ValueOfString),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v string) { return },
			}
		}
	}

	return nil
}

func mapGetter[K MapKey, V MapValue](parent *Message, field protoreflect.FieldDescriptor, getKey func(protoreflect.MapKey) K, getVal func(protoreflect.Value) V) Getter[map[K]V] {
	return GetterFunc[map[K]V](func() (nMap map[K]V) {
		pMap := parent.GetField(field).Map()
		nMap = map[K]V{}
		if pMap.Len() > 0 {
			pMap.Range(func(key protoreflect.MapKey, val protoreflect.Value) bool {
				nMap[getKey(key)] = getVal(val)
				return true
			})
		}
		return
	})
}

func mapSetter[K MapKey, V MapValue](parent *Message, field protoreflect.FieldDescriptor, setKey func(K) protoreflect.Value, setVal func(V) protoreflect.Value) Setter[map[K]V] {
	return SetterFunc[map[K]V](func(nmap map[K]V) {
		if len(nmap) > 0 {
			pMap := parent.GetField(field).Map()

			parent.mut.Lock()
			for k, v := range nmap {
				pMap.Set(setKey(k).MapKey(), setVal(v))
			}
			parent.mut.Unlock()

			parent.SetField(field, protoreflect.ValueOfMap(pMap))
		}
	})
}

func messageMapParser[K MapKey](parseKey func(string) K, newMsg func() protoreflect.Message) Parser[map[K]*Message] {
	return ParserFunc[map[K]*Message](func(s string) (map[K]*Message, error) {
		var rawMap map[string]json.RawMessage
		err := encoding.FromJSONV2([]byte(s), &rawMap)
		if err != nil {
			return nil, err
		}

		nMap := map[K]*Message{}
		for key, rawMsg := range rawMap {
			msg := newMsg()
			err = encoding.FromJSONV2(rawMsg, msg.Interface())
			if err != nil {
				return nil, err
			}

			nMap[parseKey(key)] = NewMessage(msg)
		}

		return nMap, nil
	})
}

func messageMapStringer[K MapKey](keyFunc func(K) string) Stringer[map[K]*Message] {
	return StringerFunc[map[K]*Message](func(m map[K]*Message) string {
		nMap := map[string]proto.Message{}
		for k, m := range m {
			nMap[keyFunc(k)] = m.Get().Interface()
		}
		b, _ := encoding.ToJSONV2(nMap)
		return string(b)
	})
}

func (f *MapField) Element() *Element {
	return f.element
}

func (m *Map[K, V]) Validate(env Env) error {
	pmap := m.parent.Get().NewField(m.descriptor).Map()
	mm, isMsg := m.isMessage()
	if isMsg {
		mm.rangeKV(func(k K, m *Message) bool {
			pmap.Set(protoreflect.MapKey(protoreflect.ValueOf(k)), protoreflect.ValueOfMessage(m.value))
			return true
		})
	} else {
		m.rangeKV(func(k K, v V) bool {
			pmap.Set(protoreflect.MapKey(protoreflect.ValueOf(k)), protoreflect.ValueOf(v))
			return true
		})
	}

	return m.parent.ValidateField(env, m.Descriptor(), protoreflect.ValueOfMap(pmap))
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
	return m.newValue()
}

func (m *Map[K, V]) NewKey() (k K) {
	return
}

func (m *Map[K, V]) NewVal() V {
	return m.newValue()
}

func (m *Map[K, V]) GetAnyPair(k string) (any, any, bool) {
	key := m.keyParser(k)
	val, ok := m.GetValue(key)
	return key, val, ok
}

func (m *Map[K, V]) GetValue(k K) (V, bool) {
	entries := m.Get()
	val, ok := entries[k]
	return val, ok
}

func (m *Map[K, V]) SetAnyPair(k, v any) error {
	val, ok := v.(V)
	if !ok {
		key := util.StringFormatAny(k)
		_, isMsg := m.isMessage()
		if isMsg {
			return fmt.Errorf("map invalid value for key %q: %v, expected: %T", key, v, v.(*Message).Descriptor().FullName())
		}
		return fmt.Errorf("map invalid value for key %q: %v, expected: %T", key, v, val)
	}

	m.SetPair(m.keyParser(util.StringFormatAny(k)), val)
	return nil
}

func (m *Map[K, V]) SetPair(k K, v V) {
	pMap := m.Get()
	pMap[k] = v
	m.Set(pMap)
}

func (m *Map[K, V]) RemoveKey(k string) {
	m.Remove(m.keyParser(k))
}

func (m *Map[K, V]) Remove(k K) {
	m.mut.Lock()
	defer m.mut.Unlock()

	m.parent.GetField(m.descriptor).Map().Clear(protoreflect.ValueOf(k).MapKey())
}

func (m *Map[K, V]) Get() map[K]V {
	m.mut.Lock()
	defer m.mut.Unlock()

	if m.Getter != nil {
		return m.Getter.Get()
	}

	return map[K]V{}
}

func (m *Map[K, V]) Set(v map[K]V) {
	m.mut.Lock()
	defer m.mut.Unlock()

	if m.Setter != nil {
		m.Setter.Set(v)
	}
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

func (m *Map[K, V]) Len() int {
	return len(m.Get())
}

func (m *Map[K, V]) Range(f func(string, any) bool) {
	for k, v := range m.Get() {
		if !f(m.keyStringer(k), v) {
			return
		}
	}
}

func (m *Map[K, V]) rangeKV(f func(K, V) bool) {
	for k, v := range m.Get() {
		if !f(k, v) {
			return
		}
	}
}

func (m *Map[K, V]) Parse(s string) (nMap map[K]V, err error) {
	if m.Parser != nil {
		return m.Parser.Parse(s)
	}

	var rawMap map[K]json.RawMessage
	err = encoding.FromJSONV2([]byte(s), &rawMap)
	if err != nil {
		return
	}

	if len(rawMap) > 0 {
		nMap = map[K]V{}
		for key, rawVal := range rawMap {
			var val V
			if v, ok := any(val).(*Message); ok {
				v = any(m.newValue()).(*Message)
				msg := v.Get().Interface()
				err = encoding.FromJSONV2(rawVal, msg)
				if err != nil {
					return
				}
			} else {
				err = encoding.FromJSONV2(rawVal, &val)
				if err != nil {
					return
				}
			}

			nMap[key] = val
			// nMap[m.keyParser(key)] = val
		}
	}

	// if m.isMessage() {
	// 	mMap, err := messageMapParser(m.keyParser, func() protoreflect.Message {
	// 		return m.NewValAny().(*Message).Get()
	// 	}).Parse(s)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	return any(mMap).(map[K]V), nil
	// }

	// nMap = map[K]V{}
	// for key, rawVal := range rawMap {
	// 	var val V
	// 	err = encoding.FromJSONV2(rawVal, &val)
	// 	if err != nil {
	// 		return
	// 	}
	// 	nMap[m.keyParser(key)] = val
	// }

	return
}

func (m *Map[K, V]) isMessage() (mm *Map[K, *Message], ok bool) {
	mm, ok = any(m).(*Map[K, *Message])
	return
}

func (m *Map[K, V]) StringOf(v map[K]V) string {
	if m.Stringer != nil {
		return m.Stringer.StringOf(v)
	}

	nMap := map[string]V{}
	for k, v := range m.Get() {
		nMap[m.keyStringer(k)] = v
	}

	b, _ := encoding.ToJSONV2(nMap)
	return string(b)
}

func (m *Map[K, V]) String() string {
	return m.StringOf(m.Get())
}

func (f *MapField) isFieldType() {}
