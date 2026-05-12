package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// TestCache_Var_DeliversInline verifies cache:true on a Var captures
// the first emit; downstream consumers get exactly that value. The
// downstream output here is all-cached (only cached dep), so it
// iterates once per producer message and fires exactly once.
func TestCache_Var_DeliversInline(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cache_var_to_output.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.x", int64(7))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1, "all-cached-deps consumer fires exactly once")
		assert.Equal(t, int64(14), results[0].GetValue().GetInt64Value())
	})
}

// TestCache_Var_DrainAndSkip verifies that after the first capture, a
// cache:true var drains additional upstream events without re-evaluating
// or re-emitting. The all-cached output consumer iterates exactly once
// per producer pulse, so it should publish exactly one value (the first).
func TestCache_Var_DrainAndSkip(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cache_var_to_output.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Feed three values: var should capture only the first (5*2=10),
		// drain the next two without re-emitting.
		feedInput(ps, "inputs.x", int64(5), int64(99), int64(100))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		require.Len(t, results, 1,
			"cache:true producer captures first value; subsequent events drained and skipped")
		assert.Equal(t, int64(10), results[0],
			"captured value should be 5*2 (first input), not later values")
	})
}

// TestCache_CaptureAndClear is a unit-style test of cacheBackend's
// captured-flag semantics: markCaptured sets the flag so isCaptured
// returns true; clearCapture resets it.
func TestCache_CaptureAndClear(t *testing.T) {
	t.Parallel()

	cb := &cacheBackend{cacheCapture: cacheCapture{enabled: true}}
	require.False(t, cb.isCaptured(), "fresh backend is not captured")

	cb.markCaptured()
	require.True(t, cb.isCaptured(), "captured after markCaptured")

	// markCaptured is idempotent.
	cb.markCaptured()
	require.True(t, cb.isCaptured())

	cb.clearCapture()
	require.False(t, cb.isCaptured(), "clearCapture resets the flag")

	cb.markCaptured()
	require.True(t, cb.isCaptured(), "re-captured after clear+mark")
}

// TestCache_NonEnabled is a unit test of the non-cached path: a
// disabled backend reports not-captured and no-ops on markCaptured.
func TestCache_NonEnabled(t *testing.T) {
	t.Parallel()

	cb := &cacheBackend{cacheCapture: cacheCapture{enabled: false}}
	require.False(t, cb.isCaptured())
	cb.markCaptured()
	require.False(t, cb.isCaptured(), "non-cached backend stays not-captured")
}

// TestCache_FanOut_MultipleConsumers verifies that multiple downstream
// nodes can each read the same cached value across multiple iterations.
// One Input.cache:true source feeds two distinct vars, each driving its
// own output. All iterations on both consumers read the same captured
// const value.
func TestCache_FanOut_MultipleConsumers(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cache_var_fanout.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.const", int64(10))
		feedInput(ps, "inputs.tick", int64(1), int64(2), int64(3))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		sums := outputInt64s(collectOutputs(ctx, ps, "outputs.sum"))
		products := outputInt64s(collectOutputs(ctx, ps, "outputs.product"))
		assert.Equal(t, []int64{11, 12, 13}, sums,
			"sum consumer reads cached const=10 on every iteration")
		assert.Equal(t, []int64{10, 20, 30}, products,
			"product consumer reads the SAME cached const=10 on every iteration")
	})
}

// TestCache_Var_FilterTransform_CapturesFirstPassing verifies that a
// var with cache:true AND a filter transform captures the first value
// that PASSES the filter (the first thing consumers actually see), not
// the first value fed into the pipeline. This guards against the bug
// where markCaptured fired pre-transform, causing filter-rejected
// values to "capture" and starve downstream consumers.
func TestCache_Var_FilterTransform_CapturesFirstPassing(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cache_var_with_filter.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Filter is `this.value > 10`. First two values (3, 7) are
		// dropped; 42 is the first to pass. With the post-transform
		// markCaptured fix, consumers see 42. With the pre-transform
		// bug, consumers would have seen nothing (3 captures, drains
		// 7 and 42 without re-evaluating).
		feedInput(ps, "inputs.x", int64(3), int64(7), int64(42), int64(99))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		require.Len(t, results, 1,
			"all-cached consumer fires exactly once with the first post-filter value")
		assert.Equal(t, int64(42), results[0],
			"captured value must be the first filter-passing input, not the first input fed into the pipeline")
	})
}

// TestCache_Input_MapTransform_CapturesPostMap verifies that an
// Input.cache:true with a map transform captures the post-transform
// value (what consumers see), not the raw input. Guards against the
// same pre-transform markCaptured bug in the input bridge.
func TestCache_Input_MapTransform_CapturesPostMap(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cache_input_with_map.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Map is `this.value * 100`. First input is 7 -> 700 post-map.
		// Subsequent pushes are drained.
		feedInput(ps, "inputs.x", int64(7), int64(8), int64(9))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		require.Len(t, results, 1,
			"all-cached consumer fires exactly once with the first post-map value")
		assert.Equal(t, int64(700), results[0],
			"captured value must be the post-map output, not the raw input")
	})
}

// TestCache_MixedDeps_StreamingDrivesIteration verifies a node with one
// streaming dep + one cached dep iterates per streaming event and reads
// the cached value inline on each iteration.
func TestCache_MixedDeps_StreamingDrivesIteration(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cache_var_mixed_deps.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// const is cached:true; the consumer's lastSeen captures the first
		// push. tick is streaming; each push drives one iteration.
		feedInput(ps, "inputs.const", int64(100))
		feedInput(ps, "inputs.tick", int64(1), int64(2), int64(3))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		assert.Equal(t, []int64{101, 102, 103}, results,
			"streaming tick drives iterations; cached const is read inline each time")
	})
}

