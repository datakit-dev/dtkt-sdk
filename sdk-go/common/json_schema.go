package common

import (
	"encoding/json"
	"net"
	"net/url"
	"reflect"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/protoschema"

	"github.com/google/jsonschema-go/jsonschema"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const DefaultSchemaBaseURL = "https://schemas.datakit.cloud"

var _ JSONSchemaType = (*JSONSchema[any])(nil)

type (
	JSONSchema[T any] struct {
		options JSONSchemaOptions

		schema   *jsonschema.Schema
		resolved *jsonschema.Resolved

		proto     *structpb.Struct
		protoType protoreflect.MessageType
	}
	JSONSchemaType interface {
		ID() string
		Bytes() []byte
		Proto() *structpb.Struct
		String() string
		Schema() *jsonschema.Schema
		Resolved() (*jsonschema.Resolved, error)
	}
)

func NewJSONSchemaWithOptions[T any](value T, opts *JSONSchemaOptions) (*JSONSchema[T], error) {
	s := &JSONSchema[T]{}

	if opts != nil {
		s.options = *opts
	}

	if s.options.BaseURI == "" {
		s.options.BaseURI = "mem://"
	}

	if msg, ok := any(value).(proto.Message); ok {
		s.protoType = msg.ProtoReflect().Type()
	}

	if s.options.Bytes != nil {
		var schema jsonschema.Schema
		err := json.Unmarshal(s.options.Bytes, &schema)
		if err != nil {
			return nil, err
		}

		s.schema = &schema
	} else {
		switch value := any(value).(type) {
		case time.Duration, *time.Duration, *durationpb.Duration:
			s.schema = &jsonschema.Schema{
				Type:   "string",
				Format: "duration",
			}
		case time.Time, *time.Time, *timestamppb.Timestamp:
			s.schema = &jsonschema.Schema{
				Type:   "string",
				Format: "date-time",
			}
		case *url.URL, url.URL:
			s.schema = &jsonschema.Schema{
				Type:   "string",
				Format: "uri",
			}
		case net.IP, *net.IP:
			s.schema = &jsonschema.Schema{
				Type:   "string",
				Format: "ipv4",
			}
		case proto.Message:
			if s.options.Loader == nil {
				resolved, err := protoschema.ResolvedJSONSchemaWithOptions(value.ProtoReflect().Descriptor(), &s.options.ResolveOptions)
				if err != nil {
					return nil, err
				}

				s.resolved = resolved
				s.schema = resolved.Schema().CloneSchemas()
			} else {
				schema, err := protoschema.NewJSONSchema(value.ProtoReflect().Descriptor())
				if err != nil {
					return nil, err
				}

				s.schema = schema
			}
		default:
			schema, err := jsonschema.For[T](&jsonschema.ForOptions{
				IgnoreInvalidTypes: true,
			})
			if err != nil {
				return nil, err
			}

			s.schema = schema
		}
	}

	if s.options.ID != "" {
		s.schema.ID = s.options.ID
	} else {
		s.schema.ID = s.options.BaseURI + "/" + s.schema.ID
	}

	if s.options.Callback != nil {
		err := s.options.Callback(s.schema, &s.options.ResolveOptions)
		if err != nil {
			return nil, err
		}
	}

	b, err := json.Marshal(s.schema)
	if err != nil {
		return nil, err
	}

	raw := json.RawMessage(b)
	err = raw.Canonicalize()
	if err != nil {
		return nil, err
	}

	s.options.Bytes = raw
	s.proto = new(structpb.Struct)

	err = protojson.Unmarshal(s.options.Bytes, s.proto)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func NewJSONSchema[T any](value T, opts ...JSONSchemaOption) (*JSONSchema[T], error) {
	var o JSONSchemaOptions
	if len(opts) > 0 {
		for _, opt := range opts {
			if opt != nil {
				opt(&o)
			}
		}
	}
	return NewJSONSchemaWithOptions(value, &o)
}

func JSONSchemaWithOptionsFor[T any](opts *JSONSchemaOptions) (*JSONSchema[T], error) {
	var v T
	return NewJSONSchemaWithOptions(v, opts)
}

func JSONSchemaFor[T any](opts ...JSONSchemaOption) (*JSONSchema[T], error) {
	var v T
	return NewJSONSchema(v, opts...)
}

func (s *JSONSchema[T]) ID() string {
	if s.schema.ID != "" {
		return s.schema.ID
	}
	return s.options.ID
}

func (s *JSONSchema[T]) Bytes() []byte {
	return s.options.Bytes
}

func (s *JSONSchema[T]) String() string {
	return string(s.options.Bytes)
}

func (s *JSONSchema[T]) Schema() *jsonschema.Schema {
	return s.schema
}

func (s *JSONSchema[T]) Proto() *structpb.Struct {
	return s.proto
}

func (s *JSONSchema[T]) Resolved() (*jsonschema.Resolved, error) {
	if s.resolved == nil {
		resolved, err := s.schema.Resolve(&s.options.ResolveOptions)
		if err != nil {
			return nil, err
		}

		s.resolved = resolved
	}

	return s.resolved, nil
}

func (s *JSONSchema[T]) Validate(value T) error {
	_, err := s.ValidateAny(value)
	return err
}

func (s *JSONSchema[T]) ValidateAny(input any) (value T, err error) {
	schema, err := s.Resolved()
	if err != nil {
		return
	}

	doc, err := encoding.ToJSON(input)
	if err != nil {
		return
	}

	var val any
	err = encoding.FromJSON(doc, &val)
	if err != nil {
		return
	}

	err = schema.Validate(val)
	if err != nil {
		return
	}

	if s.protoType != nil {
		msg := s.protoType.New().Interface()
		err = encoding.FromJSON(doc, msg)
		if err != nil {
			return
		}

		value = msg.(T)
	} else {
		err = encoding.FromJSON(doc, &value)
	}

	return
}

// if s.protoType != nil && v.ProtoReflect().Descriptor() == s.protoType.Descriptor() {
// 	err = protovalidate.Validate(v)
// 	if err != nil {
// 		return
// 	}

// 	value = v.(T)
// 	return
// } else if s.protoType != nil {
// 	doc, err = encoding.ToJSON(input)
// 	if err != nil {
// 		return
// 	}

// 	msg := s.protoType.New().Interface()
// 	err = encoding.FromJSON(doc, msg)
// 	if err != nil {
// 		return
// 	}

// 	value = msg.(T)
// 	return
// }

// msg := s.protoType.New().Interface()
// err = encoding.FromJSON(doc, msg)
// if err != nil {
// 	return
// }

// err = protovalidate.Validate(msg)
// if err != nil {
// 	return
// }

// value = msg.(T)

// return

// switch v := input.(type) {
// case []byte:
// 	doc = v
// case json.RawMessage:
// 	doc = v
// case string:
// 	if JSONType(s.schema.Type) != JSONString {
// 		doc = []byte(v)
// 	} else if len(v) > 0 || (v[0] != '"' && v[len(v)-1] != '"') {
// 		input = strconv.Quote(v)
// 		useInput = true
// 	}
// case map[string]any, []any:
// 	useInput = true
// }

// if doc == nil {
// 	doc, err = encoding.ToJSON(input)
// 	if err != nil {
// 		return
// 	}
// }

// if !useInput {
// 	var val any
// 	err = encoding.FromJSON(doc, &val)
// 	if err != nil {
// 		return
// 	}
// 	input = val
// }

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
	if schema.Extra == nil {
		schema.Extra = make(map[string]any)
	}
	schema.Extra["x-dtkt-format"] = goFormat

	return schema
}

// schema, err := defaultJSONSchema(value, &s.resolve)
// if err != nil {
// 	return nil, err
// } else {
// if s.reflector == nil {
// 	s.reflector = new(jsonschema.Reflector)
// 	s.reflector.ExpandedStruct = true
// 	// Add mapper to annotate fields with Go type information
// 	s.reflector.Mapper = GoTypeAnnotationMapper
// }

// if len(s.reflectorOpts) > 0 {
// 	for _, opt := range s.reflectorOpts {
// 		if opt != nil {
// 			opt(s.reflector)
// 		}
// 	}
// }

// jsonSchema, err := reflectJSONSchema(s.reflector, value)
// if err != nil {
// 	return nil, err
// }

// s.jsons.schema = &jsonSchema
// }
// s.schema = schema

// if s.reflector != nil && s.reflector.BaseSchemaID != "" && s.id != "" {
// 	s.jsonSchema.ID = s.reflector.BaseSchemaID.Add(s.id)
// } else if s.id != "" {
// 	s.jsonSchema.ID = jsonschema.ID(s.id)
// }

// func reflectJSONSchema(reflector *jsonschema.Reflector, value any) (*jsonschema.Schema, error) {
// 	reflectType := reflect.TypeOf(value)
// 	if reflectType == nil {
// 		return nil, fmt.Errorf("json schema reflect: unknown Go type")
// 	}

// 	if reflectType.Kind() == reflect.Interface {
// 		return &jsonschema.Schema{}, nil
// 	} else if reflectType.Kind() == reflect.Pointer {
// 		reflectType = reflectType.Elem()
// 	}

// 	if reflectType == nil {
// 		return nil, fmt.Errorf("json schema reflect: unknown Go type")
// 	}

// 	jsonType, ok := JSONTypeFromKind(reflectType.Kind())
// 	if ok {
// 		jsonSchema := &jsonschema.Schema{
// 			Type: jsonType.String(),
// 		}

// 		switch jsonType {
// 		case JSONInteger:
// 			// Add Go type annotation for integers
// 			if goFormat := GoTypeFormat(reflectType.Kind()); goFormat != "" {
// 				if jsonSchema.Extras == nil {
// 					jsonSchema.Extras = make(map[string]any)
// 				}
// 				jsonSchema.Extras["x-dtkt-format"] = goFormat
// 			}
// 		case JSONNumber:
// 			// Add Go type annotation for numbers
// 			if goFormat := GoTypeFormat(reflectType.Kind()); goFormat != "" {
// 				if jsonSchema.Extras == nil {
// 					jsonSchema.Extras = make(map[string]any)
// 				}
// 				jsonSchema.Extras["x-dtkt-format"] = goFormat
// 			}
// 		case JSONObject:
// 			switch reflectType.Kind() {
// 			case reflect.Map:
// 				if reflectType.Elem() == nil {
// 					return nil, fmt.Errorf("%s is not a valid JSON object", reflectType)
// 				} else if reflectType.Elem().Kind() != reflect.Interface {
// 					itemSchema, err := reflectJSONSchema(reflector, reflect.New(reflectType.Elem()).Interface())
// 					if err != nil {
// 						return nil, err
// 					}

// 					jsonSchema.ID = itemSchema.ID
// 					jsonSchema.Version = itemSchema.Version
// 					itemSchema.ID = ""
// 					itemSchema.Version = ""

// 					jsonSchema.AdditionalProperties = itemSchema
// 				}
// 			case reflect.Struct:
// 				if reflectType.NumField() > 0 {
// 					jsonSchema = reflector.ReflectFromType(reflectType)
// 				}
// 			}
// 		case JSONArray:
// 			if reflectType.Elem() == nil {
// 				return nil, fmt.Errorf("%s is not a valid JSON array", reflectType)
// 			} else if reflectType.Elem().Kind() != reflect.Interface {
// 				itemSchema, err := reflectJSONSchema(reflector, reflect.New(reflectType.Elem()).Interface())
// 				if err != nil {
// 					return nil, err
// 				}

// 				jsonSchema.ID = itemSchema.ID
// 				jsonSchema.Version = itemSchema.Version
// 				itemSchema.ID = ""
// 				itemSchema.Version = ""
// 				jsonSchema.Items = itemSchema
// 			}
// 		}

// 		return jsonSchema, nil
// 	}

// 	return nil, fmt.Errorf("%s cannot be represented as a json type", reflectType.Kind())
// }
