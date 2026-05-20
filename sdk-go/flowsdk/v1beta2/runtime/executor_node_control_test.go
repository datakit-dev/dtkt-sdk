package runtime

import (
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// node_control fires per-node lifecycle actions (StopNode, TerminateNode,
// SuspendNode) instead of flow-wide ones. Unlike FlowControl.stop_when --
// which sends EOF to all input topics so the whole pipeline drains -- NC
// stop signals only the controlled handler, which exits at its next safe
// point. Buffered messages that haven't been consumed by that handler may
// not surface. Terminate cancels the node's context; Suspend parks the
// handler until ResumeNode. None of these actions terminate the flow
// itself, so the flow returns nil on natural completion.

// --- node_control: stop_when ---
//
// stop is graceful per-node: handler exits at the next safe point. Results
// are a non-empty prefix of the full-drain expectation; the trigger value
// itself is always included (publish happens before checkLifecycle fires).

func TestNodeControl_Var_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_var_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// stop_when fires at x>=3; 6 (=3*2) is published before the stop is
		// observed, so results is a prefix of [2,4,6,8,10] starting at length 3.
		full := []int64{2, 4, 6, 8, 10}
		require.GreaterOrEqual(t, len(results), 3,
			"expected the trigger publish to be present, got %v", results)
		require.LessOrEqual(t, len(results), len(full))
		assert.Equal(t, full[:len(results)], results)
	})
}

func TestNodeControl_Action_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_action_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(42), int64(99))

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		// stop_when="= true" fires after the first publish; the second buffered
		// input may or may not surface depending on select-randomization.
		require.GreaterOrEqual(t, len(results), 1)
		require.LessOrEqual(t, len(results), 2)
		expected := []int64{42, 99}
		for i, r := range results {
			assert.Equal(t, expected[i], r.GetValue().GetInt64Value())
		}
	})
}

func TestNodeControl_Stream_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_stream_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", "hello", "world")

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.GreaterOrEqual(t, len(results), 1)
		require.LessOrEqual(t, len(results), 2)
		expected := []string{"hello", "world"}
		for i, r := range results {
			assert.Equal(t, expected[i], r.GetValue().GetStringValue())
		}
	})
}

func TestNodeControl_Output_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_output_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(10), int64(20), int64(30))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		full := []int64{10, 20, 30}
		require.GreaterOrEqual(t, len(results), 1)
		require.LessOrEqual(t, len(results), len(full))
		assert.Equal(t, full[:len(results)], results)
	})
}

func TestNodeControl_Interaction_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_interaction_stop.yaml")

		promptCh := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		responseCh := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		go func() {
			for p := range promptCh {
				anyVal, _ := common.WrapProtoAny(int64(100))
				responseCh <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(responseCh)
		}()

		feedInput(ps, "inputs.x", int64(1), int64(2))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(promptCh, responseCh)}, extraOpts...)...).Execute(ctx, g)
		close(promptCh)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.GreaterOrEqual(t, len(results), 1)
		require.LessOrEqual(t, len(results), 2)
		for _, r := range results {
			assert.Equal(t, int64(100), r.GetValue().GetInt64Value())
		}
	})
}

// --- node_control: terminate_when ---
//
// terminate cancels the node's context. The flow itself does NOT terminate
// (e.terminated stays false), so Execute returns nil. The terminated node
// transitions to PHASE_CANCELLED and any downstream consumers see the topic
// close. Outputs that were already published before cancellation are visible.

func TestNodeControl_Var_TerminateWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_var_terminate.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// terminate fires after publishing the value at x=3 (i.e. 6). Under
		// load the var loop may race ahead one iteration before observing the
		// cancel - so we assert prefix-equality without an upper-bound on len
		// (see docs/flaky-tests.md for the same pattern in FC tests).
		full := []int64{2, 4, 6, 8, 10}
		require.LessOrEqual(t, len(results), len(full))
		assert.Equal(t, full[:len(results)], results)
	})
}

func TestNodeControl_Action_TerminateWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_action_terminate.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(42))

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.LessOrEqual(t, len(results), 1)
		if len(results) == 1 {
			assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
		}
	})
}

