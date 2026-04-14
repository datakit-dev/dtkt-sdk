package runtime

import (
	"context"
	"testing"
	"time"

	expr "cel.dev/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Flatten transform

func TestGraph_Transform_Flatten(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_flatten.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 1 → [1,10], 2 → [2,20] → flattened: 1,10,2,20
		feedInput(pubsub, "inputs.x", 1, 2)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{1, 10, 2, 20}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Transform_FlattenPassthrough(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_flatten_passthrough.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{1, 2, 3}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Transform_DeepFlatten(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_deep_flatten.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// Input 10: [[10,11],[12,13]] → [10,11],[12,13] → 10,11,12,13
		feedInput(pubsub, "inputs.x", 10)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{10, 11, 12, 13}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Transform_DeepFlatten_StreamedInputs(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_deep_flatten_streamed.yaml")

		listVal := func(vals ...*expr.Value) *expr.Value {
			return &expr.Value{Kind: &expr.Value_ListValue{ListValue: &expr.ListValue{Values: vals}}}
		}
		int64Val := func(v int64) *expr.Value {
			return &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: v}}
		}

		// [[[1,2]]] and [[[3,4]]]
		nested1 := listVal(listVal(listVal(int64Val(1), int64Val(2))))
		nested2 := listVal(listVal(listVal(int64Val(3), int64Val(4))))

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", nested1, nested2)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{1, 2, 3, 4}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Multi-step pipelines: different orderings

func TestGraph_Transform_FilterThenMap(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_filter_then_map.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5, 6)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{4, 8, 12}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Transform_MapThenFilter(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_map_then_filter.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{6, 8, 10}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Transform_MapThenFlattenThenFilter(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_map_flatten_filter.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 1→[1,10], 2→[2,20] → flattened: 1,10,2,20 → >5: 10,20
		feedInput(pubsub, "inputs.x", 1, 2)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{10, 20}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Transform_FilterMapReduce(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_filter_map_scan.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 1..6 → even: 2,4,6 → doubled: 4,8,12 → running sum: 4,12,24
		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5, 6)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{4, 12, 24}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Transform_MapFlattenReduce(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_map_flatten_scan.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 10→[10,11], 20→[20,21] → flattened: 10,11,20,21 → running sum: 10,21,41,62
		feedInput(pubsub, "inputs.x", 10, 20)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{10, 21, 41, 62}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Transforms across nodes in the chain

func TestGraph_Transform_AcrossNodes(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_across_nodes.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 1,2,3,4,5 → +1: 2,3,4,5,6 → >3: 4,5,6 → *10: 40,50,60
		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{40, 50, 60}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Transform_AcrossNodes_FlattenInMiddle(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_across_nodes_flatten_middle.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 1→[1,100], 2→[2,200] → flatten: 1,100,2,200 → >=100: 100,200
		feedInput(pubsub, "inputs.x", 1, 2)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{100, 200}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Per-node-type transform sections

func TestGraph_Transform_OnInput_FullChain(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_input_full_chain.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 1..5 → +1: 2,3,4,5,6 → even: 2,4,6 → running sum: 2,6,12
		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{2, 6, 12}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Transform_OnVar_FullChain(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_var_full_chain.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 1,2,3 → >1: 2,3 → [2,4],[3,6] → 2,4,3,6 → running sum: 2,6,9,15
		feedInput(pubsub, "inputs.x", 1, 2, 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{2, 6, 9, 15}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Transform_OnOutput_FullChain(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_output_full_chain.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 1..5 → >2: 3,4,5 → *10: 30,40,50 → sum on EOF: 120
		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(120), results[0].GetValue().GetInt64Value())
	})
}

// Filter edge cases

func TestGraph_Transform_FilterRejectsAll(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_filter_rejects_all.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Empty(t, collectOutputs(ctx, pubsub, "outputs.result"))
	})
}

func TestGraph_Transform_FilterAcceptsAll(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_filter_accepts_all.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{1, 2, 3}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// String transforms (non-int64)

func TestGraph_Transform_StringMap(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_string_map.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.name", "hello", "world")
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.Equal(t, "hello!", results[0].GetValue().GetStringValue())
		assert.Equal(t, "world!", results[1].GetValue().GetStringValue())
	})
}

func TestGraph_Transform_StringFilter(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_string_filter.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.name", "hi", "hello", "yo", "world")
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.Equal(t, "hello", results[0].GetValue().GetStringValue())
		assert.Equal(t, "world", results[1].GetValue().GetStringValue())
	})
}

// Double transforms

func TestGraph_Transform_DoubleMapFilter(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_double_map_filter.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 1.0*2.5=2.5, 2.0*2.5=5.0, 3.0*2.5=7.5, 4.0*2.5=10.0 → >5.0: 7.5, 10.0
		feedInput(pubsub, "inputs.x", 1.0, 2.0, 3.0, 4.0)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.Equal(t, 7.5, results[0].GetValue().GetDoubleValue())
		assert.Equal(t, 10.0, results[1].GetValue().GetDoubleValue())
	})
}

// Phase 5.5: Flatten scalar passthrough

func TestGraph_Transform_Flatten_ScalarPassthrough(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_flatten_scalar_passthrough.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 5, 10, 15)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{5, 10, 15}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Phase 5.5: Map type change (int64 → string)

func TestGraph_Transform_Map_Int64ToString(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_map_int64_to_string.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 42, 100)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []string{"42", "100"}, outputStrings(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Phase 5.5: Map eval error at runtime

func TestGraph_Transform_Map_EvalError(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_map_eval_error.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "division by zero")
	})
}

// Phase 5.5: Context cancellation mid-pipeline

func TestGraph_Transform_ContextCancel(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_context_cancel.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		ctx, cancel := context.WithTimeout(testContext(t), 50*time.Millisecond)
		defer cancel()

		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		// Should return cleanly or with context deadline error.
		if err != nil {
			require.ErrorIs(t, err, context.DeadlineExceeded)
		}
	})
}

// Phase 5.5: Chained reduces across node types

func TestGraph_Transform_InputThenVar_ChainedReduces(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "transform_input_var_chained_reduces.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 2, 3, 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		// Input reduce: 2+3+5=10. Var scan: initial=1, 1*10=10.
		assert.Equal(t, int64(10), results[0].GetValue().GetInt64Value())
	})
}