// TestCache_Action_DrainAndSkip verifies that an Action with cache:true
// invokes the RPC on the first upstream event, publishes the response,
// and drains subsequent upstream events without calling the RPC again
// or re-publishing. The downstream output is all-cached (only cached
// dep), so it fires exactly once with the captured value.
func TestCache_Action_DrainAndSkip(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cache_action_basic.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Three values: action.echo should call the RPC on the first
		// (echo back 11), drain the next two without re-emitting.
		feedInput(ps, "inputs.x", int64(11), int64(22), int64(33))
		ctx := testContext(t)
		opts := append(mockRPCOptions(), extraOpts...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		require.Len(t, results, 1,
			"cache:true action captures first response; subsequent inputs drained and skipped")
		assert.Equal(t, int64(11), results[0],
			"captured value should be the first input's echo response, not later values")
	})
}

// TestCache_ClearCache_ResetsCaptureFlag verifies that Executor.ClearCache
// resets the producer's captured flag mid-run so the next upstream event
// is processed and re-emitted. Without the clear, drain-and-skip would
// swallow that event silently.
//
// Test shape:
//  1. Subscribe to the var topic.
//  2. Start Execute in a goroutine.
//  3. Send first input value (no EOF). Wait for the first var emit; this
//     emit is observable proof that the handler has captured and looped
//     back to await the next input. After that, the handler is parked on
//     the input channel.
//  4. Call ClearCache("vars.doubled") while the handler is parked. Safe
//     to call: clearCapture is atomic, the handler hasn't moved.
//  5. Send a second value. Handler unparks, sees the flag is clear, and
//     re-evaluates. Wait for the second emit.
//  6. Send EOF; require Execute returns.
//
// We deliberately avoid sending a "to-be-drained" middle value: there's
// no signal for "drain completed", so racing ClearCache against a
// pending message produces flakes (the handler may consume the message
// after the clear, treating it as a fresh capture).
func TestCache_ClearCache_ResetsCaptureFlag(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cache_var_to_output.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, graph) }()

		// First value triggers capture; var emits 10. Observing this emit
		// also tells us the handler has finished iteration 1 and is now
		// blocked in Resolve() awaiting the next input.
		sendInput(ps, "inputs.x", int64(5))
		got := waitForVarValueInt64(t, varCh, 1*time.Second)
		assert.Equal(t, int64(10), got, "first capture: 5*2")

		// Clear while the handler is parked. The next Resolve() will see
		// captured=false and re-evaluate.
		exec.ClearCache("vars.doubled")

		sendInput(ps, "inputs.x", int64(7))
		got = waitForVarValueInt64(t, varCh, 1*time.Second)
		assert.Equal(t, int64(14), got,
			"after ClearCache the next upstream event must re-evaluate and emit")

		// EOF so the producer exits and Execute returns.
		topic := testTopics.InputFor("inputs.x")
		err = ps.Publish(topic, pubsub.NewMessage(newEOFValue()))
		require.NoError(t, err)

		err = requireExecuteReturnsBy(t, done, 1*time.Second)
		require.NoError(t, err)
	})
}

// TestCache_NoClearCache_DrainsSubsequentEvents is the negative companion
// to TestCache_ClearCache_ResetsCaptureFlag: without ClearCache, the
// drain-and-skip semantic kicks in and a second input is silently
// dropped (no second emit on the var topic). Together with the positive
// test, this pins both halves of the ClearCache contract.
func TestCache_NoClearCache_DrainsSubsequentEvents(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cache_var_to_output.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Two values upfront; the second must be drained-and-skipped.
		feedInput(ps, "inputs.x", int64(5), int64(7))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		require.Len(t, results, 1,
			"without ClearCache, only the first value reaches consumers")
		assert.Equal(t, int64(10), results[0],
			"captured value should be 5*2; 7*2 must be drained")
	})
}

// TestCache_ClearCache_UnknownNode_NoOp verifies Executor.ClearCache
// silently no-ops for unknown node IDs (e.g. ones that aren't cache:true
// producers) instead of panicking. This is the documented behaviour --
// callers shouldn't have to gate on "is this id cached?".
func TestCache_ClearCache_UnknownNode_NoOp(t *testing.T) {
	t.Parallel()

	graph := loadFlow(t, "cache_var_to_output.yaml")

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck

	exec := NewExecutor(ps, testTopics)
	// Pre-Execute: cacheBackends is nil; ClearCache must not panic.
	exec.ClearCache("vars.doubled")
	exec.ClearCache("vars.does-not-exist")

	ctx := testContext(t)
	feedInput(ps, "inputs.x", int64(3))
	require.NoError(t, exec.Execute(ctx, graph))

	// Post-Execute: clearRunState() ran; cacheBackends is nil again.
	// Still must not panic.
	exec.ClearCache("vars.doubled")
	exec.ClearCache("inputs.x")
}

// waitForVarValueInt64 reads a NODE_OUTPUT event off ch within timeout
// and returns the inner int64 value. Skips NODE_UPDATE state events and
// EOF terminals.
func waitForVarValueInt64(t *testing.T, ch <-chan *pubsub.Message, timeout time.Duration) int64 {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for var value")
			return 0
		case msg := <-ch:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			if evt.GetEventType() != flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
				continue
			}
			node, ok := runtimeNodeFromEvent(evt).(*flowv1beta2.RunSnapshot_VarNode)
			if !ok || isEOFValue(node.GetValue()) {
				continue
			}
			return node.GetValue().GetInt64Value()
		}
	}
}
