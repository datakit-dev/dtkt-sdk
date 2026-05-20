package runtime

import (
	"context"
	"math/rand"
	"testing"
	"time"

	expr "cel.dev/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// --- Operator-initiated Suspend ---

// SuspendNode on a generator: the generator enters PHASE_SUSPENDED.

func TestCommand_SuspendNode_Generator(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_rate_limited.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		flowCh, err := ps.Subscribe(ctx, testTopics.Flow())
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)

		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		time.Sleep(50 * time.Millisecond)
		exec.Suspend()

		// Stop must wake suspended goroutines (via stopCh in waitForResume)
		// and drain. The Stop should complete promptly -- without the wake,
		// the suspended handlers would hang indefinitely.
		stopStart := time.Now()
		exec.Stop()
		err = <-done
		stopElapsed := time.Since(stopStart)
		require.NoError(t, err, "Stop should drain suspended handlers cleanly")
		assert.Less(t, stopElapsed, 2*time.Second,
			"Stop on suspended flow must wake handlers promptly; took %v", stopElapsed)

		// Flow's terminal state must be SUCCEEDED (graceful Stop, not
		// ERRORED or CANCELLED).
		var lastFlowPhase flowv1beta2.RunSnapshot_Phase
		drainCtx, drainCancel := context.WithTimeout(ctx, 1*time.Second)
		defer drainCancel()
	drain:
		for {
			select {
			case <-drainCtx.Done():
				break drain
			case msg := <-flowCh:
				evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
				msg.Ack()
				if evt.WhichData() == flowv1beta2.RunSnapshot_FlowEvent_Flow_case {
					lastFlowPhase = evt.GetFlow().GetPhase()
					if lastFlowPhase == flowv1beta2.RunSnapshot_PHASE_SUCCEEDED {
						break drain
					}
				}
			}
		}
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, lastFlowPhase,
			"flow terminal phase after Suspend+Stop must be SUCCEEDED (graceful), got %v", lastFlowPhase)
	})
}

// SuspendNode before Execute is a no-op.

func TestCommand_SuspendNode_NotRunning(t *testing.T) {
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path
	exec := NewExecutor(ps, testTopics)
	exec.SuspendNode("nonexistent") // should not panic
}

// ResumeNode on a non-suspended node is a no-op.

func TestCommand_ResumeNode_NotSuspended(t *testing.T) {
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path
	exec := NewExecutor(ps, testTopics)
	exec.ResumeNode("nonexistent", nil) // should not panic
}

// Suspend is idempotent: calling multiple times is safe.

func TestCommand_Suspend_Idempotent(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_rate_limited.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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

		// Behavioral verification: while suspended, handler must not emit
		// further outputs (proves it actually parked in waitForResume).
		assertNoOutputDuring(t, actionCh, 100*time.Millisecond)

		// Resume -- handler unparks, reads EOF from input, completes cleanly.
		exec.ResumeNode("actions.echo", nil)

		// After resume, the action's terminal phase must be SUCCEEDED
		// (proves the handler actually unparked and reached its post-loop
		// publish, not just that Execute returned).
		requirePhaseWithin(t, actionCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, 500*time.Millisecond)

		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.NoError(t, err, "after resume with EOF available, flow should complete")
	})
}

// SuspendError from retry then terminate: Execute returns ErrTerminated.

