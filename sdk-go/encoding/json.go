package encoding

import (
	"bytes"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var _ Encoder = (*JSONEncoder)(nil)
var _ Decoder = (*JSONDecoder)(nil)

type (
	JSONEncoder struct {
		buf   *bytes.Buffer
		enc   *json.Encoder
		proto protojson.MarshalOptions
		json  []func(*json.Encoder)
	}
	JSONDecoder struct {
		buf   *bytes.Buffer
		dec   *json.Decoder
		proto protojson.UnmarshalOptions
		json  []func(*json.Decoder)
	}
	JSONEncoderOption func(*JSONEncoder)
	JSONDecoderOption func(*JSONDecoder)
)

func ToJSON(v any, opts ...JSONEncoderOption) ([]byte, error) {
	return NewJSONEncoder(opts...).Encode(v)
}

func FromJSON(b []byte, v any, opts ...JSONDecoderOption) error {
	return NewJSONDecoder(opts...).Decode(b, v)
}

func NewJSONEncoder(opts ...JSONEncoderOption) *JSONEncoder {
	buf := new(bytes.Buffer)
	return (&JSONEncoder{
		buf: buf,
		enc: json.NewEncoder(buf),
	}).WithOptions(opts...)
}

func NewJSONDecoder(opts ...JSONDecoderOption) *JSONDecoder {
	buf := new(bytes.Buffer)
	return (&JSONDecoder{
		buf: buf,
		dec: json.NewDecoder(buf),
	}).WithOptions(opts...)
}

func WithJSONEncoderOptions(opts ...func(*json.Encoder)) JSONEncoderOption {
	return func(e *JSONEncoder) {
		e.json = append(e.json, opts...)
	}
}

func WithProtoJSONMarshalOptions(opts protojson.MarshalOptions) JSONEncoderOption {
	return func(e *JSONEncoder) {
		e.proto = opts
	}
}

func WithJSONDecoderOptions(opts ...func(*json.Decoder)) JSONDecoderOption {
	return func(d *JSONDecoder) {
		d.json = append(d.json, opts...)
	}
}

func WithProtoJSONUnmarshalOptions(opts protojson.UnmarshalOptions) JSONDecoderOption {
	return func(d *JSONDecoder) {
		d.proto = opts
	}
}

func (e *JSONEncoder) WithOptions(opts ...JSONEncoderOption) *JSONEncoder {
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}

	if len(e.json) > 0 {
		for _, o := range e.json {
			if o != nil {
				o(e.enc)
			}
		}
	}

	return e
}

func (d *JSONDecoder) WithOptions(opts ...JSONDecoderOption) *JSONDecoder {
	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}

	d.dec = json.NewDecoder(d.buf)
	d.dec.UseNumber()

	if len(d.json) > 0 {
		for _, opt := range d.json {
			if opt != nil {
				opt(d.dec)
			}
		}
	}

	return d
}

func (e *JSONEncoder) Encode(v any) ([]byte, error) {
	_, isMarshaler := v.(json.Marshaler)
	if p, ok := v.(proto.Message); ok && !isMarshaler {
		if p == nil {
			return nil, fmt.Errorf("proto message cannot be nil")
		}
		return e.proto.Marshal(p)
	}

	err := e.enc.Encode(v)
	if err != nil {
		return nil, err
	}

	return e.buf.Bytes(), nil
}

func (d *JSONDecoder) Decode(b []byte, v any) error {
	_, isUnmarshaler := v.(json.Unmarshaler)
	if p, ok := v.(proto.Message); ok && !isUnmarshaler {
		if p == nil {
			return fmt.Errorf("proto message cannot be nil")
		} else if !p.ProtoReflect().IsValid() {
			return fmt.Errorf("proto message must be mutable")
		}
		return d.proto.Unmarshal(b, p)
	}

	_, err := d.buf.Write(b)
	if err != nil {
		return err
	}

	return d.dec.Decode(v)
}
