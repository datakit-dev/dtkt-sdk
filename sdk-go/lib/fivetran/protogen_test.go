package fivetran

// import (
// 	_ "embed"
// 	"net/http"
// 	"regexp"
// 	"strings"
// 	"testing"

// 	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/protoschema"
// 	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
// 	"github.com/getkin/kin-openapi/openapi3"
// 	"github.com/go-openapi/inflect"
// 	"github.com/jhump/protoreflect/v2/protoprint"
// 	"google.golang.org/protobuf/reflect/protodesc"
// 	"google.golang.org/protobuf/reflect/protoregistry"
// 	"google.golang.org/protobuf/types/descriptorpb"
// )

// //go:embed openapi/openapiv1.yaml
// var oapiBytes []byte

// var pathParamPattern = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// type (
// 	resource struct {
// 		name, path string
// 		params     []string
// 	}
// 	operation struct {
// 		resource
// 		id      string
// 		method  string
// 		request []byte
// 	}
// )

// func (r resource) ServiceName() string {
// 	return r.name + "Service"
// }

// func (o operation) MethodName() string {
// 	name := util.ToPascalCase(o.id)
// 	switch o.method {
// 	case http.MethodGet:
// 		// if !strings.HasPrefix(name, "Get") && !strings.HasPrefix(name, "List") && !strings.HasSuffix(name, "List") {
// 		// 	name = "Get" + name
// 		// }

// 		// if cut, ok := strings.CutSuffix(name, "List"); ok || inflect.Pluralize(name) == name {
// 		// 	if !strings.HasPrefix(name, "List") {
// 		// 		name = "List" + cut
// 		// 	}
// 		// }
// 	case http.MethodPost:
// 		// strings.TrimSuffix(name, "Create")
// 		// if !strings.HasPrefix(name, "Create") {
// 		// 	name = "Create" + name
// 		// }
// 	case http.MethodPatch:
// 		// if cut, ok := strings.CutSuffix(name, "Update"); ok {
// 		// 	if !strings.HasPrefix(name, "Update") {
// 		// 		name = "Update" + cut
// 		// 	}
// 		// }
// 	}

// 	return name
// }

// func resourceFromPathPattern(pattern, version string) (res resource, ok bool) {
// 	segments := strings.Split(strings.Trim(pattern, "/"), "/")
// 	if len(segments) > 0 && segments[0] == version {
// 		segments = segments[1:]
// 	}

// 	if len(segments) == 0 {
// 		return
// 	}

// 	for i := 0; i < len(segments)-1; i++ {
// 		if pathParamPattern.MatchString(segments[i+1]) {
// 			res.params = append(res.params, segments[i+1])
// 			i++
// 		}
// 	}

// 	res.name = util.ToPascalCase(inflect.Singularize(segments[0]))
// 	res.path = pattern
// 	ok = true

// 	return
// }

// func TestOAPI(t *testing.T) {
// 	oapiSchema, err := openapi3.NewLoader().LoadFromData(oapiBytes)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	name := "NewDestinationRequest"
// 	schema, err := oapiSchema.Components.Schemas.JSONLookup("NewDestinationRequest")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	resourceOperations := map[string][]operation{}
// 	for _, pattern := range oapiSchema.Paths.InMatchingOrder() {
// 		res, ok := resourceFromPathPattern(pattern, oapiSchema.Info.Version)
// 		if !ok {
// 			continue
// 		}

// 		for method, op := range oapiSchema.Paths.Find(pattern).Operations() {
// 			var request []byte
// 			if op.RequestBody != nil && op.RequestBody.Value != nil && op.RequestBody.Value.Content != nil {
// 				mt := op.RequestBody.Value.Content.Get("application/json")
// 				if mt != nil && mt.Schema != nil && mt.Schema.Value != nil {
// 					request, _ = mt.Schema.Value.MarshalJSON()
// 				}
// 			}

// 			resourceOperations[res.name] = append(resourceOperations[res.name], operation{
// 				resource: res,
// 				id:       op.OperationID,
// 				method:   method,
// 				request:  request,
// 			})
// 			// op.Responses.MarshalJSON()
// 		}
// 	}

// 	for _, operations := range resourceOperations {
// 		for idx, operation := range operations {
// 			if idx == 0 {
// 				t.Log(operation.ServiceName())
// 			}

