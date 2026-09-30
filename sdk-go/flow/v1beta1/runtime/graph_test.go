package runtime_test

import (
	"bytes"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/v1beta1/runtime"
)

func TestParseGraph(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(tickerFlow)

	spec, err := flow.ReadSpec(&buf, encoding.YAML)
	if err != nil {
		t.Fatal(err)
	}

	graph, err := runtime.ParseGraph(spec.GetV1Beta1())
	if err != nil {
		t.Fatal(err)
	}

	if !graph.Parsed() {
		t.Fatal("expected graph to be parsed")
	}

	err = graph.Compile(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if !graph.Compiled() {
		t.Fatal("expected graph to be compiled")
	}
}
