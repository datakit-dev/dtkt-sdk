package protoschema

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/google/jsonschema-go/jsonschema"
	"google.golang.org/protobuf/reflect/protoreflect"
)

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

	if opts.BaseURI == "" {
		opts.BaseURI = "file://"
	}

	if ref, ok := descSchema["$ref"].(string); ok {
		ref = strings.TrimPrefix(ref, "#/$defs/")
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

	b, err := encoding.ToJSONV2(descSchema)
	if err != nil {
		return nil, err
	}

	var schema jsonschema.Schema
	err = encoding.FromJSONV2(b, &schema)
	if err != nil {
		return nil, err
	}

	return schema.Resolve(opts)
}

func ResolvedJSONSchema(desc protoreflect.MessageDescriptor) (*jsonschema.Resolved, error) {
	return ResolvedJSONSchemaWithOptions(desc, &jsonschema.ResolveOptions{})
}
