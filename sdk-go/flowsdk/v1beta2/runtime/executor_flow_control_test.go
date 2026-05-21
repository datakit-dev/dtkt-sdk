package runtime

import (
	"testing"
	"time"

	expr "cel.dev/expr"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// --- flow_control: stop_when (literal constants) ---

// Range generator with stop_when stops the flow after 5 values.

func TestFlowControl_Range_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_range_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// stop_when fires at eval_count >= 5; the output that triggers stop is
		// included (stop is graceful). Exactly 5 values: 1..5.
		assert.Equal(t, []int64{1, 2, 3, 4, 5}, results)
	})
}

// Ticker generator with stop_when stops after 3 ticks.

func TestFlowControl_Ticker_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_ticker_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		// stop_when fires at eval_count >= 3. Exactly 3 results.
		require.Len(t, results, 3)
	})
}

// Var with stop_when on a generator-sourced flow. The range generator
// emits up to 100 values; FC.stop_when on the var fires when the
// generator's value reaches 5. Without FC.stop firing, the generator
// would run to 100 and the test would hit testContext.
//
// Spec contract: FC.stop_when on the var triggers gracefulStop, which
// signals stopCh on the generator (executor_setup.go:181-190). The
// generator exits at its next safe point; the var drains any in-flight
// values; the flow exits with nil within bounded time.
//
// Fixture is generator-sourced because with the previous finite-input
// fixture, natural drain produced the same outputs as graceful stop --
// the test passed whether FC.stop fired or not (vacuous-test pattern).

func TestFlowControl_Var_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_var_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		start := time.Now()
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		elapsed := time.Since(start)
		require.NoError(t, err)

		// Without FC.stop firing, the flow would run for the full
		// 100-element range. Promptly returning proves the stop
		// path engaged.
		assert.Less(t, elapsed, 2*time.Second,
			"FC.stop_when must terminate the generator promptly; running >2s indicates stop never fired")

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// stop_when triggers at generators.seq.value>=5. With graceful
		// drain, the trigger value's emission (10) plus in-flight earlier
		// emissions surface. Upper bound is loose because publish->stopCh
		// has the same race as fc_var_terminate (publish then
		// checkLifecycle). Lower bound enforces that at least the trigger
		// value's doubled emission lands.
		require.GreaterOrEqual(t, len(results), 1,
			"trigger value's emission must surface before stop fires")
		require.Less(t, len(results), 50,
			"FC.stop_when must bound the run well below the 100-element range; "+
				"len near full range indicates stop didn't fire")
		// Each emit must be 2*N for some sequential N starting at 1.
		for i, v := range results {
			assert.Equal(t, int64((i+1)*2), v,
				"result[%d] must equal 2*(i+1) -- generator + double", i)
		}
	})
}

// Action with stop_when on a generator-sourced flow. Redesigned from the
// previous finite-input fixture which was vacuous. With a generator,
// FC.stop firing is the only way the flow exits in bounded time.

func TestFlowControl_Action_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_action_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		start := time.Now()
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		elapsed := time.Since(start)
		require.NoError(t, err)

		assert.Less(t, elapsed, 2*time.Second,
			"FC.stop_when must terminate promptly; >2s means stop never fired")

		results := collectOutputs(ctx, ps, "outputs.result")
		require.GreaterOrEqual(t, len(results), 1,
			"trigger value's emission must surface")
		require.Less(t, len(results), 50,
			"FC.stop_when must bound the run well below the 100-element range (stop never fired)")
		for i, r := range results {
			assert.Equal(t, int64(i+1), r.GetValue().GetInt64Value(),
				"result[%d] must equal i+1 (echoed generator value)", i)
		}
	})
}

// Output with stop_when on a generator-sourced flow. Redesigned from the
// previous finite-input fixture which was vacuous (natural drain ==
// graceful drain on the wire). With a generator, FC.stop firing is the
// only way the flow exits in bounded time.

