package flowsdk_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
)

var testSpec = `
apiVersion: v1beta1
kind: Flow
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

func TestSpec(t *testing.T) {
	err := v1beta1.DefaultTypeRegistry().LoadResolverTypes(api.V1Beta1)
	if err != nil {
		t.Fatal(err)
	}

	loader := flowsdk.SpecLoader()
	_, err = loader.MarshalJSONSchema()
	if err != nil {
		t.Fatal(err)
	}

	spec, err := loader.Decode(encoding.YAML, []byte(testSpec))
	if err != nil {
		t.Fatal(err)
	}

	_, err = loader.Encode(encoding.YAML, spec)
	if err != nil {
		t.Fatal(err)
	}
}
