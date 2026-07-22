package runtime_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime"
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
  vars:
    - id: tickCount
      value: = "%d ticks".format([streams.tick.count])
  outputs:
    - id: tick
      value: = streams.tick.value
    - id: iters
      value: = streams.tick.count
    - id: evenOrOdd
      value: |
        = int(streams.tick.count) % 2 == 0 ? "even" : "odd"
    - id: done
      value: |
        = streams.tick.count >= inputs.maxIters.value ? Done{reason:
        "Maximum iterations: %d".format([inputs.maxIters.value])} : null
`

func TestExecutor_Ticker(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(tickerFlow)
	flow, err := flowsdk.ReadSpec(&buf, encoding.YAML)
	if err != nil {
		t.Fatal(err)
	}

	exec, err := runtime.ExecutorFromSpec(t.Context(), flow.GetV1Beta1())
	if err != nil {
		t.Fatal(err)
	}

	// Intentionally timeout after 10 ticks (10 remaining)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	var totalIters atomic.Int32
	err = exec.Run(ctx, func(err error) {
		cancel()
	},
		shared.WithInitStart(func(context.Context, shared.Nodes[*runtime.Node]) (map[string]any, error) {
			return map[string]any{
				// Set maxIters to 20 (10 remaining after timeout)
				"inputs.maxIters": 20,
			}, nil
		}),
		shared.WithAfterEnd(func(context.Context, shared.Nodes[*runtime.Node]) error {
			totalIters.Add(1)
			return nil
		}),
	)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("expected execution to timeout after 10 ticks")
	}

	cancel()

	// Test resuming from an Env proto snapshot
	graph, err := runtime.NewGraph(exec.Graph().Proto(),
		runtime.WithEnv(exec.Snapshot().Proto()),
	)
	if err != nil {
		t.Fatal(err)
	}

	exec, err = runtime.NewExecutor(t.Context(), graph)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel = context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	err = exec.Run(ctx, func(err error) {
		cancel()
	},
		shared.WithInitStart(func(_ context.Context, _ shared.Nodes[*runtime.Node]) (map[string]any, error) {
			return exec.Snapshot().Values(), nil
		}),
		shared.WithAfterEnd(func(context.Context, shared.Nodes[*runtime.Node]) error {
			totalIters.Add(1)
			return nil
		}),
	)
	if done, ok := exec.IsDone(); !ok {
		t.Fatal("expected done message")
	} else if totalIters.Load() != 20 {
		t.Fatalf("expected 20 iterations got %d", totalIters.Load())
	} else {
		t.Log(done.String())
	}
}

var runningTotalFlow = `
kind: Flow
apiVersion: v1beta1
spec:
  name: Running Total
  inputs:
    - id: number
      uint64: {}
  streams:
    - id: tick
      generate:
        every: 0.1s
        initial: 0s
  outputs:
    - id: tick
      value: = streams.tick.value
    - id: runningTotal
      value: '= self.prev == null ? inputs.number.value : self.prev + inputs.number.value'