func TestFlowControl_Output_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_output_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		start := time.Now()
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		elapsed := time.Since(start)
		require.NoError(t, err)

		assert.Less(t, elapsed, 2*time.Second,
			"FC.stop_when must terminate promptly; >2s means stop never fired")

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// Trigger at gen value >= 5. At least one output (trigger value) must
		// surface; far below the 100-element range bound.
		require.GreaterOrEqual(t, len(results), 1,
			"trigger value's emission must surface")
		require.Less(t, len(results), 50,
			"FC.stop_when must bound the run well below the 100-element range (stop never fired)")
		for i, v := range results {
			assert.Equal(t, int64(i+1), v, "result[%d] must equal i+1 (generator value)", i)
		}
	})
}

// --- flow_control: terminate_when ---

// Range generator with terminate_when terminates the flow.
// terminate_when is on the output node: all values up to and including
// the triggering eval are already published before Terminate() fires.

func TestFlowControl_Range_TerminateWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_range_terminate.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.ErrorIs(t, err, ErrTerminated)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// terminate_when fires at eval_count >= 3. Values 1, 2, 3 are
		// published by the output handler before Terminate() executes.
		assert.Equal(t, []int64{1, 2, 3}, results)
	})
}

// --- Priority: terminate_when > stop_when ---

// When both terminate_when and stop_when are set, terminate wins.

func TestFlowControl_TerminatePriority(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_terminate_priority.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.ErrorIs(t, err, ErrTerminated)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// terminate_when fires at eval_count >= 3 (before stop_when at >= 5).
		assert.Equal(t, []int64{1, 2, 3}, results)
	})
}

// Ticker generator with stop_when driven by an input's default value.
// The stop_when is on the output so both generators.tick and inputs.maxIters
// are in the output's subscription scope. The input uses its default (10)
// via the minimum throttle fallback -- no feedInput call.

func TestFlowControl_Ticker_StopWhen_Input(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_ticker_stop_input.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		// stop_when fires at eval_count >= maxIters (default 10). Exactly 10.
		require.Len(t, results, 10)
	})
}

// --- flow_control: terminate_when on var ---

// Var with terminate_when terminates the flow when condition is met.
// terminate_when is on the var node (upstream of the output). Terminate
// cancels runCtx immediately, but in-flight iterations complete before
// the next recv() sees ctx.Done -- so 1-2 extra outputs may surface
// past the trigger value before the cancellation propagates. We assert
// only prefix-equality (no upper bound on length) and that the flow
// surfaces ErrTerminated.
//
// Without an upper bound the test was previously flaky under load
// (docs/flaky-tests.md). The prefix-equality is the meaningful invariant:
// outputs must be a valid prefix of the full-drain sequence
// [2, 4, 6, 8, 10] -- if extras surface they must still be in order.

func TestFlowControl_Var_TerminateWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_var_terminate.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.ErrorIs(t, err, ErrTerminated)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// Outputs must be a prefix of the full-drain sequence -- in-flight
		// iterations may complete before terminate propagates, so any
		// length 0..5 is acceptable as long as values appear in order.
		full := []int64{2, 4, 6, 8, 10}
		require.LessOrEqual(t, len(results), len(full))
		assert.Equal(t, full[:len(results)], results,
			"outputs must be a prefix of %v, got %v", full, results)
	})
}

// --- flow_control: terminate_when on action ---

// Action with terminate_when (always true) terminates after the first call.
// terminate_when is on the action node. Context cancel may arrive before
// the output handler processes the action result.

func TestFlowControl_Action_TerminateWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_action_terminate.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(42))

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		require.ErrorIs(t, err, ErrTerminated)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.LessOrEqual(t, len(results), 1)
		if len(results) == 1 {
			assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
		}
	})
}

// --- flow_control: stop_when on stream ---

// Stream with stop_when on a generator-sourced flow. Redesigned from the
// previous finite-input fixture which was vacuous. With a generator,
// FC.stop firing is the only way the flow exits in bounded time.

func TestFlowControl_Stream_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_stream_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		start := time.Now()
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		elapsed := time.Since(start)
		require.NoError(t, err)

		assert.Less(t, elapsed, 2*time.Second,
			"FC.stop_when must terminate promptly; >2s means stop never fired")

		results := collectOutputs(ctx, ps, "outputs.result")
		require.GreaterOrEqual(t, len(results), 1,
			"trigger value's emission must surface")
		require.Less(t, len(results), 50,
			"FC.stop_when must bound the run well below the 100-element range (stop never fired)")
		for i, r := range results {
			assert.Equal(t, int64(i+1), r.GetValue().GetInt64Value(),
				"result[%d] must equal i+1 (echoed generator value)", i)
		}
	})
}

