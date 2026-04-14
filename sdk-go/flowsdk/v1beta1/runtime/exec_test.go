package runtime

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- helpers ---------------------------------------------------------------

// parseSpec parses a YAML flow spec and returns a fully-parsed *flowsdk.Spec.
func parseSpec(t *testing.T, yaml string) *flowv1beta1.Flow {
	t.Helper()
	spec := new(flowv1beta1.Flow)
	err := encoding.FromYAMLV2([]byte(strings.TrimSpace(yaml)), spec)
	require.NoError(t, err)
	return spec
}

// newRun creates a Runtime + Graph + Executor from a parsed spec.
func newRun(t *testing.T, ctx context.Context, cancel context.CancelCauseFunc, spec *flowv1beta1.Flow) (*Runtime, *Executor) {
	t.Helper()
	run := NewFromSpec(ctx, cancel, spec)
	graph, err := GraphFromRuntime(run)
	require.NoError(t, err)
	exec, err := NewExecutor(run, graph)
	require.NoError(t, err)
	return run, exec
}

// runUntilDone starts the executor and loops on exec.Ready(), calling eval()
// on each cycle, until a Done sentinel is reached or the context is cancelled.
// It returns the final DoneError (may be nil on context cancellation).
func runUntilDone(t *testing.T, ctx context.Context, run *Runtime, exec *Executor) *DoneError {
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
			if done, ok := IsRuntimeDone(run); ok {
				return done
			}
			exec.Reset()
		}
	}
}

// ---- ticker spec (uses default input, no external events needed) ------------

const tickerSpec = `
name: Ticker
inputs:
  - id: maxIters
    cache: true
    int64:
      default: 10

streams:
  - id: tick
    generate:
      every: 0.1s

outputs:
  - id: tick
    value: = streams.tick.value
  - id: error
    value: '=
      inputs.maxIters.value <= 0 ?
        Done{reason: "inputs.maxIters must be > 0", is_error: true} : null'
  - id: success
    value: '=
      int(streams.tick.count) >= inputs.maxIters.value ?
        Done{ reason: "%d iterations completed".format([inputs.maxIters.value]) } : null'
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

	recvCh, err := run.GetRecvCh("inputs.maxIters")
	require.NoError(t, err)
	recvCh <- 20

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
      inputs.maxIters.value <= 0 ?
        Done{reason: "bad maxIters", is_error: true} : null'
  - id: success
    value: '=
      int(streams.tick.count) >= inputs.maxIters.value ?
        Done{reason: "done"} : null'
`
	spec := parseSpec(t, badSpec)
	run, exec := newRun(t, ctx, cancel, spec)

	done := runUntilDone(t, ctx, run, exec)
	require.NotNil(t, done)
	assert.True(t, done.Proto().GetIsError(), "expected error done, got: %s", done.Error())
	t.Log(done)
}

// TestExec_RequiredInputBlocks verifies that a required (no-default) input
// does NOT self-fire — the executor waits for external injection.
func TestExec_RequiredInputBlocksThenProceeds(t *testing.T) {
	const requiredInputSpec = `
name: RequiredInput
inputs:
  - id: value
    int64: {}
outputs:
  - id: result
    value: = inputs.value.value
  - id: done
    value: '= inputs.value.value > 0 ? Done{reason: "got value"} : null'
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
	done, ok := IsRuntimeDone(run)
	require.True(t, ok)
	assert.False(t, done.Proto().GetIsError())
}

// TestExec_CachedRequiredInput_PersistsAcrossCycles verifies that a required
// input with cache:true only needs to be injected once — the value is re-used
// on every subsequent cycle without blocking on sendCh again.
func TestExec_CachedRequiredInput_PersistsAcrossCycles(t *testing.T) {
	const cachedInputSpec = `
name: CachedRequiredInput
inputs:
  - id: label
    string: {}
    cache: true
streams:
  - id: tick
    generate:
      every: 0.1s
outputs:
  - id: result
    value: = inputs.label.value
  - id: done
    value: '= int(streams.tick.count) >= 3 ? Done{reason: "done"} : null'
`
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)
	time.AfterFunc(5*time.Second, func() { cancel(context.DeadlineExceeded) })

	spec := parseSpec(t, cachedInputSpec)
	run, exec := newRun(t, ctx, cancel, spec)

	require.NoError(t, exec.Start())

	// Executor must not fire before the required input arrives.
	select {
	case <-exec.Ready():
		t.Fatal("executor should not be ready before cached required input is injected")
	case <-time.After(300 * time.Millisecond):
	}

	// Inject the cached required input exactly once.
	recvCh, err := run.GetRecvCh("inputs.label")
	require.NoError(t, err)
	select {
	case recvCh <- "hello":
	case <-ctx.Done():
		t.Fatal("timed out injecting cached input")
	}

	// Run all cycles; each must produce "hello" as the result without requiring
	// another injection.
	cycleCount := 0
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out after %d cycles — cached input likely blocked on sendCh", cycleCount)
		case <-exec.Ready():
			require.NoError(t, exec.Eval())
			cycleCount++

			result, err := run.GetValue("outputs.result")
			require.NoError(t, err)
			assert.Equal(t, "hello", result, "cycle %d: expected cached input value", cycleCount)

			if _, ok := IsRuntimeDone(run); ok {
				goto done
			}
			exec.Reset()
		}
	}
done:
	assert.GreaterOrEqual(t, cycleCount, 3, "expected at least 3 cycles to verify caching")
}