func TestCommand_RetrySuspend_ThenTerminate(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_retry_suspend.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		gen1Ch, err := ps.Subscribe(ctx, testTopics.For("generators.gen1"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, append(extraOpts,
			WithErrorStrategy(flowv1beta2.ErrorStrategy_ERROR_STRATEGY_STOP))...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Suspend gen1 and verify it actually parked (PHASE_SUSPENDED).
		time.Sleep(50 * time.Millisecond)
		exec.SuspendNode("generators.gen1")
		require.True(t, waitForPhase(ctx, gen1Ch, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"gen1 must reach PHASE_SUSPENDED before flow Stop")

		// Stop must wake the suspended generator (via stopCh in its
		// select) so it exits with PHASE_SUCCEEDED. Without the wake,
		// the suspended generator would hang and Stop would never return.
		stopStart := time.Now()
		exec.Stop()
		err = <-done
		stopElapsed := time.Since(stopStart)
		require.NoError(t, err)
		assert.Less(t, stopElapsed, 2*time.Second,
			"Stop on flow with suspended generator must wake it promptly; took %v", stopElapsed)
	})
}

// --- Resume all ---

// Suspend all, then Resume all: nodes continue.

func TestCommand_Suspend_Resume_All(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_rate_limited.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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

// Suspend + Resume must preserve output stream continuity. Regression test
// for: when exec.Suspend() cancels non-self-suspendable handler contexts,
// output handlers must NOT publish a Closed:true EOF marker (their stream is
// pausing, not ending). If they do, attached subscribers exit and will not
// see any post-resume values even though the executor restarts the handler.
//
// The existing TestCommand_Suspend_Resume_All only inspects the generator
// topic; it cannot detect this bug because the generator self-suspends and
// is unaffected.

func TestCommand_Suspend_Resume_OutputContinuity(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_rate_limited.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)

		outputCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// readOutput pulls one output node off outputCh. Returns nil on timeout
		// or context cancellation.
		readOutput := func(timeout time.Duration) *flowv1beta2.RunSnapshot_OutputNode {
			select {
			case msg := <-outputCh:
				msg.Ack()
				evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
				node, _ := runtimeNodeFromEvent(evt).(*flowv1beta2.RunSnapshot_OutputNode)
				return node
			case <-time.After(timeout):
				return nil
			}
		}

		// Pre-suspend: at least one real output node arrives.
		first := readOutput(2 * time.Second)
		require.NotNil(t, first, "expected at least one output before suspend")
		require.False(t, first.GetClosed(), "first output should not be a Closed marker")

		// Suspend the entire flow.
		exec.Suspend()

		// CRITICAL: while suspended, the output stream MUST NOT publish a
		// Closed:true marker. A Closed marker means "stream ended forever",
		// which would cause subscribers to terminate their goroutines and
		// miss every post-resume value. Drain any in-flight events for a
		// short window and assert none of them are Closed.
		drainDeadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(drainDeadline) {
			node := readOutput(100 * time.Millisecond)
			if node == nil {
				continue // timeout - no event in flight, that's fine
			}
			require.False(t, node.GetClosed(),
				"output stream published Closed:true during suspend window - subscribers would exit and miss post-resume values")
		}

		// Resume and confirm a NEW output value arrives. If the suspend
		// closed the stream, this read times out and the test fails here.
		exec.Resume()

		postResume := readOutput(3 * time.Second)
		require.NotNil(t, postResume, "no output produced after resume - suspend likely closed the output stream")
		require.False(t, postResume.GetClosed(), "post-resume value should not be a Closed marker")

		exec.Stop()

		err = <-done
		assert.NoError(t, err)
	})
}

// Suspend + Resume across many cycles with variable cadence must produce
// the same value sequence as an uninterrupted run. This is the strict
// version of the contract - not just "ticks keep coming after resume" but
// "the actual sequence of values is identical."
//
// gen_rate_limited.yaml emits a deterministic counter (1, 2, 3, ...). We
// suspend/resume 10 times with random pause durations and random work
// windows in between, then assert the output values are exactly the
// monotonic sequence with no duplicates and no skips.
//
// Runs through both the direct (in-memory) pubsub and the outbox
// (persistent) paths. The outbox path was previously dropping values
// because the output handler short-circuited on ctx.Err() between
// Resolve() consuming a message and publishNode() emitting it; the fix
// is in output.go's Run/runWithTransforms loops.

func TestCommand_Suspend_Resume_ManyCycles_PreservesSequence(t *testing.T) {
	withAndWithoutOutbox(t, runManyCyclesPreservesSequence)
}