// --- flow_control: terminate_when on stream ---

// Stream with terminate_when (always true) terminates after the first RPC.
// terminate_when on the stream node cancels context immediately -- the output
// handler may or may not have processed the stream's result before cancellation.

func TestFlowControl_Stream_TerminateWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_stream_terminate.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", "hello")

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		require.ErrorIs(t, err, ErrTerminated)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.LessOrEqual(t, len(results), 1)
		if len(results) == 1 {
			assert.Equal(t, "hello", results[0].GetValue().GetStringValue())
		}
	})
}

// --- flow_control: stop_when on interaction ---

// Interaction with stop_when on a generator-sourced flow. Redesigned from
// the previous finite-input fixture which was vacuous. With a generator,
// FC.stop firing is the only way the flow exits in bounded time.

func TestFlowControl_Interaction_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_interaction_stop.yaml")

		promptCh := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		responseCh := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Auto-respond to interaction prompts.
		go func() {
			for p := range promptCh {
				anyVal, _ := common.WrapProtoAny(int64(100))
				responseCh <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(responseCh)
		}()

		ctx := testContext(t)
		start := time.Now()
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(promptCh, responseCh)}, extraOpts...)...).Execute(ctx, graph)
		elapsed := time.Since(start)
		close(promptCh) // Unblock auto-respond goroutine.
		require.NoError(t, err)

		assert.Less(t, elapsed, 2*time.Second,
			"FC.stop_when must terminate promptly; >2s means stop never fired")

		results := collectOutputs(ctx, ps, "outputs.result")
		// At least one prompt+response must surface; far below the
		// 100-element range bound.
		require.GreaterOrEqual(t, len(results), 1,
			"trigger value's emission must surface")
		require.Less(t, len(results), 50,
			"FC.stop_when must bound the run well below the 100-element range (stop never fired)")
		for i, r := range results {
			assert.Equal(t, int64(100), r.GetValue().GetInt64Value(),
				"result[%d] must equal 100 (auto-responder)", i)
		}
	})
}

// --- flow_control: terminate_when on interaction ---

// Interaction with terminate_when (always true) terminates after the first interaction.
// terminate_when on the interaction node cancels context immediately -- the output
// handler may or may not have processed the interaction's result before cancellation.

func TestFlowControl_Interaction_TerminateWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_interaction_terminate.yaml")

		promptCh := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		responseCh := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Auto-respond to interaction prompts.
		go func() {
			for p := range promptCh {
				anyVal, _ := common.WrapProtoAny(int64(100))
				responseCh <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(responseCh)
		}()

		feedInput(ps, "inputs.x", int64(1))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(promptCh, responseCh)}, extraOpts...)...).Execute(ctx, graph)
		close(promptCh) // Unblock auto-respond goroutine.
		require.ErrorIs(t, err, ErrTerminated)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.LessOrEqual(t, len(results), 1)
		if len(results) == 1 {
			assert.Equal(t, int64(100), results[0].GetValue().GetInt64Value())
		}
	})
}

// --- flow_control: suspend_when on output ---

// Output with suspend_when (always true) suspends the flow after the first output.
// We verify the suspend by observing the output node enter PHASE_SUSPENDED,
// then terminate to unblock the executor.

func TestFlowControl_Output_SuspendWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_output_suspend.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Multiple inputs queued so a "lying" suspend would emit more values.
		feedInput(ps, "inputs.x", int64(42), int64(43), int64(44))
		ctx := testContext(t)

		// Subscribe to the output node's internal topic to observe the suspend phase.
		outCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// The output should emit its value and then enter PHASE_SUSPENDED.
		require.True(t, waitForPhase(ctx, outCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected output to reach PHASE_SUSPENDED")
		// Behavioral check: suspended handler must not emit further outputs.
		assertNoOutputDuring(t, outCh, 100*time.Millisecond)

		// Terminate to unblock the executor.
		exec.Terminate()

		execErr := requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, execErr, ErrTerminated)
	})
}

