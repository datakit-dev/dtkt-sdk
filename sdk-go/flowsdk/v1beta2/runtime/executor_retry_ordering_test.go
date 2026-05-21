package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Lifecycle eval pipeline: each iteration runs retry → NC → FC in sequence.
// Retry's escalation (SuspendError) is the iteration's "starting intent".
// NC and FC may promote that intent (e.g., terminate beats suspend per the
// validity matrix in docs/flowsdk-v1beta2-cleanup-plan.md). The handler
// always reaches checkLifecycle; the combined outcome dictates what the
// handler does next.
//
// These tests pin the cross-stage behavior: retry.suspend + NC.terminate or
// FC.terminate must yield TERMINATE, not SUSPEND. (Inverted from the
// pre-pipeline behavior where retry.suspend short-circuited checkLifecycle.)

func TestRetryStrategy_FiresBeforeNodeControl_Action(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "retry_before_nc_action.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(99))
		ctx := testContext(t)

		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.echo"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		// retry.suspend + NC.terminate → NC.terminate promotes (TERMINATE >
		// SUSPEND). TerminateNode publishes PHASE_CANCELLED on the action's
		// topic. Per-node terminate is scoped, so Execute returns nil.
		require.True(t, waitForPhase(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_CANCELLED),
			"NC.terminate must promote retry.suspend; expected PHASE_CANCELLED on the action topic")

		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.NoError(t, err, "per-node terminate doesn't propagate to the flow")
	})
}

func TestRetryStrategy_FiresBeforeNodeControl_Stream(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "retry_before_nc_stream.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(99))
		ctx := testContext(t)

		streamCh, err := ps.Subscribe(ctx, testTopics.For("streams.echo"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		// retry.suspend + NC.terminate → NC.terminate promotes.
		require.True(t, waitForPhase(ctx, streamCh, flowv1beta2.RunSnapshot_PHASE_CANCELLED),
			"NC.terminate must promote retry.suspend; expected PHASE_CANCELLED on the stream topic")

		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.NoError(t, err, "per-node terminate doesn't propagate to the flow")
	})
}

func TestRetryStrategy_FiresBeforeFlowControl_Action(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "retry_before_fc_action.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(99))
		ctx := testContext(t)

		// Subscribe to the flow topic to observe the terminal state.
		flowCh, err := ps.Subscribe(ctx, testTopics.Flow())
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		// retry.suspend + FC.terminate → FC.terminate promotes (TERMINATE >
		// SUSPEND). e.Terminate() cancels runCtx; Execute returns
		// ErrTerminated and the flow-level terminal state is CANCELLED.
		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, err, ErrTerminated, "FC.terminate must promote retry.suspend and propagate to the flow")

		// Behavioral: verify the flow-level terminal state is CANCELLED
		// (proves FC.terminate's e.Terminate path actually ran, not just
		// that ErrTerminated was returned).
		states := collectFlowStates(ctx, flowCh)
		require.NotEmpty(t, states)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_CANCELLED, states[len(states)-1].GetPhase(),
			"flow terminal state must be CANCELLED when FC.terminate fires")
	})
}

func TestRetryStrategy_FiresBeforeFlowControl_Stream(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "retry_before_fc_stream.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(99))
		ctx := testContext(t)

		flowCh, err := ps.Subscribe(ctx, testTopics.Flow())
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		// retry.suspend + FC.terminate → FC.terminate promotes.
		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, err, ErrTerminated, "FC.terminate must promote retry.suspend and propagate to the flow")

		states := collectFlowStates(ctx, flowCh)
		require.NotEmpty(t, states)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_CANCELLED, states[len(states)-1].GetPhase(),
			"flow terminal state must be CANCELLED when FC.terminate fires")
	})
}

// retry.suspend + NC.stop → STOP. The handler enters retry-suspend
// escalation; checkLifecycle fires NC.stop_when which signals stopCh; the
// handler's non-blocking stopCh check after checkLifecycle catches it and
// breaks the loop. Post-loop publish emits PHASE_SUCCEEDED. Per-node stop,
// flow returns nil.
func TestRetryStrategy_StopPromotesSuspend_NodeControl(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "retry_suspend_nc_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(99))
		ctx := testContext(t)

		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.echo"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		// NC.stop wins over retry.suspend (STOP > SUSPEND in cross-stage).
		// Handler exits gracefully with PHASE_SUCCEEDED, NOT PHASE_SUSPENDED.
		phases := drainPhasesUntil(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
		for _, p := range phases {
			assert.NotEqual(t, flowv1beta2.RunSnapshot_PHASE_SUSPENDED, p,
				"NC.stop must promote retry.suspend; PHASE_SUSPENDED must not appear; phases=%v", phaseNames(phases))
		}

		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.NoError(t, err, "per-node stop doesn't propagate to the flow")
	})
}

// retry.suspend + FC.stop → STOP (per validity matrix). FC.stop's
// performStop signals stopCh on all handlers; recv()'s priority places
// stopCh below input so running handlers drain naturally, while
// about-to-suspend handlers (this test) and already-suspended handlers
// see the signal in their non-blocking check / waitForResume select.
func TestRetryStrategy_StopPromotesSuspend_FlowControl(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "retry_suspend_fc_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(99))
		ctx := testContext(t)

		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.echo"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		phases := drainPhasesUntil(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
		for _, p := range phases {
			assert.NotEqual(t, flowv1beta2.RunSnapshot_PHASE_SUSPENDED, p,
				"FC.stop must promote retry.suspend; PHASE_SUSPENDED must not appear; phases=%v", phaseNames(phases))
		}

		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.NoError(t, err, "FC.stop drains naturally; flow returns nil")
	})
}

