package common

import (
	"reflect"

	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/invopop/jsonschema"
)

const (
	JSONArray   JSONType = "array"
	JSONBoolean JSONType = "boolean"
	JSONInteger JSONType = "integer"
	JSONNull    JSONType = "null"
	JSONNumber  JSONType = "number"
	JSONObject  JSONType = "object"
	JSONString  JSONType = "string"
)

var JSONTypes = []JSONType{
	JSONArray,
	JSONBoolean,
	JSONInteger,
	JSONNull,
	JSONNumber,
	JSONObject,
	JSONString,
}

// JSONType is a string that represents a JSON type.
type JSONType string

func JSONTypeFromKind(kind reflect.Kind) (JSONType, bool) {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return JSONInteger, true
	case reflect.Float32, reflect.Float64:
		return JSONNumber, true
	case reflect.Bool:
		return JSONBoolean, true
	case reflect.String:
		return JSONString, true
	case reflect.Slice:
		return JSONArray, true
	case reflect.Struct, reflect.Map:
		return JSONObject, true
	}
	return "", false
}

func JSONTypeFromString(t string) (sharedv1beta1.JSONType, bool) {
	switch t {
	case "array":
		return sharedv1beta1.JSONType_JSON_TYPE_ARRAY, true
	case "boolean":
		return sharedv1beta1.JSONType_JSON_TYPE_BOOLEAN, true
	case "number":
		return sharedv1beta1.JSONType_JSON_TYPE_NUMBER, true
	case "integer":
		return sharedv1beta1.JSONType_JSON_TYPE_INTEGER, true
	case "object":
		return sharedv1beta1.JSONType_JSON_TYPE_OBJECT, true
	case "string":
		return sharedv1beta1.JSONType_JSON_TYPE_STRING, true
	case "null":
		return sharedv1beta1.JSONType_JSON_TYPE_NULL, true
	}
	return sharedv1beta1.JSONType_JSON_TYPE_UNSPECIFIED, false
}

func JSONTypeString(t sharedv1beta1.JSONType) (string, bool) {
	switch t {
	case sharedv1beta1.JSONType_JSON_TYPE_ARRAY:
		return "array", true
	case sharedv1beta1.JSONType_JSON_TYPE_BOOLEAN:
		return "boolean", true
	case sharedv1beta1.JSONType_JSON_TYPE_NUMBER:
		return "number", true
	case sharedv1beta1.JSONType_JSON_TYPE_INTEGER:
		return "integer", true
	case sharedv1beta1.JSONType_JSON_TYPE_OBJECT:
		return "object", true
	case sharedv1beta1.JSONType_JSON_TYPE_STRING:
		return "string", true
	case sharedv1beta1.JSONType_JSON_TYPE_NULL:
		return "null", true
	}
	return "", false
}

func (jt JSONType) String() string {
	return string(jt)
}

func (jt JSONType) ToProto() sharedv1beta1.JSONType {
	jsonType, ok := JSONTypeFromString(jt.String())
	if ok {
		return jsonType
	}
	return sharedv1beta1.JSONType_JSON_TYPE_UNSPECIFIED
}

func (JSONType) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "string",
		Title:       "JSON Type",
		Description: "JSON data type",
		Enum:        util.AnySlice(JSONTypes),
	}
}

// GoTypeFormat returns the x-dtkt-format value for a given Go reflect.Kind.
// This preserves precise Go type information in JSON Schema for round-trip conversion.
func GoTypeFormat(kind reflect.Kind) string {
	switch kind {
	case reflect.Int:
		return "int" // int is at least 32 bits, typically 64-bit
	case reflect.Int8:
		return "int8"
	case reflect.Int16:
		return "int16"
	case reflect.Int32:
		return "int32"
	case reflect.Int64:
		return "int64"
	case reflect.Uint:
		return "uint64" // uint is at least 32 bits, typically 64-bit
	case reflect.Uint8:
		return "uint8"
	case reflect.Uint16:
		return "uint16"
	case reflect.Uint32:
		return "uint32"
	case reflect.Uint64:
		return "uint64"
	case reflect.Float32:
		return "float32"
	case reflect.Float64:
		return "float64"
	default:
		return ""
	}
}
