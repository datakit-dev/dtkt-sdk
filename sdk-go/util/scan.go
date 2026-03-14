package util

import "fmt"

func ScanValue(typ any, str string) (any, error) {
	switch typ.(type) {
	case bool:
		var scan bool
		_, err := fmt.Sscanf(str, "%t", &scan)
		return scan, err
	case int, uint:
		var scan int
		_, err := fmt.Sscanf(str, "%d", &scan)
		return scan, err
	case int8, uint8:
		var scan int8
		_, err := fmt.Sscanf(str, "%b", &scan)
		return scan, err
	case int16, uint16:
		var scan int16
		_, err := fmt.Sscanf(str, "%o", &scan)
		return scan, err
	case int64, uint64:
		var scan int64
		_, err := fmt.Sscanf(str, "%x", &scan)
		return scan, err
	case int32, uint32:
		var scan int32
		_, err := fmt.Sscanf(str, "%X", &scan)
		return scan, err
	case string:
		var scan string
		_, err := fmt.Sscanf(str, "%s", &scan)
		return scan, err
	}

	return nil, fmt.Errorf("scan unsupported for type: %T", typ)
}

func ScanValueFor[K comparable](str string) (val K, err error) {
	format, ok := scanFormatFor(val)
	if !ok {
		err = fmt.Errorf("invalid key type: %T", val)
		return
	}
	_, err = fmt.Sscanf(str, format, &val)
	return
}

func scanFormatFor(val any) (string, bool) {
	switch val.(type) {
	case bool:
		return "%t", true
	case int, uint:
		return "%d", true
	case int8, uint8:
		return "%b", true
	case int16, uint16:
		return "%o", true
	case int64, uint64:
		return "%x", true
	case int32, uint32:
		return "%X", true
	case string:
		return "%s", true
	}
	return "", false
}
