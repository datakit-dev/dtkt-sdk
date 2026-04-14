package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fan-out via intermediate vars with filter

func TestGraph_FanOut_TwoVars(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fanout_two_vars.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3, 4)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		byID := outputsByID(collectMultipleOutputs(ctx, pubsub, "outputs.even", "outputs.odd"))
		assert.ElementsMatch(t, []int64{2, 4}, byID["outputs.even"])
		assert.ElementsMatch(t, []int64{1, 3}, byID["outputs.odd"])
	})
}

// Fan-out: input directly to multiple outputs

func TestGraph_FanOut_TwoOutputs(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fanout_two_outputs.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		byID := outputsByID(collectMultipleOutputs(ctx, pubsub, "outputs.doubled", "outputs.tripled"))
		assert.Equal(t, []int64{2, 4, 6}, byID["outputs.doubled"])
		assert.Equal(t, []int64{3, 6, 9}, byID["outputs.tripled"])
	})
}

// Fan-out through a chain: input → var → three outputs

func TestGraph_FanOut_VarToThreeOutputs(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fanout_var_to_three_outputs.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// x=1 → base=101, x=2 → base=102, x=5 → base=105
		feedInput(pubsub, "inputs.x", 1, 2, 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		byID := outputsByID(collectMultipleOutputs(ctx, pubsub, "outputs.raw", "outputs.doubled", "outputs.filtered"))
		assert.Equal(t, []int64{101, 102, 105}, byID["outputs.raw"])
		assert.Equal(t, []int64{202, 204, 210}, byID["outputs.doubled"])
		// Only 105 passes >103 filter
		assert.Equal(t, []int64{105}, byID["outputs.filtered"])
	})
}

// Fan-out to parallel var chains

func TestGraph_FanOut_ParallelChains(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fanout_parallel_chains.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// x=1,2,3,4,5
		// Chain A (evens *10): 2→20, 4→40
		// Chain B (+1, >3): 1→2 (drop), 2→3 (drop), 3→4 (keep), 4→5 (keep), 5→6 (keep)
		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		byID := outputsByID(collectMultipleOutputs(ctx, pubsub, "outputs.a", "outputs.b"))
		assert.Equal(t, []int64{20, 40}, byID["outputs.a"])
		assert.Equal(t, []int64{4, 5, 6}, byID["outputs.b"])
	})
}

// Wide fan-out: single input to many outputs

func TestGraph_FanOut_Wide(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "fanout_wide.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		byID := outputsByID(collectMultipleOutputs(ctx, pubsub, "outputs.identity", "outputs.doubled", "outputs.squared", "outputs.negated"))
		assert.Equal(t, []int64{3}, byID["outputs.identity"])
		assert.Equal(t, []int64{6}, byID["outputs.doubled"])
		assert.Equal(t, []int64{9}, byID["outputs.squared"])
		assert.Equal(t, []int64{-3}, byID["outputs.negated"])
	})
}
