package runtime_test

import (
	"bytes"
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

const tickerSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: Ticker
  description: |
    Example Source Ticker emitting a timestamp or expression value
    every given duration with optional initial delay.

  inputs:
    - id: maxIters
      cache: true
      int64:
        default: 10

  streams:
    - id: tick
      generate:
        every: 0.25s
        # Optional: initial delay duration (defaults to above every duration)
        initial: 0s
        # Optional: return expression value instead of current time (default)
        # value: = actions.checkStockPrice.getValue()

  outputs:
    - id: tick
      value: = streams.tick.getValue()
    - id: error
      value: '=
        inputs.maxIters.getValue() <= 0 ?
          Done{reason: "inputs.maxIters must be > 0", is_error: true} : null'
    - id: success
      value: '=
        inputs.maxIters.getValue() > 0 && int(streams.tick.getCount()) > inputs.maxIters.getValue() ?
          Done{ reason: "%d iterations completed".format([inputs.maxIters.getValue()]) } : null'
`

func TestExec(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(tickerSpec)

	spec, err := flowsdk.ReadSpec(encoding.YAML, &buf)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	run := runtime.New(ctx, spec.GetFlow())
	graph, err := runtime.NewGraph(run)
	if err != nil {
		t.Fatal(err)
	}

	exec, err := runtime.NewExecutor(graph)
	if err != nil {
		t.Fatal(err)
	}

	for idx := range 15 {
		values, err := exec.EvalAndReset(run)
		if err != nil {
			t.Fatal(err)
		}

		done := util.SliceReduce(slices.Collect(maps.Values(values)), func(v any) (any, bool) {
			done, ok := v.(*flowv1beta1.Runtime_Done)
			return done, ok
		})
		if len(done) > 0 {
			t.Log(done[0])
			break
		}

		b, err := encoding.ToJSONV2(values)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%d values: %#v", idx, string(b))
	}
}
