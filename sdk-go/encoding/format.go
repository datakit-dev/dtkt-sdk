package encoding

import (
	"fmt"
	"io"
	"log"
	"maps"
	"mime"
	"path/filepath"
	"slices"
	"strings"
)

var _ Encoder = (Format)("")
var _ Decoder = (Format)("")

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
		encoder: BinaryEncoder{},
		decoder: BinaryDecoder{},
	},
	CSV: {
		mime:    "text/csv; charset=utf-8",
		ext:     ".csv",
		encoder: NewCSVEncoder(),
		decoder: NewCSVDecoder(),
	},
	JSON: {
		mime:    "application/json",
		ext:     ".json",
		encoder: NewJSONEncoderV2(),
		decoder: NewJSONDecoderV2(),
	},
	JSONL: {
		mime: "application/jsonl",
		ext:  ".jsonl",
		altExts: []string{
			".ndjson",
		},
		encoder: NewJSONEncoderV2(),
		decoder: NewJSONDecoderV2(),
	},
	YAML: {
		mime: "application/yaml",
		ext:  ".yaml",
		altExts: []string{
			".yml",
		},
		encoder: NewYAMLEncoderV2(),
		decoder: NewYAMLDecoderV2(),
	},
}

type (
	Format    string
	Formatter struct {
		mime    string
		ext     string
		altExts []string
		encoder EncoderV2
		decoder DecoderV2
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

func FormatByExt(ext string) (Format, bool) {
	for format := range formatters {
		if format.IsExt(ext) {
			return format, true
		}
	}
	return "", false
}

func FormatByMimeType(mime string) (Format, bool) {
	for format := range formatters {
		if format.IsMimeType(mime) {
			return format, true
		}
	}
	return "", false
}

func Formats() []Format {
	return slices.Collect(maps.Keys(formatters))
}

func (f Format) HasExt(path string) bool {
	return f.IsExt(filepath.Ext(path))
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

func (f Format) Encoder() (EncoderV2, bool) {
	formatter, ok := formatters[f]
	if !ok || formatter.encoder == nil {
		return nil, false
	}
	return formatter.encoder, true
}

func (f Format) Encode(v any) ([]byte, error) {
	if enc, ok := f.Encoder(); ok {
		return enc.Encode(v)
	}
	return nil, fmt.Errorf("unknown format: %s", f)
}

func (f Format) StreamEncode(w io.Writer) func(any) error {
	if enc, ok := f.Encoder(); ok {
		return enc.StreamEncode(w)
	}
	return func(any) error {
		return fmt.Errorf("unknown format: %s", f)
	}
}

func (f Format) Decoder() (DecoderV2, bool) {
	formatter, ok := formatters[f]
	if !ok || formatter.decoder == nil {
		return nil, false
	}
	return formatter.decoder, true
}

func (f Format) Decode(b []byte, v any) error {
	if dec, ok := f.Decoder(); ok {
		return dec.Decode(b, v)
	}
	return fmt.Errorf("unknown format: %s", f)
}

func (f Format) StreamDecode(r io.Reader) func(any) error {
	if dec, ok := f.Decoder(); ok {
		return dec.StreamDecode(r)
	}
	return func(any) error {
		return fmt.Errorf("unknown format: %s", f)
	}
}