`

func TestExecutor_RunningTotal(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(runningTotalFlow)
	flow, err := flowsdk.ReadSpec(&buf, encoding.YAML)
	if err != nil {
		t.Fatal(err)
	}

	exec, err := runtime.ExecutorFromSpec(t.Context(), flow.GetV1Beta1())
	if err != nil {
		t.Fatal(err)
	}

	totalTime := 5 * time.Second
	ctx, cancel := context.WithTimeout(t.Context(), totalTime)
	defer cancel()

	var (
		inputValue atomic.Uint64
		iterations atomic.Uint64
	)

	err = exec.Run(ctx, func(error) { cancel() },
		shared.WithInitStart(func(context.Context, shared.Nodes[*runtime.Node]) (map[string]any, error) {
			return map[string]any{
				"inputs.number": inputValue.Add(1),
			}, nil
		}),
		shared.WithBeforeStart(func(context.Context, shared.Nodes[*runtime.Node]) (map[string]any, error) {
			return map[string]any{
				"inputs.number": inputValue.Add(1),
			}, nil
		}),
		shared.WithAfterEnd(func(ctx context.Context, nodes shared.Nodes[*runtime.Node]) (_ error) {
			defer iterations.Add(1)

			for _, node := range nodes {
				if node.Ident() == "outputs.runningTotal" {
					fmt.Println(node.Ident()+":", node.Value())
				}
			}
			return
		}),
	)
	if err != nil {
		done, ok := runtime.IsDoneError(err)
		if ok && done.IsError() {
			t.Fatal(err)
		} else if ok {
			t.Log(done.String())
		}
	}

	expectedIters := uint64(totalTime / (100 * time.Millisecond))
	actualIters := iterations.Load()
	if expectedIters != actualIters {
		t.Fatalf("expected %d iterations got %d", expectedIters, actualIters)
	}
}

var windowFlow = `
kind: Flow
apiVersion: v1beta1
spec:
  name: Window
  streams:
    - id: tick
      generate:
        every: 0.05s
        initial: 0s
    - id: tumbling
      window:
        tumbling:
          size: 3
        source: = streams.tick.count
    - id: sliding
      window:
        sliding:
          size: 3
          step: 2
        source: = streams.tick.count
  outputs:
    - id: tumbling
      value: = streams.tumbling.value
    - id: sliding
      value: = streams.sliding.value
`

func TestExecutor_Window(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(windowFlow)
	flow, err := flowsdk.ReadSpec(&buf, encoding.YAML)
	if err != nil {
		t.Fatal(err)
	}

	exec, err := runtime.ExecutorFromSpec(t.Context(), flow.GetV1Beta1())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 700*time.Millisecond)
	defer cancel()

	var tumblings, slidings [][]float64

	err = exec.Run(ctx, func(error) { cancel() },
		shared.WithAfterRecv(func(ctx context.Context, node *runtime.Node, value any) (_ any, _ error) {
			list, ok := value.([]any)
			if ok {
				switch node.Ident() {
				case "streams.tumbling":
					tumblings = append(tumblings, toFloats(list))
				case "streams.sliding":
					slidings = append(slidings, toFloats(list))
				}
			}
			return value, nil
		}),
	)
	if err != nil {
		done, ok := runtime.IsDoneError(err)
		if ok && done.IsError() {
			t.Fatal(err)
		} else if ok {
			t.Log(done.String())
		}
	}

	wantTumblings := [][]float64{{1, 2, 3}, {4, 5, 6}}
	if len(tumblings) < len(wantTumblings) {
		t.Fatalf("expected at least %d tumbling windows, got %d: %v", len(wantTumblings), len(tumblings), tumblings)
	}
	for i, want := range wantTumblings {
		if !reflect.DeepEqual(tumblings[i], want) {
			t.Fatalf("expected tumbling window %d to be %v, got %v", i, want, tumblings[i])
		}
	}

	wantSlidings := [][]float64{{1, 2, 3}, {3, 4, 5}, {5, 6, 7}}
	if len(slidings) < len(wantSlidings) {
		t.Fatalf("expected at least %d sliding windows, got %d: %v", len(wantSlidings), len(slidings), slidings)
	}
	for i, want := range wantSlidings {
		if !reflect.DeepEqual(slidings[i], want) {
			t.Fatalf("expected sliding window %d to be %v, got %v", i, want, slidings[i])
		}
	}
}

func toFloats(vals []any) []float64 {
	out := make([]float64, len(vals))
	for i, v := range vals {
		out[i] = reflect.ValueOf(v).Convert(reflect.TypeFor[float64]()).Float()
	}
	return out
}

// TBD:
// metrics, err := spec.NewMetrics(exec.Env(), flow.GetV1Beta1().GetMetrics())
// if err != nil {
// 	t.Fatal(err)
// }
// pipe := runtime.NewPipe(make(chan map[string]any), make(chan any))

// go func() {
// 	err := metrics.Run(ctx, exec.Env(), pipe)
// 	if err != nil {
// 		cancel()
// 	}
// }()

// go func() {
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case values := <-pipe.inCh:
// 			t.Logf("metrics: %v", values)
// 		}
// 	}
// }()
