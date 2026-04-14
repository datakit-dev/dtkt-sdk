package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// No transforms -- bare wiring.

func TestGraph_InputToOutput(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_int64_to_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 42)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
	})
}

func TestGraph_InputToOutput_MultipleValues(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_int64_to_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{1, 2, 3}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_InputToVarToOutput(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_var_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 7, 8)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{7, 8}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_EmptyInput(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_int64_to_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x")
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Empty(t, collectOutputs(ctx, pubsub, "outputs.result"))
	})
}
