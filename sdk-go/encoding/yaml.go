package encoding

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"sigs.k8s.io/yaml"
)

var _ Encoder = (*YAMLEncoder)(nil)
var _ StreamEncoder = (*YAMLEncoder)(nil)
var _ Decoder = (*YAMLDecoder)(nil)
var _ StreamDecoder = (*YAMLDecoder)(nil)

const DefaultYAMLDelim = "---\n"

type (
	YAMLEncoder struct {
		json  *JSONEncoder
		delim string
	}
	YAMLDecoder struct {
		json  *JSONDecoder
		delim string
	}
)

func NewYAMLEncoder(opts ...EncoderOption) *YAMLEncoder {
	enc := &YAMLEncoder{
		json: NewJSONEncoder(opts...),
	}
	enc.applyOptions(opts...)

	if enc.delim == "" {
		enc.delim = DefaultYAMLDelim
	}

	return enc
}

func NewYAMLDecoder(opts ...DecoderOption) *YAMLDecoder {
	dec := &YAMLDecoder{
		json: NewJSONDecoder(opts...),
	}
	dec.applyOptions(opts...)

	if dec.delim == "" {
		dec.delim = DefaultYAMLDelim
	}

	return dec
}

func ToYAML(v any, opts ...EncoderOption) ([]byte, error) {
	return NewYAMLEncoder(opts...).Encode(v)
}

func FromYAML(b []byte, v any, opts ...DecoderOption) error {
	return NewYAMLDecoder(opts...).Decode(b, v)
}

func (e *YAMLEncoder) Encode(v any) ([]byte, error) {
	b, err := e.json.Encode(v)
	if err != nil {
		return nil, err
	}

	return yaml.JSONToYAML(b)
}

func (e *YAMLEncoder) applyOptions(opts ...EncoderOption) {
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
}

func (e *YAMLDecoder) applyOptions(opts ...DecoderOption) {
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
}

func (e *YAMLEncoder) StreamEncode(w io.Writer) NextFunc {
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

func (d *YAMLDecoder) Decode(b []byte, v any) error {
	b, err := yaml.YAMLToJSON(b)
	if err != nil {
		return err
	}

	return d.json.Decode(b, v)
}

func (d *YAMLDecoder) StreamDecode(r io.Reader) NextFunc {
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
