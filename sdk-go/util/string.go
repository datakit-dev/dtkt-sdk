package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/stoewer/go-strcase"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const defaultSlugSep = '-'

var (
	defaultCharRegex = regexp.MustCompile("[^a-zA-Z0-9]+")
	defaultSlugRegex = regexp.MustCompile("[^a-zA-Z0-9" + string(defaultSlugSep) + "]+")
)

func ToString[T ~string](s T) string {
	return string(s)
}

func TidyString(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func ToPascalCase[T ~string](s T) string {
	return strcase.UpperCamelCase(string(s))
}

func ToSnakeCase[T ~string](s T) string {
	return strcase.SnakeCase(string(s))
}

func ToKebabCase[T ~string](s T) string {
	return strcase.KebabCase(string(s))
}

func ToWords[T ~string](s T) string {
	return strings.ReplaceAll(strcase.KebabCase(string(s)), "-", " ")
}

func Slugify(input string) string {
	return SlugifyWithSeparator(defaultSlugSep, input)
}

func SlugifyWithSeparator(sep rune, input string) string {
	var regex *regexp.Regexp
	if sep == defaultSlugSep {
		regex = defaultSlugRegex
	} else if (sep >= 'a' && sep <= 'z') || (sep >= 'A' && sep <= 'Z') || (sep >= '0' && sep <= '9') {
		regex = defaultCharRegex
	} else {
		regex = regexp.MustCompile("[^a-zA-Z0-9" + regexp.QuoteMeta(string(sep)) + "]+")
	}

	parts := regex.Split(input, -1)

	return strings.ToLower(
		strings.Trim(regex.ReplaceAllString(
			strings.Join(
				SliceMap(parts, func(s string) string {
					return strings.Trim(ToKebabCase(strings.TrimSpace(s)), string(sep))
				}), string(sep),
			), string(sep),
		), string(sep)),
	)
}

func StringFormatAny(val any) string {
	if val == nil {
		return ""
	}

	switch val := val.(type) {
	case protoreflect.Message:
		return prototext.Format(val.Interface())
	case proto.Message:
		return prototext.Format(val)
	case fmt.Stringer:
		return val.String()
	case string:
		return val
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%f", val)
	case []byte:
		return string(val)
	case bool:
		return strconv.FormatBool(val)
	}
	return fmt.Sprintf("%v", val)
}

func StringSplit(s string) []string {
	return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}
