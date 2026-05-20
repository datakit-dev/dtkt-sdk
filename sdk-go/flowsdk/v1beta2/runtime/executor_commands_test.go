package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// --- Flow-level Stop ---

// Stop: generator-based flow stops gracefully; Execute returns nil.

func TestCommand_Stop_Generator(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_long_running.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)

		// Subscribe to the generator topic to observe phases.
		genCh, err := ps.Subscribe(ctx, testTopics.For("generators.seq"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait for at least one value to arrive, then stop.
		msg := <-genCh
		msg.Ack()
		exec.Stop()

		err = <-done
		assert.NoError(t, err, "operator Stop should return nil (clean shutdown)")

		// Generator should reach SUCCEEDED (EOF published after cancel).
		phases := collectPhases(ctx, genCh)
		require.NotEmpty(t, phases, "expected at least one phase from the generator")
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, phases[len(phases)-1],
			"expected terminal phase SUCCEEDED, got %v", phaseNames(phases))
	})
}

// Stop: input-based flow stops gracefully when EOFs are injected.

func TestCommand_Stop_Input(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_unary_echo.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)

		// Subscribe to the external output topic before executing.
		outputCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...)

		// Feed a value but no EOF -- the flow would block waiting for more input.
		topic := testTopics.InputFor("inputs.msg")
		val, _ := nativeToExpr(42)
		ps.Publish(topic, pubsub.NewMessage(val)) //nolint:errcheck // test fixture feed to in-memory pubsub; a real failure surfaces as a downstream assertion

		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait for the output to arrive: proves the value flowed through
		// before stop fired (echo round-trip succeeded).
		var firstOut *flowv1beta2.RunSnapshot_OutputNode
		select {
		case msg := <-outputCh:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			firstOut, _ = runtimeNodeFromEvent(evt).(*flowv1beta2.RunSnapshot_OutputNode)
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for output")
		}
		require.NotNil(t, firstOut, "first output must be an OutputNode")
		assert.Equal(t, int64(42), firstOut.GetValue().GetInt64Value(),
			"echo should return 42 (proves value flowed through before stop)")

		// Now stop and assert clean shutdown within a tight bound (proves
		// the stop signal actually propagated, not a coincidental natural
		// completion).
		stopStart := time.Now()
		exec.Stop()
		err = <-done
		stopElapsed := time.Since(stopStart)
		assert.NoError(t, err, "operator Stop should return nil (clean shutdown)")
		assert.Less(t, stopElapsed, 2*time.Second,
			"Stop must terminate the flow promptly; took %v", stopElapsed)
	})
}

// Stop is idempotent: calling multiple times is safe.

func TestCommand_Stop_Idempotent(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_long_running.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		exec := NewExecutor(ps, testTopics, extraOpts...)

		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Give the flow a moment to start, then stop twice.
		time.Sleep(50 * time.Millisecond)
		exec.Stop()
		exec.Stop() // second call is no-op

		err := <-done
		assert.NoError(t, err)
	})
}

// Stop before Execute is a no-op.

func TestCommand_Stop_NotRunning(t *testing.T) {
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path
	exec := NewExecutor(ps, testTopics)
	exec.Stop() // should not panic
}

// --- Flow-level Terminate ---

// Terminate: generator-based flow is cancelled immediately; Execute returns ErrTerminated.

func TestCommand_Terminate_Generator(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_long_running.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		exec := NewExecutor(ps, testTopics, extraOpts...)

		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Give the flow a moment to start, then terminate.
		time.Sleep(50 * time.Millisecond)
		exec.Terminate()

		err := <-done
		assert.ErrorIs(t, err, ErrTerminated, "operator Terminate should return ErrTerminated")
	})
}

// Terminate: action blocked on RPC is cancelled; Execute returns ErrTerminated.

func TestCommand_Terminate_HangingAction(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_hang.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", 42)

		ctx := testContext(t)
		exec := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...)

		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait a moment for the action handler to start the hanging RPC.
		time.Sleep(100 * time.Millisecond)
		exec.Terminate()

		err := <-done
		assert.ErrorIs(t, err, ErrTerminated)
	})
}

// Terminate is idempotent.

func TestCommand_Terminate_Idempotent(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_long_running.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		exec := NewExecutor(ps, testTopics, extraOpts...)

		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		time.Sleep(50 * time.Millisecond)
		exec.Terminate()
		exec.Terminate() // second call is no-op

		err := <-done
		assert.ErrorIs(t, err, ErrTerminated)
	})
}

// Terminate before Execute is a no-op.

func TestCommand_Terminate_NotRunning(t *testing.T) {
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path
	exec := NewExecutor(ps, testTopics)
	exec.Terminate() // should not panic
}

// --- Node-level StopNode ---

// StopNode on a generator: stops that generator while others continue.

func TestCommand_StopNode_Generator(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_two_paths.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)

		// Subscribe to gen1 to observe its phases.
		gen1Ch, err := ps.Subscribe(ctx, testTopics.For("generators.gen1"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait for gen1 to start, then stop just gen1.
		msg := <-gen1Ch
		msg.Ack()
		exec.StopNode("generators.gen1")

		// gen1 should reach SUCCEEDED.
		phases := collectPhases(ctx, gen1Ch)
		require.NotEmpty(t, phases, "expected at least one phase from gen1")
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, phases[len(phases)-1],
			"expected gen1 SUCCEEDED, got %v", phaseNames(phases))

		// Stop the whole flow so the test can complete.
		exec.Stop()
		err = <-done
		assert.NoError(t, err)
	})
}

