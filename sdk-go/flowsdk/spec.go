package flowsdk

import (
	"fmt"
	"io"
	"maps"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/impl"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/protoschema"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"

	"github.com/google/jsonschema-go/jsonschema"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	SpecSchemaID       = "dtkt.core.v1.FlowSpec"
	SpecSchemaFilename = SpecSchemaID + v1beta1.JSONSchemaFileExt

	exprSchemaId = "CELExpr"
)

var (
	connectionNode *flowv1beta1.Connection
	actionNode     *flowv1beta1.Action
	actionCallNode *flowv1beta1.Action_MethodCall
	streamNode     *flowv1beta1.Stream
	streamCallNode *flowv1beta1.Stream_MethodCall
	exprSchemaRef  = jsonschema.Schema{
		Ref: "#/$defs/" + exprSchemaId,
	}
	exprSchema = &jsonschema.Schema{
		Description: "CEL expression.",
		Type:        "string",
		Pattern:     "^\\s?=\\s?",
	}
)

type SpecOptions struct {
	Resolver shared.Resolver
	Syncer   v1beta1.TypeSyncer

	schemaCache map[string]*jsonschema.Schema
}

func SpecLoader(opts ...common.JSONSchemaOption) *api.SpecLoader[*corev1.FlowSpec] {
	return api.NewSpecLoader[*corev1.FlowSpec](opts...)
}

func ReadSpec(reader io.Reader, format encoding.Format, opts ...common.JSONSchemaOption) (*corev1.FlowSpec, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	return SpecLoader(opts...).Decode(data, format)
}

func WriteSpec(writer io.Writer, format encoding.Format, spec *corev1.FlowSpec, opts ...common.JSONSchemaOption) error {
	b, err := SpecLoader(opts...).Encode(spec, format)
	if err != nil {
		return err
	}

	_, err = writer.Write(b)

	return err
}

func (o *SpecOptions) ExtendSchema(schema *jsonschema.Schema, opts *jsonschema.ResolveOptions) error {
	if o.schemaCache == nil {
		o.schemaCache = map[string]*jsonschema.Schema{}
	}

	if schema.Defs == nil {
		schema.Defs = map[string]*jsonschema.Schema{}
	}

	schema.Defs[exprSchemaId] = exprSchema

	if o.Resolver != nil {
		var (
			nodeTypes = []string{
				string(connectionNode.ProtoReflect().Descriptor().FullName()),
				string(actionCallNode.ProtoReflect().Descriptor().FullName()),
				string(streamCallNode.ProtoReflect().Descriptor().FullName()),
			}
		)

		for _, nodeType := range nodeTypes {
			schemaId := nodeType + v1beta1.JSONSchemaFileExt
			if nodeSchema, ok := schema.Defs[schemaId]; ok {
				nodeDefs := o.extendNodeSchema(nodeType, nodeSchema, opts)

				maps.Copy(schema.Defs, o.schemaCache)
				maps.DeleteFunc(nodeDefs, func(id string, schema *jsonschema.Schema) bool {
					_, exists := schema.Defs[id]
					schema.ID = ""
					schema.Schema = ""
					return exists
				})
				maps.Copy(schema.Defs, nodeDefs)
			}
		}
	}

	return nil
}

func (o *SpecOptions) extendNodeSchema(nodeType string, nodeSchema *jsonschema.Schema, opts *jsonschema.ResolveOptions) (defs map[string]*jsonschema.Schema) {
	if o.Resolver != nil {
		switch nodeType {
		case string(connectionNode.ProtoReflect().Descriptor().FullName()):
			if services, ok := nodeSchema.Properties["services"]; ok && services.Items != nil {
				services.Items.Enum = util.AnySlice(impl.ValidConnectionServices(o.Resolver))
			}
		case string(actionCallNode.ProtoReflect().Descriptor().FullName()):
			return o.extendCallNodeSchema(actionNode, nodeSchema, opts)
		case string(streamCallNode.ProtoReflect().Descriptor().FullName()):
			return o.extendCallNodeSchema(streamNode, nodeSchema, opts)
		}
	}
	return
}

