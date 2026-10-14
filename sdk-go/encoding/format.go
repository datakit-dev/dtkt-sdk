package encoding

import (
	"fmt"
	"io"
	"log"
	"maps"
	"mime"
	"net/url"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

var _ Encoder = (Format)("")
var _ StreamEncoder = (Format)("")
var _ Decoder = (Format)("")
var _ StreamDecoder = (Format)("")

const (
	Binary = Format("binary")
	CSV    = Format("csv")
	JSON   = Format("json")
	JSONL  = Format("jsonl")
	YAML   = Format("yaml")
)

var formatters = map[Format]Formatter{
	Binary: {
		mime: "application/octet-stream",
		ext:  ".bin",
		altExts: []string{
			".deploy",
			".msu",
			".msp",
		},
		getEncoder: func(...EncoderOption) FormatEncoder {
			return BinaryEncoder{}
		},
		getDecoder: func(...DecoderOption) FormatDecoder {
			return BinaryDecoder{}
		},
	},
	CSV: {
		mime: "text/csv; charset=utf-8",
		ext:  ".csv",
		getEncoder: func(...EncoderOption) FormatEncoder {
			return NewCSVEncoder()
		},
		getDecoder: func(do ...DecoderOption) FormatDecoder {
			return NewCSVDecoder()
		},
	},
	JSON: {
		mime: "application/json",
		ext:  ".json",
		getEncoder: func(opts ...EncoderOption) FormatEncoder {
			return NewJSONEncoder(opts...)
		},
		getDecoder: func(opts ...DecoderOption) FormatDecoder {
			return NewJSONDecoder(opts...)
		},
	},
	JSONL: {
		mime: "application/jsonl",
		ext:  ".jsonl",
		altExts: []string{
			".ndjson",
		},
		getEncoder: func(opts ...EncoderOption) FormatEncoder {
			return NewJSONEncoder(opts...)
		},
		getDecoder: func(opts ...DecoderOption) FormatDecoder {
			return NewJSONDecoder(opts...)
		},
	},
	YAML: {
		mime: "application/yaml",
		ext:  ".yaml",
		altExts: []string{
			".yml",
		},
		getEncoder: func(opts ...EncoderOption) FormatEncoder {
			return NewYAMLEncoder(opts...)
		},
		getDecoder: func(opts ...DecoderOption) FormatDecoder {
			return NewYAMLDecoder(opts...)
		},
	},
}

type (
	Format    string
	Formatter struct {
		mime       string
		ext        string
		altExts    []string
		getEncoder func(...EncoderOption) FormatEncoder
		getDecoder func(...DecoderOption) FormatDecoder
	}
	FormatEncoder interface {
		Encoder
		StreamEncoder
	}
	FormatDecoder interface {
		Decoder
		StreamDecoder
	}
)

func init() {
	for _, format := range formatters {
		for _, ext := range append(format.altExts, format.ext) {
			if mime.TypeByExtension(ext) == "" {
				err := mime.AddExtensionType(ext, format.mime)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}

func FormatForUri(uri string) (Format, bool) {
	for format := range formatters {
		if format.HasExt(uri) {
			return format, true
		}
	}
	return "", false
}

func FormatForExt(ext string) (Format, bool) {
	for format := range formatters {
		if format.IsExt(ext) {
			return format, true
		}
	}
	return "", false
}

func FormatForMimeType(mime string) (Format, bool) {
	for format := range formatters {
		if format.IsMimeType(mime) {
			return format, true
		}
	}
	return "", false
}

func (f Format) DelimDefault() string {
	switch f {
	case YAML:
		return DefaultYAMLDelim
	case CSV, JSON, JSONL:
		return DefaultJSONDelim
	}
	return ""
}

func Formats() []Format {
	return slices.Collect(maps.Keys(formatters))
}

func (f Format) HasExt(uri string) bool {
	url, err := url.Parse(uri)
	if err == nil {
		return f.IsExt(path.Ext(url.Path))
	}
	return f.IsExt(filepath.Ext(uri))
}

func (f Format) IsExt(ext string) bool {
	return slices.ContainsFunc(append(f.AltExts(), f.Ext()), func(e string) bool {
		return strings.EqualFold(ext, e)
	})
}

func (f Format) IsMimeType(mime string) bool {
	return formatters[f].mime == mime
}

func (f Format) MimeType() string {
	return formatters[f].mime
}

func (f Format) Ext() string {
	return formatters[f].ext
}

func (f Format) AltExts() []string {
	return formatters[f].altExts
}

func (f Format) String() string {
	return string(f)
}

func (f Format) Encoder(opts ...EncoderOption) (FormatEncoder, bool) {
	formatter, ok := formatters[f]
	if !ok || formatter.getEncoder == nil {
		return nil, false
	}
	return formatter.getEncoder(opts...), true
}

func (f Format) Encode(v any) ([]byte, error) {
	if enc, ok := f.Encoder(); ok {
		return enc.Encode(v)
	}
	return nil, fmt.Errorf("unknown format: %s", f)
}

func (f Format) StreamEncode(w io.Writer) NextFunc {
	if enc, ok := f.Encoder(); ok {
		return enc.StreamEncode(w)
	}
	return func(any) error {
		return fmt.Errorf("unknown format: %s", f)
	}
}

func (f Format) Decoder(opts ...DecoderOption) (FormatDecoder, bool) {
	formatter, ok := formatters[f]
	if !ok || formatter.getDecoder == nil {
		return nil, false
	}
	return formatter.getDecoder(opts...), true
}

func (f Format) Decode(b []byte, v any) error {
	if dec, ok := f.Decoder(); ok {
		return dec.Decode(b, v)
	}
	return fmt.Errorf("unknown format: %s", f)
}

func (f Format) StreamDecode(r io.Reader) NextFunc {
	if dec, ok := f.Decoder(); ok {
		return dec.StreamDecode(r)
	}
	return func(any) error {
		return fmt.Errorf("unknown format: %s", f)
	}
}