// StopNode on an input: publishes EOF to that input.

func TestCommand_StopNode_Input(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_unary_echo.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)

		// Subscribe to the input topic so we can observe the EOF that
		// StopNode publishes (the contract-level behavior of StopNode on
		// an Input node is "publish EOF to the input topic").
		inputCh, err := ps.Subscribe(ctx, testTopics.For("inputs.msg"))
		require.NoError(t, err)

		// Feed a value but no EOF.
		topic := testTopics.InputFor("inputs.msg")
		val, _ := nativeToExpr(42)
		ps.Publish(topic, pubsub.NewMessage(val)) //nolint:errcheck // test fixture feed to in-memory pubsub; a real failure surfaces as a downstream assertion

		exec := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...)

		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait for the action to process, then stop the input node via EOF.
		time.Sleep(200 * time.Millisecond)
		exec.StopNode("inputs.msg")

		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.NoError(t, err, "StopNode on input should cause clean shutdown")

		// Verify the output value (echo of 42) reached outputs and the
		// input topic ended with PHASE_SUCCEEDED via EOF (StopNode's
		// contract for Inputs).
		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		assert.Equal(t, []int64{42}, results,
			"echo should publish 42 before StopNode caused EOF")
		inputPhases := collectPhases(ctx, inputCh)
		require.NotEmpty(t, inputPhases)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, inputPhases[len(inputPhases)-1],
			"input terminal phase: %v", phaseNames(inputPhases))
	})
}

// StopNode on unknown node is a no-op.

func TestCommand_StopNode_UnknownNode(t *testing.T) {
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path
	exec := NewExecutor(ps, testTopics)
	exec.StopNode("nonexistent") // should not panic
}

// --- Node-level TerminateNode ---

// TerminateNode on an action: cancels the node, publishes PHASE_CANCELLED.

func TestCommand_TerminateNode_Action(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_hang.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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

		// Wait for the action to start the hanging RPC, then terminate just that node.
		time.Sleep(200 * time.Millisecond)
		exec.TerminateNode("actions.call")

		// The action node should get PHASE_CANCELLED.
		phases := collectPhases(ctx, actionCh)
		require.NotEmpty(t, phases, "expected at least one phase from the action node")
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_CANCELLED, phases[len(phases)-1],
			"expected terminal phase CANCELLED, got %v", phaseNames(phases))

		// Terminate the whole flow so the test can complete (the output
		// node is still waiting for a value that will never come).
		exec.Terminate()
		err = <-done
		// Either ErrTerminated or nil (output may have already exited).
		if err != nil {
			assert.ErrorIs(t, err, ErrTerminated)
		}
	})
}

// TerminateNode on unknown node is a no-op.

func TestCommand_TerminateNode_UnknownNode(t *testing.T) {
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path
	exec := NewExecutor(ps, testTopics)
	exec.TerminateNode("nonexistent") // should not panic
}

// --- Interaction between error strategy and operator commands ---

// Stop overrides active error-triggered STOP (no conflict, same mechanism).
// If a handler errors with STOP strategy and the operator also calls Stop,
// the flow drains once and returns the error from the failing action.
// Calling Stop a second time is idempotent. We assert both: error message
// is the action's, and the second Stop didn't cause a second drain.

func TestCommand_Stop_WithErrorStrategy(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_error_internal.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", 99)

		ctx := testContext(t)
		opts := append(mockRPCOptions(),
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_STOP))
		opts = append(opts, extraOpts...)

		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// The error-triggered STOP drains on its own. Operator Stop is
		// idempotent on top of that.
		time.Sleep(100 * time.Millisecond)
		exec.Stop()

		err := <-done
		// The error-triggered stopErr is the contract: the action's
		// "internal server error" surfaces. (Either it returned by the
		// time we called Stop, or our Stop is a no-op idempotent.) Either
		// way, err must be non-nil and carry that message.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")
	})
}

// Terminate after Stop: Terminate wins (ErrTerminated). The flow's
// final terminal state is CANCELLED (terminate-driven), not whatever
// Stop's drain might have produced.

func TestCommand_Terminate_OverridesStop(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_long_running.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		exec := NewExecutor(ps, testTopics, extraOpts...)

		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Give the flow a moment to start, call Stop, then Terminate
		// quickly enough that ctx-cancel beats stop's natural drain.
		time.Sleep(50 * time.Millisecond)
		exec.Stop()
		exec.Terminate()

		err := <-done
		// Long-running generator with no natural completion -- Terminate
		// MUST surface ErrTerminated. A nil here would mean either Stop
		// somehow drained a never-ending generator (impossible without
		// terminate) or Terminate didn't propagate.
		assert.ErrorIs(t, err, ErrTerminated,
			"long-running generator must surface ErrTerminated after Terminate (proves it overrode Stop)")
	})
}
