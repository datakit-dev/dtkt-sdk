package runtime

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- flow_control: stop_when (literal constants) ---

// Range generator with stop_when stops the flow after 5 values.

func TestFlowControl_Range_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_range_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

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
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		// stop_when fires at eval_count >= 3. Exactly 3 results.
		require.Len(t, results, 3)
	})
}

// Var with stop_when stops the flow when the condition is met.
// stop is graceful: all buffered inputs are drained before shutdown.

func TestFlowControl_Var_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_var_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Feed values 1..5. stop_when triggers at inputs.x.value >= 3, but stop
		// is graceful -- all buffered inputs (1..5) are processed and drained.
		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// All 5 inputs are processed (doubled): 2, 4, 6, 8, 10.
		assert.Equal(t, []int64{2, 4, 6, 8, 10}, results)
	})
}

// Action with stop_when (always true) stops after the first call.
// stop is graceful: all buffered inputs are drained.

func TestFlowControl_Action_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_action_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", int64(42), int64(99))

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		// stop_when is always true, but stop is graceful -- both buffered
		// inputs (42, 99) are processed. Exact outputs: [42, 99].
		require.Len(t, results, 2)
		assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
		assert.Equal(t, int64(99), results[1].GetValue().GetInt64Value())
	})
}

// Output with stop_when (always true) stops after the first output.
// stop is graceful: all buffered inputs are drained.

func TestFlowControl_Output_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_output_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.x", int64(10), int64(20), int64(30))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// stop_when is always true, but stop is graceful -- all 3 buffered
		// inputs are processed. Exact outputs: [10, 20, 30].
		assert.Equal(t, []int64{10, 20, 30}, results)
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
		defer ps.Close() //nolint:errcheck

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
		defer ps.Close() //nolint:errcheck

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
		defer ps.Close() //nolint:errcheck

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
// cancels context immediately -- the output handler may or may not have
// processed the var's values before cancellation.

func TestFlowControl_Var_TerminateWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_var_terminate.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.ErrorIs(t, err, ErrTerminated)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// terminate fires at inputs.x.value >= 3, preventing inputs 4 and 5
		// from being processed. Any outputs that were produced must be a
		// valid prefix of [2, 4, 6] (doubled values for inputs 1, 2, 3).
		expected := []int64{2, 4, 6}
		require.LessOrEqual(t, len(results), len(expected),
			"terminate should prevent processing inputs beyond the trigger")
		if len(results) > 0 {
			assert.Equal(t, expected[:len(results)], results)
		}
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
		defer ps.Close() //nolint:errcheck

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

// Stream with stop_when (always true) stops after all buffered inputs.
// stop is graceful: both buffered inputs are processed.

func TestFlowControl_Stream_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_stream_stop.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", "hello", "world")

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		// stop_when is always true, but stop is graceful -- both buffered
		// inputs are processed. Exactly 2 results.
		require.Len(t, results, 2)
		assert.Equal(t, "hello", results[0].GetValue().GetStringValue())
		assert.Equal(t, "world", results[1].GetValue().GetStringValue())
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
		defer ps.Close() //nolint:errcheck

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

// Interaction with stop_when (always true) stops after draining buffered inputs.

func TestFlowControl_Interaction_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_interaction_stop.yaml")

		promptCh := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		responseCh := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Auto-respond to interaction prompts.
		go func() {
			for p := range promptCh {
				anyVal, _ := common.WrapProtoAny(int64(100))
				responseCh <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(responseCh)
		}()

		feedInput(ps, "inputs.x", int64(1), int64(2))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(promptCh, responseCh)}, extraOpts...)...).Execute(ctx, graph)
		close(promptCh) // Unblock auto-respond goroutine.
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		// stop_when is always true, but stop is graceful -- both buffered
		// inputs are processed. Exactly 2 results, both 100.
		require.Len(t, results, 2)
		assert.Equal(t, int64(100), results[0].GetValue().GetInt64Value())
		assert.Equal(t, int64(100), results[1].GetValue().GetInt64Value())
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
		defer ps.Close() //nolint:errcheck

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
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.x", int64(42))
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

		// Terminate to unblock the executor.
		exec.Terminate()

		execErr := <-done
		assert.ErrorIs(t, execErr, ErrTerminated)
	})
}

// --- flow_control: suspend_when on var ---

// Var with suspend_when (always true) suspends the flow after the first var evaluation.

func TestFlowControl_Var_SuspendWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_var_suspend.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.x", int64(5))
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

		exec.Terminate()

		execErr := <-done
		assert.ErrorIs(t, execErr, ErrTerminated)
	})
}

// --- flow_control: suspend_when on action ---

// Action with suspend_when (always true) suspends the flow after the first RPC.

func TestFlowControl_Action_SuspendWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_action_suspend.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", int64(42))
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

		exec.Terminate()

		execErr := <-done
		assert.ErrorIs(t, execErr, ErrTerminated)
	})
}

// --- flow_control: suspend_when on stream ---

// Stream with suspend_when (always true) suspends the flow after the first RPC.

func TestFlowControl_Stream_SuspendWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fc_stream_suspend.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", "hello")
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

		exec.Terminate()

		execErr := <-done
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
		defer ps.Close() //nolint:errcheck

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

		exec.Terminate()

		execErr := <-done
		close(promptCh) // Unblock auto-respond goroutine.
		assert.ErrorIs(t, execErr, ErrTerminated)
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
		defer ps.Close() //nolint:errcheck

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