// --- flow_control: suspend_when on var ---

// Var with suspend_when (always true) suspends the flow after the first var evaluation.

func TestFlowControl_Var_SuspendWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_var_suspend.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5))
		ctx := testContext(t)

		// Subscribe to the var node's internal topic to observe the suspend phase.
		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		require.True(t, waitForPhase(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected var to reach PHASE_SUSPENDED")
		assertNoOutputDuring(t, varCh, 100*time.Millisecond)

		exec.Terminate()

		execErr := requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, execErr, ErrTerminated)
	})
}

// --- flow_control: suspend_when on action ---

// Action with suspend_when (always true) suspends the flow after the first RPC.

func TestFlowControl_Action_SuspendWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_action_suspend.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(42), int64(99), int64(7))
		ctx := testContext(t)

		// Subscribe to the action node's internal topic to observe the suspend phase.
		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.call"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		require.True(t, waitForPhase(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected action to reach PHASE_SUSPENDED")
		assertNoOutputDuring(t, actionCh, 100*time.Millisecond)

		exec.Terminate()

		execErr := requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, execErr, ErrTerminated)
	})
}

// --- flow_control: suspend_when on stream ---

// Stream with suspend_when (always true) suspends the flow after the first RPC.

func TestFlowControl_Stream_SuspendWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_stream_suspend.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", "hello", "world", "again")
		ctx := testContext(t)

		// Subscribe to the stream node's internal topic to observe the suspend phase.
		streamCh, err := ps.Subscribe(ctx, testTopics.For("streams.echo"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		require.True(t, waitForPhase(ctx, streamCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected stream to reach PHASE_SUSPENDED")
		assertNoOutputDuring(t, streamCh, 100*time.Millisecond)

		exec.Terminate()

		execErr := requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, execErr, ErrTerminated)
	})
}

// --- flow_control: suspend_when on interaction ---

// Interaction with suspend_when (always true) suspends the flow after the first interaction.

func TestFlowControl_Interaction_SuspendWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_interaction_suspend.yaml")

		promptCh := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		responseCh := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Auto-respond to interaction prompts.
		go func() {
			for p := range promptCh {
				anyVal, _ := common.WrapProtoAny(int64(100))
				responseCh <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(responseCh)
		}()

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3))
		ctx := testContext(t)

		// Subscribe to the interaction node's internal topic to observe the suspend phase.
		interCh, err := ps.Subscribe(ctx, testTopics.For("interactions.confirm"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, append([]Option{WithInteractions(promptCh, responseCh)}, extraOpts...)...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		require.True(t, waitForPhase(ctx, interCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected interaction to reach PHASE_SUSPENDED")
		assertNoOutputDuring(t, interCh, 100*time.Millisecond)

		exec.Terminate()

		close(promptCh) // Unblock auto-respond goroutine.
		execErr := requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, execErr, ErrTerminated)
	})
}

// --- flow_control: output with suspend_when then stop_when (CEL-suspend + operator-resume + CEL-stop) ---

// Output with two flow_control conditions evaluated in sequence on the same
// handler. The ticker drives eval_count; suspend_when==3 parks the handler
// via the FC callback (e.Suspend()), the test calls exec.Resume(), and then
// stop_when>=6 fires a gracefulStop that drains the flow cleanly.
//
// This covers a gap left by TestFlowControl_Output_SuspendWhen and friends:
// those tests terminate immediately after observing PHASE_SUSPENDED and never
// prove that Resume() actually unparks a handler that was suspended via a CEL
// expression (as opposed to operator-driven exec.Suspend()). It also proves
// that a later flow_control condition on the same node continues to fire
// after the resume.
//
// Expected outputs.result sequence: [1, 2, 3, 4, 5, 6].

