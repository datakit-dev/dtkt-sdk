package flowsdk

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"buf.build/go/protovalidate"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/invopop/jsonschema"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	SpecAPIVersion     = api.V1Beta1
	SpecKind           = "Flow"
	SpecSchemaID       = "dtkt.flowsdk.v1beta1.Flow"
	SpecSchemaFilename = SpecSchemaID + v1beta1.JSONSchemaFileExt
)

type (
	Spec struct {
		flow *flowv1beta1.Flow
		raw  []byte
		opts SpecOptions
	}
	SpecOptions struct {
		Resolver shared.Resolver
		Syncer   v1beta1.TypeSyncer
	}
)

func SpecLoader(opts ...api.SpecLoaderOpt[*Spec]) *api.SpecLoader[*Spec] {
	spec := &Spec{
		opts: SpecOptions{
			Resolver: api.V1Beta1,
			Syncer:   v1beta1.DefaultTypeSyncer(),
		},
	}

	registry := v1beta1.DefaultTypeRegistry()
	opts = append([]api.SpecLoaderOpt[*Spec]{
		api.WithSchemaOpts[*Spec](
			common.WithSchemaCompiler(registry.Compiler()),
			common.WithSchemaID(registry.BaseUri().JoinPath(SpecSchemaFilename).String()),
			common.WithJSONSchemaCallback(spec.opts.ExtendSchema),
		),
	}, opts...)
	return api.NewLoader(spec, opts...)
}

func NewSpecWithFlow(flow *flowv1beta1.Flow) *Spec {
	return &Spec{
		flow: flow,
	}
}

func ReadSpec(format encoding.Format, reader io.Reader) (*Spec, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	spec, err := SpecLoader().Decode(format, raw)
	if err != nil {
		return nil, fmt.Errorf("failed to decode: %w", err)
	}

	spec.raw = raw

	return spec, nil
}

func WriteSpec(spec *Spec, format encoding.Format, writer io.Writer) (int, error) {
	b, err := SpecLoader().Encode(format, spec)
	if err != nil {
		return 0, err
	}
	spec.raw = b
	return writer.Write(b)
}

func (f *Spec) GetFlow() *flowv1beta1.Flow {
	return f.flow
}

func (f *Spec) GetRaw() []byte {
	return f.raw
}

func (*Spec) SpecKind() string {
	return SpecKind
}

func (*Spec) SpecID() string {
	return SpecSchemaID
}

func (*Spec) APIVersion() api.Version {
	return SpecAPIVersion
}

func (f *Spec) Validate() error {
	if f == nil {
		return fmt.Errorf("invalid spec: failed to load")
	} else if f.flow == nil {
		return fmt.Errorf("invalid spec: missing flow")
	}
	return protovalidate.Validate(f.flow)
}

func (s *Spec) Parse() (*flowv1beta1.Graph, error) {
	graph, err := runtime.SpecGraph(s.flow)
	if err != nil {
		return nil, err
	}
	return graph.Proto(), nil
}

func (f Spec) MarshalJSON() ([]byte, error) {
	return encoding.ToJSONV2(f.flow)
}

func (f *Spec) UnmarshalJSON(data []byte) error {
	flow := new(flowv1beta1.Flow)
	if err := encoding.FromJSONV2(data, flow); err != nil {
		return err
	}

	*f = Spec{
		flow: flow,
	}

	return nil
}

func (Spec) Filename() string {
	return SpecSchemaFilename
}

func (o SpecOptions) ExtendSchemaWithContext(ctx context.Context, schema *jsonschema.Schema) error {
	var (
		nodeSchemas = map[string]*jsonschema.Schema{}
		typeNodes   = map[string]shared.SpecNode{
			spec.GetIDPrefix((*flowv1beta1.Connection)(nil)): (*flowv1beta1.Connection)(nil),
			spec.GetIDPrefix((*flowv1beta1.Action)(nil)):     (*flowv1beta1.Action)(nil),
			spec.GetIDPrefix((*flowv1beta1.Stream)(nil)):     (*flowv1beta1.Stream)(nil),
		}
	)

	if schema.Definitions == nil {
		schema.Definitions = jsonschema.Definitions{}
	}

	for prefix, node := range typeNodes {
		typeName := string(node.ProtoReflect().Descriptor().FullName())
		typeSchema, err := o.Syncer.GetType(typeName)
		if err != nil {
			return fmt.Errorf("failed to load type schema %s: %s", typeName, err)
		} else {
			b, err := encoding.ToJSONV2(typeSchema.JsonSchema)
			if err != nil {
				return err
			}

			var nodeSchema jsonschema.Schema
			err = encoding.FromJSONV2(b, &nodeSchema)
			if err != nil {
				return err
			}

			switch node := node.(type) {
			case *flowv1beta1.Connection:
				if services, ok := nodeSchema.Properties.Load("services"); ok && services.Items != nil {
					var names []string
					o.Resolver.RangeServices(func(sd protoreflect.ServiceDescriptor) bool {
						name := string(sd.FullName())
						if !slices.Contains(names, name) {
							names = append(names, name)
						}
						return true
					})
					services.Items.Enum = util.AnySlice(names)
				}
			case *flowv1beta1.Action, *flowv1beta1.Stream:
				typeSchema, err := o.Syncer.GetType("dtkt.flow.v1beta1.MethodCall")
				if err != nil {
					return fmt.Errorf("failed to load service call schema: %s", err)
				}

				b, err := encoding.ToJSONV2(typeSchema.JsonSchema)
				if err != nil {
					return err
				}

				var callSchema jsonschema.Schema
				err = encoding.FromJSONV2(b, &callSchema)
				if err != nil {
					return err
				}

				if callProp, ok := nodeSchema.Properties.Load("call"); ok {
					if methodProp, ok := callSchema.Properties.Load("method"); ok {
						methodProp.Enum = util.AnySlice(spec.ValidCallNodeMethods(o.Resolver, node))
					}
					schema.Definitions[prefix+".call"] = &callSchema
					callProp.Ref = "#/$defs/" + prefix + ".call"
				}
				callSchema.ID = ""
				callSchema.Version = ""
			}

			nodeSchemas[prefix] = &nodeSchema
		}
	}

	typeSchema, err := o.Syncer.GetType("dtkt.flow.v1beta1.Flow")
	if err != nil {
		return fmt.Errorf("failed to load schema: dtkt.flow.v1beta1.Flow")
	}

	b, err := encoding.ToJSONV2(typeSchema.JsonSchema)
	if err != nil {
		return err
	}

	var specSchema jsonschema.Schema
	err = encoding.FromJSONV2(b, &specSchema)
	if err != nil {
		return err
	}

	specSchema.ID = ""
	specSchema.Version = ""

	for prefix, nodeSchema := range nodeSchemas {
		schema.Definitions[prefix] = nodeSchema

		if base, ok := specSchema.Properties.Load(prefix); ok {
			if base.Items != nil && strings.HasSuffix(nodeSchema.ID.String(), base.Items.Ref) {
				base.Items.Ref = "#/$defs/" + prefix
			} else {
				return fmt.Errorf("node schema ref missing for prefix: %s", prefix)
			}
		} else {
			return fmt.Errorf("node schema missing for prefix: %s", prefix)
		}

		nodeSchema.ID = ""
		nodeSchema.Version = ""
	}

	schema.Definitions["Spec"] = &specSchema

	return nil
}

func (o SpecOptions) ExtendSchema(schema *jsonschema.Schema) error {
	return o.ExtendSchemaWithContext(context.Background(), schema)
}