func runManyCyclesPreservesSequence(t *testing.T, extraOpts []Option) {
	t.Helper()
	{
		graph := loadFlow(t, "gen_rate_limited.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)

		outputCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// readOutput pulls one real output VALUE off outputCh, skipping any
		// EVENT_TYPE_NODE_UPDATE phase-change events (PENDING/SUSPENDED)
		// that are published on the same topic during transitions.
		readOutput := func(timeout time.Duration) (int64, bool) {
			deadline := time.After(timeout)
			for {
				select {
				case msg := <-outputCh:
					msg.Ack()
					evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
					if evt.GetEventType() != flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
						continue
					}
					node, _ := runtimeNodeFromEvent(evt).(*flowv1beta2.RunSnapshot_OutputNode)
					if node == nil {
						continue
					}
					if node.GetClosed() {
						t.Fatalf("output stream closed mid-test (Closed:true received)")
					}
					return node.GetValue().GetInt64Value(), true
				case <-deadline:
					return 0, false
				}
			}
		}

		var collected []int64

		// Use a deterministic seed so failures are reproducible. We're not
		// trying to exhaustively fuzz, just exercise variable cadence.
		rng := rand.New(rand.NewSource(42))

		const cycles = 10
		for i := 0; i < cycles; i++ {
			// Read a variable number of values (1-5) before suspending.
			workCount := 1 + rng.Intn(5)
			for j := 0; j < workCount; j++ {
				v, ok := readOutput(2 * time.Second)
				if !ok {
					t.Fatalf("cycle %d: timed out reading work value %d", i, j)
				}
				collected = append(collected, v)
			}

			exec.Suspend()

			// Variable pause: 50ms - 500ms.
			pauseMs := 50 + rng.Intn(450)
			time.Sleep(time.Duration(pauseMs) * time.Millisecond)

			exec.Resume()
		}

		// Drain a final batch to confirm the stream survives the last resume.
		for j := 0; j < 3; j++ {
			v, ok := readOutput(2 * time.Second)
			if !ok {
				t.Fatalf("post-cycles: timed out reading tail value %d", j)
			}
			collected = append(collected, v)
		}

		exec.Stop()

		err = <-done
		require.NoError(t, err)

		// The contract: values are exactly 1, 2, 3, ..., len(collected) - same
		// as if no suspend/resume cycle had happened. No duplicates, no skips,
		// strictly monotonic. The exact length depends on how many values
		// snuck through during transitions, but every position must equal
		// (index + 1).
		require.NotEmpty(t, collected, "must have collected at least one value")
		t.Logf("collected %d values across %d suspend/resume cycles: %v",
			len(collected), cycles, collected)

		for i, v := range collected {
			expected := int64(i + 1)
			require.Equalf(t, expected, v,
				"position %d: got value %d, want %d (suspend/resume corrupted the value sequence)",
				i, v, expected)
		}
	}
}

// Per-handler-type suspend/resume coverage. Each entry is a handler
// type + a fixture that drives a deterministic counter (1, 2, 3, ...)
// through that handler. The shared assertion is the same as the output
// handler test: 10 suspend/resume cycles must produce the exact monotonic
// value sequence with no duplicates and no skips.
//
// Why this matters: the consumed-but-not-published race that bit
// output.go is a class of bug each consumer handler is at risk for.
// An audit found the other handlers structurally safe, but tests that
// actually exercise each path are the only durable guarantee.
//
// Coverage spans every handler type that consumes upstream values: var,
// switch, output, unary action, ticker generator, range generator (via
// gen_rate_limited used by ManyCycles), bidi stream. server_stream and
// client_stream have asymmetric stream semantics (one→many, many→one)
// and are covered by separate "stream survives suspend" tests rather
// than sequence preservation. Cron is timing-sparse and gets a lighter
// "publishes EOF on suspended-then-stopped" test.

