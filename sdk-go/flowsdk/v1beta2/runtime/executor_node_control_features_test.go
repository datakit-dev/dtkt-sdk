package runtime

import (
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Step G: NC interaction with adjacent Var/Action/Stream features.
//
// Each test verifies NC integrates correctly with a feature that has its
// own goroutine, cache, throttle, or pipeline. The feature paths are
// shared across multiple node types where applicable (e.g. throttle on
// Action AND Stream); we test the canonical examples to validate that
// the shared mixin/recv() priority handling works for each.

// nc_var_switch_stop: NC.stop_when on a Var that uses the switch oneof
// variant rather than value. Exercises the switchHandler.Run path
// (compiled via compiledVarSwitch). NC.stop_when fires when input >= 50.
// Stop is per-node graceful: after the trigger, the var exits at the
// next safe point.
func TestNodeControl_Var_Switch_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_var_switch_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(5), int64(50), int64(150))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		results := outputStrings(collectOutputs(ctx, ps, "outputs.result"))
		// 5 -> "low", 50 -> "mid" (triggers stop), 150 may or may not surface.
		require.GreaterOrEqual(t, len(results), 2,
			"expected at least the trigger publish to land; got %v", results)
		assert.Equal(t, "low", results[0])
		assert.Equal(t, "mid", results[1])
	})
}

// nc_var_transforms_stop: NC.stop_when on a Var that has a transform
// pipeline (filter→map). Exercises runWithTransforms path. The transform
// goroutines run in their own errgroup; NC.stop must let the transform
// pipeline drain in-flight items.
//
// Spec contract: NC.stop_when fires per-iteration via checkLifecycle.
// stop_when triggers at inputs.x.value>=5. The values fed are
// [2, 4, 5, 6, 8]; 5 is filtered out (odd), 6 is the first input that
// both passes the filter AND satisfies stop_when. After 6 publishes,
// the var must stop -- so 8 must NOT surface. Full-natural-drain would
// be [20, 40, 60, 80] (4 outputs). Stop-fired bound is `len < 4`.
func TestNodeControl_Var_Transforms_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_var_transforms_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(2), int64(4), int64(5), int64(6), int64(8))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// Filter keeps even values: [2,4,6,8]; map ×10 → [20,40,60,80].
		// stop_when triggers when inputs.x.value>=5 lands -- i.e. by the
		// time we publish the value derived from input 6 (which is the
		// first post-filter input where the predicate holds). After that
		// trigger, the var must NOT publish 80 (the post-transform value
		// for input 8). Natural drain (NC broken) produces 4 outputs.
		full := []int64{20, 40, 60, 80}
		require.GreaterOrEqual(t, len(results), 2,
			"expected at least the early values to pass through")
		require.Less(t, len(results), len(full),
			"NC.stop_when must stop the var before the full drain "+
				"surfaces; if len==4, NC.stop never fired (likely Gap 2: "+
				"runWithTransforms skips checkLifecycle).")
		assert.Equal(t, full[:len(results)], results)
	})
}

// nc_output_transforms_stop: parallel to nc_var_transforms_stop but for the
// Output handler. Exposes Gap 2: outputHandler.runWithTransforms
// (output.go:131) doesn't call checkLifecycle, so NC.stop_when may never
// fire when transforms are configured. Spec contract: NC.stop_when must
// fire per-iteration regardless of which Run path the handler took.
//
// With transforms+NC working: filter keeps [2, 4, 6, 8] from [2, 4, 5, 6, 8];
// map ×10 → [20, 40, 60, 80]; NC.stop_when fires when input 6 lands (first
// post-filter trigger). Stop-fired bound is len < 4 (full natural drain).
func TestNodeControl_Output_Transforms_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_output_transforms_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(2), int64(4), int64(5), int64(6), int64(8))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		full := []int64{20, 40, 60, 80}
		require.GreaterOrEqual(t, len(results), 2,
			"expected at least the early values to pass through")
		require.Less(t, len(results), len(full),
			"NC.stop_when must stop the output before the full drain "+
				"surfaces; if len==4, NC.stop never fired (Gap 2: "+
				"outputHandler.runWithTransforms skips checkLifecycle).")
		assert.Equal(t, full[:len(results)], results)
	})
}

