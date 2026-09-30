package api

import (
	"fmt"
	"slices"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/protoschema"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/jsonschema-go/jsonschema"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	// SpecLoader encapsulate multiple API versions and SDK-compatible types for
	// uniform encoding, decoding, and validation. Primarily, the Go SDK expects
	// an opaque proto with the following fields:
	//  - `kind`: string with default and buf validate const set to expected kind;
	//  - `apiVersion`: string with buf validate `in` of compatible API versions;
	//  - `spec`: google.protobuf.Struct (JSON) with INPUT_ONLY field behavior;
	//  - `version`: a oneof with field names corresponding with the compatible
	// 		versions of the annotated `apiVersion` field validation values.
	// During runtime validation, the `spec` JSON is transcoded into the oneof
	// field corresponding with the value of the `apiVersion` field.
	SpecLoader[T SpecType] struct {
		specType     T
		apiVersions  []string
		specVersions map[string]SpecVersion

		kindDefault,
		apiVersionDefault string

		typedSchema *common.JSONSchema[T]
		schemaOpts  common.JSONSchemaOptions
	}
	// SpecType enforces the proto message must be generated using Go Opaque API
	// which provides a richer set of accessors making them ideal for ergonomic
	// and uniform spec message shapes having kind, apiVersion, spec, and version.
	SpecType interface {
		proto.Message

		ClearApiVersion()
		ClearKind()
		ClearSpec()
		ClearVersion()

		HasApiVersion() bool
		HasKind() bool
		HasSpec() bool
		HasVersion() bool

		GetApiVersion() string
		GetKind() string
		GetSpec() *structpb.Struct

		SetApiVersion(string)
		SetKind(string)
		SetSpec(*structpb.Struct)
	}
	// SpecVersion contains a field from the SpecType's oneof `version` field
	// and a mutable value of its type.
	SpecVersion struct {
		field protoreflect.FieldDescriptor
		value protoreflect.Value
	}
)

func NewSpecLoader[T SpecType](schemaOpts ...common.JSONSchemaOption) *SpecLoader[T] {
	loader := &SpecLoader[T]{
		specVersions: map[string]SpecVersion{},
	}

	for _, opt := range schemaOpts {
		if opt != nil {
			opt(&loader.schemaOpts)
		}
	}

	// Opaque API Get* methods return default value if default field option is set.
	loader.kindDefault = loader.specType.GetKind()
	loader.apiVersionDefault = loader.specType.GetApiVersion()

	specType := loader.specType.ProtoReflect().Descriptor()
	for idx := range specType.Fields().Len() {
		field := specType.Fields().Get(idx)

		switch field.Name() {
		case "kind":
			if loader.kindDefault == "" && field.HasDefault() {
				loader.kindDefault = field.Default().String()
			}
		case "api_version":
			if loader.apiVersionDefault == "" && field.HasDefault() {
				loader.apiVersionDefault = field.Default().String()
			}

			if opts, ok := field.Options().(*descriptorpb.FieldOptions); ok && opts != nil && proto.HasExtension(opts, validate.E_Field) {
				if rules, ok := proto.GetExtension(opts, validate.E_Field).(*validate.FieldRules); ok && len(rules.GetString().GetIn()) > 0 {
					loader.apiVersions = rules.GetString().GetIn()
				}
			}
		}
	}

	if version := specType.Oneofs().ByName("version"); version != nil {
		versions := util.IntMap(version.Fields().Len(), func(idx int) SpecVersion {
			field := version.Fields().Get(idx)
			return SpecVersion{
				field: field,
				value: loader.specType.ProtoReflect().NewField(field),
			}
		})

		if len(loader.apiVersions) == 0 {
			loader.apiVersions = util.SliceOfFunc(versions, func(v SpecVersion) string {
				return v.String()
			})
		}

		loader.specVersions = util.SliceReduceToMap(versions,
			func(_ int, v SpecVersion) (name string, version SpecVersion, ok bool) {
				name = v.String()
				version = v
				ok = slices.Contains(loader.apiVersions, name)
				return
			},
		)
	}

	return loader
}

func (l *SpecLoader[T]) MarshalJSONSchema() ([]byte, error) {
	schema, err := l.TypedSchema()
	if err != nil {
		return nil, err
	}

	return schema.Bytes(), nil
}

func (v SpecVersion) String() string {
	return string(v.field.Name())
}

