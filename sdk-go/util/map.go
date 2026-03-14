package util

import (
	"encoding/json/jsontext"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
)

var _ MapType[string, any] = (*SyncMap[string, any])(nil)
var _ MapType[string, any] = (*OrderedMap[string, any])(nil)

type (
	OrderedMap[K comparable, V any] struct {
		keys []K
		vals []V
	}
	SyncMap[K comparable, V any] struct {
		m sync.Map
	}
	MapPair[K comparable, V any] struct {
		Key K
		Val V
	}
	MapType[K comparable, V any] interface {
		Clear()
		Delete(key K)
		FindFunc(find func(K, V) bool) (key K, value V, found bool)
		Keys() (keys []K)
		Len() (count int)
		Load(key K) (value V, ok bool)
		LoadAndDelete(key K) (value V, loaded bool)
		LoadOrStore(key K, value V) (actual V, loaded bool)
		Store(key K, value V)
		StorePairs(...MapPair[K, V])
		Values() (values []V)
	}
)

func NewOrderedMap[K comparable, V any](pairs ...MapPair[K, V]) (m *OrderedMap[K, V]) {
	return (&OrderedMap[K, V]{}).WithPairs(pairs...)
}

func NewSyncMap[K comparable, V any](pairs ...MapPair[K, V]) (m *SyncMap[K, V]) {
	m = &SyncMap[K, V]{}
	m.StorePairs(pairs...)
	return
}

func NewSyncMapFromNative[K comparable, V any](native map[K]V) (m *SyncMap[K, V]) {
	m = &SyncMap[K, V]{}
	m.StoreNative(native)
	return
}

func NewMapPair[K comparable, V any](k K, v V) MapPair[K, V] {
	return MapPair[K, V]{
		Key: k,
		Val: v,
	}
}

func AnyMap[M map[K]V, K comparable, V any](m M) map[K]any {
	result := map[K]any{}
	for k, v := range m {
		result[k] = v
	}
	return result
}

func MapToMap[V2 any, M map[K]V, K comparable, V any](m1 M, fn func(K, V) V2) map[K]V2 {
	result := map[K]V2{}
	for k, v := range m1 {
		result[k] = fn(k, v)
	}
	return result
}

func MergeMaps[M ~map[K]V, K comparable, V any](dst, src M) M {
	if dst == nil {
		return src
	}
	if src == nil {
		return dst
	}
	dst = maps.Clone(dst)
	maps.Copy(dst, src)
	return dst
}

// ReduceMap returns a map r for any key/value pairs from m for which fn returns true.
func ReduceMap[Map ~map[K]V, K comparable, V any](m Map, fn func(K, V) bool) (r Map) {
	r = maps.Clone(m)
	maps.DeleteFunc(r, func(k K, v V) bool {
		return !fn(k, v)
	})
	return
}

// ReduceMapToSlice returns a slice r for any key/value pairs from m for which fn returns true.
func ReduceMapToSlice[Map ~map[K]V, K comparable, V any, V2 any](m Map, fn func(K, V) (V2, bool)) (r []V2) {
	for k, v := range m {
		if v2, ok := fn(k, v); ok {
			r = append(r, v2)
		}
	}
	return
}

func (m *OrderedMap[K, V]) Clear() {
	m.keys = nil
	m.vals = nil
}

func (m *OrderedMap[K, V]) Clone() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{
		keys: slices.Clone(m.keys),
		vals: slices.Clone(m.vals),
	}
}

func (m *SyncMap[K, V]) Clear() {
	m.m.Clear()
}

func (m *OrderedMap[K, V]) WithPairs(pairs ...MapPair[K, V]) *OrderedMap[K, V] {
	if len(pairs) > 0 {
		for _, pair := range pairs {
			m.Store(pair.Key, pair.Val)
		}
	}
	return m
}

func (m *OrderedMap[K, V]) StorePairs(pairs ...MapPair[K, V]) {
	m.WithPairs(pairs...)
}

func (m *SyncMap[K, V]) StorePairs(pairs ...MapPair[K, V]) {
	if len(pairs) > 0 {
		for _, pair := range pairs {
			m.Store(pair.Key, pair.Val)
		}
	}
}

func (m *SyncMap[K, V]) StoreNative(native map[K]V) {
	for k, v := range native {
		m.Store(k, v)
	}
}

func (m *OrderedMap[K, V]) First() (key K, val V, ok bool) {
	ok = len(m.keys) > 0
	if ok {
		key = m.keys[0]
		val = m.vals[0]
	}
	return
}

func (m *OrderedMap[K, V]) Last() (key K, val V, ok bool) {
	return m.Index(-1)
}

func (m *OrderedMap[K, V]) Index(idx int) (key K, val V, ok bool) {
	if idx < 0 {
		idx = len(m.keys) + idx
	}

	ok = idx >= 0 && idx < len(m.keys)
	if ok {
		key = m.keys[idx]
		val = m.vals[idx]
	}
	return
}

func (m *OrderedMap[K, V]) Pop() (key K, val V, ok bool) {
	ok = len(m.keys) > 0
	if ok {
		key, m.keys = m.keys[len(m.keys)-1], m.keys[:len(m.keys)-1]
		val, m.vals = m.vals[len(m.vals)-1], m.vals[:len(m.vals)-1]
	}
	return
}

func (m *OrderedMap[K, V]) Keys() []K {
	return m.keys
}

func (m *OrderedMap[K, V]) Values() []V {
	return m.vals
}

func (m *OrderedMap[K, V]) AnyKeys() []any {
	return AnySlice(m.Keys())
}

func (m *OrderedMap[K, V]) AnyVals() []any {
	return AnySlice(m.Values())
}

func (m *OrderedMap[K, V]) Len() int {
	return len(m.keys)
}