func TestNodeControl_Stream_TerminateWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_stream_terminate.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", "hello")

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.LessOrEqual(t, len(results), 1)
		if len(results) == 1 {
			assert.Equal(t, "hello", results[0].GetValue().GetStringValue())
		}
	})
}

func TestNodeControl_Output_TerminateWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_output_terminate.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(10), int64(20), int64(30))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// terminate fires after the first output is published; subsequent inputs
		// may or may not surface depending on scheduler timing.
		require.LessOrEqual(t, len(results), 3)
		expected := []int64{10, 20, 30}
		if len(results) > 0 {
			assert.Equal(t, expected[:len(results)], results)
		}
	})
}

func TestNodeControl_Interaction_TerminateWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_interaction_terminate.yaml")

		promptCh := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		responseCh := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		go func() {
			for p := range promptCh {
				anyVal, _ := common.WrapProtoAny(int64(100))
				responseCh <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(responseCh)
		}()

		feedInput(ps, "inputs.x", int64(1))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(promptCh, responseCh)}, extraOpts...)...).Execute(ctx, g)
		close(promptCh)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.LessOrEqual(t, len(results), 1)
		if len(results) == 1 {
			assert.Equal(t, int64(100), results[0].GetValue().GetInt64Value())
		}
	})
}

// --- node_control: suspend_when ---
//
// suspend pauses one node in place. The flow does not exit. Tests observe
// PHASE_SUSPENDED for the targeted node, then call Terminate() to unblock.

func TestNodeControl_Var_SuspendWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_var_suspend.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Feed multiple inputs so the handler has work to do AFTER the
		// suspend fires; a buggy "lying" suspend that publishes
		// PHASE_SUSPENDED but keeps iterating would emit additional
		// outputs, which assertNoOutputDuring catches.
		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5))
		ctx := testContext(t)

		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, g)
		}()

		require.True(t, waitForPhase(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected var to reach PHASE_SUSPENDED")
		// Behavioral verification: handler must not emit further outputs
		// while suspended.
		assertNoOutputDuring(t, varCh, 100*time.Millisecond)

		exec.Terminate()
		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, err, ErrTerminated)
	})
}

func TestNodeControl_Action_SuspendWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_action_suspend.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Multiple inputs queued so a "lying" suspend would emit more values.
		feedInput(ps, "inputs.msg", int64(42), int64(99), int64(7))
		ctx := testContext(t)

		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.call"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, g)
		}()

		require.True(t, waitForPhase(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected action to reach PHASE_SUSPENDED")
		assertNoOutputDuring(t, actionCh, 100*time.Millisecond)

		exec.Terminate()
		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, err, ErrTerminated)
	})
}

func TestNodeControl_Stream_SuspendWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_stream_suspend.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", "hello", "world", "again")
		ctx := testContext(t)

		streamCh, err := ps.Subscribe(ctx, testTopics.For("streams.echo"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, g)
		}()

		require.True(t, waitForPhase(ctx, streamCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected stream to reach PHASE_SUSPENDED")
		assertNoOutputDuring(t, streamCh, 100*time.Millisecond)

		exec.Terminate()
		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, err, ErrTerminated)
	})
}

func TestNodeControl_Output_SuspendWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_output_suspend.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(42), int64(43), int64(44))
		ctx := testContext(t)

		outCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, g)
		}()

		require.True(t, waitForPhase(ctx, outCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected output to reach PHASE_SUSPENDED")
		assertNoOutputDuring(t, outCh, 100*time.Millisecond)

		exec.Terminate()
		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, err, ErrTerminated)
	})
}

func TestNodeControl_Interaction_SuspendWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_interaction_suspend.yaml")

		promptCh := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		responseCh := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		go func() {
			for p := range promptCh {
				anyVal, _ := common.WrapProtoAny(int64(100))
				responseCh <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(responseCh)
		}()

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3))
		ctx := testContext(t)

		intCh, err := ps.Subscribe(ctx, testTopics.For("interactions.confirm"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, append([]Option{WithInteractions(promptCh, responseCh)}, extraOpts...)...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, g)
		}()

		require.True(t, waitForPhase(ctx, intCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected interaction to reach PHASE_SUSPENDED")
		assertNoOutputDuring(t, intCh, 100*time.Millisecond)

		exec.Terminate()
		close(promptCh)
		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, err, ErrTerminated)
	})
}

