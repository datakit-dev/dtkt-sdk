package encoding

import (
	"fmt"

	"buf.build/go/protoyaml"
	"google.golang.org/protobuf/proto"
	"sigs.k8s.io/yaml"
)

var _ Encoder = (*YAMLEncoder)(nil)
var _ Decoder = (*YAMLDecoder)(nil)

type (
	YAMLEncoder struct {
		json  *JSONEncoder
		proto protoyaml.MarshalOptions
	}
	YAMLDecoder struct {
		json  *JSONDecoder
		proto protoyaml.UnmarshalOptions
	}
	YAMLEncoderOption func(*YAMLEncoder)
	YAMLDecoderOption func(*YAMLDecoder)
)

func ToYAML(v any, opts ...YAMLEncoderOption) ([]byte, error) {
	return NewYAMLEncoder(opts...).Encode(v)
}

func FromYAML(b []byte, v any, opts ...YAMLDecoderOption) error {
	return NewYAMLDecoder(opts...).Decode(b, v)
}

func NewYAMLEncoder(opts ...YAMLEncoderOption) *YAMLEncoder {
	return (&YAMLEncoder{
		json: NewJSONEncoder(),
	}).WithOptions(opts...)
}

func NewYAMLDecoder(opts ...YAMLDecoderOption) *YAMLDecoder {
	return (&YAMLDecoder{
		json: NewJSONDecoder(),
	}).WithOptions(opts...)
}

func WithProtoYAMLMarshalOptions(opts protoyaml.MarshalOptions) YAMLEncoderOption {
	return func(e *YAMLEncoder) {
		e.proto = opts
	}
}

func WithProtoYAMLUnmarshalOptions(opts protoyaml.UnmarshalOptions) YAMLDecoderOption {
	return func(d *YAMLDecoder) {
		d.proto = opts
	}
}

func (e *YAMLEncoder) WithOptions(opts ...YAMLEncoderOption) *YAMLEncoder {
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	return e
}

func (d *YAMLDecoder) WithOptions(opts ...YAMLDecoderOption) *YAMLDecoder {
	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}
	return d
}

func (e *YAMLEncoder) Encode(v any) ([]byte, error) {
	if p, ok := v.(proto.Message); ok {
		if p == nil {
			return nil, fmt.Errorf("proto message cannot be nil")
		}
		return e.proto.Marshal(p)
	}

	b, err := e.json.Encode(v)
	if err != nil {
		return nil, err
	}

	return yaml.JSONToYAML(b)
}

func (d *YAMLDecoder) Decode(b []byte, v any) error {
	if p, ok := v.(proto.Message); ok {
		if p == nil {
			return fmt.Errorf("proto message cannot be nil")
		} else if !p.ProtoReflect().IsValid() {
			return fmt.Errorf("proto message must be mutable")
		}
		return d.proto.Unmarshal(b, p)
	}

	b, err := yaml.YAMLToJSON(b)
	if err != nil {
		return err
	}

	return d.json.Decode(b, v)
}
