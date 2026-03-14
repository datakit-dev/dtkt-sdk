package tippecanoe

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

const maxGeoJSONLineSize = 10 * 1024 * 1024 // 10MB

func convertNDJSONToFeatureCollection(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, maxGeoJSONLineSize)

	_, _ = w.Write([]byte(`{"type":"FeatureCollection","features":[`))

	first := true
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		lineNum++

		// Validate each line as a Feature
		var f rawFeature
		if err := json.Unmarshal(line, &f); err != nil {
			return fmt.Errorf("invalid JSON at line %d: %w", lineNum, err)
		}
		if f.Type != "Feature" {
			return fmt.Errorf("invalid GeoJSON type at line %d: expected 'Feature', got '%s'", lineNum, f.Type)
		}
		if len(f.Geometry) == 0 || string(f.Geometry) == "null" {
			return fmt.Errorf("missing geometry at line %d", lineNum)
		}

		// Write the validated line to the output
		if !first {
			_, _ = w.Write([]byte(","))
		}
		first = false
		_, _ = w.Write(line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading NDJSON: %w", err)
	}

	_, _ = w.Write([]byte(`]}`))
	return nil
}
