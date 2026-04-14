package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Running scan (no window) — emits on every input

func TestGraph_Scan_RunningTotal(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "accum_scan_running_total.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

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
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 10, 20, 30, 40, 50)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(150), results[0].GetValue().GetInt64Value())
	})
}
