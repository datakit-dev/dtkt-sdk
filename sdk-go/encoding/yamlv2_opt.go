package encoding

type (
	YAMLEncoderV2Option func(*YAMLEncoderV2)
	YAMLDecoderV2Option func(*YAMLDecoderV2)
)

func WithYAMLEncoderV2JSONOptions(opts ...JSONEncoderV2Option) YAMLEncoderV2Option {
	return func(e *YAMLEncoderV2) {
		e.jsonOpts = append(e.jsonOpts, opts...)
	}
}

func WithYAMLDecoderV2JSONOptions(opts ...JSONDecoderV2Option) YAMLDecoderV2Option {
	return func(d *YAMLDecoderV2) {
		d.jsonOpts = append(d.jsonOpts, opts...)
	}
}

func WithYAMLEncodeDelim(delim string) YAMLEncoderV2Option {
	return func(e *YAMLEncoderV2) {
		e.delim = delim
	}
}
func WithYAMLDecodeDelim(split string) YAMLDecoderV2Option {
	return func(d *YAMLDecoderV2) {
		d.delim = split
	}
}