// retry.terminate + NC.suspend → TERMINATE. The handler catches
// TerminateError; checkLifecycle is intentionally skipped (terminate is
// sticky, NC.suspend would be invalid on a terminating node per the
// validity matrix). NC.suspend's PHASE_SUSPENDED must not appear on the
// node's topic.
func TestRetryStrategy_TerminateBlocksSuspend(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "retry_terminate_nc_suspend.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(99))
		ctx := testContext(t)

		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.echo"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		err = NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		require.Error(t, err)
		var termErr *TerminateError
		assert.True(t, errors.As(err, &termErr), "expected TerminateError, got %T: %v", err, err)

		// NC.suspend must NOT have fired (would be invalid on a terminating
		// node). Drain whatever phases were published; assert SUSPENDED is
		// not among them.
		drainCtx, drainCancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer drainCancel()
		var phases []flowv1beta2.RunSnapshot_Phase
		for {
			select {
			case <-drainCtx.Done():
				goto done
			case msg := <-actionCh:
				evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
				msg.Ack()
				phases = append(phases, phaseOf(runtimeNodeFromEvent(evt)))
			}
		}
	done:
		for _, p := range phases {
			assert.NotEqual(t, flowv1beta2.RunSnapshot_PHASE_SUSPENDED, p,
				"NC.suspend must not fire on a terminating node; phases=%v", phaseNames(phases))
		}
	})
}

// retry.skip + NC.stop -> STOP. The lifecycle pipeline requires every
// iteration to reach checkLifecycle, even when retry chose to skip the
// item (no value published this iteration). NC.stop_when="= true" must
// fire and the handler must exit gracefully.
//
// Pre-fix bug: the skip path used to `continue` directly, bypassing
// checkLifecycle. NC.stop_when never fired and the handler kept skipping
// forever (until input EOF). The fixture feeds 3 inputs; without the fix
// the handler would process all 3 (skipping each) before exiting via
// EOF; with the fix it exits on iter 1's checkLifecycle.
func TestRetryStrategy_SkipReachesCheckLifecycle(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "retry_skip_nc_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(1), int64(2), int64(3))
		ctx := testContext(t)

		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.echo"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		err = NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		require.NoError(t, err)

		phases := drainPhasesUntil(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
		// PHASE_STOPPING must appear -- proof NC.stop_when fired via
		// checkLifecycle. PHASE_SUCCEEDED is the terminal phase.
		assertPhaseOrder(t, phases,
			flowv1beta2.RunSnapshot_PHASE_STOPPING,
			flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
	})
}

// retry.continue + NC.stop -> CONTINUE publishes value, then STOP. The
// pipeline must reach checkLifecycle after the continue's value emit so
// NC.stop_when can fire normally. Verifies NC sees the iteration as if a
// regular value had been emitted.
func TestRetryStrategy_ContinueReachesCheckLifecycle(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "retry_continue_nc_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(1), int64(2), int64(3))
		ctx := testContext(t)

		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.echo"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		err = NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		require.NoError(t, err)

		phases := drainPhasesUntil(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
		// continue_when emits the value on iter 1 (PHASE_RUNNING with the
		// converted error message), then NC.stop fires (PHASE_STOPPING),
		// then handler exits (PHASE_SUCCEEDED).
		assertPhaseOrder(t, phases,
			flowv1beta2.RunSnapshot_PHASE_STOPPING,
			flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)

		// The value continue_when emits (the converted error.message) is
		// the load-bearing observable for "continue_when fired". Without
		// this assertion, a regression that published the wrong value (or
		// nothing) and still ran NC.stop would pass the phase-order check.
		// The action calls error.NotFound (mock returns "resource not found").
		results := collectOutputs(ctx, ps, "outputs.result")
		require.GreaterOrEqual(t, len(results), 1,
			"continue_when must publish at least one converted value before NC.stop fires")
		assert.Equal(t, "resource not found", results[0].GetValue().GetStringValue(),
			"continue_when's converted value (error.message from error.NotFound) must surface as the action's emitted value")
	})
}

// Negative-case sanity: retry_strategy.suspend_when is configured but the
// RPC succeeds, so retry never fires. NC.stop_when must still take effect
// on the next iteration's checkLifecycle. Confirms that NC fires normally
// when retry is dormant.
func TestRetryStrategy_DoesNotFire_NodeControlStillFires(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "retry_inactive_nc_active.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(42))
		ctx := testContext(t)

		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.call"))
		require.NoError(t, err)

		opts := append(mockRPCOptions(), extraOpts...)
		err = NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		require.NoError(t, err)

		phases := drainPhasesUntil(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
		// NC.stop_when fired (PHASE_STOPPING then PHASE_SUCCEEDED). retry
		// suspend never fired, so PHASE_SUSPENDED must NOT appear.
		for _, p := range phases {
			assert.NotEqual(t, flowv1beta2.RunSnapshot_PHASE_SUSPENDED, p,
				"retry inactive (RPC succeeded) - PHASE_SUSPENDED must not appear; phases=%v", phaseNames(phases))
		}
		assertPhaseOrder(t, phases,
			flowv1beta2.RunSnapshot_PHASE_STOPPING,
			flowv1beta2.RunSnapshot_PHASE_SUCCEEDED)
	})
}
