package common

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"slices"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/gowebpki/jcs"
	"github.com/invopop/jsonschema"
	compileschema "github.com/santhosh-tekuri/jsonschema/v6"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const DefaultSchemaBaseURL = "https://schemas.datakit.cloud"

var _ JSONSchemaType = (*JSONSchema[any])(nil)

type (
	JSONSchema[T any] struct {
		id             string
		reflector      *jsonschema.Reflector
		reflectorOpts  []func(*jsonschema.Reflector)
		rawSchema      []byte
		jsonSchema     *jsonschema.Schema
		protoSchema    *structpb.Struct
		compiler       *compileschema.Compiler
		compiledSchema *compileschema.Schema
		callback       JSONSchemaCallbackFunc
	}
	JSONSchemaType interface {
		GetID() string
		ValidateAny(any) error
		JSONSchema() *jsonschema.Schema
		ToProto() *structpb.Struct
		Bytes() []byte
		String() string
		setID(string)
		setRawSchema([]byte)
		setCallback(JSONSchemaCallbackFunc)
		setCompiler(*compileschema.Compiler)
		setReflector(*jsonschema.Reflector)
		setReflectorOpts(...func(*jsonschema.Reflector))
	}
)

func JSONSchemaFor[T any](opts ...JSONSchemaOpt) (*JSONSchema[T], error) {
	var v T
	return NewJSONSchema(v, opts...)
}

func NewJSONSchema[T any](value T, opts ...JSONSchemaOpt) (*JSONSchema[T], error) {
	var s = &JSONSchema[T]{}

	if len(opts) > 0 {
		for opt := range slices.Values(opts) {
			if opt != nil {
				opt(s)
			}
		}
	}

	if s.reflector == nil {
		s.reflector = new(jsonschema.Reflector)
		s.reflector.ExpandedStruct = true
		// Add mapper to annotate fields with Go type information
		s.reflector.Mapper = GoTypeAnnotationMapper
	}

	if len(s.reflectorOpts) > 0 {
		for _, opt := range s.reflectorOpts {
			if opt != nil {
				opt(s.reflector)
			}
		}
	}

	if s.rawSchema != nil {
		var jsonSchema jsonschema.Schema
		err := json.Unmarshal(s.rawSchema, &jsonSchema)
		if err != nil {
			return nil, err
		}
		s.jsonSchema = &jsonSchema
	} else {
		jsonSchema, ok := DefaultJSONSchema(value)
		if ok {
			s.jsonSchema = jsonSchema
		} else {
			jsonSchema, err := reflectJSONSchema(s.reflector, value)
			if err != nil {
				return nil, err
			}

			s.jsonSchema = jsonSchema
		}
	}

	if s.reflector.BaseSchemaID != "" && s.id != "" {
		s.jsonSchema.ID = s.reflector.BaseSchemaID.Add(s.id)
	} else if s.id != "" {
		s.jsonSchema.ID = jsonschema.ID(s.id)
	}

	if s.callback != nil {
		err := s.callback(s.jsonSchema)
		if err != nil {
			return nil, err
		}
	}

	b, err := json.Marshal(s.jsonSchema)
	if err != nil {
		return nil, err
	}

	b, err = jcs.Transform(b)
	if err != nil {
		return nil, err
	}

	s.rawSchema = b

	protoSchema := new(structpb.Struct)
	err = protojson.Unmarshal(b, protoSchema)
	if err != nil {
		return nil, err
	}

	s.protoSchema = protoSchema

	if s.compiler == nil {
		s.compiler = compileschema.NewCompiler()
	}

	var exists bool
	err = s.compiler.AddResource(s.jsonSchema.ID.String(), protoSchema.AsMap())
	if err != nil {
		if _, exists = err.(*compileschema.ResourceExistsError); !exists {
			return nil, fmt.Errorf("add resource: %w", err)
		}
	}

	if !exists {
		s.compiledSchema, err = s.compiler.Compile(s.jsonSchema.ID.String())
		if err != nil {
			return nil, fmt.Errorf("compile: %w", err)
		}
	}

	return s, nil
}

func DefaultJSONSchema(v any) (*jsonschema.Schema, bool) {
	switch v.(type) {
	case time.Duration, *time.Duration, *durationpb.Duration:
		return &jsonschema.Schema{
			Type:   "string",
			Format: "duration",
		}, true
	case time.Time, *time.Time, *timestamppb.Timestamp:
		return &jsonschema.Schema{
			Type:   "string",
			Format: "date-time",
		}, true
	case *url.URL, url.URL:
		return &jsonschema.Schema{
			Type:   "string",
			Format: "uri",
		}, true
	case net.IP, *net.IP:
		return &jsonschema.Schema{
			Type:   "string",
			Format: "ipv4",
		}, true
	}
	return nil, false
}

