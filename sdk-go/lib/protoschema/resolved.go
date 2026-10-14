package protoschema

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func NewJSONSchema(desc protoreflect.MessageDescriptor) (*jsonschema.Schema, error) {
	b, err := JSONSchemaBytes(desc)
	if err != nil {
		return nil, err
	}

	var schema jsonschema.Schema
	err = json.Unmarshal(b, &schema)
	if err != nil {
		return nil, err
	}

	return &schema, nil
}

func NewJSONSchemaMap(desc protoreflect.MessageDescriptor) (map[string]any, error) {
	gen := NewGenerator(
		WithJSONNames(),
	)

	err := gen.Add(desc)
	if err != nil {
		return nil, err
	}

	genSchemas := gen.Generate()

	schemaMap, ok := genSchemas[desc.FullName()]
	if !ok {
		return nil, fmt.Errorf("failed to generate schema for: %s", desc.FullName())
	}

	return schemaMap, nil
}

func JSONSchemaBytes(desc protoreflect.MessageDescriptor) ([]byte, error) {
	schemaMap, err := NewJSONSchemaMap(desc)
	if err != nil {
		return nil, err
	}

	return json.Marshal(schemaMap)
}

func ResolvedJSONSchema(desc protoreflect.MessageDescriptor) (*jsonschema.Resolved, error) {
	return ResolvedJSONSchemaWithOptions(desc, &jsonschema.ResolveOptions{})
}

func ResolvedJSONSchemaWithOptions(desc protoreflect.MessageDescriptor, opts *jsonschema.ResolveOptions) (*jsonschema.Resolved, error) {
	gen := NewGenerator(
		WithJSONNames(),
		WithBundle(),
	)

	err := gen.Add(desc)
	if err != nil {
		return nil, err
	}

	schemaMap := gen.Generate()

	descSchema, ok := schemaMap[desc.FullName()]
	if !ok {
		return nil, fmt.Errorf("failed to generate schema for: %s", desc.FullName())
	}

	if opts == nil {
		opts = &jsonschema.ResolveOptions{}
	}

	if opts.BaseURI == "" {
		opts.BaseURI = "file://"
	}

	if ref, ok := descSchema["$ref"].(string); ok {
		ref = strings.TrimPrefix(ref, defsPrefix)
		if defs, ok := descSchema["$defs"].(map[string]any); ok {
			for name, def := range defs {
				if name == ref {
					continue
				} else if def, ok := def.(map[string]any); ok {
					delete(def, "$schema")
				}
			}

			if def, ok := defs[ref].(map[string]any); ok {
				delete(defs, ref)
				def["$defs"] = defs
				descSchema = def

				uri, err := url.Parse(fmt.Sprintf("%s/%s", opts.BaseURI, ref))
				if err != nil {
					return nil, err
				}
				descSchema["$id"] = uri.String()
			}
		}
	}

	b, err := json.Marshal(descSchema)
	if err != nil {
		return nil, err
	}

	var schema jsonschema.Schema
	err = json.Unmarshal(b, &schema)
	if err != nil {
		return nil, err
	}

	return schema.Resolve(opts)
}