func TestCommand_Suspend_Resume_PreservesSequence_Var(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		runHandlerSuspendResumeSequence(t, "suspend_resume_var.yaml", extraOpts)
	})
}

func TestCommand_Suspend_Resume_PreservesSequence_Ticker(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		runHandlerSuspendResumeSequence(t, "suspend_resume_ticker.yaml", extraOpts)
	})
}

func TestCommand_Suspend_Resume_PreservesSequence_BidiStream(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		opts := append(mockRPCOptions(), extraOpts...)
		runHandlerSuspendResumeSequence(t, "suspend_resume_bidi_stream.yaml", opts)
	})
}

// Server-stream, client-stream, cron, and interaction handlers don't fit
// the 1:1 sequence-preservation model (one→many, many→one, sparse-timing,
// or external-prompt-driven). For them the suspend/resume contract is
// "cycle through suspend N times without hanging or breaking the flow."

// Server stream: one request, many responses. Suspend mid-flow,
// resume, eventually stop. Verifies the stream connection isn't
// torn down on suspend.

func TestCommand_Suspend_Resume_DoesNotHang_ServerStream(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_server_stream.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Feed input that asks for 5 responses.
		feedInput(ps, "inputs.count", int64(5))

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Cycle 3 times, hitting the stream while it's potentially
		// in flight.
		for i := 0; i < 3; i++ {
			time.Sleep(20 * time.Millisecond)
			exec.Suspend()
			time.Sleep(50 * time.Millisecond)
			exec.Resume()
		}

		exec.Stop()
		require.NoError(t, <-done)
	})
}

// Client stream: many requests, one response. Same shape as above -
// just verify suspend/resume doesn't break it.

func TestCommand_Suspend_Resume_DoesNotHang_ClientStream(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_client_stream.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Feed several values; the client stream collects them.
		feedInput(ps, "inputs.msg", int64(1), int64(2), int64(3))

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		for i := 0; i < 3; i++ {
			time.Sleep(20 * time.Millisecond)
			exec.Suspend()
			time.Sleep(50 * time.Millisecond)
			exec.Resume()
		}

		exec.Stop()
		require.NoError(t, <-done)
	})
}

// Cron: schedule-driven, sparse timing. Just verify suspend/resume
// cycles do not deadlock the flow.

func TestCommand_Suspend_Resume_DoesNotHang_Cron(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_cron.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// gen_cron has stop_when at eval_count >= 2; cycle through
		// suspend a few times within that window.
		time.Sleep(100 * time.Millisecond)
		exec.Suspend()
		time.Sleep(50 * time.Millisecond)
		exec.Resume()
		exec.Suspend()
		time.Sleep(50 * time.Millisecond)
		exec.Resume()

		// Let it run to completion via stop_when.
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-ctx.Done():
			exec.Stop()
			t.Fatal("cron flow did not complete after suspend/resume cycles")
		}
	})
}

// Interaction: prompt-driven. Use an auto-responding mock so the
// interaction completes; verify suspend/resume cycles in between
// don't break the prompt/response cycle.

func TestCommand_Suspend_Resume_DoesNotHang_Interaction(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_basic.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Channel for prompts; auto-responder will deliver values back
		// via the executor's interaction routing.
		promptCh := make(chan *flowv1beta2.InteractionRequestEvent, 16)
		respCh := make(chan *flowv1beta2.InteractionResponseEvent, 16)

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3))

		ctx := testContext(t)
		opts := append([]Option{
			WithInteractions(promptCh, respCh),
		}, extraOpts...)
		exec := NewExecutor(ps, testTopics, opts...)

		// Auto-responder: every prompt gets a response. Must close
		// promptCh explicitly before <-responderDone or the responder
		// will block on `for req := range promptCh` forever.
		responderDone := make(chan struct{})
		go func() {
			defer close(responderDone)
			for req := range promptCh {
				anyVal, _ := common.WrapProtoAny(int64(42))
				select {
				case respCh <- &flowv1beta2.InteractionResponseEvent{
					Id:    req.GetId(),
					Token: req.GetToken(),
					Value: anyVal,
				}:
				case <-ctx.Done():
					return
				}
			}
		}()

		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Cycle suspend/resume.
		for i := 0; i < 3; i++ {
			time.Sleep(20 * time.Millisecond)
			exec.Suspend()
			time.Sleep(50 * time.Millisecond)
			exec.Resume()
		}

		exec.Stop()
		require.NoError(t, <-done)
		close(promptCh) // signal responder to exit
		<-responderDone
	})
}

