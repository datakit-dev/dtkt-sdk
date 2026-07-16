package encoding

import "io"

type (
	Encoder interface {
		Encode(any) ([]byte, error)
	}
	StreamEncoder interface {
		StreamEncode(io.Writer) func(any) error
	}
	EncoderV2 interface {
		Encoder
		StreamEncoder
	}
	Decoder interface {
		Decode([]byte, any) error
	}
	StreamDecoder interface {
		StreamDecode(io.Reader) func(any) error
	}
	DecoderV2 interface {
		Decoder
		StreamDecoder
	}
	EncodeFunc       func(any) ([]byte, error)
	DecodeFunc       func([]byte, any) error
	StreamEncodeFunc func(io.Writer) func(any) error
	StreamDecodeFunc func(io.Reader) func(any) error
)

func (f EncodeFunc) Encode(v any) ([]byte, error) {
	return f(v)
}

func (f DecodeFunc) Decode(b []byte, v any) error {
	return f(b, v)
}

func (f StreamEncodeFunc) StreamEncode(w io.Writer) func(any) error {
	return f(w)
}

func (f StreamDecodeFunc) StreamDecode(r io.Reader) func(any) error {
	return f(r)
}
