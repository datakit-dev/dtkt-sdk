package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// --- Operator-initiated Suspend ---

// SuspendNode on a generator: the generator enters PHASE_SUSPENDED.

func TestCommand_SuspendNode_Generator(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_rate_limited.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)

		// Subscribe to the generator's topic to observe phases.
		genCh, err := ps.Subscribe(ctx, testTopics.For("generators.seq"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait for at least one value, then suspend the generator.
		msg := <-genCh
		msg.Ack()
		exec.SuspendNode("generators.seq")

		// Should reach PHASE_SUSPENDED.
		require.True(t, waitForPhase(ctx, genCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected generator to reach PHASE_SUSPENDED")

		// Stop the flow to unblock Execute.
		exec.Stop()

		err = <-done
		assert.NoError(t, err, "Stop after suspend should return nil")
	})
}

// SuspendNode then ResumeNode on a generator: resumes producing values.

func TestCommand_SuspendNode_Resume_Generator(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_rate_limited.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)

		// Subscribe to the generator's topic to observe phases.
		genCh, err := ps.Subscribe(ctx, testTopics.For("generators.seq"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait for a value, then suspend.
		msg := <-genCh
		msg.Ack()
		exec.SuspendNode("generators.seq")

		require.True(t, waitForPhase(ctx, genCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected PHASE_SUSPENDED")

		// Resume the generator.
		exec.ResumeNode("generators.seq", nil)

		// Should get PHASE_PENDING followed by RUNNING values.
		require.True(t, waitForPhase(ctx, genCh, flowv1beta2.RunSnapshot_PHASE_PENDING),
			"expected PHASE_PENDING after resume")

		// Wait for a RUNNING value to confirm the generator is active again.
		require.True(t, waitForPhase(ctx, genCh, flowv1beta2.RunSnapshot_PHASE_RUNNING),
			"expected PHASE_RUNNING after resume")

		// Stop the flow.
		exec.Stop()

		err = <-done
		assert.NoError(t, err)
	})
}

// Suspend all nodes, then Terminate: Execute returns ErrTerminated.

func TestCommand_Suspend_ThenTerminate(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_rate_limited.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		exec := NewExecutor(ps, testTopics, extraOpts...)

		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Give the flow a moment to start, then suspend all.
		time.Sleep(50 * time.Millisecond)
		exec.Suspend()

		// Terminate while suspended.
		exec.Terminate()

		err := <-done
		assert.ErrorIs(t, err, ErrTerminated)
	})
}

// Suspend all nodes, then Stop: Execute returns nil (parked goroutines exit).

func TestCommand_Suspend_ThenStop(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_rate_limited.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		exec := NewExecutor(ps, testTopics, extraOpts...)

		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		time.Sleep(50 * time.Millisecond)
		exec.Suspend()

		// Stop should wake parked goroutines and drain.
		exec.Stop()

		err := <-done
		assert.NoError(t, err)
	})
}

// SuspendNode before Execute is a no-op.

func TestCommand_SuspendNode_NotRunning(t *testing.T) {
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck
	exec := NewExecutor(ps, testTopics)
	exec.SuspendNode("nonexistent") // should not panic
}

// ResumeNode on a non-suspended node is a no-op.

func TestCommand_ResumeNode_NotSuspended(t *testing.T) {
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck
	exec := NewExecutor(ps, testTopics)
	exec.ResumeNode("nonexistent", nil) // should not panic
}

// Suspend is idempotent: calling multiple times is safe.

func TestCommand_Suspend_Idempotent(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_rate_limited.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		exec := NewExecutor(ps, testTopics, extraOpts...)

		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		time.Sleep(50 * time.Millisecond)
		exec.Suspend()
		exec.Suspend() // second call is no-op

		exec.Stop()

		err := <-done
		assert.NoError(t, err)
	})
}

// --- Suspend action (hanging RPC) ---

// SuspendNode on a hanging action, then terminate.

