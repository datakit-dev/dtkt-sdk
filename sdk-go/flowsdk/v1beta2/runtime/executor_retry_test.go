package runtime

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// Retry: backoff succeeds after transient error (UnavailableThenOK).

func TestGraph_Action_RetryBackoff_Success(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_backoff.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 42)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
	})
}

// Retry: skip_when skips the item on NotFound (code 5).

func TestGraph_Action_RetrySkip(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_skip.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 99)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		assert.Empty(t, results)
	})
}

// Retry: terminate_when terminates the flow on Internal (code 13).

func TestGraph_Action_RetryTerminate(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_terminate.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 99)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.Error(t, err)

		var termErr *TerminateError
		assert.True(t, errors.As(err, &termErr), "expected TerminateError, got %T: %v", err, err)
	})
}

// Retry: retries exhausted propagates the last error.

func TestGraph_Action_RetryExhausted(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_exhaust.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 99)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")
	})
}

// Retry: no backoff defined, only skip_when -- skips immediately on first error.

func TestGraph_Action_RetryNoBackoff_Skip(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_no_backoff_skip.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 99)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		assert.Empty(t, results)
	})
}

// Retry: when guard doesn't match the error -> error propagates immediately.

func TestGraph_Action_RetryWhenNoMatch(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_when_no_match.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 99)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		// when = code==14 (Unavailable) but error is NotFound (code 5) -- doesn't match, immediate fail.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource not found")
	})
}

// Retry: when guard matches the error -> strategy activates and retries succeed.

func TestGraph_Action_RetryWhenMatch(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_when_match.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 42)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		// when = code==14 (Unavailable), UnavailableThenOK returns code 14 first -> retries -> succeeds.
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
	})
}

// Retry: suspend_when suspends the node on PermissionDenied (code 7).

func TestGraph_Action_RetrySuspend(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_suspend.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 99)
		ctx := testContext(t)

		// Subscribe to the action node's topic to observe phases.
		actionCh, err := pubsub.Subscribe(ctx, testTopics.For("actions.echo"))
		require.NoError(t, err)

		exec := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait for the action to reach PHASE_SUSPENDED.
		require.True(t, waitForPhase(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected action to reach PHASE_SUSPENDED")

		// Resume the node -- it will read EOF from the input and complete.
		exec.ResumeNode("actions.echo", nil)

		err = <-done
		assert.NoError(t, err, "after resume, flow should complete cleanly (input EOF)")
	})
}

// Retry: server stream with backoff retry.

func TestGraph_Stream_RetryBackoff_Success(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_retry_backoff.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 42)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
	})
}

// Retry: multiple escalation paths -- skip_when takes priority over terminate_when
// when skip_when matches first (NotFound code 5 matches skip_when).

func TestGraph_Action_RetryMultiEscalation_Skip(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_multi_escalation.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 99)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		// skip_when matches (code 5) -> item skipped, no error.
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		assert.Empty(t, results)
	})
}

// Retry: multiple escalation paths -- terminate_when fires when skip_when doesn't match.

func TestGraph_Action_RetryMultiEscalation_Terminate(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_multi_escalation_terminate.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 99)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		// terminate_when matches (code 13) -> flow terminated.
		require.Error(t, err)

		var termErr *TerminateError
		assert.True(t, errors.As(err, &termErr), "expected TerminateError, got %T: %v", err, err)
	})
}

// Retry: no retry strategy, action succeeds normally.

func TestGraph_Action_NoRetryStrategy(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_unary_echo.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 42)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)
	})
}

// Unit test: backoffDelay computation.

func TestBackoffDelay(t *testing.T) {
	b := flowv1beta2.Backoff_builder{
		InitialBackoff:    durationpb.New(100 * time.Millisecond),
		BackoffMultiplier: 2.0,
		MaxBackoff:        durationpb.New(500 * time.Millisecond),
		MaxAttempts:       5,
	}.Build()

	// attempt 1: 100ms * 2^0 = 100ms
	assert.Equal(t, 100*time.Millisecond, backoffDelay(b, 1))
	// attempt 2: 100ms * 2^1 = 200ms
	assert.Equal(t, 200*time.Millisecond, backoffDelay(b, 2))
	// attempt 3: 100ms * 2^2 = 400ms
	assert.Equal(t, 400*time.Millisecond, backoffDelay(b, 3))
	// attempt 4: 100ms * 2^3 = 800ms, capped at 500ms
	assert.Equal(t, 500*time.Millisecond, backoffDelay(b, 4))
}

func TestBackoffDelay_DefaultMultiplier(t *testing.T) {
	b := flowv1beta2.Backoff_builder{
		InitialBackoff: durationpb.New(50 * time.Millisecond),
		MaxAttempts:    3,
	}.Build()

	// multiplier defaults to 2 when < 1
	assert.Equal(t, 50*time.Millisecond, backoffDelay(b, 1))
	assert.Equal(t, 100*time.Millisecond, backoffDelay(b, 2))
}

func TestBackoffDelay_NilBackoff(t *testing.T) {
	assert.Equal(t, time.Duration(0), backoffDelay(nil, 1))
}

// --- continue_when ---

// Retry: continue_when matches -- error is converted to a value output.

func TestGraph_Action_RetryContinue(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_continue.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 99)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		// continue_when emits this.error.message as the output value.
		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, "resource not found", results[0].GetValue().GetStringValue())
	})
}

// Retry: continue_when doesn't match (false) -- error propagates normally.

func TestGraph_Action_RetryContinue_NoMatch(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_continue_no_match.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 99)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)

		// continue_when expects code==5 (NotFound) but error is code==13 (Internal).
		// Expression returns false, so error propagates.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")
	})
}

// Retry: continue_when emits error info downstream alongside a healthy path.

func TestGraph_Action_RetryContinue_MultiPath(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_continue_multi.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.fail_input", 99)
		feedInput(ps, "inputs.ok_input", 42)

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		// The failed action emits the error message via continue_when.
		failResults := collectOutputs(ctx, ps, "outputs.fail_result")
		require.Len(t, failResults, 1)
		assert.Equal(t, "internal server error", failResults[0].GetValue().GetStringValue())

		// The healthy path produces the echoed value.
		okResults := collectOutputs(ctx, ps, "outputs.ok_result")
		require.Len(t, okResults, 1)
		assert.Equal(t, int64(42), okResults[0].GetValue().GetInt64Value())
	})
}

// Retry: continue_when with node phase -- node succeeds (not errored).

func TestGraph_Action_RetryContinue_NodePhase(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_continue.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 99)

		ctx := testContext(t)
		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.echo"))
		require.NoError(t, err)

		err = NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		// Node should succeed (not error) because continue_when converted the error.
		phases := collectPhases(ctx, actionCh)
		require.NotEmpty(t, phases)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, phases[len(phases)-1],
			"expected terminal phase SUCCEEDED, got %v", phaseNames(phases))
	})
}