func TestFlowControl_Output_SuspendResumeStop(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_output_suspend_resume_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)

		// Subscribe BEFORE Execute starts so we capture every event from
		// the first tick onwards. The output topic carries both
		// NODE_OUTPUT events (the emitted values) and NODE_STATE events
		// (phase transitions including PHASE_SUSPENDED).
		outCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// The output handler publishes the tick value, then calls
		// checkLifecycle (output.go:103-106) which may fire suspend_when
		// after the third emission. We expect values 1, 2, 3 to land,
		// then the handler to park.

		// First three values arrive before suspend_when fires.
		for i, want := range []int64{1, 2, 3} {
			got := readNextInt64Output(t, outCh, 2*time.Second)
			require.Equalf(t, want, got, "pre-suspend value %d: got %d, want %d", i, got, want)
		}

		// After the third emit, suspend_when (eval_count==3) fires and
		// parks the whole flow. No further NODE_OUTPUT must arrive until
		// we Resume. 200ms is a comfortable margin above the 50ms tick
		// interval -- if Resume is broken or the handler isn't actually
		// parked, we'd see tick 4 land here.
		assertNoOutputDuring(t, outCh, 200*time.Millisecond)

		// Resume. This is the path no existing FC suspend test exercises:
		// the handler was parked by the FC callback, and we're proving
		// the operator Resume() wakes it.
		exec.Resume()

		// Ticks 4, 5, 6 must land. Value 6 is emitted before stop_when
		// fires (checkLifecycle runs after the publish), so all six
		// values surface in the [1..6] sequence.
		for i, want := range []int64{4, 5, 6} {
			got := readNextInt64Output(t, outCh, 2*time.Second)
			require.Equalf(t, want, got, "post-resume value %d: got %d, want %d", i, got, want)
		}

		// stop_when (eval_count>=6) calls gracefulStop, which cascades
		// EOF through the flow. Execute must return nil (clean drain),
		// not ErrTerminated.
		execErr := requireExecuteReturnsBy(t, done, 2*time.Second)
		require.NoError(t, execErr,
			"stop_when on output must trigger a clean drain (gracefulStop), "+
				"not a terminate; if this returns ErrTerminated the FC stop "+
				"callback is wired to e.Terminate instead of gracefulStop")
	})
}

// readNextInt64Output is the shared reader for the suspend/resume/stop
// tests. It pulls the next NODE_OUTPUT event off the subscribed topic
// (skipping phase-only events), unwraps the typed node's value to int64,
// and fails the test on timeout, closed-stream, or EOF. Works against any
// handler topic (vars.*, actions.*, streams.*, interactions.*, outputs.*)
// via runtimeNodeFromEvent + the shared StateNode.GetValue() interface.
//
// Subscribe to the handler's OWN topic, not a downstream output's topic,
// to avoid races where e.Suspend() publishes a PHASE_SUSPENDED to the
// downstream node before the upstream's data events have been forwarded
// through the outbox relay.
//
// Two value forms are handled: raw `Value_Int64Value` (vars/actions/
// streams whose CEL returns the input value directly) and `ObjectValue`
// wrapping a `wrapperspb.Int64Value` Any (interactions, where the
// response value is delivered as an Any via WrapProtoAny on the
// responder side). Downstream consumers see both forms transparently
// because CEL eval unwraps Any; this helper does the same so tests can
// subscribe directly to the producing handler's topic.
func readNextInt64Output(t *testing.T, ch <-chan *pubsub.Message, budget time.Duration) int64 {
	t.Helper()
	deadline := time.After(budget)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for next NODE_OUTPUT within %v", budget)
		case msg := <-ch:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			if evt.GetEventType() != flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
				continue
			}
			node := runtimeNodeFromEvent(evt)
			if node == nil || isEOFValue(node.GetValue()) {
				t.Fatalf("topic closed before sequence drained (saw EOF/closed mid-test)")
			}
			v := node.GetValue()
			if i := v.GetInt64Value(); i != 0 || v.GetKind() != nil {
				if _, ok := v.GetKind().(*expr.Value_Int64Value); ok {
					return i
				}
			}
			if obj := v.GetObjectValue(); obj != nil {
				var wrap wrapperspb.Int64Value
				if err := obj.UnmarshalTo(&wrap); err == nil {
					return wrap.GetValue()
				}
				t.Fatalf("NODE_OUTPUT value is an Any but not Int64Value: %v", obj.GetTypeUrl())
			}
			t.Fatalf("NODE_OUTPUT value is neither Int64Value nor Any{Int64Value}: %v", v)
			return 0
		}
	}
}