func reflectJSONSchema(reflector *jsonschema.Reflector, value any) (*jsonschema.Schema, error) {
	reflectType := reflect.TypeOf(value)
	if reflectType == nil {
		return nil, fmt.Errorf("json schema reflect: unknown Go type")
	}

	if reflectType.Kind() == reflect.Interface {
		return &jsonschema.Schema{}, nil
	} else if reflectType.Kind() == reflect.Pointer {
		reflectType = reflectType.Elem()
	}

	if reflectType == nil {
		return nil, fmt.Errorf("json schema reflect: unknown Go type")
	}

	jsonType, ok := JSONTypeFromKind(reflectType.Kind())
	if ok {
		jsonSchema := &jsonschema.Schema{
			Type: jsonType.String(),
		}

		switch jsonType {
		case JSONInteger:
			// Add Go type annotation for integers
			if goFormat := GoTypeFormat(reflectType.Kind()); goFormat != "" {
				if jsonSchema.Extras == nil {
					jsonSchema.Extras = make(map[string]any)
				}
				jsonSchema.Extras["x-dtkt-format"] = goFormat
			}
		case JSONNumber:
			// Add Go type annotation for numbers
			if goFormat := GoTypeFormat(reflectType.Kind()); goFormat != "" {
				if jsonSchema.Extras == nil {
					jsonSchema.Extras = make(map[string]any)
				}
				jsonSchema.Extras["x-dtkt-format"] = goFormat
			}
		case JSONObject:
			switch reflectType.Kind() {
			case reflect.Map:
				if reflectType.Elem() == nil {
					return nil, fmt.Errorf("%s is not a valid JSON object", reflectType)
				} else if reflectType.Elem().Kind() != reflect.Interface {
					itemSchema, err := reflectJSONSchema(reflector, reflect.New(reflectType.Elem()).Interface())
					if err != nil {
						return nil, err
					}

					jsonSchema.ID = itemSchema.ID
					jsonSchema.Version = itemSchema.Version
					itemSchema.ID = ""
					itemSchema.Version = ""

					jsonSchema.AdditionalProperties = itemSchema
				}
			case reflect.Struct:
				if reflectType.NumField() > 0 {
					jsonSchema = reflector.ReflectFromType(reflectType)
				}
			}
		case JSONArray:
			if reflectType.Elem() == nil {
				return nil, fmt.Errorf("%s is not a valid JSON array", reflectType)
			} else if reflectType.Elem().Kind() != reflect.Interface {
				itemSchema, err := reflectJSONSchema(reflector, reflect.New(reflectType.Elem()).Interface())
				if err != nil {
					return nil, err
				}

				jsonSchema.ID = itemSchema.ID
				jsonSchema.Version = itemSchema.Version
				itemSchema.ID = ""
				itemSchema.Version = ""
				jsonSchema.Items = itemSchema
			}
		}

		return jsonSchema, nil
	}

	return nil, fmt.Errorf("%s cannot be represented as a json type", reflectType.Kind())
}

func (s *JSONSchema[T]) setCallback(f JSONSchemaCallbackFunc) {
	s.callback = f
}

func (s *JSONSchema[T]) setID(id string) {
	s.id = id
	if s.jsonSchema != nil {
		s.jsonSchema.ID = jsonschema.ID(id)
	}
}

func (s *JSONSchema[T]) setRawSchema(raw []byte) {
	s.rawSchema = raw
}

func (s *JSONSchema[T]) setReflector(r *jsonschema.Reflector) {
	s.reflector = r
}

func (s *JSONSchema[T]) setReflectorOpts(opts ...func(*jsonschema.Reflector)) {
	s.reflectorOpts = opts
}

func (s *JSONSchema[T]) setCompiler(c *compileschema.Compiler) {
	s.compiler = c
}

func (s *JSONSchema[T]) GetID() string {
	if s.jsonSchema != nil {
		return s.jsonSchema.ID.String()
	}
	return s.id
}

func (s *JSONSchema[T]) Bytes() []byte {
	return s.rawSchema
}

func (s *JSONSchema[T]) String() string {
	return string(s.rawSchema)
}

func (s *JSONSchema[T]) JSONSchema() *jsonschema.Schema {
	return s.jsonSchema
}

func (s *JSONSchema[T]) ToProto() *structpb.Struct {
	return s.protoSchema
}

func (s *JSONSchema[T]) ValidateString(doc string) (v T, err error) {
	return s.ValidateBytes([]byte(doc))
}

func (s *JSONSchema[T]) ValidateBytes(doc []byte) (v T, err error) {
	err = s.ValidateAny(doc)
	if err != nil {
		return
	}
	return UnmarshalJSON[T](doc)
}

func (s *JSONSchema[T]) Validate(value T) error {
	doc, err := encoding.ToJSON(value)
	if err != nil {
		return err
	}

	return s.ValidateAny(doc)
}

// GoTypeAnnotationMapper is a jsonschema.Mapper that annotates schema fields
// with x-dtkt-format to preserve Go type information for round-trip conversion.
func GoTypeAnnotationMapper(t reflect.Type) *jsonschema.Schema {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Get the Go type format if applicable
	goFormat := GoTypeFormat(t.Kind())
	if goFormat == "" {
		return nil // Let default handling proceed
	}

	schema := &jsonschema.Schema{}

	// Map to JSON Schema type
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = "integer"
	case reflect.Float32, reflect.Float64:
		schema.Type = "number"
	}

	// Add x-dtkt-format extension
	if schema.Extras == nil {
		schema.Extras = make(map[string]any)
	}
	schema.Extras["x-dtkt-format"] = goFormat

	return schema
}

func (s *JSONSchema[T]) ValidateAny(value any) error {
	var err error
	switch JSONType(s.jsonSchema.Type) {
	case JSONArray:
		switch doc := value.(type) {
		case string:
			value, err = UnmarshalJSON[[]any](doc)
		case []byte:
			value, err = UnmarshalJSON[[]any](doc)
		}
		if err != nil {
			return err
		}
	case JSONObject:
		switch doc := value.(type) {
		case string:
			value, err = UnmarshalJSON[map[string]any](doc)
		case []byte:
			value, err = UnmarshalJSON[map[string]any](doc)
		}
		if err != nil {
			return err
		}
	default:
		switch doc := value.(type) {
		case string:
			value, err = UnmarshalJSON[any](doc)
		case []byte:
			value, err = UnmarshalJSON[any](doc)
		}
		if err != nil {
			return err
		}
	}

	return s.compiledSchema.Validate(value)
}
