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
	var buf bytes.Buffer
	stream := encoding.CSV.StreamEncode(&buf)
	err := stream([]string{"foo", "bar"})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(buf.String())

	b, err := encoding.CSV.Encode(json.RawMessage(`["foo", 123, 6789.789, true, null, {"bar": "baz"}, [123.456, -78]]`))
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(b))

	list, _ := structpb.NewValue([]any{123.456, "foo", true, nil})
	b, err = encoding.CSV.Encode(list)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(b))
}

func TestCSVDecoder(t *testing.T) {
	var buf bytes.Buffer
	stream := encoding.CSV.StreamDecode(&buf)

	_, err := fmt.Fprintln(&buf, `foo,bar`)
	if err != nil {
		t.Fatal(err)
	}

	var record []string
	err = stream(&record)
	if err != nil {
		t.Fatal(err)
	} else if len(record) < 2 || record[0] != "foo" || record[1] != "bar" {
		t.Fatalf(`expected: [foo, bar], got: %v`, record)
	}

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

	buf.Reset()

	_, err = fmt.Fprintln(&buf, `123,456`)
	if err != nil {
		t.Fatal(err)
	}

	protoVal := new(structpb.Value)
	err = encoding.CSV.Decode(buf.Bytes(), protoVal)
	if err != nil {
		t.Fatal(err)
	}

	anySlice = protoVal.GetListValue().AsSlice()
	if len(anySlice) < 2 || anySlice[0] != "123" || anySlice[1] != "456" {
		t.Fatalf(`expected: [123, 456], got: %v`, anySlice)
	}

	buf.Reset()

	_, err = fmt.Fprintln(&buf, `hello,world`)
	if err != nil {
		t.Fatal(err)
	}

	protoList := new(structpb.ListValue)
	err = encoding.CSV.Decode(buf.Bytes(), protoList)
	if err != nil {
		t.Fatal(err)
	}

	anySlice = protoList.AsSlice()
	if len(anySlice) < 2 || anySlice[0] != "hello" || anySlice[1] != "world" {
		t.Fatalf(`expected: [hello, world], got: %v`, anySlice)
	}
}
