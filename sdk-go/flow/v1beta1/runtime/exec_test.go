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
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/v1beta1/runtime"
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
    - id: tickDone
      value: |
        = streams.tick.count >= inputs.maxIters.value ? Done{
            message: "Maximum iterations: %d".format([inputs.maxIters.value])
          } : null
  outputs:
    - id: tick
      value: = streams.tick.value
    - id: iters
      value: = streams.tick.count
    - id: evenOrOdd
      value: |
        = int(streams.tick.count) % 2 == 0 ? "even" : "odd"
    - id: isDone
      value: |
        = isEOF(vars.tickDone) ? vars.tickDone.value : null
`

func TestExecutor_Ticker(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(tickerFlow)
	flow, err := flow.ReadSpec(&buf, encoding.YAML)
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
	err = exec.Run(ctx,
		shared.WithInitStart(func(context.Context, shared.GraphNodes[*runtime.Node]) (map[string]any, error) {
			return map[string]any{
				// Set maxIters to 20 (10 remaining after timeout)
				"inputs.maxIters": 20,
			}, nil
		}),
		shared.WithAfterEnd(func(_ context.Context, nodes shared.GraphNodes[*runtime.Node]) error {
			totalIters.Add(1)
			return nil
		}),
	)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("expected execution to timeout after 10 ticks")
	}

	cancel()

	// Test resuming from an env snapshot
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

	err = exec.Run(ctx,
		shared.WithInitStart(func(_ context.Context, _ shared.GraphNodes[*runtime.Node]) (map[string]any, error) {
			return exec.Snapshot().Values(), nil
		}),
		shared.WithAfterEnd(func(_ context.Context, nodes shared.GraphNodes[*runtime.Node]) error {
			totalIters.Add(1)
			return nil
		}),
	)
	if _, ok := exec.IsDone(); !ok {
		t.Fatal("expected done message")
	} else if totalIters.Load() != 20 {
		t.Fatalf("expected 20 iterations got %d", totalIters.Load())
	}

	isDone, ok := exec.Graph().Node("outputs.isDone")
	if !ok {
		t.Fatal("expected outputs.isDone")
	} else {
		t.Logf("%s: %v", isDone.Ident(), isDone.Value())
	}
}

var eofFlow = `
kind: Flow
apiVersion: v1beta1
spec:
  name: EOF
  streams:
    - id: tick
      generate:
        initial: 0s
        every: 0.1s
  outputs:
    - id: test
      value: '= streams.tick.count >= 10 ? Done{} : null'
`

func TestExecutor_EOF(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(eofFlow)
	flow, err := flow.ReadSpec(&buf, encoding.YAML)
	if err != nil {
		t.Fatal(err)
	}

	exec, err := runtime.ExecutorFromSpec(t.Context(), flow.GetV1Beta1())
	if err != nil {
		t.Fatal(err)
	}

	totalTime := 2 * time.Second
	ctx, cancel := context.WithTimeout(t.Context(), totalTime)
	defer cancel()

	var inputValue atomic.Uint64
	err = exec.Run(ctx,
		shared.WithInitStart(func(context.Context, shared.GraphNodes[*runtime.Node]) (map[string]any, error) {
			return map[string]any{
				"inputs.number": inputValue.Add(1),
			}, nil
		}),
		shared.WithBeforeStart(func(context.Context, shared.GraphNodes[*runtime.Node]) (map[string]any, error) {
			return map[string]any{
				"inputs.number": inputValue.Add(1),
			}, nil
		}),
	)

	done, ok := exec.IsDone()
	if !ok {
		t.Fatal("expected done")
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
      value: '= self.prev != null ? self.prev + inputs.number.value : inputs.number.value'
`