func TestCommand_Suspend_Resume_PreservesSequence_UnaryAction(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		opts := append(mockRPCOptions(), extraOpts...)
		runHandlerSuspendResumeSequence(t, "suspend_resume_unary.yaml", opts)
	})
}

func TestCommand_Suspend_Resume_PreservesSequence_Switch(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		runHandlerSuspendResumeSequence(t, "suspend_resume_switch.yaml", extraOpts)
	})
}

// runHandlerSuspendResumeSequence is the shared body of the per-handler
// suspend/resume tests. Each fixture must publish exactly the monotonic
// integer sequence 1, 2, 3, ... on outputs.result.
//
// The suspend/resume operation is parameterized: pass exec.Suspend / exec.Resume
// for whole-flow tests, or closures over exec.SuspendNode / exec.ResumeNode for
// single-node tests. Either way, the contract is the same: 10 cycles of
// suspend → variable pause → resume must preserve the exact value sequence.
func runHandlerSuspendResumeSequence(t *testing.T, fixture string, extraOpts []Option) {
	t.Helper()
	runHandlerSuspendResumeSequenceWith(t, fixture, extraOpts, nil, nil)
}

func runHandlerSuspendResumeSequenceWith(
	t *testing.T,
	fixture string,
	extraOpts []Option,
	suspendOverride func(*Executor),
	resumeOverride func(*Executor),
) {
	t.Helper()
	graph := loadFlow(t, fixture)

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	ctx := testContext(t)

	outputCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
	require.NoError(t, err)

	exec := NewExecutor(ps, testTopics, extraOpts...)
	done := make(chan error, 1)
	go func() {
		done <- exec.Execute(ctx, graph)
	}()

	doSuspend := func() {
		if suspendOverride != nil {
			suspendOverride(exec)
		} else {
			exec.Suspend()
		}
	}
	doResume := func() {
		if resumeOverride != nil {
			resumeOverride(exec)
		} else {
			exec.Resume()
		}
	}

	readOutput := func(timeout time.Duration) (int64, bool) {
		deadline := time.After(timeout)
		for {
			select {
			case msg := <-outputCh:
				msg.Ack()
				evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
				if evt.GetEventType() != flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
					continue
				}
				node, _ := runtimeNodeFromEvent(evt).(*flowv1beta2.RunSnapshot_OutputNode)
				if node == nil {
					continue
				}
				if node.GetClosed() {
					t.Fatalf("output stream closed mid-test (Closed:true received)")
				}
				return node.GetValue().GetInt64Value(), true
			case <-deadline:
				return 0, false
			}
		}
	}

	var collected []int64

	rng := rand.New(rand.NewSource(42))
	const cycles = 10
	for i := 0; i < cycles; i++ {
		workCount := 1 + rng.Intn(5)
		for j := 0; j < workCount; j++ {
			v, ok := readOutput(2 * time.Second)
			if !ok {
				t.Fatalf("%s cycle %d: timed out reading work value %d", fixture, i, j)
			}
			collected = append(collected, v)
		}

		doSuspend()
		time.Sleep(time.Duration(50+rng.Intn(450)) * time.Millisecond)
		doResume()
	}

	for j := 0; j < 3; j++ {
		v, ok := readOutput(2 * time.Second)
		if !ok {
			t.Fatalf("%s post-cycles: timed out reading tail value %d", fixture, j)
		}
		collected = append(collected, v)
	}

	exec.Stop()
	require.NoError(t, <-done)

	require.NotEmptyf(t, collected, "%s: must collect at least one value", fixture)
	t.Logf("%s: collected %d values across %d cycles: %v",
		fixture, len(collected), cycles, collected)
	for i, v := range collected {
		require.Equalf(t, int64(i+1), v,
			"%s position %d: got %d, want %d (suspend/resume corrupted sequence)",
			fixture, i, v, i+1)
	}
}

