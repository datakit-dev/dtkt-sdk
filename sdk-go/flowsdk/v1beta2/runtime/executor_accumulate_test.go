package runtime

import (
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Running scan (no window) - emits on every input

func TestGraph_Scan_RunningTotal(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_scan_running_total.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(pubsub, "inputs.x", 10, 20, 30)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{10, 30, 60}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.sum")))
	})
}

func TestGraph_Scan_RunningProduct(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_scan_running_product.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// 2,3,4 → product: 2, 6, 24
		feedInput(pubsub, "inputs.x", 2, 3, 4)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{2, 6, 24}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Non-zero initial value

func TestGraph_Scan_NonZeroInitial(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_scan_nonzero_initial.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// start=100, +1=101, +2=103, +3=106
		feedInput(pubsub, "inputs.x", 1, 2, 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{101, 103, 106}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Event window

func TestGraph_Reduce_EventWindow(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_event_window.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(15), results[0].GetValue().GetInt64Value())
	})
}

func TestGraph_Reduce_EventWindow_NonZeroInitial(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_event_nonzero.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// initial=50, +10+20 = 80
		feedInput(pubsub, "inputs.x", 10, 20)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(80), results[0].GetValue().GetInt64Value())
	})
}

// Fixed window: range 1..3 all land in a single 1s window, sum=6.

func TestGraph_Reduce_FixedWindow(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_fixed_window.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(6), results[0].GetValue().GetInt64Value())
	})
}

// Sliding window: range 1..3 all land in a single 1s/500ms window, sum=6.

func TestGraph_Reduce_SlidingWindow(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_sliding_window.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(6), results[0].GetValue().GetInt64Value())
	})
}

// Session window: range 1..3 arrive instantly (< 200ms timeout), sum=6.

func TestGraph_Reduce_SessionWindow(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_session_window.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(6), results[0].GetValue().GetInt64Value())
	})
}

// GroupBy key

func TestGraph_Reduce_GroupByKey(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_group_by_key.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// 1..6: even=2+4+6=12, odd=1+3+5=9 → two results
		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5, 6)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		vals := make([]int64, len(results))
		for i, r := range results {
			vals[i] = r.GetValue().GetInt64Value()
		}
		assert.ElementsMatch(t, []int64{9, 12}, vals)
	})
}

// Reduce edge cases

func TestGraph_Reduce_SingleValue(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_single_value.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(pubsub, "inputs.x", 42)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{42}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Reduce_EmptyInput(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_empty_input.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(pubsub, "inputs.x")
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Empty(t, collectOutputs(ctx, pubsub, "outputs.result"))
	})
}

func TestGraph_Reduce_FilterThenReduce(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_filter_then_reduce.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(pubsub, "inputs.x", 1, 2, 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Empty(t, collectOutputs(ctx, pubsub, "outputs.result"))
	})
}

// Scan with string concatenation

func TestGraph_Scan_StringConcat(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_scan_string_concat.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(pubsub, "inputs.word", "hello", "world", "!")
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 3)
		assert.Equal(t, "hello ", results[0].GetValue().GetStringValue())
		assert.Equal(t, "hello world ", results[1].GetValue().GetStringValue())
		assert.Equal(t, "hello world ! ", results[2].GetValue().GetStringValue())
	})
}

// Phase 5.5: GroupBy with fixed window

func TestGraph_Reduce_GroupBy_FixedWindow(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_group_by_fixed.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// 1..6: even=(2+4+6)=12, odd=(1+3+5)=9
		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5, 6)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		vals := make([]int64, len(results))
		for i, r := range results {
			vals[i] = r.GetValue().GetInt64Value()
		}
		assert.ElementsMatch(t, []int64{9, 12}, vals)
	})
}

// Phase 5.5: Bare reduce (no window) with multiple values

func TestGraph_Reduce_MultiValueNoWindow(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_multivalue_no_window.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(pubsub, "inputs.x", 10, 20, 30, 40, 50)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(150), results[0].GetValue().GetInt64Value())
	})
}

// --- Gap 1 window-behavior tests (multi-window emissions) ---
//
// The four tests below exercise Transform.Reduce.GroupBy.Window with input
// timing that SPANS multiple windows. They are the discriminating cases
// the docs/flowsdk-v1beta2-test-and-runtime-audit.md Gap 1 finding
// requires: `compileTransforms` (transform.go:466-482) reads only
// gb.GetKey() and never consults gb.GetWindow(), so the runtime emits a
// single sum at EOF regardless of how many windows the input timing
// spans.
//
// All four tests EXPECT the runtime to emit one result per window. Until
// Gap 1 is fixed, all four fail with `Len == 1` (the EOF-only behavior).
// The error message names which window variant's emission contract is
// not honored, so the failure points directly at which code path needs
// implementing.

