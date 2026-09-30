package encoding

import (
	"bufio"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"slices"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const DefaultJSONDelim = "\n"

var _ Encoder = (*JSONEncoder)(nil)
var _ StreamEncoder = (*JSONEncoder)(nil)
var _ Decoder = (*JSONDecoder)(nil)
var _ StreamDecoder = (*JSONDecoder)(nil)

var defaultJsonEncoder = NewJSONEncoder()
var defaultJsonDecoder = NewJSONDecoder()

type (
	JSONEncoder struct {
		marshalers []*json.Marshalers
		options    []json.Options
		delim      string
		raw        bool
	}
	JSONDecoder struct {
		unmarshalers []*json.Unmarshalers
		options      []json.Options
		delim        string
		stream       bool
	}
)

func NewJSONEncoder(opts ...EncoderOption) *JSONEncoder {
	enc := &JSONEncoder{
		options: []json.Options{
			json.DefaultOptionsV2(),
		},
	}

	opts = append([]EncoderOption{
		WithEncodeDurationString(),
		WithProtoEncode(),
	}, opts...)

	enc.applyOptions(opts...)

	if enc.delim == "" {
		enc.delim = DefaultJSONDelim
	}

	slices.Reverse(enc.marshalers)
	enc.options = append(enc.options, json.WithMarshalers(json.JoinMarshalers(enc.marshalers...)))

	return enc
}

func NewJSONDecoder(opts ...DecoderOption) *JSONDecoder {
	dec := &JSONDecoder{
		options: []json.Options{
			json.DefaultOptionsV2(),
		},
	}

	opts = append([]DecoderOption{
		WithDecodeDurationString(),
		WithProtoDecode(),
	}, opts...)

	dec.applyOptions(opts...)

	if dec.delim == "" {
		dec.delim = DefaultJSONDelim
	}

	slices.Reverse(dec.unmarshalers)
	dec.options = append(dec.options, json.WithUnmarshalers(json.JoinUnmarshalers(dec.unmarshalers...)))

	return dec
}

func ToJSON(v any, opts ...EncoderOption) ([]byte, error) {
	return NewJSONEncoder(opts...).Encode(v)
}

func FromJSON(b []byte, v any, opts ...DecoderOption) error {
	return NewJSONDecoder(opts...).Decode(b, v)
}

func (e *JSONEncoder) Encode(v any) ([]byte, error) {
	if raw, ok := e.encodeRaw(v); ok {
		return raw, nil
	}
	return json.Marshal(v, e.options...)
}

func (e *JSONEncoder) StreamEncode(w io.Writer) func(any) error {
	enc := jsontext.NewEncoder(w, e.options...)
	if e.delim != DefaultJSONDelim {
		return func(v any) error {
			if raw, ok := e.encodeRaw(v); ok {
				_, err := fmt.Fprint(w, string(raw))
				if err != nil {
					return err
				}
			} else {
				err := json.MarshalWrite(w, v, e.options...)
				if err != nil {
					return err
				}
			}

			_, err := fmt.Fprint(w, e.delim)
			return err
		}
	}

	return func(v any) error {
		if raw, ok := e.encodeRaw(v); ok {
			_, err := fmt.Fprintln(w, string(raw))
			return err
		}

		return json.MarshalEncode(enc, v)
	}
}

func (d *JSONDecoder) Decode(b []byte, v any) error {
	return json.Unmarshal(b, v, d.options...)
}

func (d *JSONDecoder) StreamDecode(r io.Reader) func(any) error {
	scan := d.StreamScanner(r)
	return func(v any) error {
		b, err := scan()
		if err != nil {
			return err
		}
		return d.Decode(b, v)
	}
}

func (d *JSONDecoder) StreamScanner(r io.Reader) func() ([]byte, error) {
	// Custom delimiter with stream mode is not supported (uncommon combination)
	if d.delim != DefaultJSONDelim && d.stream {
		return func() ([]byte, error) {
			return nil, fmt.Errorf("stream mode with custom delimiter is not supported")
		}
	}

	if d.delim != DefaultJSONDelim {
		scanner := bufio.NewScanner(r)
		scanner.Split(DelimSplitFunc(d.delim))

		return func() ([]byte, error) {
			if scanner.Scan() {
				return scanner.Bytes(), nil
			}

			if err := scanner.Err(); err != nil {
				return nil, err
			}

			return nil, io.EOF
		}
	} else {
		var (
			decoder = jsontext.NewDecoder(r, d.options...)
			streamOpen,
			streamClose bool
		)

		return func() ([]byte, error) {
			if d.stream && streamOpen && streamClose {
				return nil, io.EOF
			}

			if d.stream && !streamOpen {
				tok, err := decoder.ReadToken()
				if err != nil {
					return nil, fmt.Errorf("stream decode start: %w", err)
				} else if tok.Kind() != '[' {
					return nil, fmt.Errorf("invalid stream start token: %s, expected: `[`", tok)
				}
				streamOpen = true
			}

			if d.stream && !streamClose {
				kind := decoder.PeekKind()
				if kind == ']' {
					_, err := decoder.ReadToken()
					if err != nil {
						return nil, fmt.Errorf("stream decode end: %w", err)
					}
					streamClose = true
				}
			}

			return decoder.ReadValue()
		}
	}
}

func (e *JSONEncoder) applyOptions(opts ...EncoderOption) {
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
}

func (d *JSONDecoder) applyOptions(opts ...DecoderOption) {
	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}
}

func (e *JSONEncoder) encodeRaw(v any) (jsontext.Value, bool) {
	switch v := v.(type) {
	case string:
		if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
			v = v[1 : len(v)-1]
		}

		if e.raw {
			return []byte(v), true
		}

		return []byte(strconv.Quote(v)), true
	case time.Time:
		return e.encodeRaw(v.Format(time.RFC3339))
	case time.Duration:
		return e.encodeRaw(v.String())
	case *wrapperspb.BytesValue:
		return e.encodeRaw(string(v.GetValue()))
	case *wrapperspb.StringValue:
		return e.encodeRaw(v.GetValue())
	case *timestamppb.Timestamp:
		return e.encodeRaw(v.AsTime())
	case *durationpb.Duration:
		return e.encodeRaw(v.AsDuration())
	case proto.Message:
		switch v.ProtoReflect().Descriptor().FullName() {
		case "google.protobuf.Timestamp":
			b, err := proto.Marshal(v)
			if err == nil {
				d := &timestamppb.Timestamp{}
				err = proto.Unmarshal(b, d)
				if err == nil {
					return e.encodeRaw(d.AsTime())
				}
			}
		case "google.protobuf.Duration":
			b, err := proto.Marshal(v)
			if err == nil {
				d := &durationpb.Duration{}
				err = proto.Unmarshal(b, d)
				if err == nil {
					return e.encodeRaw(d.AsDuration())
				}
			}
		case "google.protobuf.StringValue", "google.protobuf.BytesValue":
			field := v.ProtoReflect().Descriptor().Fields().ByName("value")
			if field != nil && v.ProtoReflect().Has(field) {
				return e.encodeRaw(v.ProtoReflect().Get(field).String())
			}
		case "google.protobuf.Value":
			field := v.ProtoReflect().Descriptor().Fields().ByName("string_value")
			if field != nil && v.ProtoReflect().Has(field) {
				return e.encodeRaw(v.ProtoReflect().Get(field).String())
			}
		}
	}
	return nil, false
}