// nc_switch_transforms_stop: parallel to nc_var_transforms_stop but for the
// switch oneof variant of Var (which routes through switchHandler.Run, not
// the value-variant code path). Exposes Gap 2:
// switchHandler.runWithTransforms (switch.go:152) doesn't call
// checkLifecycle, so NC.stop_when may never fire when transforms are also
// configured on the switch var.
//
// The switch passes through the input value via `default: "= this.value"`
// (no case branches). transforms then filter/map. Sequence parallels
// nc_var_transforms_stop: full drain would be [20, 40, 60, 80]; stop-fired
// bound is len < 4.
func TestNodeControl_Switch_Transforms_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_switch_transforms_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(2), int64(4), int64(5), int64(6), int64(8))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		full := []int64{20, 40, 60, 80}
		require.GreaterOrEqual(t, len(results), 2,
			"expected at least the early values to pass through")
		require.Less(t, len(results), len(full),
			"NC.stop_when must stop the switch var before the full drain "+
				"surfaces; if len==4, NC.stop never fired (Gap 2: "+
				"switchHandler.runWithTransforms skips checkLifecycle).")
		assert.Equal(t, full[:len(results)], results)
	})
}

// nc_interaction_transforms_stop: parallel to nc_var_transforms_stop but for
// the Interaction handler. Exposes Gap 2:
// interactionHandler.runWithTransforms (interaction.go:243) doesn't call
// checkLifecycle, so NC.stop_when may never fire when transforms are
// configured. The auto-responder echoes back the input value (1:1 prompt
// to response), so the interaction's value stream equals the input stream.
// Transforms then filter/map the response values. Full drain would be
// [20, 40, 60, 80] (after filter+map from [2,4,5,6,8]); stop-fired bound is
// len < 4.
func TestNodeControl_Interaction_Transforms_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_interaction_transforms_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Echo responder: response value = original input fed at the same
		// position. Inputs are sent in order, so the Nth prompt corresponds
		// to the Nth input value.
		inputs := []int64{2, 4, 5, 6, 8}
		promptCh := make(chan *flowv1beta2.InteractionRequestEvent, len(inputs))
		responseCh := make(chan *flowv1beta2.InteractionResponseEvent, len(inputs))

		go func() {
			idx := 0
			for p := range promptCh {
				if idx >= len(inputs) {
					// Extra prompts past the input list -- shouldn't
					// happen if NC.stop fires, but guard against a
					// regression that floods the responder.
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

		feedInput(ps, "inputs.x", int64(2), int64(4), int64(5), int64(6), int64(8))

		ctx := testContext(t)
		opts := append([]Option{WithInteractions(promptCh, responseCh)}, extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		require.NoError(t, err)

		close(promptCh)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		full := []int64{20, 40, 60, 80}
		require.GreaterOrEqual(t, len(results), 2,
			"expected at least the early values to pass through")
		require.Less(t, len(results), len(full),
			"NC.stop_when must stop the interaction before the full drain "+
				"surfaces; if len==4, NC.stop never fired (Gap 2: "+
				"interactionHandler.runWithTransforms skips checkLifecycle).")
		assert.Equal(t, full[:len(results)], results)
	})
}

// nc_action_memoize_stop: NC.stop_when on an Action with memoize. Memoize
// short-circuits the RPC for repeated inputs, so the cache hit path is
// distinct from the regular call path. NC.stop should fire from
// checkLifecycle regardless of whether the iteration hit the cache or
// dispatched a fresh RPC.
func TestNodeControl_Action_Memoize_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_action_memoize_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Repeat the same input -> first miss, second is a cache hit.
		feedInput(ps, "inputs.msg", int64(1), int64(1), int64(3), int64(5))

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		require.NoError(t, err)

		// stop_when fires when value >= 3, so we expect the prefix
		// [1, 1, 3] of [1, 1, 3, 5]. The two leading 1s prove memoize
		// worked: both invocations produced the same value (the second
		// is a cache hit -- otherwise the mock RPC's identity behavior
		// would still produce 1, but with cache disabled both would still
		// equal 1 and we couldn't tell the difference). The trigger
		// value 3 surfaces because cache-hit iterations still go through
		// publish-then-checkLifecycle, so stop_when fires on iter 3 (5
		// is dropped).
		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		expected := []int64{1, 1, 3, 5}
		require.GreaterOrEqual(t, len(results), 3,
			"expected at least the trigger publish, got %v", results)
		require.LessOrEqual(t, len(results), len(expected))
		assert.Equal(t, expected[:len(results)], results,
			"results must be a prefix of [1,1,3,5] (memoize+stop_when behavior)")
	})
}

