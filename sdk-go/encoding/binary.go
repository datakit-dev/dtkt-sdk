package encoding

import (
	"bufio"
	"encoding"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/jhump/protoreflect/v2/protoresolve"
	"google.golang.org/protobuf/proto"
)

type (
	BinaryEncoder struct {
		Delim    string
		Resolver protoresolve.SerializationResolver
	}
	BinaryDecoder struct {
		Delim    string
		Resolver protoresolve.SerializationResolver
	}
)

func (e BinaryEncoder) StreamEncode(w io.Writer) func(any) error {
	return func(v any) error {
		b, err := e.encode(v)
		if err != nil {
			return err
		}

		_, err = w.Write(b)
		return err
	}
}

func (e BinaryEncoder) Encode(v any) ([]byte, error) {
	return e.encode(v)
}

func (d BinaryDecoder) StreamDecode(r io.Reader) func(any) error {
	if d.Delim == "" {
		buf := make([]byte, bufio.MaxScanTokenSize)
		return func(v any) error {
			n, err := r.Read(buf)
			if n > 0 {
				err := d.decode(buf[:n], v)
				if err != nil {
					return err
				}
			}
			return err
		}
	}

	scanner := bufio.NewScanner(r)
	if d.Delim != "\n" {
		scanner.Split(DelimSplitFunc(d.Delim))
	}

	return func(v any) error {
		if scanner.Scan() {
			return d.decode(scanner.Bytes(), v)
		}

		err := scanner.Err()
		if err != nil {
			return err
		}

		return io.EOF
	}
}

func (d BinaryDecoder) Decode(b []byte, v any) error {
	return d.decode(b, v)
}

func (e BinaryEncoder) encode(v any) ([]byte, error) {
	switch v := v.(type) {
	case encoding.BinaryMarshaler:
		return v.MarshalBinary()
	case encoding.TextMarshaler:
		return v.MarshalText()
	case json.RawMessage:
		return v, nil
	case []byte:
		return v, nil
	case string:
		return base64.StdEncoding.DecodeString(v)
	case proto.Message:
		return proto.MarshalOptions{
			Deterministic: true,
		}.Marshal(v)
	}
	return nil, fmt.Errorf("cannot encode %T to bytes", v)
}

func (d BinaryDecoder) decode(b []byte, v any) error {
	switch v := v.(type) {
	case encoding.BinaryUnmarshaler:
		return v.UnmarshalBinary(b)
	case encoding.TextUnmarshaler:
		return v.UnmarshalText(b)
	case *any:
		*v = b
	case *json.RawMessage:
		*v = b
	case *[]byte:
		*v = b
	case *string:
		*v = base64.StdEncoding.EncodeToString(b)
	case proto.Message:
		return proto.UnmarshalOptions{
			Resolver: d.Resolver,
		}.Unmarshal(b, v)
	default:
		return fmt.Errorf("cannot decode bytes to %T", v)
	}
	return nil
}