func (m *OrderedMap[K, V]) FindFunc(find func(K, V) bool) (key K, value V, found bool) {
	m.Range(func(k K, v V) bool {
		found = find(k, v)
		if found {
			key = k
			value = v
		}
		return !found
	})
	return
}

func (m *OrderedMap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	value, loaded = m.Load(key)
	if loaded {
		m.Delete(key)
	}
	return
}

func (m *OrderedMap[K, V]) LoadOrStore(key K, val V) (actual V, loaded bool) {
	actual, loaded = m.Load(key)
	if !loaded {
		m.Store(key, val)
		actual = val
	}
	return
}

func (m *OrderedMap[K, V]) LoadOrStoreFunc(key K, val V, onActual func(actual V, loaded bool)) {
	onActual(m.LoadOrStore(key, val))
}

func (m *OrderedMap[K, V]) Range(callback func(key K, val V) bool) {
	for idx, key := range m.keys {
		if !callback(key, m.vals[idx]) {
			return
		}
	}
}

func (m *OrderedMap[K, V]) Store(key K, val V) {
	idx := slices.Index(m.keys, key)
	if idx >= 0 {
		m.vals[idx] = val
	} else {
		m.keys = append(m.keys, key)
		m.vals = append(m.vals, val)
	}
}

func (m *OrderedMap[K, V]) Load(key K) (val V, ok bool) {
	idx := slices.Index(m.keys, key)
	ok = idx >= 0
	if ok {
		val = m.vals[idx]
	}
	return
}

func (m *OrderedMap[K, V]) Delete(key K) {
	for idx, k := range m.keys {
		if k == key {
			m.keys = append(m.keys[:idx], m.keys[idx+1:]...)
			m.vals = append(m.vals[:idx], m.vals[idx+1:]...)
			break
		}
	}
}

func (m *OrderedMap[K, V]) ToNativeMap() (native map[K]V) {
	native = make(map[K]V, len(m.keys))
	m.Range(func(key K, val V) bool {
		native[key] = val
		return true
	})
	return
}

func (m *OrderedMap[K, V]) MarshalJSONTo(enc *jsontext.Encoder) error {
	err := enc.WriteToken(jsontext.BeginObject)
	if err != nil {
		return err
	}

	for idx, key := range m.keys {
		err = enc.WriteToken(jsontext.String(StringFormatAny(key)))
		if err != nil {
			return err
		}

		val, err := encoding.ToJSONV2(m.vals[idx], encoding.WithEncodeJSONOptions(enc.Options()))
		if err != nil {
			return err
		}

		err = enc.WriteValue(val)
		if err != nil {
			return err
		}
	}

	return enc.WriteToken(jsontext.EndObject)
}

func (m *OrderedMap[K, V]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var start bool
	for {
		if !start && dec.PeekKind() == '{' {
			_, err := dec.ReadToken()
			if err != nil {
				return err
			}
			start = true
		}

		// Unmarshal key
		raw, err := dec.ReadValue()
		if err != nil {
			return err
		}

		str, err := strconv.Unquote(raw.String())
		if err != nil {
			return err
		}

		key, err := ScanValueFor[K](str)
		if err != nil {
			return err
		}

		idx := slices.Index(m.keys, key)
		if idx == -1 {
			m.keys = append(m.keys, key)
			idx = len(m.keys) - 1
		}

		// Unmarshal value
		raw, err = dec.ReadValue()
		if err != nil {
			return err
		}

		var val V
		err = encoding.FromJSONV2(raw, &val, encoding.WithDecodeJSONOptions(dec.Options()))
		if err != nil {
			return err
		}

		fmt.Println("unmarshal key:", key, "val:", val)

		if len(m.vals) > idx {
			m.vals[idx] = val
		} else {
			m.vals = append(m.vals, val)
		}

		if start && dec.PeekKind() == '}' {
			_, err := dec.ReadToken()
			if err != nil {
				return err
			}
			break
		}
	}

	return nil
}

func (m *SyncMap[K, V]) Len() (count int) {
	m.m.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

func (m *SyncMap[K, V]) Delete(key K) { m.m.Delete(key) }

func (m *SyncMap[K, V]) Load(key K) (value V, ok bool) {
	v, ok := m.m.Load(key)
	if !ok || v == nil {
		return
	}
	return v.(V), ok
}

func (m *SyncMap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, loaded := m.m.LoadAndDelete(key)
	if !loaded || v == nil {
		return
	}
	return v.(V), loaded
}

func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	v, loaded := m.m.LoadOrStore(key, value)
	if !loaded || v == nil {
		return
	}
	return v.(V), loaded
}

func (m *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(key, value any) bool {
		if k, ok := key.(K); ok {
			if v, ok := value.(V); ok {
				return f(k, v)
			}
		}
		return true
	})
}

func (m *SyncMap[K, V]) Store(key K, value V) { m.m.Store(key, value) }

func (m *SyncMap[K, V]) FindFunc(find func(K, V) bool) (key K, value V, found bool) {
	m.Range(func(k K, v V) bool {
		found = find(k, v)
		if found {
			key = k
			value = v
		}
		return !found
	})
	return
}

func (m *SyncMap[K, V]) Keys() (keys []K) {
	m.Range(func(key K, _ V) bool {
		keys = append(keys, key)
		return true
	})
	return
}

func (m *SyncMap[K, V]) Values() (values []V) {
	m.Range(func(_ K, value V) bool {
		values = append(values, value)
		return true
	})
	return
}

func (m *SyncMap[K, V]) ToNativeMap() (native map[K]V) {
	native = map[K]V{}
	if m.Len() > 0 {
		m.Range(func(key K, val V) bool {
			native[key] = val
			return true
		})
	}
	return
}
