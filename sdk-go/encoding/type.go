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
)