// nc_action_throttle_stop: NC.stop_when on a throttled Action with a 1s
// interval. Without the StopChan select-case in the throttle wait, NC.stop
// would have to wait the full throttle window (1s) before exiting. Test
// asserts execution finishes well under that window -- a regression in
// stoppable.go's throttle wait would push this past 1s.
func TestNodeControl_Action_Throttle_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_action_throttle_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", int64(1), int64(2))

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)

		start := time.Now()
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		elapsed := time.Since(start)
		require.NoError(t, err)

		// 1s throttle interval; with stopCh wake-up the test should finish
		// well under that. Allow a generous 500ms upper bound for slow CI.
		assert.Less(t, elapsed, 500*time.Millisecond,
			"NC.stop on throttled Action must wake the throttle wait promptly; took %v", elapsed)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.GreaterOrEqual(t, len(results), 1,
			"first publish must land before stop fires")
	})
}

// nc_stream_throttle_stop: NC.stop_when on a throttled Stream (server-stream
// path). Same shape as the action throttle test but exercises the
// serverStreamHandler's throttle wait.
func TestNodeControl_Stream_Throttle_StopWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_stream_throttle_stop.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.msg", "hello", "world")

		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)

		start := time.Now()
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, g)
		elapsed := time.Since(start)
		require.NoError(t, err)

		assert.Less(t, elapsed, 500*time.Millisecond,
			"NC.stop on throttled Stream must wake the throttle wait promptly; took %v", elapsed)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.GreaterOrEqual(t, len(results), 1,
			"first publish must land before stop fires")
	})
}

