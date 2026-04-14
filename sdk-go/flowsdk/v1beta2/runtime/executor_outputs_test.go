package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Single transforms on Output nodes.

func TestGraph_Output_MapTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "output_map.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{10, 20, 30}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Output_FilterTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "output_filter.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{4, 5}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Output_ReduceTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// Reduce on output with event window -- only emits final sum.
		graph := loadFlow(t, "output_reduce.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 10, 20, 30)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(60), results[0].GetValue().GetInt64Value())
	})
}

// Output throttle: rate-limits how often the output CEL evaluates

func TestGraph_Output_Throttle(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "output_throttle.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// 5 values at 5 per 500ms = 100ms apart. Should take ~400ms (first fires immediately).
		start := time.Now()
		feedInput(ps, "inputs.x", 1, 2, 3, 4, 5)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		elapsed := time.Since(start)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		assert.Equal(t, []int64{1, 2, 3, 4, 5}, outputInt64s(results))
		// 4 throttle waits of ~100ms each
		assert.GreaterOrEqual(t, elapsed, 300*time.Millisecond)
	})
}
