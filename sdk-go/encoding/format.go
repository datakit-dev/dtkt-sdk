package encoding

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

var _ Encoder = (Format)("")
var _ Decoder = (Format)("")

const (
	YAML = Format("yaml")
	JSON = Format("json")
)

type Format string

func (f Format) HasExt(path string) bool {
	return f.IsExt(filepath.Ext(path))
}

func (f Format) IsExt(ext string) bool {
	return slices.ContainsFunc(f.Exts(), func(e string) bool {
		return strings.EqualFold(ext, e)
	})
}

func (f Format) Exts() []string {
	switch f {
	case JSON:
		return []string{".json"}
	case YAML:
		return []string{".yaml", ".yml"}
	}
	return nil
}

func (f Format) String() string {
	return string(f)
}

func (f Format) Encoder() (Encoder, bool) {
	switch f {
	case JSON:
		return NewJSONEncoderV2(), true
	case YAML:
		return NewYAMLEncoderV2(), true
	}

	return nil, false
}

func (f Format) Decoder() (Decoder, bool) {
	switch f {
	case JSON:
		return NewJSONDecoderV2(), true
	case YAML:
		return NewYAMLDecoderV2(), true
	}

	return nil, false
}

func (f Format) Encode(v any) ([]byte, error) {
	if enc, ok := f.Encoder(); ok {
		return enc.Encode(v)
	}
	return nil, fmt.Errorf("unknown format: %s", f)
}

func (f Format) Decode(b []byte, v any) error {
	if dec, ok := f.Decoder(); ok {
		return dec.Decode(b, v)
	}
	return fmt.Errorf("unknown format: %s", f)
}
