package flowsdk_test

import (
	"net/url"
	"os"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
)

var completeFlowSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: User Action
  description: User action test flow.

  actions:
    - id: echo
      user:
        inputs:
          - id: value
            title: Echo
            input: {}

  outputs:
    - id: time
      value: = sources.tick.getValue()
    - id: foo
      value: = vars.foo.getValue()
`

var bareFlowSpec = `
spec:
  name: User Action
  description: User action test flow.

  actions:
    - id: echo
      user:
        inputs:
          - id: value
            title: Echo
            input: {}

  outputs:
    - id: time
      value: = sources.tick.getValue()
    - id: foo
      value: = vars.foo.getValue()
`

func TestSpec_Schema(t *testing.T) {
	registry := v1beta1.NewTypeRegistry(v1beta1.NewMemoryTypeSyncer(), &url.URL{
		Scheme: "mem",
		Path:   "/types",
	})
	specOpts := flowsdk.SpecOptions{
		Resolver: api.GlobalResolver(),
		Syncer:   registry.Syncer(),
	}
	loader := api.NewSpecLoader[*corev1.FlowSpec](
		common.WithJSONSchemaBaseURI(registry.BaseUri().String()),
		common.WithJSONSchemaID(registry.BaseUri().JoinPath(flowsdk.SpecSchemaFilename).String()),
		common.WithJSONSchemaLoader(registry.SchemaLoader()),
		common.WithJSONSchemaCallback(specOpts.ExtendSchema),
	)

	schema, err := loader.TypedSchema()
	if err != nil {
		t.Fatal(err)
	} else {
		os.WriteFile("/tmp/flowspec.json", schema.Bytes(), 0o600)
	}

	for id := range schema.Schema().Defs {
		t.Log(id)
	}

	spec, err := schema.ValidateAny(map[string]any{
		"kind":       "Flow",
		"apiVersion": "v1beta1",
		"spec": map[string]any{
			"name": "Test Flow",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = loader.Validate(spec)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSpec_BareSpec(t *testing.T) {
	loader := flowsdk.SpecLoader()

	spec, err := loader.Decode([]byte(bareFlowSpec), encoding.YAML)
	if err != nil {
		t.Fatal(err)
	}

	b, err := loader.Encode(spec, encoding.YAML)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(string(b))
	}
}

func TestSpec_CompleteSpec(t *testing.T) {
	loader := flowsdk.SpecLoader()

	spec, err := loader.Decode([]byte(completeFlowSpec), encoding.YAML)
	if err != nil {
		t.Fatal(err)
	}

	b, err := loader.Encode(spec, encoding.YAML)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(string(b))
	}
}
