package runtime

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	flowsdkv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// --- TERMINATE strategy (default) ---

// TERMINATE: action error terminates the flow immediately.

func TestErrorStrategy_Terminate_ActionError(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_error_internal.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", 99)

		ctx := testContext(t)
		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.call"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_STOP))
		opts = append(opts, extraOpts...)
		err = NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		// STOP returns the error after draining.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")

		// Action's terminal phase is PHASE_ERRORED -- proves the error
		// surfaced through the node-level lifecycle, not just the
		// flow-level error.
		actionPhases := collectPhases(ctx, actionCh)
		require.NotEmpty(t, actionPhases)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_ERRORED, actionPhases[len(actionPhases)-1],
			"action terminal phase: %v", phaseNames(actionPhases))

		// No successful outputs (the action errored, no value to forward).
		results := collectOutputs(ctx, ps, "outputs.result")
		assert.Empty(t, results, "no values forwarded when action errors under STOP")
	})
}

// STOP: multi-path flow -- the non-errored path completes and produces output.

func TestErrorStrategy_Stop_MultiPath(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stop_multi_path.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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

// STOP: generator + action error -- the action errors on first call and
// triggers STOP. Behavioral assertions:
//   - Execute returns the underlying error
//   - The errored action publishes PHASE_ERRORED as its terminal phase
//   - The output topic produces NO successful values (the only path is
//     generator -> action -> output and the action always errors)

func TestErrorStrategy_Stop_Generator(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stop_generator_action.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.call"))
		require.NoError(t, err)
		// Subscribe to the generator topic BEFORE running so we can verify
		// STOP actually halted the generator. Range end=100; a regression
		// where stopCh wasn't signalled would let it run all iterations
		// and we'd never see PHASE_SUCCEEDED (or it would arrive much
		// later). The load-bearing check is that the generator's terminal
		// phase is SUCCEEDED (graceful stop), proving stopCore signalled
		// it (executor_setup.go:181-190).
		genCh, err := ps.Subscribe(ctx, testTopics.For("generators.gen"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_STOP))
		opts = append(opts, extraOpts...)
		err = NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")

		// Action publishes PHASE_ERRORED as terminal phase.
		actionPhases := collectPhases(ctx, actionCh)
		require.NotEmpty(t, actionPhases)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_ERRORED, actionPhases[len(actionPhases)-1],
			"action terminal phase must be ERRORED; got %v", phaseNames(actionPhases))

		// Generator must terminate via STOP, not run to natural completion.
		// PHASE_SUCCEEDED on the generator topic indicates the generator
		// exited cleanly via stopCh (gracefulStop -> stopCore signals stopCh
		// on generators). A regression that didn't signal the generator
		// would either leave it running to end=100 or never publish a
		// terminal phase.
		genPhases := collectPhases(ctx, genCh)
		require.NotEmpty(t, genPhases, "expected generator phase events")
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, genPhases[len(genPhases)-1],
			"generator must reach PHASE_SUCCEEDED via STOP signal; got %v", phaseNames(genPhases))

		// No successful output values: the action errors before publishing
		// any value, so the downstream output gets nothing to forward.
		results := collectOutputs(ctx, ps, "outputs.result")
		assert.Empty(t, results, "no values should be forwarded when action always errors")
	})
}

// STOP + TerminateError: TerminateError overrides STOP strategy.

func TestErrorStrategy_Stop_TerminateErrorOverrides(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stop_terminate_error.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
// after all nodes complete. Behavioral assertions:
//   - Execute returns the error
//   - The errored action's terminal phase is PHASE_ERRORED
//   - Output topic gets no value (the action errored, no value to forward)

func TestErrorStrategy_Continue_ActionError(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_error_internal.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", 99)

		ctx := testContext(t)
		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.call"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_CONTINUE))
		opts = append(opts, extraOpts...)
		err = NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")

		actionPhases := collectPhases(ctx, actionCh)
		require.NotEmpty(t, actionPhases)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_ERRORED, actionPhases[len(actionPhases)-1],
			"action terminal phase: %v", phaseNames(actionPhases))

		results := collectOutputs(ctx, ps, "outputs.result")
		assert.Empty(t, results, "no values should be forwarded when action errors")
	})
}