// Fixed (tumbling) window: range 1..6 spaced 50ms apart, 150ms windows.
// Expected: [6, 15] = [1+2+3, 4+5+6] (two windows of three values each).
//
// Vacuous-test sister TestGraph_Reduce_FixedWindow uses a 1s window with
// all 3 values arriving in ~microseconds, so a single emission is correct
// regardless of whether windowing is implemented or not.

func TestGraph_Reduce_FixedWindow_MultiEmission(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_fixed_window_multi.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		require.Equal(t, []int64{6, 15}, results,
			"fixed window must emit one sum per window close. Got len=%d, "+
				"values=%v. A single value of 21 means the runtime collapsed "+
				"all events into one EOF emission, ignoring the 150ms window. "+
				"Gap 1: compileTransforms (transform.go:466-482) never reads "+
				"gb.GetWindow().", len(results), results)
	})
}

// Sliding window: range 1..6 spaced 50ms apart; length=200ms, slide=100ms.
// Each slide tick (100ms) closes a window and emits the values active in
// that window. Exact emission count depends on alignment of window
// boundaries with event arrival; non-vacuous lower bound is 2.

func TestGraph_Reduce_SlidingWindow_MultiEmission(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_sliding_window_multi.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		require.GreaterOrEqual(t, len(results), 2,
			"sliding window must emit one result per slide; got %d emissions "+
				"with values %v. A single emission means the runtime collapsed "+
				"all events into one EOF emission, ignoring the 200ms/100ms "+
				"sliding config. Gap 1: compileTransforms doesn't read the "+
				"Sliding variant.", len(results), results)
	})
}

// Session window: two timed bursts separated by a 250ms gap; 100ms
// timeout. First session closes after burst 1's last value sits idle for
// 100ms; second session emitted after burst 2. Expected: [6, 60].

func TestGraph_Reduce_SessionWindow_MultiEmission(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_session_window_multi.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		// Burst 1: 1, 2, 3 immediately. Gap. Burst 2: 10, 20, 30. The
		// 250ms inter-burst gap exceeds the 100ms session timeout so a
		// correctly-implemented session closes between the bursts.
		//
		// CRITICAL: must use sendInput (no auto-EOF) rather than
		// feedInput. feedInput publishes an EOF marker after the values,
		// which would close the input subscription before burst 2 lands
		// (Execute terminates at t=~15ms instead of waiting the 250ms
		// gap). An explicit EOF is published after burst 2 to terminate
		// Execute cleanly.
		for _, v := range []int64{1, 2, 3} {
			sendInput(ps, "inputs.x", v)
		}
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(250 * time.Millisecond):
			}
			for _, v := range []int64{10, 20, 30} {
				sendInput(ps, "inputs.x", v)
			}
			_ = ps.Publish(testTopics.InputFor("inputs.x"), pubsub.NewMessage(newEOFValue()))
		}()

		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		require.Equal(t, []int64{6, 60}, results,
			"session window must emit one sum per session close. Got len=%d, "+
				"values=%v. A single value of 66 means the 250ms idle gap "+
				"did not close the first session, so the runtime is ignoring "+
				"the session.timeout config. Gap 1: compileTransforms doesn't "+
				"read the Session variant.", len(results), results)
	})
}

// --- Gap 1 key + window combined tests ---
//
// The window-only tests above (TestGraph_Reduce_FixedWindow_MultiEmission
// etc.) prove the window LAYER is unimplemented when group_by has no key.
// The combined case -- key AND window together -- is a separate code
// path: per-key bucketing IS implemented, but per-key emission at window
// close is gated by the same Gap 1 window-time logic.
//
// Existing tests (TestGraph_Reduce_GroupBy_FixedWindow and its YAML
// accum_reduce_group_by_fixed.yaml) have all values arrive in
// microseconds so they fit in any window. Those tests pass with exactly
// the per-key emission count regardless of whether windowing fires. The
// four tests below add timing/gating that SPANS multiple windows: per
// (key, window) pair the spec requires one emission. The current
// implementation emits per-key only at EOF (one per key), so all four
// fail with `len == 2` rather than the expected 4.

// GroupBy key + fixed window: range 1..6 at 50ms spacing, 150ms window.
// Window 1 [0, 150ms) holds 1, 2, 3 -> odd=1+3=4, even=2.
// Window 2 [150, 300ms) holds 4, 5, 6 -> odd=5, even=4+6=10.
// Expected: 4 emissions summing to [4, 2, 5, 10] in some order.

