package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Single transforms on Var nodes

func TestGraph_Var_MapTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "var_map_x2.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 5, 10)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{10, 20}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Var_FilterTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "var_filter_even.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3, 4)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{2, 4}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Var_ScanTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// Running scan: emits accumulated value on each input.
		graph := loadFlow(t, "var_scan_sum.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		// Running total: 1, 3, 6
		assert.Equal(t, []int64{1, 3, 6}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Var_FlattenTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "var_flatten.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 10, 20)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{10, 11, 20, 21}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Deep var chains

func TestGraph_Var_DeepChain_MapAtEveryLevel(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// input -> var1(+1) -> var2(*2) -> var3(+100) -> output
		graph := loadFlow(t, "var_deep_chain_3levels.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 5 -> +1=6 -> *2=12 -> +100=112
		feedInput(pubsub, "inputs.x", 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{112}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Var_DeepChain_FilterInMiddle(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// input -> var1(filter even) -> var2(*10) -> output
		graph := loadFlow(t, "var_deep_chain_filter_middle.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 1,2,3,4,5 -> even: 2,4 -> *10: 20,40
		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{20, 40}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Var_DeepChain_ScanAtEnd(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// input -> var1(filter) -> var2(map) -> var3(scan sum) -> output
		graph := loadFlow(t, "var_deep_chain_scan_end.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 1..5 -> >2: 3,4,5 -> *2: 6,8,10 -> running sum: 6,14,24
		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{6, 14, 24}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Var_DeepChain_FlattenAcrossVars(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// input -> var1(map to list) -> var2(flatten + filter) -> output
		graph := loadFlow(t, "var_deep_chain_flatten_across.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 1->[1,10], 2->[2,20] -> flatten: 1,10,2,20 -> >=10: 10,20
		feedInput(pubsub, "inputs.x", 1, 2)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{10, 20}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Var with Switch (case/default)

func TestGraph_Var_Switch(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// Use switch to categorize values.
		graph := loadFlow(t, "var_switch.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 5, 50, 500)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 3)
		assert.Equal(t, "small", results[0].GetValue().GetStringValue())
		assert.Equal(t, "medium", results[1].GetValue().GetStringValue())
		assert.Equal(t, "big", results[2].GetValue().GetStringValue())
	})
}

func TestGraph_Var_SwitchWithTransforms(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// Switch -> filter (only "big") -> output.
		graph := loadFlow(t, "var_switch_with_filter.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 5, 200, 50, 300)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.Equal(t, "big", results[0].GetValue().GetStringValue())
		assert.Equal(t, "big", results[1].GetValue().GetStringValue())
	})
}

// Var passthrough (no transforms)

func TestGraph_Var_Passthrough(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_var_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 7, 8, 9)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{7, 8, 9}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// String var operations

func TestGraph_Var_StringMapFilter(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "var_string_map_filter.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// "ab"->"ab_suffix"(9>8 yes), "x"->"x_suffix"(8=8 no), "hello"->"hello_suffix"(12>8 yes)
		feedInput(pubsub, "inputs.name", "ab", "x", "hello")
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.Equal(t, "ab_suffix", results[0].GetValue().GetStringValue())
		assert.Equal(t, "hello_suffix", results[1].GetValue().GetStringValue())
	})
}
