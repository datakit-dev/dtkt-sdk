package protoform

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

var _ Binding[map[string]string] = (*MapField[string, string])(nil)
var _ Field = (*MapField[string, string])(nil)
var _ MapFieldType = (*MapField[string, string])(nil)

type (
	MapField[K MapKeyType, V MapValueType] struct {
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
	MapFieldType interface {
		Field
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
		isMap()
	}
	MapKeyType interface {
		bool | int32 | int64 | uint32 | uint64 | string
	}
	MapValueType interface {
		ScalarValueType | *Message
	}
)

func NewMapField(parent *Message, field protoreflect.FieldDescriptor) (MapFieldType, bool) {
	if parent == nil || field == nil || !field.IsMap() || field.MapKey() == nil || field.MapValue() == nil {
		return nil, false
	}

	var m MapFieldType
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

	return m, true
}

func newMapOf[K MapKeyType](parent *Message, field protoreflect.FieldDescriptor, keyParser func(string) K, keyStringer func(K) string, getKey func(protoreflect.MapKey) K, setKey func(K) protoreflect.Value) MapFieldType {
	if field.MapValue().Message() != nil {
		return &MapField[K, *Message]{
			Getter: getMapValue(parent, field, getKey, func(v protoreflect.Value) *Message {
				return newMapMessage(parent, v.Message())
			}),
			Setter: setMapValue(parent, field, setKey, func(m *Message) protoreflect.Value {
				return protoreflect.ValueOfMessage(m.Get())
			}),
			Stringer:    messageMapStringer(keyStringer),
			parent:      parent,
			descriptor:  field,
			keyParser:   keyParser,
			keyStringer: keyStringer,
			newValue: func() *Message {
				return newMapMessage(parent, parent.Get().NewField(field).Map().NewValue().Message())
			},
		}
	} else {
		switch field.MapValue().Kind() {
		case protoreflect.BoolKind:
			return &MapField[K, bool]{
				Getter:      getMapValue(parent, field, getKey, func(v protoreflect.Value) bool { return v.Bool() }),
				Setter:      setMapValue(parent, field, setKey, protoreflect.ValueOfBool),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v bool) { return },
			}
		case protoreflect.BytesKind:
			return &MapField[K, []byte]{
				Getter:      getMapValue(parent, field, getKey, func(v protoreflect.Value) []byte { return v.Bytes() }),
				Setter:      setMapValue(parent, field, setKey, protoreflect.ValueOfBytes),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v []byte) { return },
			}
		case protoreflect.EnumKind:
			return &MapField[K, protoreflect.EnumNumber]{
				Getter:      getMapValue(parent, field, getKey, func(v protoreflect.Value) protoreflect.EnumNumber { return v.Enum() }),
				Setter:      setMapValue(parent, field, setKey, protoreflect.ValueOfEnum),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v protoreflect.EnumNumber) { return },
			}
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			return &MapField[K, int32]{
				Getter:      getMapValue(parent, field, getKey, func(v protoreflect.Value) int32 { return int32(v.Int()) }),
				Setter:      setMapValue(parent, field, setKey, protoreflect.ValueOfInt32),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v int32) { return },
			}
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			return &MapField[K, int64]{
				Getter:      getMapValue(parent, field, getKey, func(v protoreflect.Value) int64 { return v.Int() }),
				Setter:      setMapValue(parent, field, setKey, protoreflect.ValueOfInt64),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v int64) { return },
			}
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			return &MapField[K, uint32]{
				Getter:      getMapValue(parent, field, getKey, func(v protoreflect.Value) uint32 { return uint32(v.Uint()) }),
				Setter:      setMapValue(parent, field, setKey, protoreflect.ValueOfUint32),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v uint32) { return },
			}
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			return &MapField[K, uint64]{
				Getter:      getMapValue(parent, field, getKey, func(v protoreflect.Value) uint64 { return uint64(v.Uint()) }),
				Setter:      setMapValue(parent, field, setKey, protoreflect.ValueOfUint64),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v uint64) { return },
			}
		case protoreflect.FloatKind:
			return &MapField[K, float32]{
				Getter:      getMapValue(parent, field, getKey, func(v protoreflect.Value) float32 { return float32(v.Float()) }),
				Setter:      setMapValue(parent, field, setKey, protoreflect.ValueOfFloat32),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v float32) { return },
			}
		case protoreflect.DoubleKind:
			return &MapField[K, float64]{
				Getter:      getMapValue(parent, field, getKey, func(v protoreflect.Value) float64 { return v.Float() }),
				Setter:      setMapValue(parent, field, setKey, protoreflect.ValueOfFloat64),
				parent:      parent,
				descriptor:  field,
				keyParser:   keyParser,
				keyStringer: keyStringer,
				newValue:    func() (v float64) { return },
			}
		case protoreflect.StringKind:
			return &MapField[K, string]{
				Getter:      getMapValue(parent, field, getKey, func(v protoreflect.Value) string { return v.String() }),
				Setter:      setMapValue(parent, field, setKey, protoreflect.ValueOfString),
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

func newMapMessage(parent *Message, value protoreflect.Message) *Message {
	return &Message{
		value: value,
		depth: parent.depth + 1,
	}
}

func getMapValue[K MapKeyType, V MapValueType](parent *Message, field protoreflect.FieldDescriptor, getKey func(protoreflect.MapKey) K, getVal func(protoreflect.Value) V) Getter[map[K]V] {
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

func setMapValue[K MapKeyType, V MapValueType](parent *Message, field protoreflect.FieldDescriptor, setKey func(K) protoreflect.Value, setVal func(V) protoreflect.Value) Setter[map[K]V] {
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

func messageMapStringer[K MapKeyType](keyFunc func(K) string) Stringer[map[K]*Message] {
	return StringerFunc[map[K]*Message](func(m map[K]*Message) string {
		nMap := map[string]proto.Message{}
		for k, m := range m {
			nMap[keyFunc(k)] = m.Get().Interface()
		}
		b, _ := encoding.ToJSON(nMap)
		return string(b)
	})
}

func (m *MapField[K, V]) Validate(env Env) error {
	protoMap := m.parent.Get().NewField(m.descriptor).Map()
	if msgMap, isMsg := m.isMessage(); isMsg {
		msgMap.rangeKV(func(k K, m *Message) bool {
			protoMap.Set(protoreflect.MapKey(protoreflect.ValueOf(k)), protoreflect.ValueOfMessage(m.value))
			return true
		})
	} else {
		m.rangeKV(func(k K, v V) bool {
			protoMap.Set(protoreflect.MapKey(protoreflect.ValueOf(k)), protoreflect.ValueOf(v))
			return true
		})
	}

	return m.parent.ValidateField(env, m.Descriptor(), protoreflect.ValueOfMap(protoMap))
}

func (m *MapField[K, V]) Parent() *Message {
	return m.parent
}

func (m *MapField[K, V]) Descriptor() protoreflect.FieldDescriptor {
	return m.descriptor
}

func (m *MapField[K, V]) NewKeyAny() any {
	return m.NewKey()
}

func (m *MapField[K, V]) NewValAny() any {
	return m.newValue()
}

func (m *MapField[K, V]) NewKey() (k K) {
	return
}

func (m *MapField[K, V]) NewVal() V {
	return m.newValue()
}

func (m *MapField[K, V]) GetAnyPair(k string) (any, any, bool) {
	key := m.keyParser(k)
	val, ok := m.GetValue(key)
	return key, val, ok
}

func (m *MapField[K, V]) GetValue(k K) (V, bool) {
	entries := m.Get()
	val, ok := entries[k]
	return val, ok
}

func (m *MapField[K, V]) SetAnyPair(k, v any) error {
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

func (m *MapField[K, V]) SetPair(k K, v V) {
	pMap := m.Get()
	pMap[k] = v
	m.Set(pMap)
}

func (m *MapField[K, V]) RemoveKey(k string) {
	m.Remove(m.keyParser(k))
}

func (m *MapField[K, V]) Remove(k K) {
	m.mut.Lock()
	defer m.mut.Unlock()

	m.parent.GetField(m.descriptor).Map().Clear(protoreflect.ValueOf(k).MapKey())
}

func (m *MapField[K, V]) Get() map[K]V {
	m.mut.Lock()
	defer m.mut.Unlock()

	if m.Getter != nil {
		return m.Getter.Get()
	}

	return map[K]V{}
}

func (m *MapField[K, V]) Set(v map[K]V) {
	m.mut.Lock()
	defer m.mut.Unlock()

	if m.Setter != nil {
		m.Setter.Set(v)
	}
}

func (m *MapField[K, V]) Keys() (keys []string) {
	m.Range(func(k string, _ any) bool {
		keys = append(keys, k)
		return true
	})
	return
}

func (m *MapField[K, V]) Vals() (vals []any) {
	m.Range(func(_ string, v any) bool {
		vals = append(vals, v)
		return true
	})
	return
}

func (m *MapField[K, V]) Len() int {
	return len(m.Get())
}

func (m *MapField[K, V]) Range(f func(string, any) bool) {
	for k, v := range m.Get() {
		if !f(m.keyStringer(k), v) {
			return
		}
	}
}

func (m *MapField[K, V]) rangeKV(f func(K, V) bool) {
	for k, v := range m.Get() {
		if !f(k, v) {
			return
		}
	}
}

func (m *MapField[K, V]) Parse(s string) (nMap map[K]V, err error) {
	if m.Parser != nil {
		return m.Parser.Parse(s)
	}

	var rawMap map[K]json.RawMessage
	err = encoding.FromJSON([]byte(s), &rawMap)
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

				err = encoding.FromJSON(rawVal, msg)
				if err != nil {
					return
				}
			} else {
				err = encoding.FromJSON(rawVal, &val)
				if err != nil {
					return
				}
			}

			nMap[key] = val
		}
	}

	return
}

func (m *MapField[K, V]) isMessage() (mm *MapField[K, *Message], ok bool) {
	mm, ok = any(m).(*MapField[K, *Message])
	return
}

func (m *MapField[K, V]) StringOf(v map[K]V) string {
	if m.Stringer != nil {
		return m.Stringer.StringOf(v)
	}

	nMap := map[string]V{}
	for k, v := range m.Get() {
		nMap[m.keyStringer(k)] = v
	}

	b, _ := encoding.ToJSON(nMap)
	return string(b)
}

func (m *MapField[K, V]) String() string {
	return m.StringOf(m.Get())
}

func (f *MapField[K, V]) isField() {}
func (f *MapField[K, V]) isMap()   {}
