package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Full even/odd flow: fan-out + filter + reduce + event window

func TestGraph_Join_EvenOddSum(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "join_even_odd_sum.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// Inputs with duplicates to exercise filter deduplication behavior.
		feedInput(pubsub, "inputs.number", 1, 2, 3, 3, 3, 4, 5, 6, 7, 8, 8, 8, 9, 10)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		// Collect all outputs individually for exact assertions.
		evenOddPairs := collectOutputs(ctx, pubsub, "outputs.evenOddPairs")
		byID := outputsByID(collectMultipleOutputs(ctx, pubsub, "outputs.evenNumbers", "outputs.oddNumbers", "outputs.evenSum", "outputs.oddSum"))

		// evenNumbers: only even inputs pass the filter (duplicates preserved).
		assert.Equal(t, []int64{2, 4, 6, 8, 8, 8, 10}, byID["outputs.evenNumbers"])

		// oddNumbers: only odd inputs pass the filter (duplicates preserved).
		assert.Equal(t, []int64{1, 3, 3, 3, 5, 7, 9}, byID["outputs.oddNumbers"])

		// evenSum: scan accumulates even values -- 2, 6, 12, 20, 28, 36, 46.
		assert.Equal(t, []int64{2, 6, 12, 20, 28, 36, 46}, byID["outputs.evenSum"])

		// oddSum: reduce with event window (fires once on close).
		require.Len(t, byID["outputs.oddSum"], 1)
		assert.Equal(t, int64(31), byID["outputs.oddSum"][0])

		// evenOddPairs: list-typed output [even, odd]. Each emission is a pair
		// of the latest even and odd values.
		expectedPairs := [][2]int64{
			{2, 1}, {4, 3}, {6, 3}, {8, 3}, {8, 5}, {8, 7}, {10, 9},
		}
		require.Len(t, evenOddPairs, len(expectedPairs), "expected exactly %d evenOddPairs emissions", len(expectedPairs))
		for i, pair := range evenOddPairs {
			listVals := pair.GetValue().GetListValue().GetValues()
			require.Len(t, listVals, 2, "evenOddPairs[%d] should be a 2-element list", i)
			got := [2]int64{listVals[0].GetInt64Value(), listVals[1].GetInt64Value()}
			assert.Equal(t, expectedPairs[i], got, "evenOddPairs[%d]", i)
		}
	})
}

// Diamond topology: A → B, A → C, B → D, C → D

func TestGraph_Join_Diamond(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// Input → var_left(*2) + var_right(*3) → output(left + right)
		graph := loadFlow(t, "join_diamond.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// x=10 → left=20, right=30 → combined=50
		feedInput(pubsub, "inputs.x", 10)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.combined")
		require.Len(t, results, 1)
		assert.Equal(t, int64(50), results[0].GetValue().GetInt64Value())
	})
}

func TestGraph_Join_DiamondMultipleValues(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// Diamond with multiple input values.
		graph := loadFlow(t, "join_diamond_multiple.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// x=5 → doubled=10, negated=-5 → sum=5
		// x=10 → doubled=20, negated=-10 → sum=10
		feedInput(pubsub, "inputs.x", 5, 10)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{5, 10}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.sum")))
	})
}

// Two independent inputs joining at output

func TestGraph_Join_TwoInputs(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "join_two_inputs.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.a", int64(7))
		feedInput(pubsub, "inputs.b", int64(6))

		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.product")
		require.Len(t, results, 1)
		assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
	})
}

// Fan-in: multiple vars feed into one output with reduce

func TestGraph_Join_FanInWithReduce(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// Two vars fan into a single output that sums everything.
		graph := loadFlow(t, "join_fan_in.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 3,7,1,10 → small: 3,1 → large: 70,100
		feedInput(pubsub, "inputs.x", 3, 7, 1, 10)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		byID := outputsByID(collectMultipleOutputs(ctx, pubsub, "outputs.small", "outputs.large"))
		assert.ElementsMatch(t, []int64{3, 1}, byID["outputs.small"])
		assert.ElementsMatch(t, []int64{70, 100}, byID["outputs.large"])
	})
}

// Shared intermediary: one var used by multiple outputs

func TestGraph_Join_SharedIntermediary(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "join_shared_intermediary.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// x=5 → shared=10 → plusOne=11, plusTen=20
		// x=10 → shared=20 → plusOne=21, plusTen=30
		feedInput(pubsub, "inputs.x", 5, 10)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		byID := outputsByID(collectMultipleOutputs(ctx, pubsub, "outputs.plusOne", "outputs.plusTen"))
		assert.Equal(t, []int64{11, 21}, byID["outputs.plusOne"])
		assert.Equal(t, []int64{20, 30}, byID["outputs.plusTen"])
	})
}