// --- flow_control: var with suspend_when then stop_when (CEL-suspend + operator-resume + CEL-stop) ---

// Var with two flow_control conditions evaluated in sequence on the same
// handler. Parallel to TestFlowControl_Output_SuspendResumeStop: the
// existing TestFlowControl_Var_SuspendWhen terminates after observing
// PHASE_SUSPENDED and never proves Resume() unparks a CEL-suspended var.
// This test does. Expected outputs.result sequence: [1, 2, 3, 4, 5, 6].

func TestFlowControl_Var_SuspendResumeStop(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_var_suspend_resume_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)

		// Subscribe to the VAR's own topic (where its data events publish).
		// Subscribing to outputs.result would race in outbox mode: when the
		// var's checkLifecycle fires e.Suspend(), publishSuspendedPhase
		// commits a PHASE_SUSPENDED to outputs.result via the outbox; the
		// relay may forward that phase event before the var's earlier data
		// events have been forwarded to the output handler for processing.
		outCh, err := ps.Subscribe(ctx, testTopics.For("vars.passthrough"))
		require.NoError(t, err)

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5), int64(6))

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// First three values land before suspend_when (inputs.x.value==3) fires.
		for i, want := range []int64{1, 2, 3} {
			got := readNextInt64Output(t, outCh, 2*time.Second)
			require.Equalf(t, want, got, "pre-suspend value %d: got %d, want %d", i, got, want)
		}

		// Handler parks after the third emit. No further NODE_OUTPUT must
		// land until we Resume. 200ms is comfortable margin -- if Resume
		// is broken or the handler isn't parked, input 4 would surface.
		assertNoOutputDuring(t, outCh, 200*time.Millisecond)

		exec.Resume()

		// 4, 5, 6 land post-resume. stop_when fires after 6 publishes.
		for i, want := range []int64{4, 5, 6} {
			got := readNextInt64Output(t, outCh, 2*time.Second)
			require.Equalf(t, want, got, "post-resume value %d: got %d, want %d", i, got, want)
		}

		execErr := requireExecuteReturnsBy(t, done, 2*time.Second)
		require.NoError(t, execErr,
			"stop_when on var must trigger a clean drain (gracefulStop), "+
				"not a terminate")
	})
}

// --- flow_control: action with suspend_when then stop_when ---

// Action equivalent of TestFlowControl_Var_SuspendResumeStop. echo.Echo is
// a registered unary that returns the request value unchanged, so feeding
// [1..6] yields outputs [1..6] with the same suspend/resume/stop boundaries.

func TestFlowControl_Action_SuspendResumeStop(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_action_suspend_resume_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		// Subscribe to the ACTION's own topic. See var test for the
		// rationale -- avoids the outbox-relay race against
		// publishSuspendedPhase on downstream output handlers.
		outCh, err := ps.Subscribe(ctx, testTopics.For("actions.call"))
		require.NoError(t, err)

		feedInput(ps, "inputs.msg", int64(1), int64(2), int64(3), int64(4), int64(5), int64(6))

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		for i, want := range []int64{1, 2, 3} {
			got := readNextInt64Output(t, outCh, 2*time.Second)
			require.Equalf(t, want, got, "pre-suspend value %d: got %d, want %d", i, got, want)
		}

		assertNoOutputDuring(t, outCh, 200*time.Millisecond)

		exec.Resume()

		for i, want := range []int64{4, 5, 6} {
			got := readNextInt64Output(t, outCh, 2*time.Second)
			require.Equalf(t, want, got, "post-resume value %d: got %d, want %d", i, got, want)
		}

		execErr := requireExecuteReturnsBy(t, done, 2*time.Second)
		require.NoError(t, execErr,
			"stop_when on action must trigger a clean drain (gracefulStop), "+
				"not a terminate")
	})
}

// --- flow_control: stream with suspend_when then stop_when ---

// Stream equivalent of TestFlowControl_Var_SuspendResumeStop. Uses
// echo.Echo (the unary echo wrapped as a stream node, same pattern as
// fc_stream_stop.yaml and fc_stream_suspend.yaml).