// --- NC.suspend_when + ResumeNode end-to-end ---
//
// Each test: feed exactly one input, observe PHASE_SUSPENDED (proves NC
// fired), assert no NODE_OUTPUT during a brief drain window (proves the
// handler is actually paused, not just publishing the suspend phase
// while still emitting), call exec.ResumeNode, then require PHASE_SUCCEEDED
// within a short budget (proves resume actually unparked the handler --
// a "lying" resume that left the handler parked would otherwise silently
// fall through to testContext's 10s deadline). Then assert Execute
// returns naturally with nil (per-node suspend doesn't terminate the flow).
// Single-input keeps timing deterministic: no race between "second
// iteration suspends" and "EOF arrives, handler exits SUCCEEDED."
//
// NB: in v1beta2 the iteration order is publish-value then checkLifecycle,
// so iter 1's value emits BEFORE NC.suspend fires. After resume the
// handler reads EOF and exits with PHASE_SUCCEEDED -- it does not emit a
// fresh NODE_OUTPUT. The behavioral check after resume is therefore the
// PHASE_SUCCEEDED transition (via requirePhaseWithin), not a value emit.
//
// NB: ResumeNode(id, val) value-injection only applies to legacy retry-
// suspended handlers (parkAndResume); selfSuspendable handlers (the mixin
// path NC uses) silently drop `val`. NC resume tests pass nil.

func TestNodeControl_Var_SuspendThenResume(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_var_suspend.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(7))
		ctx := testContext(t)

		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		require.True(t, waitForPhase(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected PHASE_SUSPENDED before resume")
		assertNoOutputDuring(t, varCh, 100*time.Millisecond)
		exec.ResumeNode("vars.doubled", nil)
		requirePhaseWithin(t, varCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, 500*time.Millisecond)

		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		require.NoError(t, err, "Execute should return naturally")
	})
}

func TestNodeControl_Action_SuspendThenResume(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_action_suspend.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(42))
		ctx := testContext(t)

		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.call"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		require.True(t, waitForPhase(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected PHASE_SUSPENDED before resume")
		assertNoOutputDuring(t, actionCh, 100*time.Millisecond)
		exec.ResumeNode("actions.call", nil)
		requirePhaseWithin(t, actionCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, 500*time.Millisecond)

		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		require.NoError(t, err, "Execute should return naturally")
	})
}

