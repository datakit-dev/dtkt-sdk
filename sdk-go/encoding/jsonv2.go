package encoding

import (
	"bufio"
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"slices"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
)

const DefaultJSONDelim = "\n"

var _ EncoderV2 = (*JSONEncoderV2)(nil)
var _ DecoderV2 = (*JSONDecoderV2)(nil)

type (
	JSONEncoderV2 struct {
		marshalers []*json.Marshalers
		opts       []json.Options
		delim      string
		raw        bool
	}
	JSONDecoderV2 struct {
		unmarshalers []*json.Unmarshalers
		opts         []json.Options
		delim        string
		stream       bool
	}
)

func ToJSONV2(v any, opts ...JSONEncoderV2Option) ([]byte, error) {
	return NewJSONEncoderV2(opts...).Encode(v)
}

func FromJSONV2(b []byte, v any, opts ...JSONDecoderV2Option) error {
	return NewJSONDecoderV2(opts...).Decode(b, v)
}

func NewJSONEncoderV2(opts ...JSONEncoderV2Option) *JSONEncoderV2 {
	enc := &JSONEncoderV2{
		opts: []json.Options{
			json.DefaultOptionsV2(),
		},
	}

	opts = append([]JSONEncoderV2Option{
		WithEncodeDurationString(),
		WithEncodeProtoJSON(),
	}, opts...)

	for _, opt := range opts {
		if opt != nil {
			opt(enc)
		}
	}

	if enc.delim == "" {
		enc.delim = DefaultJSONDelim
	}

	slices.Reverse(enc.marshalers)
	enc.opts = append(enc.opts, json.WithMarshalers(json.JoinMarshalers(enc.marshalers...)))

	return enc
}

func NewJSONDecoderV2(opts ...JSONDecoderV2Option) *JSONDecoderV2 {
	dec := &JSONDecoderV2{
		opts: []json.Options{
			json.DefaultOptionsV2(),
		},
	}

	opts = append([]JSONDecoderV2Option{
		WithDecodeDurationString(),
		WithDecodeProtoJSON(),
	}, opts...)

	for _, opt := range opts {
		if opt != nil {
			opt(dec)
		}
	}

	if dec.delim == "" {
		dec.delim = DefaultJSONDelim
	}

	slices.Reverse(dec.unmarshalers)
	dec.opts = append(dec.opts, json.WithUnmarshalers(json.JoinUnmarshalers(dec.unmarshalers...)))

	return dec
}

func (e *JSONEncoderV2) encodeRaw(v any) (jsontext.Value, bool) {
	switch v := v.(type) {
	case string:
		if e.raw {
			return []byte(v), true
		}
	case time.Duration:
		if e.raw {
			return []byte(v.String()), true
		} else {
			return []byte(strconv.Quote(v.String())), true
		}
	case *durationpb.Duration:
		return e.encodeRaw(v.AsDuration())
	}
	return nil, false
}

func (e *JSONEncoderV2) Encode(v any) ([]byte, error) {
	if raw, ok := e.encodeRaw(v); ok {
		return raw, nil
	}

	buf := &bytes.Buffer{}
	err := json.MarshalWrite(buf, v, e.opts...)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (e *JSONEncoderV2) StreamEncode(w io.Writer) func(any) error {
	enc := jsontext.NewEncoder(w, e.opts...)
	return func(v any) error {
		if e.delim != DefaultJSONDelim {
			if raw, ok := e.encodeRaw(v); ok {
				_, err := fmt.Fprint(w, string(raw))
				if err != nil {
					return err
				}
				return nil
			} else {
				err := json.MarshalWrite(w, v, e.opts...)
				if err != nil {
					return err
				}
			}

			_, err := fmt.Fprint(w, e.delim)
			return err
		} else {
			if raw, ok := e.encodeRaw(v); ok {
				_, err := fmt.Fprintln(w, string(raw))
				return err
			}

			return json.MarshalEncode(enc, v)
		}
	}
}

func (d *JSONDecoderV2) Decode(b []byte, v any) error {
	return json.Unmarshal(b, v, d.opts...)
}

func (d *JSONDecoderV2) StreamDecode(r io.Reader) func(any) error {
	// Custom delimiter with stream mode is not supported (uncommon combination)
	if d.delim != DefaultJSONDelim && d.stream {
		return func(any) error {
			return fmt.Errorf("stream mode with custom delimiter is not supported")
		}
	}

	if d.delim != DefaultJSONDelim {
		scan := bufio.NewScanner(r)
		scan.Split(DelimSplitFunc(d.delim))

		return func(v any) error {
			if scan.Scan() {
				err := d.Decode(scan.Bytes(), v)
				if err != nil {
					return err
				}
			}

			if err := scan.Err(); err != nil {
				return err
			}

			return io.EOF
		}
	} else {
		var (
			dec = jsontext.NewDecoder(r, d.opts...)
			streamOpen,
			streamClose bool
		)

		return func(v any) error {
			if d.stream && streamOpen && streamClose {
				return io.EOF
			}

			if d.stream && !streamOpen {
				tok, err := dec.ReadToken()
				if err != nil {
					return fmt.Errorf("stream decode start: %w", err)
				} else if tok.Kind() != '[' {
					return fmt.Errorf("invalid stream start token: %s, expected: `[`", tok)
				}
				streamOpen = true
			}

			if d.stream && !streamClose {
				kind := dec.PeekKind()
				if kind == ']' {
					_, err := dec.ReadToken()
					if err != nil {
						return fmt.Errorf("stream decode end: %w", err)
					}
					streamClose = true
				}
			}

			return json.UnmarshalDecode(dec, v, d.opts...)
		}
	}
}