// --- Single-node SuspendNode/ResumeNode multi-cycle tests ---
//
// These mirror the whole-flow PreservesSequence tests but suspend/resume
// only ONE node at a time. Two contracts must hold:
//
//   1. Final sequence at the output is exactly 1, 2, 3, ... (no
//      skips, no dupes) - same as whole-flow suspend.
//   2. While the target node is suspended, the OTHER nodes in the flow
//      MUST continue running. We verify this by subscribing to a
//      witness topic upstream of the suspended node and asserting it
//      received values during the suspend window. Suspending one node
//      must not cascade-pause the rest.

// Suspend the intermediate var. Verify:
//   - Generator (upstream) keeps producing during the suspend window
//   - Output (downstream) pauses because var is no longer feeding it
//   - After resume, var drains its input buffer and sequence is preserved
func TestCommand_SuspendNode_Resume_Var_OtherNodesKeepRunning(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		runSingleNodeSuspendIsolationTest(t, singleNodeSuspendIsolationCase{
			fixture:       "suspend_resume_var.yaml",
			suspendedNode: "vars.passthrough",
			witnessTopic:  "generators.seq", // must keep flowing
			expectWitness: true,
			pauseDuration: 300 * time.Millisecond,
			extraOpts:     extraOpts,
		})
	})
}

// Suspend the terminal output. Verify:
//   - Generator AND var continue running during the suspend window
//   - After resume, output drains var's buffer and sequence is preserved
func TestCommand_SuspendNode_Resume_Output_OtherNodesKeepRunning(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		runSingleNodeSuspendIsolationTest(t, singleNodeSuspendIsolationCase{
			fixture:       "suspend_resume_var.yaml",
			suspendedNode: "outputs.result",
			witnessTopic:  "vars.passthrough", // must keep flowing
			expectWitness: true,
			pauseDuration: 300 * time.Millisecond,
			extraOpts:     extraOpts,
		})
	})
}

// Suspend the generator (the producer). Verify:
//   - Generator's own topic does NOT receive new values during suspend
//     (this is the suspend semantic for the source; everything downstream
//     is a consequence of nothing flowing in).
//   - After resume, generator picks up preserved eval_count and the
//     output sequence continues from where it left off, no gaps.
//
// This is the only single-node case where the witness assertion is
// "the suspended node itself stays quiet" rather than "an unaffected
// node keeps moving" - because suspending the source IS supposed to
// stop the data flow.
func TestCommand_SuspendNode_Resume_Generator_StopsProducing(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		runSingleNodeSuspendIsolationTest(t, singleNodeSuspendIsolationCase{
			fixture:       "suspend_resume_var.yaml",
			suspendedNode: "generators.seq",
			witnessTopic:  "generators.seq", // suspended node's OWN topic
			expectWitness: false,            // must NOT flow during suspend
			pauseDuration: 300 * time.Millisecond,
			extraOpts:     extraOpts,
		})
	})
}

type singleNodeSuspendIsolationCase struct {
	fixture       string
	suspendedNode string
	witnessTopic  string // upstream topic to observe during suspend
	expectWitness bool   // true: witness should receive; false: must NOT receive
	pauseDuration time.Duration
	extraOpts     []Option
}