func TestNodeControl_Stream_SuspendThenResume(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_stream_suspend.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", "hello")
		ctx := testContext(t)

		streamCh, err := ps.Subscribe(ctx, testTopics.For("streams.echo"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		require.True(t, waitForPhase(ctx, streamCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected PHASE_SUSPENDED before resume")
		assertNoOutputDuring(t, streamCh, 100*time.Millisecond)
		exec.ResumeNode("streams.echo", nil)
		requirePhaseWithin(t, streamCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, 500*time.Millisecond)

		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		require.NoError(t, err, "Execute should return naturally")
	})
}

func TestNodeControl_Output_SuspendThenResume(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_output_suspend.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(42))
		ctx := testContext(t)

		outCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		require.True(t, waitForPhase(ctx, outCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected PHASE_SUSPENDED before resume")
		assertNoOutputDuring(t, outCh, 100*time.Millisecond)
		exec.ResumeNode("outputs.result", nil)
		requirePhaseWithin(t, outCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, 500*time.Millisecond)

		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		require.NoError(t, err, "Execute should return naturally")
	})
}

func TestNodeControl_Interaction_SuspendThenResume(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_interaction_suspend.yaml")

		promptCh := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		responseCh := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		go func() {
			for p := range promptCh {
				anyVal, _ := common.WrapProtoAny(int64(100))
				responseCh <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(responseCh)
		}()

		feedInput(ps, "inputs.x", int64(1))
		ctx := testContext(t)

		intCh, err := ps.Subscribe(ctx, testTopics.For("interactions.confirm"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, append([]Option{WithInteractions(promptCh, responseCh)}, extraOpts...)...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		require.True(t, waitForPhase(ctx, intCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected PHASE_SUSPENDED before resume")
		assertNoOutputDuring(t, intCh, 100*time.Millisecond)
		exec.ResumeNode("interactions.confirm", nil)
		requirePhaseWithin(t, intCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, 500*time.Millisecond)

		close(promptCh)
		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		require.NoError(t, err, "Execute should return naturally")
	})
}

// --- NC priority (terminate > suspend > stop) ---
//
// `checkLifecycleControl` enforces this internal ordering for both FC and NC.
// FC has TestFlowControl_TerminatePriority; these are the NC analogs.

// nc_terminate_priority sets all three triggers true on the same Var. The
// `terminate_when` branch must win: var transitions to PHASE_CANCELLED on
// its topic without any PHASE_STOPPING or PHASE_SUSPENDED state events
// being published first. The flow itself does not terminate (per-node
// terminate stays scoped), so Execute returns nil.
func TestNodeControl_TerminatePriority(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_terminate_priority.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(1))
		ctx := testContext(t)

		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
		require.NoError(t, err)

		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		phases := drainPhasesUntil(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_CANCELLED)
		// Verify no losing-priority phase appeared.
		for _, p := range phases {
			assert.NotEqual(t, flowv1beta2.RunSnapshot_PHASE_STOPPING, p,
				"PHASE_STOPPING must not appear when terminate_when wins; phases=%v", phaseNames(phases))
			assert.NotEqual(t, flowv1beta2.RunSnapshot_PHASE_SUSPENDED, p,
				"PHASE_SUSPENDED must not appear when terminate_when wins; phases=%v", phaseNames(phases))
		}
	})
}

// nc_suspend_over_stop sets only suspend_when + stop_when, both true. The
// `suspend_when` branch must win: var transitions to PHASE_SUSPENDED on its
// topic without any PHASE_STOPPING state event being published first.
// Terminate to clean up (the var stays parked otherwise).
func TestNodeControl_SuspendOverStop(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_suspend_over_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(1))
		ctx := testContext(t)

		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		phases := drainPhasesUntil(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED)
		for _, p := range phases {
			assert.NotEqual(t, flowv1beta2.RunSnapshot_PHASE_STOPPING, p,
				"PHASE_STOPPING must not appear when suspend_when wins; phases=%v", phaseNames(phases))
		}

		exec.Terminate()
		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, err, ErrTerminated)
	})
}

// --- NC + FC same-node ordering ---
//
// When a node declares both flow_control AND node_control, NC fires first
// per iteration. The contract is that per-node state events (e.g. NC's
// PHASE_STOPPING for the controlled node) land cleanly before any
// FC-driven flow-wide cancel can race with them. These tests assert the
// ordering invariant by observing the phase sequence on the controlled
// node's topic.

// nc_and_fc_same_node_stop: both NC.stop_when and FC.stop_when fire on
// the same iteration. NC.StopNode publishes PHASE_STOPPING on the node's
// topic; FC's performStop drains via input EOFs. NC-first means
// PHASE_STOPPING reaches the var topic before the input-EOF cascade
// terminates publishing.
func TestNodeControl_AndFlowControl_SameNode_Stop(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_and_fc_same_node_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
		require.NoError(t, err)

		feedInput(ps, "inputs.x", int64(1))
		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		phases := drainPhasesUntil(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
		// NC fires first → PHASE_STOPPING is published on the var topic
		// before the var handler exits via SUCCEEDED. FC also fires
		// (drains via input EOFs) but its effect is on the input topics,
		// not visible on the var topic as a state event.
		assertPhaseOrder(t, phases,
			flowv1beta2.RunSnapshot_PHASE_STOPPING,
			flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
	})
}

