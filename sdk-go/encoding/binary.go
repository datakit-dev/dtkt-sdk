package encoding

import (
	"bufio"
	"bytes"
	"encoding"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
)

var _ Encoder = (*BinaryEncoder)(nil)
var _ StreamEncoder = (*BinaryEncoder)(nil)
var _ Decoder = (*BinaryDecoder)(nil)
var _ StreamDecoder = (*BinaryDecoder)(nil)

type (
	BinaryEncoder struct {
		Delim string
		Proto protodelim.MarshalOptions
		buf   bytes.Buffer
	}
	BinaryDecoder struct {
		Delim    string
		Proto    protodelim.UnmarshalOptions
		buf      bytes.Buffer
		readers  map[io.Reader]protodelim.Reader
		scanners map[io.Reader]*bufio.Scanner
	}
)

func (e BinaryEncoder) StreamEncode(w io.Writer) NextFunc {
	return func(v any) error {
		return e.encode(w, v)
	}
}

func (e BinaryEncoder) Encode(v any) ([]byte, error) {
	e.buf.Reset()

	err := e.encode(&e.buf, v)
	if err != nil {
		return nil, err
	}

	return e.buf.Bytes(), nil
}

func (d BinaryDecoder) StreamDecode(r io.Reader) NextFunc {
	return func(v any) error {
		return d.decode(r, v)
	}
}

func (d BinaryDecoder) Decode(b []byte, v any) error {
	d.buf.Reset()
	_, err := d.buf.Write(b)
	if err != nil {
		return err
	}

	return d.decode(&d.buf, v)
}

func (e BinaryEncoder) encode(w io.Writer, v any) (err error) {
	var b []byte
	switch v := v.(type) {
	case proto.Message:
		_, err = protodelim.MarshalTo(w, v)
	case encoding.BinaryMarshaler:
		b, err = v.MarshalBinary()
		if err != nil {
			return err
		}

		_, err = w.Write(b)
	case encoding.TextMarshaler:
		b, err = v.MarshalText()
		if err != nil {
			return err
		}

		_, err = w.Write(b)
	case json.RawMessage:
		_, err = w.Write(v)
	case []byte:
		_, err = w.Write(v)
	case string:
		b, err = base64.StdEncoding.DecodeString(v)
		if err != nil {
			return err
		}

		_, err = w.Write(b)
	default:
		return fmt.Errorf("cannot encode %T to bytes", v)
	}
	return
}

func (d BinaryDecoder) decode(r io.Reader, v any) error {
	if msg, ok := v.(proto.Message); ok {
		return d.Proto.UnmarshalFrom(d.getReader(r), msg)
	} else if d.Delim == "" {
		return fmt.Errorf("cannot decode binary without a delimiter")
		// 	buf := make([]byte, bufio.MaxScanTokenSize)
		// return func(v any) error {
		// 	n, err := r.Read(buf)
		// 	if n > 0 {
		// 		err := d.decode(buf[:n], v)
		// 		if err != nil {
		// 			return err
		// 		}
		// 	}
		// 	return err
		// }
	}

	scanner := d.getScanner(r)
	if !scanner.Scan() {
		return io.EOF
	} else if err := scanner.Err(); err != nil {
		return err
	}

	switch v := v.(type) {
	case encoding.BinaryUnmarshaler:
		return v.UnmarshalBinary(scanner.Bytes())
	case encoding.TextUnmarshaler:
		return v.UnmarshalText(scanner.Bytes())
	case *any:
		*v = scanner.Bytes()
	case *json.RawMessage:
		*v = scanner.Bytes()
	case *[]byte:
		*v = scanner.Bytes()
	case *string:
		*v = base64.StdEncoding.EncodeToString(scanner.Bytes())
	default:
		return fmt.Errorf("cannot decode bytes to %T", v)
	}

	return nil
}

func (d *BinaryDecoder) getReader(r io.Reader) protodelim.Reader {
	reader, ok := r.(protodelim.Reader)
	if ok {
		return reader
	}

	if d.readers == nil {
		d.readers = map[io.Reader]protodelim.Reader{}
	}

	reader, ok = d.readers[r]
	if !ok {
		reader = bufio.NewReader(r)
		d.readers[r] = reader
	}

	return reader
}

func (d *BinaryDecoder) getScanner(r io.Reader) *bufio.Scanner {
	if d.scanners == nil {
		d.scanners = map[io.Reader]*bufio.Scanner{}
	}

	scanner, ok := d.scanners[r]
	if !ok {
		scanner = bufio.NewScanner(r)
		if d.Delim != "\n" {
			scanner.Split(DelimSplitFunc(d.Delim))
		}

		d.scanners[r] = scanner
	}

	return scanner
}