// runSingleNodeSuspendIsolationTest runs a single suspend/resume cycle on a
// specific node and asserts both:
//
//	(a) the witness topic upstream behaves correctly during the pause window
//	    (keeps flowing if expectWitness=true; stays quiet if false), and
//	(b) the final output sequence is preserved (1..N strictly monotonic).
func runSingleNodeSuspendIsolationTest(t *testing.T, tc singleNodeSuspendIsolationCase) {
	t.Helper()
	graph := loadFlow(t, tc.fixture)

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	ctx := testContext(t)

	outputCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
	require.NoError(t, err)

	witnessCh, err := ps.Subscribe(ctx, testTopics.For(tc.witnessTopic))
	require.NoError(t, err)

	exec := NewExecutor(ps, testTopics, tc.extraOpts...)
	done := make(chan error, 1)
	go func() {
		done <- exec.Execute(ctx, graph)
	}()

	// Drain helper: pulls one OutputNode value (skipping NODE_UPDATE phase
	// events). Returns 0,false on timeout.
	readOutput := func(timeout time.Duration) (int64, bool) {
		deadline := time.After(timeout)
		for {
			select {
			case msg := <-outputCh:
				msg.Ack()
				evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
				if evt.GetEventType() != flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
					continue
				}
				node, _ := runtimeNodeFromEvent(evt).(*flowv1beta2.RunSnapshot_OutputNode)
				if node == nil {
					continue
				}
				if node.GetClosed() {
					t.Fatalf("output stream closed mid-test (Closed:true received)")
				}
				return node.GetValue().GetInt64Value(), true
			case <-deadline:
				return 0, false
			}
		}
	}

	// Wait for at least 2 outputs before suspending so the flow is warm.
	var collected []int64
	for j := 0; j < 2; j++ {
		v, ok := readOutput(2 * time.Second)
		require.Truef(t, ok, "warmup: timed out reading value %d", j)
		collected = append(collected, v)
	}

	// Drain witnessCh of any pre-suspend messages (NODE_OUTPUT count only).
	witnessCount := 0
	countWitness := func(msg *pubsub.Message) {
		evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
		msg.Ack()
		if evt.GetEventType() != flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
			return
		}
		witnessCount++
	}
	// Drain whatever is buffered.
	drainDeadline := time.After(20 * time.Millisecond)
draindrain:
	for {
		select {
		case msg := <-witnessCh:
			countWitness(msg)
		case <-drainDeadline:
			break draindrain
		}
	}
	witnessBefore := witnessCount

	// Suspend the target node. Track witness count in the background
	// during the pause window.
	exec.SuspendNode(tc.suspendedNode)

	// Bookkeeping invariant: SuspendNode must mark ONLY the target node
	// as suspended. Downstream nodes that go quiet because they have no
	// input to consume are "starved" - they're still running their
	// goroutines, just blocked on a channel read. They must NOT be in
	// suspendedNodes.
	exec.mu.Lock()
	suspendedSet := make(map[string]bool, len(exec.suspendedNodes))
	for id := range exec.suspendedNodes {
		suspendedSet[id] = true
	}
	exec.mu.Unlock()
	require.Truef(t, suspendedSet[tc.suspendedNode],
		"%s should be in suspendedNodes after SuspendNode call", tc.suspendedNode)
	require.Lenf(t, suspendedSet, 1,
		"only %s should be suspended; got %v (downstream nodes should be starved, not suspended)",
		tc.suspendedNode, suspendedSet)

	witnessDone := make(chan int)
	go func() {
		count := 0
		t := time.NewTimer(tc.pauseDuration)
		defer t.Stop()
		for {
			select {
			case msg := <-witnessCh:
				evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
				msg.Ack()
				if evt.GetEventType() == flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
					count++
				}
			case <-t.C:
				witnessDone <- count
				return
			}
		}
	}()
	witnessDuringPause := <-witnessDone

	if tc.expectWitness {
		require.Greaterf(t, witnessDuringPause, 0,
			"witness topic %s expected to keep receiving values while %s is suspended, got %d (before=%d)",
			tc.witnessTopic, tc.suspendedNode, witnessDuringPause, witnessBefore)
	} else {
		// Allow up to one in-flight value to drain after suspend - the
		// outbox relay may have committed/queued a value just before
		// the suspend signal was processed. Anything more than 1 means
		// the suspend didn't actually take effect.
		require.LessOrEqualf(t, witnessDuringPause, 1,
			"witness topic %s expected at most 1 in-flight value while %s is suspended, got %d",
			tc.witnessTopic, tc.suspendedNode, witnessDuringPause)
	}

	exec.ResumeNode(tc.suspendedNode, nil)

	// Drain a few more values post-resume and verify the entire collected
	// sequence is exactly 1..N.
	for j := 0; j < 3; j++ {
		v, ok := readOutput(3 * time.Second)
		require.Truef(t, ok, "post-resume: timed out reading tail value %d", j)
		collected = append(collected, v)
	}

	exec.Stop()
	require.NoError(t, <-done)

	t.Logf("%s suspended for %v; collected %d output values: %v; witness on %s during pause: %d",
		tc.suspendedNode, tc.pauseDuration, len(collected), collected, tc.witnessTopic, witnessDuringPause)
	for i, v := range collected {
		require.Equalf(t, int64(i+1), v,
			"position %d: got %d, want %d (single-node suspend corrupted sequence)",
			i, v, i+1)
	}
}