// 			t.Logf("\t%s [%s %s]", operation.MethodName(), operation.method, operation.path)
// 			// t.Logf("\t%s", string(operation.request))
// 		}
// 	}

// 	var (
// 		schemaBytes []byte
// 		// schemaDiscr *openapi3.Discriminator
// 	)

// 	switch schema := schema.(type) {
// 	case *openapi3.Schema:
// 		// schemaDiscr = schema.Discriminator
// 		schema.Discriminator = nil

// 		schemaBytes, err = schema.MarshalJSON()
// 		if err != nil {
// 			t.Fatalf("marshal schema %s json: %s", name, err)
// 		}
// 	case *openapi3.Ref:
// 		// TODO: handle Ref
// 	}

// 	// if schemaDiscr != nil {
// 	// 	var jsonSchema jsonschema.Schema
// 	// 	err = json.Unmarshal(schemaBytes, &jsonSchema)
// 	// 	if err != nil {
// 	// 		t.Fatalf("unmarshal schema %s json: %s", name, err)
// 	// 	}

// 	// 	discrProp, ok := jsonSchema.Properties.Get(schemaDiscr.PropertyName)
// 	// 	if !ok {
// 	// 		t.Fatalf("schema %s missing property: %s", name, schemaDiscr.PropertyName)
// 	// 	}

// 	// 	if jsonSchema.Definitions == nil {
// 	// 		jsonSchema.Definitions = jsonschema.Definitions{}
// 	// 	}

// 	// 	for key, ref := range schemaDiscr.Mapping {
// 	// 		discrProp.Enum = append(discrProp.Enum, key)

// 	// 		refSchema, err := oapiSchema.Components.Schemas.JSONLookup(path.Base(ref))
// 	// 		if err != nil {
// 	// 			t.Fatalf("resolve discriminator ref %s: %s", key, err)
// 	// 		}

// 	// 		if refSchema, ok := refSchema.(*openapi3.Schema); ok {
// 	// 			switch {
// 	// 			case refSchema.AllOf != nil:
// 	// 				for _, ref := range refSchema.AllOf {
// 	// 					if path.Base(ref.Ref) == name {
// 	// 						continue
// 	// 					}

// 	// 					b, err := json.Marshal(ref.Value)
// 	// 					if err != nil {
// 	// 						t.Fatalf("marshal discriminator schema %s: %s", ref.Ref, err)
// 	// 					}

// 	// 					var subSchema jsonschema.Schema
// 	// 					err = json.Unmarshal(b, &subSchema)
// 	// 					if err != nil {
// 	// 						t.Fatalf("unmarshal discriminator schema %s: %s", ref.Ref, err)
// 	// 					}

// 	// 					jsonSchema.Definitions[path.Base(ref.Ref)] = &subSchema
// 	// 				}
// 	// 			}
// 	// 		}
// 	// 	}
// 	// 	// schemaDiscr.Mapping

// 	// 	schemaBytes, err = json.Marshal(jsonSchema)
// 	// 	if err != nil {
// 	// 		t.Fatalf("marshal schema %s json: %s", name, err)
// 	// 	}
// 	// }

// 	t.Log(string(schemaBytes))

// 	fileProto, err := protoschema.NewParser(protoschema.ParserOptions{
// 		PackageName:      "dtkt.lib.fivetran.v1",
// 		MessageName:      name,
// 		EnableValidation: true,
// 		UseJSONNames:     true,
// 	}).Parse(schemaBytes)
// 	if err != nil {
// 		t.Fatalf("protoschema parse: %s", err)
// 	}

// 	fileProto.Name = new(util.ToSnakeCase(name) + ".proto")
// 	if fileProto.Options == nil {
// 		fileProto.Options = new(descriptorpb.FileOptions)
// 	}

// 	fileProto.Options.GoPackage = new("github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/lib/fivetran/v1;fivetranv1")

// 	file, err := protodesc.NewFile(fileProto, protoregistry.GlobalFiles)
// 	if err != nil {
// 		t.Fatalf("protodesc new file: %s", err)
// 	}

// 	printer := protoprint.Printer{
// 		SortElements: true,
// 	}

// 	str, err := printer.PrintProtoToString(file)
// 	if err != nil {
// 		t.Fatalf("protoprint print to string: %s", err)
// 	}

// 	t.Log(str)
// }
