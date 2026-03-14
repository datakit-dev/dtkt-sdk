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
)

type (
	JSONEncoderV2Option func(*JSONEncoderV2)
	JSONDecoderV2Option func(*JSONDecoderV2)
)

func WithEncodeJSONOptions(opts ...json.Options) JSONEncoderV2Option {
	return func(e *JSONEncoderV2) {
		e.opts = append(e.opts, opts...)
	}
}

func WithEncodeDelim(delim string) JSONEncoderV2Option {
	return func(e *JSONEncoderV2) {
		e.delim = delim
	}
}

func WithEncodeRaw(raw bool) JSONEncoderV2Option {
	return func(e *JSONEncoderV2) {
		e.raw = raw
	}
}

func WithDecodeDelim(delim string) JSONDecoderV2Option {
	return func(e *JSONDecoderV2) {
		e.delim = delim
	}
}

func WithEncodeIndent(prefix, indent string) JSONEncoderV2Option {
	return func(e *JSONEncoderV2) {
		e.opts = append(e.opts,
			jsontext.WithIndentPrefix(prefix),
			jsontext.WithIndent(indent),
		)
	}
}

func WithEncodeMultiline(multiline bool) JSONEncoderV2Option {
	return func(e *JSONEncoderV2) {
		e.opts = append(e.opts, jsontext.Multiline(multiline))
	}
}

func WithEncodeProtoJSON() JSONEncoderV2Option {
	return func(e *JSONEncoderV2) {
		e.marshalers = append(e.marshalers, json.MarshalFunc(protojson.Marshal))
	}
}

// WithEncodeDurationString adds support for encoding time.Duration to JSON strings.
func WithEncodeDurationString() JSONEncoderV2Option {
	return func(e *JSONEncoderV2) {
		e.marshalers = append(e.marshalers, json.MarshalFunc(func(d time.Duration) ([]byte, error) {
			return jsontext.Value(strconv.Quote(d.String())), nil
		}))
	}
}

func WithEncodeProtoJSONOptions(opts protojson.MarshalOptions) JSONEncoderV2Option {
	return func(e *JSONEncoderV2) {
		e.marshalers = append(e.marshalers, json.MarshalFunc(opts.Marshal))
	}
}

func WithEncodeProtoJSONRedact() JSONEncoderV2Option {
	return WithEncodeProtoJSONRedactOptions(protojson.MarshalOptions{})
}

func WithEncodeProtoJSONRedactOptions(opts protojson.MarshalOptions) JSONEncoderV2Option {
	return func(e *JSONEncoderV2) {
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

func WithDecodeJSONOptions(opts ...json.Options) JSONDecoderV2Option {
	return func(d *JSONDecoderV2) {
		d.opts = append(d.opts, opts...)
	}
}

// WithDecodeJSONStream when stream is true expects the first decoded token to
// be an array open bracket `[`, followed by zero or more json values, and a
// final array close bracket `]`. Each json value within this array is decoded
// and emitted to caller.
func WithDecodeJSONStream(stream bool) JSONDecoderV2Option {
	return func(d *JSONDecoderV2) {
		d.stream = stream
	}
}

// WithDecodeDurationString adds support for decoding time.Duration from JSON strings.
func WithDecodeDurationString() JSONDecoderV2Option {
	return func(d *JSONDecoderV2) {
		d.unmarshalers = append(d.unmarshalers,
			json.UnmarshalFunc(func(b []byte, p *time.Duration) error {
				d, err := time.ParseDuration(string(b))
				if err != nil {
					d, err = time.ParseDuration(string(b[1 : len(b)-1]))
					if err != nil {
						return err
					}
				}
				*p = d
				return nil
			}),
		)
	}
}

func WithDecodeProtoJSON() JSONDecoderV2Option {
	return func(d *JSONDecoderV2) {
		d.unmarshalers = append(d.unmarshalers, json.UnmarshalFunc(protojson.Unmarshal))
	}
}

func WithDecodeProtoJSONOptions(opts protojson.UnmarshalOptions) JSONDecoderV2Option {
	return func(d *JSONDecoderV2) {
		d.unmarshalers = append(d.unmarshalers, json.UnmarshalFunc(opts.Unmarshal))
	}
}

func WithDecodeProtoJSONFunc(f func(b []byte, m proto.Message) error) JSONDecoderV2Option {
	return func(d *JSONDecoderV2) {
		d.unmarshalers = append(d.unmarshalers, json.UnmarshalFunc(f))
	}
}