func TestExecutor_RunningTotal(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(runningTotalFlow)
	flow, err := flow.ReadSpec(&buf, encoding.YAML)
	if err != nil {
		t.Fatal(err)
	}

	exec, err := runtime.ExecutorFromSpec(t.Context(), flow.GetV1Beta1())
	if err != nil {
		t.Fatal(err)
	}

	totalTime := time.Second
	ctx, cancel := context.WithTimeout(t.Context(), totalTime)
	defer cancel()

	var (
		inputValue atomic.Uint64
		iterations atomic.Uint64
	)

	err = exec.Run(ctx,
		shared.WithInitStart(func(context.Context, shared.GraphNodes[*runtime.Node]) (map[string]any, error) {
			return map[string]any{
				"inputs.number": inputValue.Add(1),
			}, nil
		}),
		shared.WithBeforeStart(func(context.Context, shared.GraphNodes[*runtime.Node]) (map[string]any, error) {
			return map[string]any{
				"inputs.number": inputValue.Add(1),
			}, nil
		}),
		shared.WithAfterEnd(func(ctx context.Context, nodes shared.GraphNodes[*runtime.Node]) (_ error) {
			iterations.Add(1)
			return
		}),
	)
	if err != nil {
		done, ok := runtime.IsDone(err)
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

	runningTotal, ok := exec.Graph().Nodes().Load("outputs.runningTotal")
	if ok {
		t.Log("runningTotal:", runningTotal.Value())
	}
}

var runningTotalInteriorFlow = `
kind: Flow
apiVersion: v1beta1
spec:
  name: Running Total Interior
  inputs:
    - id: number
      uint64: {}
  streams:
    - id: tick
      generate:
        every: 0.1s
        initial: 0s
  vars:
    - id: runningTotal
      value: '= self.prev != null ? self.prev + inputs.number.value : inputs.number.value'
  outputs:
    - id: tick
      value: = streams.tick.value
    - id: runningTotal
      value: = vars.runningTotal.value
`

// TestExecutor_RunningTotal_InteriorNode verifies that self.prev correctly
// rotates and accumulates on an interior (non-sink) node, i.e. a node that
// has an outbound edge and is therefore never collected as an "end" node.
// Prior to fixing Node.Reset()'s scope, prev/curr only rotated for sink
// nodes, so an accumulator expressed on an interior var would never see a
// non-null self.prev after the first round, and its expression would
// collapse to just the latest input every round instead of accumulating.
//
// The exact per-round cadence/timing of this executor is independent of
// this fix, so rather than assert an exact formula tied to round count,
// this test asserts the discriminating property directly: a broken
// self.prev on an interior node can never exceed the largest single input
// it ever observed (since it would just keep reflecting the latest input
// alone), whereas a working accumulator must exceed it once more than one
// round has been observed.
func TestExecutor_RunningTotal_InteriorNode(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(runningTotalInteriorFlow)
	flow, err := flow.ReadSpec(&buf, encoding.YAML)
	if err != nil {
		t.Fatal(err)
	}

	exec, err := runtime.ExecutorFromSpec(t.Context(), flow.GetV1Beta1())
	if err != nil {
		t.Fatal(err)
	}

	totalTime := time.Second
	ctx, cancel := context.WithTimeout(t.Context(), totalTime)
	defer cancel()

	var (
		inputValue atomic.Uint64
		iterations atomic.Uint64
	)

	err = exec.Run(ctx,
		shared.WithInitStart(func(context.Context, shared.GraphNodes[*runtime.Node]) (map[string]any, error) {
			return map[string]any{
				"inputs.number": inputValue.Add(1),
			}, nil
		}),
		shared.WithBeforeStart(func(context.Context, shared.GraphNodes[*runtime.Node]) (map[string]any, error) {
			return map[string]any{
				"inputs.number": inputValue.Add(1),
			}, nil
		}),
		shared.WithAfterEnd(func(context.Context, shared.GraphNodes[*runtime.Node]) (_ error) {
			iterations.Add(1)
			return
		}),
	)
	if err != nil {
		done, ok := runtime.IsDone(err)
		if ok && done.IsError() {
			t.Fatal(err)
		} else if ok {
			t.Log(done.String())
		}
	}

	if iterations.Load() < 2 {
		t.Fatalf("expected at least 2 completed rounds to meaningfully exercise accumulation, got %d", iterations.Load())
	}

	maxInput := inputValue.Load()

	interior, ok := exec.Graph().Nodes().Load("vars.runningTotal")
	if !ok {
		t.Fatal("expected to find vars.runningTotal node")
	}

	actual, err := toUint64(interior.Value())
	if err != nil {
		t.Fatalf("vars.runningTotal: %v", err)
	}
	if actual <= maxInput {
		t.Fatalf("expected interior running total to exceed the largest single input observed (%d), got %d — self.prev does not appear to be accumulating on this interior node", maxInput, actual)
	}

	sink, ok := exec.Graph().Nodes().Load("outputs.runningTotal")
	if !ok {
		t.Fatal("expected to find outputs.runningTotal node")
	}

	sinkValue, err := toUint64(sink.Value())
	if err != nil {
		t.Fatalf("outputs.runningTotal: %v", err)
	}
	if sinkValue != actual {
		t.Fatalf("expected sink to mirror interior node value %d, got %d", actual, sinkValue)
	}
}

func toUint64(value any) (uint64, error) {
	switch v := value.(type) {
	case uint64:
		return v, nil
	case int64:
		return uint64(v), nil
	case float64:
		return uint64(v), nil
	default:
		return 0, fmt.Errorf("unexpected type %T (%v)", value, value)
	}
}

var windowFlow = `
kind: Flow
apiVersion: v1beta1
spec:
  name: Window Flow
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
	flow, err := flow.ReadSpec(&buf, encoding.YAML)
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

	err = exec.Run(ctx,
		shared.WithAfterRecv(func(ctx context.Context, node *runtime.Node, value any) (_ any, _ error) {
			list, ok := value.([]any)
			if ok {
				switch node.Ident() {
				case "streams.tumbling":
					vals := toFloats(list)
					tumblings = append(tumblings, vals)
					t.Logf("streams.tumbling: %v", vals)
				case "streams.sliding":
					vals := toFloats(list)
					slidings = append(slidings, vals)
					t.Logf("streams.sliding: %v", vals)
				}
			}
			return value, nil
		}),
	)
	if err != nil {
		done, ok := runtime.IsDone(err)
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