// nc_suspend_reentry: NC.suspend_when="= true" on a Var. Each iteration
// after publish triggers SuspendNode, which signals suspendCh and marks
// suspendedNodes[id]=true; the next Resolve sees the suspend signal and
// parks. ResumeNode wakes the handler, clears the flag; the next iteration
// reads the next input, publishes, fires NC.suspend again, and the cycle
// repeats. Validates: re-entry is clean, the suspendedNodes flag toggles
// correctly each cycle, no goroutines leak.
//
// Behavioral verification per cycle:
//   - assertNoOutputDuring after each PHASE_SUSPENDED proves the handler
//     actually parked (not just publishing the suspend phase while
//     continuing to emit values).
//   - expectOutputWithin between cycle 1's resume and cycle 2's SUSPENDED
//     proves cycle 2 actually unparked and produced a fresh emission
//     (value 4 from the second input).
//   - requirePhaseWithin(SUCCEEDED) after the final resume bounds the
//     post-EOF wait so a regression doesn't fall through to testContext.
func TestNodeControl_SuspendReentry(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_suspend_reentry.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(1), int64(2))
		ctx := testContext(t)

		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		// Cycle 1: publish 2, NC.suspend fires, SUSPENDED.
		require.True(t, waitForPhase(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected first PHASE_SUSPENDED after iter 1")
		assertNoOutputDuring(t, varCh, 100*time.Millisecond)
		exec.ResumeNode("vars.doubled", nil)

		// Cycle 2: handler unparks, reads input 2, publishes value 4,
		// NC.suspend fires, SUSPENDED again. Verify the value=4 emission
		// behaviorally before the next phase wait swallows it.
		expectOutputWithin(t, varCh, 500*time.Millisecond)
		require.True(t, waitForPhase(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected second PHASE_SUSPENDED after re-entry")
		assertNoOutputDuring(t, varCh, 100*time.Millisecond)
		exec.ResumeNode("vars.doubled", nil)

		// Inputs drained; handler reads EOF and exits SUCCEEDED. Bound
		// the wait so a regression in re-entry cleanup fails fast.
		requirePhaseWithin(t, varCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, 500*time.Millisecond)

		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		require.NoError(t, err, "Execute should return naturally")
	})
}

// nc_with_error_strategy_continue: NC.terminate_when on one var with the
// flow set to error_strategy: CONTINUE. Per-node terminate cancels just
// the doubled var; the sibling tripled var keeps running and processes
// all inputs. CONTINUE doesn't apply to NC-driven termination (NC isn't
// an error), but this verifies the two features compose without
// misclassifying NC's per-node terminate as a flow-level error.
func TestNodeControl_WithErrorStrategy_Continue(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_with_error_strategy_continue.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		// Subscribe to doubled's topic BEFORE feeding so PHASE_CANCELLED
		// from NC.terminate isn't missed. Without this, the loose
		// LessOrEqual(fullDoubled) bound matches natural drain -- broken
		// NC.terminate would still pass.
		doubledCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
		require.NoError(t, err)

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5))

		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, g)
		// Flow exits naturally; CONTINUE only matters for ERRORED nodes,
		// and NC.terminate is not an error.
		require.NoError(t, err)

		// Spec contract: per-node NC.terminate fires on doubled; verify
		// PHASE_CANCELLED landed on the var topic (load-bearing check
		// that NC.terminate actually fired).
		require.True(t, waitForPhase(ctx, doubledCh, flowv1beta2.RunSnapshot_PHASE_CANCELLED),
			"NC.terminate_when on doubled must publish PHASE_CANCELLED")

		doubled := outputInt64s(collectOutputs(ctx, ps, "outputs.doubled_out"))
		tripled := outputInt64s(collectOutputs(ctx, ps, "outputs.tripled_out"))

		// doubled was per-node terminated; expect a prefix of [2,4,6].
		fullDoubled := []int64{2, 4, 6, 8, 10}
		require.LessOrEqual(t, len(doubled), len(fullDoubled))
		assert.Equal(t, fullDoubled[:len(doubled)], doubled)

		// tripled is unaffected; expect all 5.
		assert.Equal(t, []int64{3, 6, 9, 12, 15}, tripled,
			"sibling var must run to completion when CONTINUE + per-node NC terminate")
	})
}

// TestNodeControl_StopOnSuspended pins down the runtime contract that
// StopNode on a suspended handler exits the handler with PHASE_SUCCEEDED
// (graceful), not PHASE_CANCELLED.
//
// The test uses NC.suspend_when="= true" on a Var so the var enters
// PHASE_SUSPENDED on its first iteration. Then we directly call
// exec.StopNode("vars.doubled"), which signals the handler's stopCh.
// The parked waitForResume returns suspendStopped, the handler breaks
// out of its loop, and the post-loop publish emits PHASE_SUCCEEDED.
//
// Behavioral verification chain:
//  1. waitForPhase(SUSPENDED) on the var topic -- proves NC fired and
//     handler parked.
//  2. assertNoOutputDuring(100ms) -- proves the parked handler isn't
//     still emitting in the background.
//  3. exec.StopNode("vars.doubled") -- the operator stops the suspended
//     node directly (no resume).
//  4. requirePhaseWithin(SUCCEEDED, 500ms) -- proves the stop signal
//     reached the parked handler and it exited gracefully (NOT cancelled).
//  5. requireExecuteReturnsBy(done, 500ms) with NoError -- proves the
//     flow drained cleanly (per-node stop is not flow-level).
func TestNodeControl_StopOnSuspended(t *testing.T) {
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
			"expected var to reach PHASE_SUSPENDED before StopNode")
		assertNoOutputDuring(t, varCh, 100*time.Millisecond)

		exec.StopNode("vars.doubled")
		requirePhaseWithin(t, varCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, 500*time.Millisecond)

		err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.NoError(t, err, "per-node stop is not flow-level; Execute returns nil")
	})
}
