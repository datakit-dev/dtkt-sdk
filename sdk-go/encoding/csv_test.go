package encoding_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCSVEncoder(t *testing.T) {
	var (
		enc = encoding.NewCSVEncoder()
		buf bytes.Buffer
	)

	stream := enc.StreamEncode(&buf)
	err := stream([]string{"foo", "bar"})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(buf.String())

	b, err := enc.Encode(json.RawMessage(`["foo", 123, 6789.789, true, null, {"bar": "baz"}, [123.456, -78]]`))
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(b))

	list, _ := structpb.NewValue([]any{123.456, "foo", true, nil})
	b, err = enc.Encode(list)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(b))
}

func TestCSVDecoder(t *testing.T) {
	var (
		dec    = encoding.NewCSVDecoder()
		buf    bytes.Buffer
		record []string
	)

	_, err := fmt.Fprintln(&buf, `foo,bar`)
	if err != nil {
		t.Fatal(err)
	}

	stream := dec.StreamDecode(&buf)
	err = stream(&record)
	if err != nil {
		t.Fatal(err)
	} else if len(record) < 2 || record[0] != "foo" || record[1] != "bar" {
		t.Fatalf(`expected: [foo, bar], got: %v`, record)
	}

	t.Log(record)

	_, err = fmt.Fprintln(&buf, `bar,baz`)
	if err != nil {
		t.Fatal(err)
	}

	var anySlice []any
	err = stream(&anySlice)
	if err != nil {
		t.Fatal(err)
	} else if len(anySlice) < 2 || anySlice[0] != "bar" || anySlice[1] != "baz" {
		t.Fatalf(`expected: [bar, baz], got: %v`, anySlice)
	}

	t.Log(record)
	buf.Reset()

	_, err = fmt.Fprintln(&buf, `123,456`)
	if err != nil {
		t.Fatal(err)
	}

	protoVal := new(structpb.Value)
	err = dec.Decode(buf.Bytes(), protoVal)
	if err != nil {
		t.Fatal(err)
	}

	anySlice = protoVal.GetListValue().AsSlice()
	if len(anySlice) < 2 || anySlice[0] != "123" || anySlice[1] != "456" {
		t.Fatalf(`expected: [123, 456], got: %v`, anySlice)
	}

	t.Log(protoVal.AsInterface())
	buf.Reset()

	_, err = fmt.Fprintln(&buf, `hello,world`)
	if err != nil {
		t.Fatal(err)
	}

	protoList := new(structpb.ListValue)
	err = dec.Decode(buf.Bytes(), protoList)
	if err != nil {
		t.Fatal(err)
	}

	anySlice = protoList.AsSlice()
	if len(anySlice) < 2 || anySlice[0] != "hello" || anySlice[1] != "world" {
		t.Fatalf(`expected: [hello, world], got: %v`, anySlice)
	}

	t.Log(anySlice)
	buf.Reset()

	_, err = fmt.Fprintln(&buf, `hello,world`)
	if err != nil {
		t.Fatal(err)
	}

	var rawJson []byte
	err = stream(&rawJson)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(rawJson))

	err = stream(nil)
	if err == nil {
		t.Fatal("expected value cannot be nil error")
	}
}
