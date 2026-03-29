package runtime_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- helpers ---------------------------------------------------------------

// parseSpec parses a YAML flow spec and returns a fully-parsed *flowsdk.Spec.
func parseSpec(t *testing.T, yaml string) *flowsdk.Spec {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteString(yaml)
	spec, err := flowsdk.ReadSpec(encoding.YAML, &buf)
	require.NoError(t, err)
	_, err = spec.Parse()
	require.NoError(t, err)
	return spec
}

// newRun creates a Runtime + Graph + Executor from a parsed spec.
func newRun(t *testing.T, ctx context.Context, cancel context.CancelCauseFunc, spec *flowsdk.Spec) (*runtime.Runtime, *runtime.Executor) {
	t.Helper()
	run := runtime.New(ctx, cancel, spec.GetFlow())
	graph, err := runtime.RuntimeGraph(run)
	require.NoError(t, err)
	exec, err := runtime.NewExecutor(run, graph)
	require.NoError(t, err)
	return run, exec
}

// runUntilDone starts the executor and loops on exec.Ready(), calling eval()
// on each cycle, until a Done sentinel is reached or the context is cancelled.
// It returns the final DoneError (may be nil on context cancellation).
func runUntilDone(t *testing.T, ctx context.Context, run *runtime.Runtime, exec *runtime.Executor) *runtime.DoneError {
	t.Helper()
	err := exec.Start()
	require.NoError(t, err)

	for {
		select {
		case <-ctx.Done():
			t.Fatal("execution timed out")
			return nil
		case <-exec.Ready():
			require.NoError(t, exec.Eval())
			if done, ok := runtime.IsRuntimeDone(run); ok {
				return done
			}
			exec.Reset()
		}
	}
}

// ---- ticker spec (uses default input, no external events needed) ------------

const tickerSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: Ticker

  inputs:
    - id: maxIters
      int64:
        default: 3

  streams:
    - id: tick
      generate:
        every: 0.1s

  outputs:
    - id: tick
      value: = streams.tick.getValue()
    - id: error
      value: '=
        inputs.maxIters.getValue() <= 0 ?
          Done{reason: "inputs.maxIters must be > 0", is_error: true} : null'
    - id: success
      value: '=
        int(streams.tick.getCount()) >= inputs.maxIters.getValue() ?
          Done{ reason: "%d iterations completed".format([inputs.maxIters.getValue()]) } : null'
`

// TestExec_TickerWithDefault verifies that an input with a default value does
// not require external injection — the executor self-triggers it and the flow
// runs to completion.
func TestExec_TickerWithDefault(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)
	time.AfterFunc(5*time.Second, func() { cancel(context.DeadlineExceeded) })

	spec := parseSpec(t, tickerSpec)
	run, exec := newRun(t, ctx, cancel, spec)

	done := runUntilDone(t, ctx, run, exec)
	require.NotNil(t, done)
	assert.False(t, done.Proto().GetIsError(), "expected clean completion, got: %s", done.Error())
	t.Log(done.Error())
}

// TestExec_TickerErrorOutput verifies the error output fires when maxIters ≤ 0.
// We inject 0 via the input channel before starting.
func TestExec_TickerErrorOutput(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)
	time.AfterFunc(5*time.Second, func() { cancel(context.DeadlineExceeded) })

	// Set default to -1 inline so the error branch fires immediately.
	const badSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: TickerError
  inputs:
    - id: maxIters
      int64:
        default: -1
  streams:
    - id: tick
      generate:
        every: 0.1s
  outputs:
    - id: error
      value: '=
        inputs.maxIters.getValue() <= 0 ?
          Done{reason: "bad maxIters", is_error: true} : null'
    - id: success
      value: '=
        int(streams.tick.getCount()) >= inputs.maxIters.getValue() ?
          Done{reason: "done"} : null'
`
	spec := parseSpec(t, badSpec)
	run, exec := newRun(t, ctx, cancel, spec)

	done := runUntilDone(t, ctx, run, exec)
	require.NotNil(t, done)
	assert.True(t, done.Proto().GetIsError(), "expected error done, got: %s", done.Error())
}

// TestExec_RequiredInputBlocks verifies that a required (no-default) input
// does NOT self-fire — the executor waits for external injection.
func TestExec_RequiredInputBlocksThenProceeds(t *testing.T) {
	const requiredInputSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: RequiredInput
  inputs:
    - id: value
      int64: {}
  outputs:
    - id: result
      value: = inputs.value.getValue()
    - id: done
      value: '= inputs.value.getValue() > 0 ? Done{reason: "got value"} : null'
`
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)
	time.AfterFunc(5*time.Second, func() { cancel(context.DeadlineExceeded) })

	spec := parseSpec(t, requiredInputSpec)
	run, exec := newRun(t, ctx, cancel, spec)

	require.NoError(t, exec.Start())

	// Confirm the executor does NOT become ready on its own within a short window.
	select {
	case <-exec.Ready():
		t.Fatal("executor should not be ready before required input is injected")
	case <-time.After(300 * time.Millisecond):
		// expected — no spurious trigger
	}

	// Now inject the required value via the input's send channel.
	env, err := run.Env()
	require.NoError(t, err)
	sendCh, err := run.GetSendCh("inputs.value")
	require.NoError(t, err)

	select {
	case sendCh <- env.TypeAdapter().NativeToValue(int64(42)):
	case <-ctx.Done():
		t.Fatal("timed out sending input")
	}

	// Now it should become ready.
	select {
	case <-exec.Ready():
	case <-ctx.Done():
		t.Fatal("timed out waiting for ready after input injection")
	}

	require.NoError(t, exec.Eval())
	done, ok := runtime.IsRuntimeDone(run)
	require.True(t, ok)
	assert.False(t, done.Proto().GetIsError())
}