func TestGraph_Reduce_GroupBy_FixedWindow_MultiEmission(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_group_by_fixed_multi.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		require.Len(t, results, 4,
			"key+fixed-window must emit one result per (key, window) pair. "+
				"Got len=%d, values=%v. A len of 2 means per-key emission is "+
				"firing at EOF rather than at each window close: per-key works "+
				"(Gap 1's window layer is what's broken on the combined path).",
			len(results), results)
		assert.ElementsMatch(t, []int64{4, 2, 5, 10}, results,
			"per-(key, window) sums must be [4 (odd-w1), 2 (even-w1), 5 (odd-w2), 10 (even-w2)]")
	})
}

// GroupBy key + sliding window: range 1..6 at 50ms spacing, length=200ms,
// slide=100ms. Per-key emission per window close as each slide fires.
// Non-vacuous bound: > 2 emissions (since 2 is the per-key-at-EOF Gap-1
// outcome).

func TestGraph_Reduce_GroupBy_SlidingWindow_MultiEmission(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_group_by_sliding_multi.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		require.Greater(t, len(results), 2,
			"key+sliding-window must emit per (key, window) as each slide "+
				"closes. Got len=%d, values=%v. A len of 2 means per-key "+
				"emission fires at EOF only: per-key works, the sliding "+
				"window layer doesn't.",
			len(results), results)
	})
}

// GroupBy key + session window: two timed bursts separated by 250ms
// gap, 100ms session timeout. Per-key session closes after each key's
// idle period. Burst 1 -> odd=4, even=2. Burst 2 -> odd=5, even=10.
// Expected: 4 emissions.

func TestGraph_Reduce_GroupBy_SessionWindow_MultiEmission(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_group_by_session_multi.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		// Burst 1 then 250ms gap then burst 2. Must use sendInput (no
		// auto-EOF) -- see sister TestGraph_Reduce_SessionWindow_MultiEmission
		// for why feedInput would terminate Execute prematurely.
		for _, v := range []int64{1, 2, 3} {
			sendInput(ps, "inputs.x", v)
		}
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(250 * time.Millisecond):
			}
			for _, v := range []int64{4, 5, 6} {
				sendInput(ps, "inputs.x", v)
			}
			_ = ps.Publish(testTopics.InputFor("inputs.x"), pubsub.NewMessage(newEOFValue()))
		}()

		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		require.Len(t, results, 4,
			"key+session-window must emit per (key, session-close) pair. "+
				"Got len=%d, values=%v. A len of 2 means per-key emission "+
				"fires at EOF only: per-key works, the session window layer "+
				"doesn't see the 250ms idle gap.",
			len(results), results)
		assert.ElementsMatch(t, []int64{4, 2, 5, 10}, results,
			"per-(key, session) sums must be [4 (odd-s1), 2 (even-s1), 5 (odd-s2), 10 (even-s2)]")
	})
}

// GroupBy key + event window: range 1..6 with `when: this.value % 3 == 0`.
// Value 3 closes the first windows for both keys (odd=1+3=4, even=2);
// value 6 closes the second windows (odd=5, even=4+6=10). Expected: 4
// emissions.

func TestGraph_Reduce_GroupBy_EventWindow_MultiEmission(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_group_by_event_multi.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		require.Len(t, results, 4,
			"key+event-window must emit per (key, event-gate) pair. "+
				"Got len=%d, values=%v. A len of 2 means per-key emission "+
				"fires at EOF only: per-key works, the event window's `when` "+
				"CEL isn't gating emission.",
			len(results), results)
		assert.ElementsMatch(t, []int64{4, 2, 5, 10}, results,
			"per-(key, event-window) sums must be [4 (odd-w1), 2 (even-w1), 5 (odd-w2), 10 (even-w2)]")
	})
}

// Event window: range 1..6; `when: this.value % 3 == 0` closes the
// window when the CEL evaluates true on the incoming value. Expected:
// [6, 15] = [1+2+3, 4+5+6].

func TestGraph_Reduce_EventWindow_MultiEmission(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_reduce_event_window_multi.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		require.Equal(t, []int64{6, 15}, results,
			"event window must emit when the `when` CEL fires (every third "+
				"value here). Got len=%d, values=%v. A single value of 21 "+
				"means the runtime is ignoring the event.when CEL and only "+
				"emitting at EOF. Gap 1: compileTransforms doesn't read the "+
				"Event variant; graph.go:303 reads `when` for edge inference "+
				"only, not for emission gating.", len(results), results)
	})
}