func TestCommand_SuspendNode_HangingAction(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_hang.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 42)
		ctx := testContext(t)

		// Subscribe to the action node's topic to observe phases.
		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.call"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait for the action to start (Hang blocks), then suspend.
		time.Sleep(200 * time.Millisecond)
		exec.SuspendNode("actions.call")

		// Should reach PHASE_SUSPENDED.
		require.True(t, waitForPhase(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected action to reach PHASE_SUSPENDED")

		// Terminate the whole flow.
		exec.Terminate()

		err = <-done
		if err != nil {
			assert.ErrorIs(t, err, ErrTerminated)
		}
	})
}

// --- Retry-initiated suspend + resume ---

// SuspendError from retry: node suspends, resume causes re-run, reads EOF, completes.

func TestCommand_RetrySuspend_Resume(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_suspend.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 99)
		ctx := testContext(t)

		// Subscribe to the action node's topic to observe phases.
		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.echo"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait for PHASE_SUSPENDED (retry suspend_when triggers).
		require.True(t, waitForPhase(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected action to reach PHASE_SUSPENDED")

		// Resume -- handler re-runs, reads EOF from input, completes cleanly.
		exec.ResumeNode("actions.echo", nil)

		err = <-done
		assert.NoError(t, err, "after resume with EOF available, flow should complete")
	})
}

// SuspendError from retry then terminate: Execute returns ErrTerminated.

func TestCommand_RetrySuspend_ThenTerminate(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_suspend.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 99)
		ctx := testContext(t)

		// Subscribe to the action node's topic to observe PHASE_SUSPENDED.
		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.echo"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		require.True(t, waitForPhase(ctx, actionCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected PHASE_SUSPENDED")

		exec.Terminate()

		err = <-done
		assert.ErrorIs(t, err, ErrTerminated)
	})
}

// --- Suspend + error strategy interaction ---

// STOP strategy + suspend: suspended node wakes on stop drain.

func TestCommand_Suspend_WithErrorStrategy_Stop(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_two_paths_rate_limited.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)

		exec := NewExecutor(ps, testTopics, append(extraOpts,
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_STOP))...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Suspend one generator, then stop the flow.
		time.Sleep(50 * time.Millisecond)
		exec.SuspendNode("generators.gen1")
		time.Sleep(50 * time.Millisecond)
		exec.Stop()

		err := <-done
		assert.NoError(t, err)
	})
}

// --- Resume all ---

// Suspend all, then Resume all: nodes continue.

func TestCommand_Suspend_Resume_All(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_rate_limited.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)

		genCh, err := ps.Subscribe(ctx, testTopics.For("generators.seq"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait for a value, then suspend all.
		msg := <-genCh
		msg.Ack()
		exec.Suspend()

		require.True(t, waitForPhase(ctx, genCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected PHASE_SUSPENDED")

		// Resume all.
		exec.Resume()

		// Generator should resume producing values.
		require.True(t, waitForPhase(ctx, genCh, flowv1beta2.RunSnapshot_PHASE_RUNNING),
			"expected PHASE_RUNNING after Resume")

		exec.Stop()

		err = <-done
		assert.NoError(t, err)
	})
}

// --- Edge case: SuspendNode on input (no-op) ---

func TestCommand_SuspendNode_Input(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_unary_echo.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Feed a value with no EOF so the flow stays alive.
		topic := testTopics.InputFor("inputs.msg")
		val, _ := nativeToExpr(42)
		ps.Publish(topic, pubsub.NewMessage(val)) //nolint:errcheck

		ctx := testContext(t)

		// Subscribe to output to confirm the action processed.
		outputCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait for output.
		select {
		case msg := <-outputCh:
			msg.Ack()
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for output")
		}

		// SuspendNode on input is a no-op (inputs don't have resume channels).
		exec.SuspendNode("inputs.msg")

		// Stop should still work normally.
		exec.Stop()

		err = <-done
		assert.NoError(t, err)
	})
}