func (o *SpecOptions) extendCallNodeSchema(node shared.SpecNode, schema *jsonschema.Schema, opts *jsonschema.ResolveOptions) (defs map[string]*jsonschema.Schema) {
	if o.Resolver == nil || len(schema.Properties) == 0 {
		return
	}

	defs = map[string]*jsonschema.Schema{}
	impl.RangeCallNodeMethods(o.Resolver, node, func(md protoreflect.MethodDescriptor) bool {
		resolved, err := protoschema.ResolvedJSONSchemaWithOptions(md.Input(), opts)
		if err != nil {
			return true
		}

		reqName := string(md.Input().FullName())
		reqId := reqName + v1beta1.JSONSchemaFileExt
		reqSchema := o.extendMessageSchema(resolved.Schema().CloneSchemas())

		schema.AllOf = append(schema.AllOf, &jsonschema.Schema{
			If: &jsonschema.Schema{
				Properties: map[string]*jsonschema.Schema{
					"method": {
						Const: new(any(string(md.FullName()))),
					},
				},
				Required: []string{"method"},
			},
			Then: &jsonschema.Schema{
				Properties: map[string]*jsonschema.Schema{
					"request": {
						AnyOf: []*jsonschema.Schema{
							new(exprSchemaRef),
							{Ref: "#/$defs/" + reqId},
						},
					},
				},
			},
		})

		maps.Copy(defs, reqSchema.Defs)

		reqSchema.Defs = nil
		defs[reqId] = reqSchema

		return true
	})

	return
}

func (o *SpecOptions) extendMessageSchema(schema *jsonschema.Schema) *jsonschema.Schema {
	if len(schema.Properties) > 0 {
		props := map[string]*jsonschema.Schema{}
		for fieldName, fieldSchema := range schema.Properties {
			props[fieldName] = o.extendFieldSchema(fieldSchema)
		}
		maps.Copy(schema.Properties, props)
	}
	return schema
}

func (o *SpecOptions) extendFieldSchema(schema *jsonschema.Schema) *jsonschema.Schema {
	switch {
	case schema.Ref != "":
		schema.Ref = "#/$defs/" + o.getRefOrExprId(schema)
	case schema.Type == "array" && schema.Items != nil:
		if schema.Items.Ref != "" {
			schema.Items.Ref = "#/$defs/" + o.getRefOrExprId(schema.Items)
		} else {
			schema.Items.AnyOf = append(schema.Items.AnyOf,
				new(exprSchemaRef),
				new(*schema.Items),
			)
		}

		schema.AnyOf = append(schema.AnyOf,
			new(exprSchemaRef),
			&jsonschema.Schema{
				Type:  schema.Type,
				Items: schema.Items,
			},
		)

		schema.Type = ""
		schema.Items = nil
	case schema.Type == "object" && schema.PropertyNames != nil && schema.AdditionalProperties != nil:
		if schema.AdditionalProperties.Ref != "" {
			schema.AdditionalProperties.Ref = "#/$defs/" + o.getRefOrExprId(schema.AdditionalProperties)
		} else {
			schema.AdditionalProperties.AnyOf = append(schema.AdditionalProperties.AnyOf,
				new(exprSchemaRef),
				new(*schema.AdditionalProperties),
			)
		}

		schema.AnyOf = append(schema.AnyOf,
			new(exprSchemaRef),
			&jsonschema.Schema{
				Type:                 schema.Type,
				PropertyNames:        schema.PropertyNames,
				AdditionalProperties: schema.AdditionalProperties,
			},
		)

		schema.Type = ""
		schema.PropertyNames = nil
		schema.AdditionalProperties = nil
	case len(schema.AnyOf) > 0:
		schema.AnyOf = append(schema.AnyOf, new(exprSchemaRef))
	case schema.Type != "":
		schema.AnyOf = append(schema.AnyOf,
			new(exprSchemaRef),
			&jsonschema.Schema{Type: schema.Type},
		)
		schema.Type = ""
	}

	return schema
}

func (o *SpecOptions) getRefOrExprId(schema *jsonschema.Schema) string {
	id := "CELExprOr_" + strings.TrimSuffix(
		strings.TrimPrefix(schema.Ref, "#/$defs/"),
		v1beta1.JSONSchemaFileExt,
	)

	def, ok := o.schemaCache[id]
	if !ok {
		def = newRefOrExprSchema(schema)
		o.schemaCache[id] = def
	}

	return id
}

func newRefOrExprSchema(schema *jsonschema.Schema) *jsonschema.Schema {
	return &jsonschema.Schema{
		AnyOf: []*jsonschema.Schema{
			new(exprSchemaRef),
			{Ref: schema.Ref},
		},
	}
}
