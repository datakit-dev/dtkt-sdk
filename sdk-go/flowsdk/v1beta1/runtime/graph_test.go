package runtime_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	// "github.com/google/cel-go/common/types/ref"
)

var tickerFlow = `
kind: Flow
apiVersion: v1beta1
spec:
  name: Ticker
  inputs:
    - id: maxIters
      uint64:
        default: 10
      cache: true
  streams:
    - id: tick
      generate:
        every: 0.1s
        initial: 0s
  outputs:
    - id: tick
      value: = streams.tick.value
    - id: evenOrOdd
      value: |
        = int(streams.tick.count) % 2 == 0 ? "even" : "odd"
    - id: done
      value: |
        = streams.tick.count >= inputs.maxIters.value ? Done{reason:
        "Maximum iterations: %d".format([inputs.maxIters.value])} : null
`

func TestGraph_Execute(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(tickerFlow)
	flow, err := flowsdk.ReadSpec(&buf, encoding.YAML)
	if err != nil {
		t.Fatal(err)
	}

	graph, err := runtime.NewGraphFromSpec(flow.GetV1Beta1())
	if err != nil {
		t.Fatal(err)
	}

	// reduceProg, err := shared.CompileExpr(graph.Env(), "= streams.tick.value")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// reducer := streamflow.NewReduce(func(prev, curr time.Time) time.Time {
	// 	if curr.After(prev) {
	// 		return curr
	// 	}
	// 	return prev
	// })

	// go func() {
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 		case value, ok := <-reducer.Out():
	// 			if !ok {
	// 				return
	// 			}

	// 			log.FromCtx(ctx).Info("streams.tick", slog.Any("value", value))
	// 		}
	// 	}
	// }()

	err = graph.Execute(ctx,
		shared.WithInitCycle(func(ctx context.Context, t ...*runtime.Node) (map[string]any, error) {
			return map[string]any{
				"inputs.maxIters": 11,
			}, nil
		}),
		shared.WithAfterRecv(func(ctx context.Context, node *runtime.Node, value any) (err error) {
			proto := node.Proto()
			if proto.GetState() <= flowv1beta1.Node_STATE_PENDING {
				return fmt.Errorf("expected node state > pending, got: %s", proto.GetState())
			} else if proto.GetStartTime() == nil || proto.GetStartTime().AsTime().IsZero() {
				return fmt.Errorf("expected node start time to be set")
			} else if proto.GetFinishTime() == nil || proto.GetFinishTime().AsTime().IsZero() {
				return fmt.Errorf("expected node finish time to be set")
			}

			if strings.HasPrefix(node.Ident(), shared.OutputPrefix) {
				t.Logf("%s: %v (%T)", node.Ident(), value, value)
			}

			// if node.Ident() == "streams.tick" {
			// 	if t, ok := value.(time.Time); ok {
			// 		reducer.In() <- t
			// 	}
			// }

			return
		}),
	)
	if err != nil {
		// close(reducer.In())

		done, ok := runtime.IsDoneError(err)
		if ok && done.Proto().GetIsError() {
			t.Fatal(err)
		} else if ok {
			t.Log(done.Error())
		}
	}
}