func TestFlowControl_Stream_SuspendResumeStop(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_stream_suspend_resume_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		// Subscribe to the STREAM's own topic. See var test for the
		// rationale -- avoids the outbox-relay race against
		// publishSuspendedPhase on downstream output handlers.
		outCh, err := ps.Subscribe(ctx, testTopics.For("streams.echo"))
		require.NoError(t, err)

		feedInput(ps, "inputs.msg", int64(1), int64(2), int64(3), int64(4), int64(5), int64(6))

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		for i, want := range []int64{1, 2, 3} {
			got := readNextInt64Output(t, outCh, 2*time.Second)
			require.Equalf(t, want, got, "pre-suspend value %d: got %d, want %d", i, got, want)
		}

		assertNoOutputDuring(t, outCh, 200*time.Millisecond)

		exec.Resume()

		for i, want := range []int64{4, 5, 6} {
			got := readNextInt64Output(t, outCh, 2*time.Second)
			require.Equalf(t, want, got, "post-resume value %d: got %d, want %d", i, got, want)
		}

		execErr := requireExecuteReturnsBy(t, done, 2*time.Second)
		require.NoError(t, execErr,
			"stop_when on stream must trigger a clean drain (gracefulStop), "+
				"not a terminate")
	})
}

// --- flow_control: interaction with suspend_when then stop_when ---

// Interaction equivalent of TestFlowControl_Var_SuspendResumeStop. The
// auto-responder echoes back the original input value 1:1, so feeding
// [1..6] yields interaction responses [1..6] and outputs [1..6]. Suspend
// fires AFTER the response for value 3 has been delivered and published,
// since checkLifecycle runs after the publish (interaction.go:234).

func TestFlowControl_Interaction_SuspendResumeStop(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_interaction_suspend_resume_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Echo responder: response value = original input fed at the same
		// position. Inputs are sent in order, so the Nth prompt's response
		// is the Nth input value.
		inputs := []int64{1, 2, 3, 4, 5, 6}
		promptCh := make(chan *flowv1beta2.InteractionRequestEvent, len(inputs))
		responseCh := make(chan *flowv1beta2.InteractionResponseEvent, len(inputs))

		go func() {
			idx := 0
			for p := range promptCh {
				if idx >= len(inputs) {
					return
				}
				anyVal, _ := common.WrapProtoAny(inputs[idx])
				responseCh <- &flowv1beta2.InteractionResponseEvent{
					Id:    p.GetId(),
					Token: p.GetToken(),
					Value: anyVal,
				}
				idx++
			}
		}()

		ctx := testContext(t)
		// Subscribe to the INTERACTION's own topic. See var test for the
		// rationale -- avoids the outbox-relay race against
		// publishSuspendedPhase on downstream output handlers.
		outCh, err := ps.Subscribe(ctx, testTopics.For("interactions.confirm"))
		require.NoError(t, err)

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5), int64(6))

		opts := append([]Option{WithInteractions(promptCh, responseCh)}, extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		for i, want := range []int64{1, 2, 3} {
			got := readNextInt64Output(t, outCh, 2*time.Second)
			require.Equalf(t, want, got, "pre-suspend value %d: got %d, want %d", i, got, want)
		}

		assertNoOutputDuring(t, outCh, 200*time.Millisecond)

		exec.Resume()

		for i, want := range []int64{4, 5, 6} {
			got := readNextInt64Output(t, outCh, 2*time.Second)
			require.Equalf(t, want, got, "post-resume value %d: got %d, want %d", i, got, want)
		}

		execErr := requireExecuteReturnsBy(t, done, 2*time.Second)
		require.NoError(t, execErr,
			"stop_when on interaction must trigger a clean drain (gracefulStop), "+
				"not a terminate")

		close(promptCh)
	})
}

// --- flow_control: same-iteration suspend-vs-stop priority ---

// FlowControl priority contract (checkLifecycleControl in flow_control.go):
//   terminate > suspend > stop
//
// Test: feed an input value that satisfies BOTH suspend_when and stop_when
// on the same iteration. If suspend wins (correct), the handler parks. If
// stop wins (regression), the handler drains and Execute returns.
//
// Discriminator: PHASE_SUSPENDED must land for the var before Execute
// returns. Then a SECOND input that satisfies ONLY stop_when (not
// suspend_when) is fed after Resume to confirm stop_when still works on
// its own and the run completes cleanly.