func (l *SpecLoader[T]) TypedSchema() (_ *common.JSONSchema[T], err error) {
	if l.typedSchema == nil {
		resolved, err := protoschema.ResolvedJSONSchemaWithOptions(
			l.specType.ProtoReflect().Descriptor(),
			&l.schemaOpts.ResolveOptions,
		)
		if err != nil {
			return nil, err
		}

		root := resolved.Schema().CloneSchemas()
		props := root.Properties
		if len(props) > 0 {
			var (
				specSchema     *jsonschema.Schema
				versionSchemas = map[string]*jsonschema.Schema{}
			)

			for field, schema := range props {
				switch {
				case field == "spec":
					specSchema = schema
				case slices.Contains(l.apiVersions, field):
					versionSchemas[field] = schema
				case slices.Contains(util.MapKeys(l.specVersions), field):
					versionSchemas[field] = schema
				}
			}

			if specSchema != nil && len(versionSchemas) > 0 {
				// Protobuf JSON name is probably "apiVersion", but tolerate either.
				apiVersionField := "apiVersion"
				if _, ok := props[apiVersionField]; !ok {
					if _, ok := props["api_version"]; ok {
						apiVersionField = "api_version"
					}
				}

				// Remove SDK/internal oneof version fields from the public JSON Schema.
				for field := range versionSchemas {
					delete(props, field)

					root.Required = removeString(root.Required, field)
					root.PropertyOrder = removeString(root.PropertyOrder, field)
				}

				// Keep the original spec property as the generic fallback/annotation shell.
				// The allOf conditionals below will narrow it based on apiVersion.
				props["spec"] = &jsonschema.Schema{
					Type: "object",
				}

				// For each api version:
				//
				//   if apiVersion == "v1beta1"
				//   then properties.spec = <schema formerly under v1beta1>
				//
				// Note: "required: [apiVersion]" inside if is important. Without it,
				// the `if` schema would match when apiVersion is absent.
				for version, schema := range versionSchemas {
					root.AllOf = append(root.AllOf, &jsonschema.Schema{
						If: &jsonschema.Schema{
							Properties: map[string]*jsonschema.Schema{
								apiVersionField: {
									Const: new(any(version)),
								},
							},
							Required: []string{apiVersionField},
						},
						Then: &jsonschema.Schema{
							Properties: map[string]*jsonschema.Schema{
								"spec": schema.CloneSchemas(),
							},
						},
					})
				}
			}
		}

		schemaBytes, err := root.MarshalJSON()
		if err != nil {
			return nil, err
		}

		l.schemaOpts.Bytes = schemaBytes

		typedSchema, err := common.JSONSchemaWithOptionsFor[T](&l.schemaOpts)
		if err != nil {
			return nil, err
		}

		l.typedSchema = typedSchema
	}

	return l.typedSchema, nil
}

func (l *SpecLoader[T]) Validate(inst T) error {
	if !inst.HasVersion() {
		err := l.structToVersion(inst)
		if err != nil {
			return err
		}
	}

	return protovalidate.Validate(inst)
}

func (l *SpecLoader[T]) Decode(data []byte, format encoding.Format) (inst T, err error) {
	inst = l.specType.ProtoReflect().New().Interface().(T)
	err = format.Decode(data, inst)
	if err != nil {
		return
	}

	err = l.Validate(inst)
	if err != nil {
		return
	}

	return inst, nil
}

func (l *SpecLoader[T]) Encode(inst T, format encoding.Format) ([]byte, error) {
	inst = proto.CloneOf(inst)

	err := l.Validate(inst)
	if err != nil {
		return nil, err
	}

	if !inst.HasSpec() {
		err = l.versionToStruct(inst)
		if err != nil {
			return nil, err
		}
	}

	return format.Encode(inst)
}

func (l *SpecLoader[T]) versionToStruct(inst T) error {
	if !inst.HasKind() {
		inst.SetKind(l.kindDefault)
	}

	if !inst.HasApiVersion() {
		inst.SetApiVersion(l.apiVersionDefault)
	}

	err := ValidVersion(inst.GetApiVersion())
	if err != nil {
		return err
	}

	version, ok := l.specVersions[inst.GetApiVersion()]
	if !ok {
		return fmt.Errorf("invalid api version: %s", inst.GetApiVersion())
	}

	if version.field.Message() != nil {
		specData, err := encoding.ToJSON(inst.ProtoReflect().Get(version.field).Message().Interface())
		if err != nil {
			return err
		}

		specStruct := new(structpb.Struct)
		err = encoding.FromJSON(specData, specStruct)
		if err != nil {
			return err
		}

		inst.SetSpec(specStruct)
		inst.ClearVersion()
	}

	return nil
}

func (l *SpecLoader[T]) structToVersion(inst T) error {
	if !inst.HasKind() {
		inst.SetKind(l.kindDefault)
	}

	if !inst.HasApiVersion() {
		inst.SetApiVersion(l.apiVersionDefault)
	}

	err := ValidVersion(inst.GetApiVersion())
	if err != nil {
		return err
	}

	version, ok := l.specVersions[inst.GetApiVersion()]
	if !ok {
		return fmt.Errorf("invalid api version: %s", inst.GetApiVersion())
	}

	if version.field.Message() != nil {
		specData, err := inst.GetSpec().MarshalJSON()
		if err != nil {
			return err
		}

		specVersion := version.value.Message().New()
		err = encoding.FromJSON(specData, specVersion.Interface())
		if err != nil {
			return err
		}

		inst.ProtoReflect().Set(version.field, protoreflect.ValueOfMessage(specVersion))
		inst.ClearSpec()
	}

	return nil
}

func removeString(xs []string, x string) (out []string) {
	if len(xs) == 0 {
		return xs
	}

	for _, y := range xs {
		if y != x {
			out = append(out, y)
		}
	}
	return out
}