// --- Edge case: SuspendNode on input (no-op) ---

func TestCommand_SuspendNode_Input(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_unary_echo.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Feed a value with no EOF so the flow stays alive.
		topic := testTopics.InputFor("inputs.msg")
		val, _ := nativeToExpr(42)
		ps.Publish(topic, pubsub.NewMessage(val)) //nolint:errcheck // test fixture feed to in-memory pubsub; a real failure surfaces as a downstream assertion

		ctx := testContext(t)

		// Subscribe to output to confirm the action processed AND to a
		// second value AFTER SuspendNode("inputs.msg") to prove the
		// suspend was a no-op (otherwise the flow would have stopped
		// processing further inputs).
		outputCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...)
		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait for the first echo (input value 42) to flow through.
		first := waitForOutputValue(t, outputCh, 5*time.Second)
		assert.Equal(t, int64(42), first.GetInt64Value(), "first echo before suspend")

		// SuspendNode on an Input is documented as a no-op (inputs aren't
		// in the handlers map, they're managed by bridges). Verify by
		// pushing another value AFTER the suspend call: it must still
		// flow through, proving the suspend didn't pause the input.
		exec.SuspendNode("inputs.msg")
		val2, _ := nativeToExpr(99)
		ps.Publish(topic, pubsub.NewMessage(val2)) //nolint:errcheck // test fixture feed to in-memory pubsub; a real failure surfaces as a downstream assertion

		second := waitForOutputValue(t, outputCh, 1*time.Second)
		assert.Equal(t, int64(99), second.GetInt64Value(),
			"second echo after SuspendNode(input) must still flow -- input suspend is a no-op")

		// Stop should still work normally.
		exec.Stop()

		err = <-done
		assert.NoError(t, err)
	})
}

// waitForOutputValue reads a NODE_OUTPUT event off ch within timeout and
// returns the inner value. Skips NODE_UPDATE state events and EOF/Closed
// terminals.
func waitForOutputValue(t *testing.T, ch <-chan *pubsub.Message, timeout time.Duration) *expr.Value {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for output value")
		case msg := <-ch:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			if evt.GetEventType() != flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
				continue
			}
			node := runtimeNodeFromEvent(evt)
			out, ok := node.(*flowv1beta2.RunSnapshot_OutputNode)
			if !ok || out.GetClosed() || isEOFValue(out.GetValue()) {
				continue
			}
			return out.GetValue()
		}
	}
}