func TestFlowControl_SuspendOverStop_Priority(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_suspend_over_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)

		// Subscribe to the var's own topic before Execute starts.
		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.passthrough"))
		require.NoError(t, err)

		// Value 5 fires both suspend_when (==5) and stop_when (>=5).
		// Value 6 fires only stop_when. The contract: iteration on
		// value 5 must park (suspend wins), iteration on value 6 must
		// stop. If stop won on value 5 we'd never see PHASE_SUSPENDED
		// and Execute would return without our Resume.
		feedInput(ps, "inputs.x", int64(5), int64(6))

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Value 5 must publish before checkLifecycle fires.
		got := readNextInt64Output(t, varCh, 2*time.Second)
		require.Equal(t, int64(5), got,
			"value 5 must publish before suspend_when fires (publish-then-checkLifecycle order)")

		// Suspend wins over stop on the same iteration -> handler
		// reaches PHASE_SUSPENDED. If stop won instead, the var would
		// drain and we'd see PHASE_SUCCEEDED here (and Execute would
		// return nil immediately).
		requirePhaseWithin(t, varCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED, 1*time.Second)

		// Execute must NOT have returned yet -- suspend parks the flow,
		// it does not terminate it. A regression where stop wins would
		// have closed Execute by now.
		select {
		case err := <-done:
			t.Fatalf("Execute returned during suspend; suspend_when priority over stop_when must keep the flow alive. err=%v", err)
		default:
		}

		exec.Resume()

		// Value 6 publishes, then stop_when (only) fires -> graceful
		// stop -> handler drains and exits.
		got = readNextInt64Output(t, varCh, 2*time.Second)
		require.Equal(t, int64(6), got,
			"value 6 must publish after Resume before stop_when fires")

		execErr := requireExecuteReturnsBy(t, done, 2*time.Second)
		require.NoError(t, execErr,
			"after stop_when fires on iter 2 (without a co-firing suspend), gracefulStop "+
				"must drain the flow cleanly; if this returns non-nil, stop_when's drain "+
				"path is broken once a prior suspend has been resumed")
	})
}

// --- flow_control: ticker + var stop_when referencing input ---

// Ticker generator with stop_when on a var that references both the generator
// (via its value expression) and an input (via stop_when). This mirrors the
// ticker.yaml example flow: the var computes even/odd from the tick count and
// stops when the tick count reaches the input-provided max iterations.

func TestFlowControl_Ticker_Var_StopWhen_Input(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_ticker_var_stop_input.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		// Collect outputs from all three output nodes.
		ticks := collectOutputs(ctx, ps, "outputs.tick")
		evenOrOdds := collectOutputs(ctx, ps, "outputs.evenOrOdd")
		tickAndEvenOrOdds := collectOutputs(ctx, ps, "outputs.tickAndEvenOrOdd")

		// stop_when fires at eval_count >= 10 (default): exactly 10 outputs.
		require.Len(t, ticks, 10)
		require.Len(t, evenOrOdds, 10)
		require.Len(t, tickAndEvenOrOdds, 10)

		// evenOrOdd alternates: odd(1), even(2), odd(3), ...
		expectedEvenOrOdd := []string{"odd", "even", "odd", "even", "odd", "even", "odd", "even", "odd", "even"}
		assert.Equal(t, expectedEvenOrOdd, outputStrings(evenOrOdds))

		// tickAndEvenOrOdd: [eval_count, evenOrOdd] pairs.
		for i, out := range tickAndEvenOrOdds {
			listVals := out.GetValue().GetListValue().GetValues()
			require.Len(t, listVals, 2, "tickAndEvenOrOdd[%d]", i)
			assert.Equal(t, int64(i+1), listVals[0].GetInt64Value(), "tickAndEvenOrOdd[%d] eval_count", i)
			assert.Equal(t, expectedEvenOrOdd[i], listVals[1].GetStringValue(), "tickAndEvenOrOdd[%d] value", i)
		}

		// collectOutputs reads until Closed: true (EOF), proving each output
		// published its terminal marker and the flow cleaned up properly.
		// Execute returning nil error confirms no handler leaked or hung.
	})
}
