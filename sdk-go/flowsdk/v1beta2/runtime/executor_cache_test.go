package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
