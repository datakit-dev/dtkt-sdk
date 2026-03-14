package tippecanoe

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type FeatureCollectionStream struct {
	dec    *json.Decoder
	prefix io.Reader
	suffix io.Reader
	first  bool
	done   bool
	curr   io.Reader
}

type rawFeature struct {
	Type     string          `json:"type"`
	Geometry json.RawMessage `json:"geometry"`
}

// NewFeatureCollectionStream returns an io.Reader that wraps an NDJSON stream
// into a GeoJSON FeatureCollection.
func NewFeatureCollectionStream(r io.Reader) io.Reader {
	dec := json.NewDecoder(r)
	return &FeatureCollectionStream{
		dec:    dec,
		prefix: strings.NewReader(`{"type":"FeatureCollection","features":[`),
		suffix: strings.NewReader(`]}`),
		first:  true,
	}
}

func (f *FeatureCollectionStream) Read(p []byte) (int, error) {
	// Emit prefix
	if f.prefix != nil {
		n, err := f.prefix.Read(p)
		if errors.Is(err, io.EOF) {
			f.prefix = nil
		}
		if n > 0 || err != nil {
			return n, err
		}
	}

	// Emit any buffered feature content
	if f.curr != nil {
		n, err := f.curr.Read(p)
		if errors.Is(err, io.EOF) {
			f.curr = nil
		}
		if n > 0 || err != nil {
			return n, err
		}
	}

	// Loop to load next feature or suffix
	for {
		// If we already finished, emit suffix
		if f.done {
			if f.suffix != nil {
				n, err := f.suffix.Read(p)
				if errors.Is(err, io.EOF) {
					f.suffix = nil
				}
				return n, err
			}
			return 0, io.EOF
		}

		// Decode next raw line
		var raw json.RawMessage
		err := f.dec.Decode(&raw)
		if err != nil {
			if errors.Is(err, io.EOF) {
				if f.first {
					return 0, fmt.Errorf("NDJSON is valid, but no features found")
				}
				f.done = true
				continue // loop again to emit suffix
			}
			return 0, fmt.Errorf("JSON decode error: %w", err)
		}

		// Validate as Feature
		var ft rawFeature
		if err := json.Unmarshal(raw, &ft); err != nil {
			return 0, fmt.Errorf("invalid feature at line: %w", err)
		}
		if ft.Type != "Feature" {
			return 0, fmt.Errorf("unexpected type %q — only GeoJSON Features are supported", ft.Type)
		}

		// Set up the next chunk to emit (feature with optional comma)
		comma := ""
		if !f.first {
			comma = ","
		}
		f.first = false
		f.curr = strings.NewReader(comma + string(raw))

		// Loop again — now `f.curr != nil`, so it'll emit in the next `Read(p)`
	}
}
