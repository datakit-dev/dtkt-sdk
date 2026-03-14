package encoding

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"sigs.k8s.io/yaml"
)

var _ EncoderV2 = (*YAMLEncoderV2)(nil)
var _ DecoderV2 = (*YAMLDecoderV2)(nil)

const DefaultYAMLDelim = "---\n"

type (
	YAMLEncoderV2 struct {
		jsonOpts []JSONEncoderV2Option
		json     *JSONEncoderV2
		delim    string
	}
	YAMLDecoderV2 struct {
		jsonOpts []JSONDecoderV2Option
		json     *JSONDecoderV2
		delim    string
	}
)

func ToYAMLV2(v any, opts ...YAMLEncoderV2Option) ([]byte, error) {
	return NewYAMLEncoderV2(opts...).Encode(v)
}

func FromYAMLV2(b []byte, v any, opts ...YAMLDecoderV2Option) error {
	return NewYAMLDecoderV2(opts...).Decode(b, v)
}

func NewYAMLEncoderV2(opts ...YAMLEncoderV2Option) *YAMLEncoderV2 {
	enc := &YAMLEncoderV2{}

	for _, opt := range opts {
		if opt != nil {
			opt(enc)
		}
	}

	if enc.delim == "" {
		enc.delim = DefaultYAMLDelim
	}

	enc.json = NewJSONEncoderV2(enc.jsonOpts...)

	return enc
}

func NewYAMLDecoderV2(opts ...YAMLDecoderV2Option) *YAMLDecoderV2 {
	dec := &YAMLDecoderV2{}

	for _, opt := range opts {
		if opt != nil {
			opt(dec)
		}
	}

	if dec.delim == "" {
		dec.delim = DefaultYAMLDelim
	}

	dec.json = NewJSONDecoderV2(dec.jsonOpts...)

	return dec
}

func (e *YAMLEncoderV2) Encode(v any) ([]byte, error) {
	b, err := e.json.Encode(v)
	if err != nil {
		return nil, err
	}

	return yaml.JSONToYAML(b)
}

func (d *YAMLDecoderV2) Decode(b []byte, v any) error {
	b, err := yaml.YAMLToJSON(b)
	if err != nil {
		return err
	}

	return d.json.Decode(b, v)
}

func (e *YAMLEncoderV2) StreamEncode(w io.Writer) func(any) error {
	return func(v any) error {
		b, err := e.Encode(v)
		if err != nil {
			return err
		}

		b, err = yaml.JSONToYAML(b)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(w, "%s%s", e.delim, b)
		return err
	}
}

func (d *YAMLDecoderV2) StreamDecode(r io.Reader) func(any) error {
	scan := bufio.NewScanner(r)
	scan.Split(DelimSplitFunc(d.delim))

	return func(v any) error {
		if scan.Scan() {
			b := scan.Bytes()
			if len(bytes.TrimSpace(b)) == 0 {
				return nil
			}

			err := d.Decode(b, v)
			return err
		}

		if err := scan.Err(); err != nil {
			return err
		}

		return io.EOF
	}
}
