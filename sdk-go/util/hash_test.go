package util

import (
	"bytes"
	"testing"
)

// testDeterministicBytes is a small helper that runs encodeDeterministic
// and returns the resulting buffer contents, failing the test on error.
func testDeterministicBytes(t *testing.T, value any) []byte {
	t.Helper()
	var buf bytes.Buffer
	if _, err := deterministicBytes(&buf, value); err != nil {
		t.Fatalf("encodeDeterministic(%#v) returned error: %v", value, err)
	}
	return buf.Bytes()
}

// assertDeterministic encodes the value twice and asserts the output is
// byte-for-byte identical both times.
func assertDeterministic(t *testing.T, value any) []byte {
	t.Helper()
	first := testDeterministicBytes(t, value)
	second := testDeterministicBytes(t, value)
	if !bytes.Equal(first, second) {
		t.Fatalf("encodeDeterministic(%#v) not deterministic: %q != %q", value, first, second)
	}
	return first
}

func TestEncodeDeterministic_Scalars(t *testing.T) {
	tests := []struct {
		name string
		a    any
		b    any
	}{
		{"bool", true, true},
		{"int", 42, 42},
		{"int8", int8(-8), int8(-8)},
		{"int16", int16(1234), int16(1234)},
		{"int32", int32(-123456), int32(-123456)},
		{"int64", int64(9223372036854775807), int64(9223372036854775807)},
		{"uint", uint(7), uint(7)},
		{"uint8", uint8(255), uint8(255)},
		{"uint16", uint16(65535), uint16(65535)},
		{"uint32", uint32(4294967295), uint32(4294967295)},
		{"uint64", uint64(18446744073709551615), uint64(18446744073709551615)},
		{"float32", float32(3.14), float32(3.14)},
		{"float64", 2.71828, 2.71828},
		{"string", "hello world", "hello world"},
		{"nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out1 := assertDeterministic(t, tt.a)
			out2 := assertDeterministic(t, tt.b)
			if !bytes.Equal(out1, out2) {
				t.Fatalf("equal scalar values encoded differently: %q != %q", out1, out2)
			}
		})
	}
}

func TestEncodeDeterministic_ScalarsDiffer(t *testing.T) {
	a := assertDeterministic(t, 1)
	b := assertDeterministic(t, 2)
	if bytes.Equal(a, b) {
		t.Fatalf("expected different scalar values to encode differently, both got %q", a)
	}
}

func TestEncodeDeterministic_Bytes(t *testing.T) {
	value := []byte("raw bytes")
	out1 := assertDeterministic(t, value)
	out2 := assertDeterministic(t, []byte("raw bytes"))
	if !bytes.Equal(out1, out2) {
		t.Fatalf("[]byte encoding not deterministic: %q != %q", out1, out2)
	}
}

func TestEncodeDeterministic_Reader(t *testing.T) {
	// io.Reader is stateful/single-use, so unlike the other cases we can't
	// encode it twice; just verify the bytes are copied through verbatim.
	out := testDeterministicBytes(t, bytes.NewReader([]byte("stream data")))
	if !bytes.Equal(out, []byte("stream data")) {
		t.Fatalf("io.Reader value should be copied verbatim, got %q", out)
	}
}

func TestEncodeDeterministic_Slices(t *testing.T) {
	a := []int{1, 2, 3}
	b := []int{1, 2, 3}
	out1 := assertDeterministic(t, a)
	out2 := assertDeterministic(t, b)
	if !bytes.Equal(out1, out2) {
		t.Fatalf("equal slices encoded differently: %q != %q", out1, out2)
	}

	// Slice element order matters and must change the output.
	reordered := []int{3, 2, 1}
	out3 := assertDeterministic(t, reordered)
	if bytes.Equal(out1, out3) {
		t.Fatalf("reordered slice should encode differently, both got %q", out1)
	}
}

func TestEncodeDeterministic_NestedSlices(t *testing.T) {
	a := [][]string{{"a", "b"}, {"c"}}
	b := [][]string{{"a", "b"}, {"c"}}
	out1 := assertDeterministic(t, a)
	out2 := assertDeterministic(t, b)
	if !bytes.Equal(out1, out2) {
		t.Fatalf("equal nested slices encoded differently: %q != %q", out1, out2)
	}
}

