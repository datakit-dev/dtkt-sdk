package util_test

import (
	"slices"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

type testStruct struct {
	Str  string `json:"str,omitzero"`
	Nil  any    `json:"nil,omitzero"`
	Num  int    `json:"num,omitzero"`
	Bool bool   `json:"bool,omitzero"`
}

func TestOrderedMap_MarshalJSON(t *testing.T) {
	m1 := util.NewOrderedMap(
		util.NewMapPair("str", testStruct{Str: "def"}),
		util.NewMapPair("nil", testStruct{Nil: nil}),
		util.NewMapPair("num", testStruct{Num: 456}),
		util.NewMapPair("bool", testStruct{Bool: true}),
	)

	b1, err := encoding.ToJSONV2(m1)
	if err != nil {
		t.Fatal(err)
	}

	m2 := util.NewOrderedMap(
		util.NewMapPair("str", testStruct{}),
		util.NewMapPair("nil", testStruct{}),
		util.NewMapPair("num", testStruct{}),
		util.NewMapPair("bool", testStruct{}),
	)
	err = encoding.FromJSONV2(b1, m2)
	if err != nil {
		t.Fatal(err)
	}

	b2, err := encoding.ToJSONV2(m2)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(b1))
	t.Log(string(b2))

	if !slices.Equal(m1.Keys(), m2.Keys()) {
		t.Fatalf("expected keys %v == %v", m1.Keys(), m2.Keys())
	}

	// expected values:
	// [{def <nil> 0 false} { <nil> 0 false} { <nil> 456 false} { <nil> 0 true}]
	// [{ <nil> 0 false} { <nil> 0 false} { <nil> 0 false} { <nil> 0 false}]

	if !slices.Equal(m1.Values(), m2.Values()) {
		t.Fatalf("expected values %v == %v", m1.Values(), m2.Values())
	}
}

type mapType[K comparable, V any] struct {
	*util.OrderedMap[K, V]
}

func (m mapType[K, V]) Clone() mapWrapper {
	return mapType[K, V]{m.OrderedMap.Clone()}
}

type mapWrapper interface {
	AnyKeys() []any
	AnyVals() []any
	Clear()
	Clone() mapWrapper
	Len() int
}

func TestOrderedMap_NonStringKeys_MarshalJSON(t *testing.T) {
	for _, m1 := range []mapWrapper{
		mapType[bool, string]{util.NewOrderedMap(util.NewMapPair(bool(true), "bool"))},
		mapType[int32, string]{util.NewOrderedMap(util.NewMapPair(int32(1), "int32"))},
		mapType[int64, string]{util.NewOrderedMap(util.NewMapPair(int64(2), "int64"))},
		mapType[uint32, string]{util.NewOrderedMap(util.NewMapPair(uint32(3), "uint32"))},
		mapType[uint64, string]{util.NewOrderedMap(util.NewMapPair(uint64(4), "uint64"))},
		mapType[string, time.Duration]{util.NewOrderedMap(util.NewMapPair("dur", time.Minute))},
	} {
		b, err := encoding.ToJSONV2(m1)
		if err != nil {
			t.Fatal(err)
		}

		t.Log(string(b))

		m2 := m1.Clone()
		m2.Clear()

		if m2.Len() > 0 {
			t.Fatal("expected empty map")
		}

		err = encoding.FromJSONV2(b, &m2)
		if err != nil {
			t.Fatal(err)
		}

		if !slices.Equal(m1.AnyKeys(), m2.AnyKeys()) {
			t.Fatalf("expected keys %v == %v", m1.AnyKeys(), m2.AnyKeys())
		}

		if !slices.Equal(m1.AnyVals(), m2.AnyVals()) {
			t.Fatalf("expected values %v == %v", m1.AnyVals(), m2.AnyVals())
		}
	}
}

func TestOrderedMap_Pop(t *testing.T) {
	var (
		pairs = []util.MapPair[string, string]{
			util.NewMapPair("foo", "bar"),
			util.NewMapPair("bar", "baz"),
		}
		m = util.NewOrderedMap(pairs...)
	)

	slices.Reverse(pairs)
	for idx := range m.Len() {
		key, val, ok := m.Pop()
		if !ok || key != pairs[idx].Key || val != pairs[idx].Val {
			t.Fatalf("pop invalid at index %d (ok=%t): %s != %s, %s != %s", idx, ok, key, pairs[idx].Key, val, pairs[idx].Val)
		} else {
			t.Log(key, val)
		}
	}
}

func TestOrderedMap_Index(t *testing.T) {
	var (
		pairs = []util.MapPair[string, string]{
			util.NewMapPair("foo", "bar"),
			util.NewMapPair("bar", "baz"),
			util.NewMapPair("baz", "boo"),
		}
		m = util.NewOrderedMap(pairs...)
	)

	idx := 0
	key, val, ok := m.First()
	if !ok || key != pairs[idx].Key || val != pairs[idx].Val {
		t.Fatalf("first invalid at index %d (ok=%t): %s != %s, %s != %s", idx, ok, key, pairs[idx].Key, val, pairs[idx].Val)
	}

	idx = len(pairs) - 1
	key, val, ok = m.Last()
	if !ok || key != pairs[idx].Key || val != pairs[idx].Val {
		t.Fatalf("pop invalid at index %d (ok=%t): %s != %s, %s != %s", idx, ok, key, pairs[idx].Key, val, pairs[idx].Val)
	}

	for idx := range m.Len() {
		key, val, ok := m.Index(idx)
		if !ok || key != pairs[idx].Key || val != pairs[idx].Val {
			t.Fatalf("pop invalid at index %d (ok=%t): %s != %s, %s != %s", idx, ok, key, pairs[idx].Key, val, pairs[idx].Val)
		} else {
			t.Log(key, val)
		}
	}

	for idx := range m.Len() {
		key, val, ok := m.Index(-idx)
		if idx > 0 {
			idx = m.Len() - idx
		}
		if !ok || key != pairs[idx].Key || val != pairs[idx].Val {
			t.Fatalf("pop invalid at index %d (ok=%t): %s != %s, %s != %s", idx, ok, key, pairs[idx].Key, val, pairs[idx].Val)
		} else {
			t.Log(key, val)
		}
	}

	len := m.Len() + 1
	_, _, ok = m.Index(-len)
	if ok {
		t.Fatalf("expected index out of range: %d > %d", len, m.Len())
	}

	for m.Len() > 0 {
		_, _, ok = m.Last()
		if !ok {
			t.Fatalf("expected last value")
		}

		m.Pop()
	}

	_, _, ok = m.Last()
	if ok {
		t.Fatalf("expected no last value")
	}
}
