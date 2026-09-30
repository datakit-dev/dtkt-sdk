package encoding

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strconv"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protorange"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/durationpb"
)

type (
	EncoderOption     func(EncoderOptionee)
	DecoderOption     func(DecoderOptionee)
	YAMLEncoderOption func(*YAMLEncoder)
	YAMLDecoderOption func(*YAMLDecoder)
	EncoderOptionee   interface {
		applyOptions(...EncoderOption)
	}
	DecoderOptionee interface {
		applyOptions(...DecoderOption)
	}
)

func WithJSONEncoderOptions(opts ...json.Options) EncoderOption {
	return func(e EncoderOptionee) {
		if e, ok := e.(*JSONEncoder); ok {
			e.options = append(e.options, opts...)
		}
	}
}

func WithEncodeDelim(delim string) EncoderOption {
	return func(e EncoderOptionee) {
		switch e := e.(type) {
		case *JSONEncoder:
			e.delim = delim
		case *YAMLEncoder:
			e.delim = delim
		}
	}
}

func WithEncodeRaw(raw bool) EncoderOption {
	return func(e EncoderOptionee) {
		if e, ok := e.(*JSONEncoder); ok {
			e.raw = raw
		}
	}
}

func WithDecodeDelim(delim string) DecoderOption {
	return func(d DecoderOptionee) {
		if d, ok := d.(*JSONDecoder); ok {
			d.delim = delim
		}
	}
}

func WithEncodeIndent(prefix, indent string) EncoderOption {
	return func(e EncoderOptionee) {
		if e, ok := e.(*JSONEncoder); ok {
			e.options = append(e.options,
				jsontext.WithIndentPrefix(prefix),
				jsontext.WithIndent(indent),
			)
		}
	}
}

func WithEncodeMultiline(multiline bool) EncoderOption {
	return func(e EncoderOptionee) {
		if e, ok := e.(*JSONEncoder); ok {
			e.options = append(e.options, jsontext.Multiline(multiline))
		}
	}
}

// WithEncodeDurationString adds support for encoding time.Duration to JSON strings.
func WithEncodeDurationString() EncoderOption {
	return func(e EncoderOptionee) {
		if e, ok := e.(*JSONEncoder); ok {
			e.marshalers = append(e.marshalers,
				json.MarshalFunc(func(dur time.Duration) ([]byte, error) {
					return jsontext.Value(strconv.Quote(dur.String())), nil
				}),
				json.MarshalFunc(func(dur *durationpb.Duration) ([]byte, error) {
					return jsontext.Value(strconv.Quote(dur.AsDuration().String())), nil
				}),
			)
		}
	}
}

func WithProtoEncode() EncoderOption {
	return func(e EncoderOptionee) {
		if e, ok := e.(*JSONEncoder); ok {
			e.marshalers = append(e.marshalers, json.MarshalFunc(protojson.Marshal))
		}
	}
}

func WithProtoEncodeOptions(opts protojson.MarshalOptions) EncoderOption {
	return func(e EncoderOptionee) {
		if e, ok := e.(*JSONEncoder); ok {
			e.marshalers = append(e.marshalers, json.MarshalFunc(opts.Marshal))
		}
	}
}

func WithProtoEncodeRedacted() EncoderOption {
	return WithProtoEncodeRedactedOptions(protojson.MarshalOptions{})
}

func WithProtoEncodeRedactedOptions(opts protojson.MarshalOptions) EncoderOption {
	return func(e EncoderOptionee) {
		if e, ok := e.(*JSONEncoder); ok {
			e.marshalers = append(e.marshalers,
				json.MarshalFunc(func(msg proto.Message) ([]byte, error) {
					err := protorange.Range(msg.ProtoReflect(), func(v protopath.Values) error {
						idx := v.Index(-1)
						switch idx.Step.Kind() {
						case protopath.FieldAccessStep:
							if idx.Step.FieldDescriptor().Options().(*descriptorpb.FieldOptions).GetDebugRedact() {
								v.Index(-2).Value.Message().Clear(idx.Step.FieldDescriptor())
							}
						}
						return nil
					})
					if err != nil {
						return nil, err
					}

					return opts.Marshal(msg)
				}),
			)
		}
	}
}

func WithJSONDecoderOptions(opts ...json.Options) DecoderOption {
	return func(d DecoderOptionee) {
		if d, ok := d.(*JSONDecoder); ok {
			d.options = append(d.options, opts...)
		}
	}
}

// WithDecodeJSONStream when stream is true expects the first decoded token to
// be an array open bracket `[`, followed by zero or more json values, and a
// final array close bracket `]`. Each json value within this array is decoded
// and emitted to caller.
func WithDecodeJSONStream(stream bool) DecoderOption {
	return func(d DecoderOptionee) {
		if d, ok := d.(*JSONDecoder); ok {
			d.stream = stream
		}
	}
}

// WithDecodeDurationString adds support for decoding time.Duration from JSON strings.
func WithDecodeDurationString() DecoderOption {
	return func(d DecoderOptionee) {
		if d, ok := d.(*JSONDecoder); ok {
			d.unmarshalers = append(d.unmarshalers,
				json.UnmarshalFunc(func(b []byte, dur *time.Duration) error {
					d, err := time.ParseDuration(string(b))
					if err != nil {
						d, err = time.ParseDuration(string(b[1 : len(b)-1]))
						if err != nil {
							return err
						}
					}
					*dur = d
					return nil
				}),
				json.UnmarshalFunc(func(b []byte, p *durationpb.Duration) error {
					d, err := time.ParseDuration(string(b))
					if err != nil {
						d, err = time.ParseDuration(string(b[1 : len(b)-1]))
						if err != nil {
							return err
						}
					}
					*p = *durationpb.New(d)
					return nil
				}),
			)
		}
	}
}

func WithProtoDecode() DecoderOption {
	return func(d DecoderOptionee) {
		if d, ok := d.(*JSONDecoder); ok {
			d.unmarshalers = append(d.unmarshalers, json.UnmarshalFunc(protojson.Unmarshal))
		}
	}
}

func WithProtoDecodeOptions(opts protojson.UnmarshalOptions) DecoderOption {
	return func(d DecoderOptionee) {
		if d, ok := d.(*JSONDecoder); ok {
			d.unmarshalers = append(d.unmarshalers, json.UnmarshalFunc(opts.Unmarshal))
		}
	}
}

func WithProtoDecodeFunc(f func(b []byte, m proto.Message) error) DecoderOption {
	return func(d DecoderOptionee) {
		if d, ok := d.(*JSONDecoder); ok {
			d.unmarshalers = append(d.unmarshalers, json.UnmarshalFunc(f))
		}
	}
}