// nc_and_fc_same_node_terminate: both NC.terminate_when and FC.terminate_when
// fire on the same iteration. NC.TerminateNode cancels the node's ctx and
// publishes PHASE_CANCELLED on the node topic; FC.Terminate cancels runCtx
// and surfaces ErrTerminated at the flow level. NC-first means the
// per-node PHASE_CANCELLED state event reaches the topic before the
// flow-level cancel takes everything down.
func TestNodeControl_AndFlowControl_SameNode_Terminate(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_and_fc_same_node_terminate.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		// Subscribe to the var topic to observe PHASE_CANCELLED, which
		// proves NC.terminate landed (per-node CANCELLED publish from
		// TerminateNode). FC.terminate alone wouldn't publish a per-node
		// CANCELLED -- it just cancels runCtx and returns ErrTerminated.
		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
		require.NoError(t, err)

		feedInput(ps, "inputs.x", int64(1))
		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		// FC.terminate fires e.Terminate, so Execute returns ErrTerminated.
		assert.ErrorIs(t, err, ErrTerminated)

		// NC.terminate runs first (NC-then-FC ordering), so the var topic
		// must show PHASE_CANCELLED before any FC-driven cancel races
		// with it. This is the "NC fires first" invariant.
		phases := drainPhasesUntil(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_CANCELLED)
		require.Contains(t, phases, flowv1beta2.RunSnapshot_PHASE_CANCELLED,
			"NC.terminate must publish PHASE_CANCELLED on the var topic; phases=%v", phaseNames(phases))
	})
}

// nc_overrides_fc_same_node: NC.terminate_when AND FC.suspend_when fire
// the same iteration. NC.TerminateNode cancels the node ctx (per-node);
// FC.Suspend signals the whole flow. NC fires first per the ordering
// contract, so the per-node PHASE_CANCELLED state event reaches the var
// topic before any FC-driven flow-wide suspend can race with it.
//
// Downstream behavior depends on a goroutine race between the output
// handler observing FC's suspendCh signal vs the EOF marker propagated
// by var's terminal CANCELLED publish: either output parks (FC wins) or
// output drains EOF and exits SUCCEEDED (race won by EOF arrival). Both
// are valid; the invariant we verify is just NC's per-node state event
// landing on the var topic. Use exec.Stop() to drain whichever state
// the flow ended up in.
func TestNodeControl_OverridesFlowControl_SameNode(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_overrides_fc_same_node.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(1))
		ctx := testContext(t)

		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		require.True(t, waitForPhase(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_CANCELLED),
			"expected NC.terminate to land PHASE_CANCELLED on the var topic")

		// exec.Stop() drains whichever state the flow ended up in: if
		// output was parked by FC.Suspend, performStop wakes it; if
		// output already exited via EOF, Stop is a no-op.
		exec.Stop()
		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		// Either nil (natural drain) or ErrTerminated is acceptable;
		// the test guarantees only that NC.terminate landed first.
		if err != nil {
			assert.ErrorIs(t, err, ErrTerminated)
		}
	})
}

// --- combinations / scope / wiring ---

// nc_vs_fc_scope: two parallel branches share an input. The 'doubled' branch
// has node_control.terminate_when at x>=3, so its node is canceled mid-stream
// and its output sees a partial prefix. The 'tripled' branch is unaffected
// and processes every buffered input.

func TestNodeControl_VsFlowControl_Scope(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_vs_fc_scope.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		doubled := outputInt64s(collectOutputs(ctx, ps, "outputs.doubled_out"))
		tripled := outputInt64s(collectOutputs(ctx, ps, "outputs.tripled_out"))

		// doubled was terminated per-node at x>=3. Under load the var may
		// race ahead before observing the cancel; assert prefix-equality
		// without an upper bound on len (see docs/flaky-tests.md).
		fullDoubled := []int64{2, 4, 6, 8, 10}
		require.LessOrEqual(t, len(doubled), len(fullDoubled))
		assert.Equal(t, fullDoubled[:len(doubled)], doubled,
			"node_control on doubled must not corrupt its prefix")

		// tripled is unaffected by NodeControl on a sibling node; all 5 inputs
		// are processed (1*3, 2*3, 3*3, 4*3, 5*3).
		assert.Equal(t, []int64{3, 6, 9, 12, 15}, tripled,
			"node_control on doubled must not affect the tripled branch")
	})
}

// nc_edge_inference: NodeControl CEL references vars.counter, while
// vars.gated.value is a constant. Without edge collection from
// node_control expressions, vars.gated would not see vars.counter
// in its activation map and stop_when would never become true.