func TestEncodeDeterministic_Maps(t *testing.T) {
	// Map key order in Go is randomized, but the encoding must be stable
	// across repeated encodes of the same logical map, regardless of the
	// order in which the runtime happens to iterate it.
	value := map[string]int{
		"zebra":   1,
		"apple":   2,
		"mango":   3,
		"banana":  4,
		"cherry":  5,
		"delta":   6,
		"echo":    7,
		"foxtrot": 8,
	}

	var outputs [][]byte
	for i := 0; i < 20; i++ {
		outputs = append(outputs, testDeterministicBytes(t, value))
	}
	for i := 1; i < len(outputs); i++ {
		if !bytes.Equal(outputs[0], outputs[i]) {
			t.Fatalf("map encoding not deterministic across repeated encodes: %q != %q", outputs[0], outputs[i])
		}
	}
}

func TestEncodeDeterministic_NestedMaps(t *testing.T) {
	a := map[string]any{
		"name":  "widget",
		"tags":  map[string]any{"color": "red", "size": "large"},
		"count": 3,
	}
	b := map[string]any{
		"count": 3,
		"tags":  map[string]any{"size": "large", "color": "red"},
		"name":  "widget",
	}
	out1 := assertDeterministic(t, a)
	out2 := assertDeterministic(t, b)
	if !bytes.Equal(out1, out2) {
		t.Fatalf("equal nested maps (different construction order) encoded differently: %q != %q", out1, out2)
	}
}

func TestEncodeDeterministic_Structs(t *testing.T) {
	type Address struct {
		City string `json:"city"`
		Zip  string `json:"zip"`
	}
	type Person struct {
		Name    string   `json:"name"`
		Age     int      `json:"age"`
		Emails  []string `json:"emails"`
		Address Address  `json:"address"`
	}

	a := Person{
		Name:   "Ada",
		Age:    30,
		Emails: []string{"ada@example.com", "lovelace@example.com"},
		Address: Address{
			City: "London",
			Zip:  "SW1A",
		},
	}
	b := Person{
		Name:   "Ada",
		Age:    30,
		Emails: []string{"ada@example.com", "lovelace@example.com"},
		Address: Address{
			City: "London",
			Zip:  "SW1A",
		},
	}

	out1 := assertDeterministic(t, a)
	out2 := assertDeterministic(t, b)
	if !bytes.Equal(out1, out2) {
		t.Fatalf("equal structs encoded differently: %q != %q", out1, out2)
	}

	c := b
	c.Age = 31
	out3 := assertDeterministic(t, c)
	if bytes.Equal(out1, out3) {
		t.Fatalf("differing structs should encode differently, both got %q", out1)
	}
}

func TestEncodeDeterministic_PointerToStruct(t *testing.T) {
	type Point struct {
		X int `json:"x"`
		Y int `json:"y"`
	}

	a := &Point{X: 1, Y: 2}
	b := &Point{X: 1, Y: 2}

	out1 := assertDeterministic(t, a)
	out2 := assertDeterministic(t, b)
	if !bytes.Equal(out1, out2) {
		t.Fatalf("equal pointers-to-struct encoded differently: %q != %q", out1, out2)
	}
}

func TestHashAny_DeterministicAcrossTypes(t *testing.T) {
	values := []any{
		42,
		"hello",
		true,
		3.14,
		[]int{1, 2, 3},
		map[string]int{"a": 1, "b": 2},
		struct {
			Name string
			Age  int
		}{Name: "test", Age: 1},
		[]byte("raw"),
	}

	for _, v := range values {
		h1, err := HashAnyBytes(v)
		if err != nil {
			t.Fatal(err)
		}
		h2, err := HashAnyBytes(v)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(h1, h2) {
			t.Fatalf("HashAny(%#v) not deterministic: %x != %x", v, h1, h2)
		}
	}
}

func TestHashAny_DifferentValuesDifferentHashes(t *testing.T) {
	h1, err := HashAnyBytes(map[string]int{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	h2, err := HashAnyBytes(map[string]int{"a": 2})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(h1, h2) {
		t.Fatalf("expected different maps to hash differently, both got %x", h1)
	}
}