// CONTINUE: multi-path flow -- the non-errored path completes and produces
// output; the flow returns the error from the failed path.

func TestErrorStrategy_Continue_MultiPath(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stop_multi_path.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Feed both failing inputs.
		feedInput(ps, "inputs.fail_input", 99)
		feedInput(ps, "inputs.fail_input2", 99)

		ctx := testContext(t)
		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_CONTINUE))
		opts = append(opts, extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		// Flow returns the error(s) from the failed actions.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")

		// The generator->ok_action->ok_result path should have produced all 3 values
		// even though two siblings errored.
		results := collectOutputs(ctx, ps, "outputs.ok_result")
		require.Len(t, results, 3, "generator path should produce all values")
		assert.Equal(t, []int64{1, 2, 3}, outputInt64s(results))
	})
}

// CONTINUE: multiple nodes error -- the spec contract is that all errors
// are collected and surfaced in the returned error. With one failing action
// the previous test was vacuous (single error trivially "all" of one). This
// version feeds both fail inputs and asserts the returned error references
// BOTH failing nodes.

func TestErrorStrategy_Continue_MultipleErrors(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "continue_gen_and_error.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.fail_input", 99)
		feedInput(ps, "inputs.fail_input2", 99)

		ctx := testContext(t)

		// Subscribe to both failing action nodes.
		failCh, err := ps.Subscribe(ctx, testTopics.For("actions.fail_action"))
		require.NoError(t, err)
		failCh2, err := ps.Subscribe(ctx, testTopics.For("actions.fail_action2"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_CONTINUE))
		opts = append(opts, extraOpts...)
		execErr := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)

		// The returned error must reference BOTH failing action ids.
		// (Spec contract: CONTINUE accumulates errors; the load-bearing
		// proof of "multiple errors collected" is that both names appear.)
		require.Error(t, execErr)
		assert.Contains(t, execErr.Error(), "fail_action",
			"first failing action id must appear in the aggregated error")
		assert.Contains(t, execErr.Error(), "fail_action2",
			"second failing action id must appear in the aggregated error")

		// Both errored nodes should have PHASE_ERRORED.
		phases := collectPhases(ctx, failCh)
		require.NotEmpty(t, phases)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_ERRORED, phases[len(phases)-1],
			"fail_action terminal phase must be ERRORED")

		phases2 := collectPhases(ctx, failCh2)
		require.NotEmpty(t, phases2)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_ERRORED, phases2[len(phases2)-1],
			"fail_action2 terminal phase must be ERRORED")
	})
}

// Flow.error_strategy round-trips through the YAML decoder: a positive
// assertion that the spec field is parsed (independent of whether the
// runtime honors it).

func TestErrorStrategy_FromSpec_Decodes(t *testing.T) {
	f, err := os.Open("testdata/error_strategy_continue.yaml")
	require.NoError(t, err)
	defer f.Close() //nolint:errcheck // fixture opened read-only and fully read below; close error cannot affect the parsed result
	spec, err := flowsdkv1beta2.ReadSpec(encoding.YAML, f)
	require.NoError(t, err)
	require.Equal(t,
		flowv1beta2.ErrorStrategy_ERROR_STRATEGY_CONTINUE,
		spec.GetFlow().GetErrorStrategy(),
		"Flow.error_strategy must round-trip through YAML decode")
}

// CONTINUE: error_strategy declared in the Flow spec is honored end-to-end.
// No WithErrorStrategy option -- the runtime picks up the strategy from the
// spec (carried through Graph.error_strategy by graph.Build()) and lets the
// generator path complete despite the failed input path.

func TestErrorStrategy_Continue_FromSpec(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "error_strategy_continue.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.fail_input", 99)

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")

		results := collectOutputs(ctx, ps, "outputs.ok_result")
		require.Len(t, results, 3, "generator path should produce all values")
		assert.Equal(t, []int64{1, 2, 3}, outputInt64s(results))
	})
}