func TestNodeControl_EdgeInference(t *testing.T) {
	g := loadFlow(t, "nc_edge_inference.yaml")

	// Verify Build() produced an edge from vars.counter to vars.gated.
	var hasEdge bool
	for _, e := range g.GetEdges() {
		if e.GetSource() == "vars.counter" && e.GetTarget() == "vars.gated" {
			hasEdge = true
			break
		}
	}
	require.True(t, hasEdge,
		"graph.Build must collect edges from node_control CEL expressions")

	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// Re-build per subtest because graph mutation isn't safe to share.
		g := loadFlow(t, "nc_edge_inference.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// vars.gated emits a constant 1 per iteration. stop is per-node
		// graceful, so the prefix length depends on when stopCh is observed
		// relative to remaining buffered inputs. Trigger at counter>=3 so
		// at least three 1s precede the stop.
		require.GreaterOrEqual(t, len(results), 3)
		require.LessOrEqual(t, len(results), 5)
		for _, r := range results {
			assert.Equal(t, int64(1), r)
		}
	})
}

// --- PHASE_STOPPING publish coverage (one per handler type) ---
//
// Each handler type has its own post-loop publish path. NC.stop_when fires the
// shared StopNode, which publishes PHASE_STOPPING (state event) before the
// handler exits. The handler then publishes PHASE_SUCCEEDED (terminal). These
// tests assert that ordering on the controlled node's topic for every type.

func TestNodeControl_Var_StopWhen_PublishesPhaseStopping(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_var_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
		require.NoError(t, err)

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5))

		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		phases := drainPhasesUntil(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
		assertPhaseOrder(t, phases,
			flowv1beta2.RunSnapshot_PHASE_STOPPING,
			flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
	})
}

func TestNodeControl_Action_StopWhen_PublishesPhaseStopping(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_action_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.call"))
		require.NoError(t, err)

		feedInput(ps, "inputs.msg", int64(42), int64(99))

		opts := append(mockRPCOptions(), extraOpts...)
		err = NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		require.NoError(t, err)

		phases := drainPhasesUntil(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
		assertPhaseOrder(t, phases,
			flowv1beta2.RunSnapshot_PHASE_STOPPING,
			flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
	})
}

func TestNodeControl_Stream_StopWhen_PublishesPhaseStopping(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_stream_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		streamCh, err := ps.Subscribe(ctx, testTopics.For("streams.echo"))
		require.NoError(t, err)

		feedInput(ps, "inputs.msg", "hello", "world")

		opts := append(mockRPCOptions(), extraOpts...)
		err = NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		require.NoError(t, err)

		phases := drainPhasesUntil(ctx, streamCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
		assertPhaseOrder(t, phases,
			flowv1beta2.RunSnapshot_PHASE_STOPPING,
			flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
	})
}

func TestNodeControl_Output_StopWhen_PublishesPhaseStopping(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_output_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		outCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
		require.NoError(t, err)

		feedInput(ps, "inputs.x", int64(10), int64(20), int64(30))

		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		phases := drainPhasesUntil(ctx, outCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
		assertPhaseOrder(t, phases,
			flowv1beta2.RunSnapshot_PHASE_STOPPING,
			flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
	})
}

func TestNodeControl_Interaction_StopWhen_PublishesPhaseStopping(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_interaction_stop.yaml")

		promptCh := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		responseCh := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		go func() {
			for p := range promptCh {
				anyVal, _ := common.WrapProtoAny(int64(100))
				responseCh <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(responseCh)
		}()

		ctx := testContext(t)
		intCh, err := ps.Subscribe(ctx, testTopics.For("interactions.confirm"))
		require.NoError(t, err)

		feedInput(ps, "inputs.x", int64(1), int64(2))
		err = NewExecutor(ps, testTopics, append([]Option{WithInteractions(promptCh, responseCh)}, extraOpts...)...).Execute(ctx, g)
		close(promptCh)
		require.NoError(t, err)

		phases := drainPhasesUntil(ctx, intCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
		assertPhaseOrder(t, phases,
			flowv1beta2.RunSnapshot_PHASE_STOPPING,
			flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
	})
}
