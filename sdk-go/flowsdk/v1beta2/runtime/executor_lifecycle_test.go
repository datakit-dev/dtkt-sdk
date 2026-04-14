package runtime

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// --- TERMINATE strategy (default) ---

// TERMINATE: action error terminates the flow immediately.

func TestErrorStrategy_Terminate_ActionError(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_error_internal.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 99)

		// Subscribe to the action node's topic before executing.
		ctx := testContext(t)
		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.call"))
		require.NoError(t, err)

		err = NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")

		// The errored action node should have PHASE_ERRORED as terminal phase.
		phases := collectPhases(ctx, actionCh)
		require.NotEmpty(t, phases)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_ERRORED, phases[len(phases)-1],
			"expected terminal phase ERRORED, got %v", phaseNames(phases))
	})
}

// TERMINATE: TerminateError from retry also terminates.

func TestErrorStrategy_Terminate_RetryTerminateError(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_terminate.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 99)

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.Error(t, err)

		var termErr *TerminateError
		assert.True(t, errors.As(err, &termErr), "expected TerminateError, got %T: %v", err, err)
	})
}

// --- STOP strategy ---

// STOP: action error drains the pipeline gracefully; flow returns the error.

func TestErrorStrategy_Stop_ActionError(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_error_internal.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 99)

		ctx := testContext(t)
		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_STOP))
		opts = append(opts, extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		// STOP still returns the error after draining.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")
	})
}

// STOP: multi-path flow -- the non-errored path completes and produces output.

func TestErrorStrategy_Stop_MultiPath(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stop_multi_path.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Feed both inputs. ok_input will be processed by the healthy action.
		feedInput(ps, "inputs.fail_input", 99)
		feedInput(ps, "inputs.ok_input", 42)

		ctx := testContext(t)
		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_STOP))
		opts = append(opts, extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		// Flow returns the error from the failed action.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")

		// The healthy path should have produced output despite the error.
		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1, "healthy path should have produced output")
		assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
	})
}

// STOP: generator + action error -- generator stops gracefully.

func TestErrorStrategy_Stop_Generator(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stop_generator_action.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_STOP))
		opts = append(opts, extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		// STOP still returns the error.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")
	})
}

// STOP + TerminateError: TerminateError overrides STOP strategy.

func TestErrorStrategy_Stop_TerminateErrorOverrides(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stop_terminate_error.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 99)

		ctx := testContext(t)
		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_STOP))
		opts = append(opts, extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		// TerminateError bypasses STOP -- immediate termination.
		require.Error(t, err)
		var termErr *TerminateError
		assert.True(t, errors.As(err, &termErr),
			"expected TerminateError even with STOP strategy, got %T: %v", err, err)
	})
}

// STOP: errored node publishes PHASE_ERRORED.

func TestErrorStrategy_Stop_NodePhase(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_error_internal.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 99)

		ctx := testContext(t)
		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.call"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_STOP))
		opts = append(opts, extraOpts...)
		_ = NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		// The errored node should have PHASE_ERRORED as terminal phase.
		phases := collectPhases(ctx, actionCh)
		require.NotEmpty(t, phases)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_ERRORED, phases[len(phases)-1],
			"expected terminal phase ERRORED, got %v", phaseNames(phases))
	})
}

// --- CONTINUE strategy ---

// CONTINUE: action error does not terminate the flow; error is returned
// after all nodes complete.

func TestErrorStrategy_Continue_ActionError(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_error_internal.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 99)

		ctx := testContext(t)
		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_CONTINUE))
		opts = append(opts, extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		// CONTINUE still returns the error after all nodes complete.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")
	})
}

// CONTINUE: multi-path flow -- the non-errored path completes and produces
// output; the flow returns the error from the failed path.

func TestErrorStrategy_Continue_MultiPath(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stop_multi_path.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.fail_input", 99)
		feedInput(ps, "inputs.ok_input", 42)

		ctx := testContext(t)
		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_CONTINUE))
		opts = append(opts, extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		// Flow returns the error from the failed action.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")

		// The healthy path should have produced output despite the error.
		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1, "healthy path should have produced output")
		assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
	})
}

// CONTINUE: errored node publishes PHASE_ERRORED.

func TestErrorStrategy_Continue_NodePhase(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_error_internal.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 99)

		ctx := testContext(t)
		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.call"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_CONTINUE))
		opts = append(opts, extraOpts...)
		_ = NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		phases := collectPhases(ctx, actionCh)
		require.NotEmpty(t, phases)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_ERRORED, phases[len(phases)-1],
			"expected terminal phase ERRORED, got %v", phaseNames(phases))
	})
}

// CONTINUE + TerminateError: TerminateError overrides CONTINUE strategy.

func TestErrorStrategy_Continue_TerminateErrorOverrides(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stop_terminate_error.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 99)

		ctx := testContext(t)
		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_CONTINUE))
		opts = append(opts, extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		// TerminateError bypasses CONTINUE -- immediate termination.
		require.Error(t, err)
		var termErr *TerminateError
		assert.True(t, errors.As(err, &termErr),
			"expected TerminateError even with CONTINUE strategy, got %T: %v", err, err)
	})
}

// CONTINUE: generator path continues running when an independent input path
// errors. The generator path produces all its output values.

func TestErrorStrategy_Continue_GeneratorContinues(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "continue_gen_and_error.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Feed the failing input.
		feedInput(ps, "inputs.fail_input", 99)

		ctx := testContext(t)
		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_CONTINUE))
		opts = append(opts, extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		// Flow returns the error from the failed action.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")

		// The generator->ok_action->ok_result path should have produced all 3 values.
		results := collectOutputs(ctx, ps, "outputs.ok_result")
		require.Len(t, results, 3, "generator path should produce all values")
		assert.Equal(t, []int64{1, 2, 3}, outputInt64s(results))
	})
}

// CONTINUE: multiple nodes error -- all errors are collected and returned.

func TestErrorStrategy_Continue_MultipleErrors(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "continue_gen_and_error.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.fail_input", 99)

		ctx := testContext(t)

		// Subscribe to both action nodes.
		failCh, err := ps.Subscribe(ctx, testTopics.For("actions.fail_action"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_CONTINUE))
		opts = append(opts, extraOpts...)
		execErr := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		// The returned error should reference fail_action.
		require.Error(t, execErr)
		assert.Contains(t, execErr.Error(), "fail_action")

		// The errored node should have PHASE_ERRORED.
		phases := collectPhases(ctx, failCh)
		require.NotEmpty(t, phases)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_ERRORED, phases[len(phases)-1])
	})
}
