package runtime

import (
	"sort"
	"testing"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Input value types -- passthrough (no transforms)

func TestGraph_Input_Int64(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_int64_to_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", int64(42), int64(-1), int64(0))
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{42, -1, 0}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Input_Int32(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_int32_to_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", int32(100), int32(-5))
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.Equal(t, int64(100), results[0].GetValue().GetInt64Value())
		assert.Equal(t, int64(-5), results[1].GetValue().GetInt64Value())
	})
}

func TestGraph_Input_Uint64(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_uint64_to_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", uint64(999), uint64(0))
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.Equal(t, uint64(999), results[0].GetValue().GetUint64Value())
		assert.Equal(t, uint64(0), results[1].GetValue().GetUint64Value())
	})
}

func TestGraph_Input_Uint32(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_uint32_to_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", uint32(42))
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, uint64(42), results[0].GetValue().GetUint64Value())
	})
}

func TestGraph_Input_Double(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_double_to_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 3.14, -0.5, 0.0)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 3)
		assert.Equal(t, 3.14, results[0].GetValue().GetDoubleValue())
		assert.Equal(t, -0.5, results[1].GetValue().GetDoubleValue())
		assert.Equal(t, 0.0, results[2].GetValue().GetDoubleValue())
	})
}

func TestGraph_Input_Float(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_float_to_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", float32(1.5), float32(2.5))
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.InDelta(t, 1.5, results[0].GetValue().GetDoubleValue(), 0.001)
		assert.InDelta(t, 2.5, results[1].GetValue().GetDoubleValue(), 0.001)
	})
}

func TestGraph_Input_Bool(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_bool_to_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", true, false, true)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 3)
		assert.True(t, results[0].GetValue().GetBoolValue())
		assert.False(t, results[1].GetValue().GetBoolValue())
		assert.True(t, results[2].GetValue().GetBoolValue())
	})
}

func TestGraph_Input_String(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_string_to_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", "hello", "", "world")
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 3)
		assert.Equal(t, "hello", results[0].GetValue().GetStringValue())
		assert.Equal(t, "", results[1].GetValue().GetStringValue())
		assert.Equal(t, "world", results[2].GetValue().GetStringValue())
	})
}

func TestGraph_Input_Bytes(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_bytes_to_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", []byte("abc"), []byte{0x01, 0x02})
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.Equal(t, []byte("abc"), results[0].GetValue().GetBytesValue())
		assert.Equal(t, []byte{0x01, 0x02}, results[1].GetValue().GetBytesValue())
	})
}

// Input with transforms

func TestGraph_Input_MapTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_int64_map_x2.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{2, 4, 6}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Input_FilterTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_int64_filter_even.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5, 6)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{2, 4, 6}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Input_MapTransform_Chain(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// Input(map +100) -> Var (pass-through) -> Output.
		graph := loadFlow(t, "input_int64_map_plus100_var_output.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 2)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{101, 102}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Non-int64 input with transforms

func TestGraph_Input_String_MapTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_string_map_world.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.name", "hello", "goodbye")
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.Equal(t, "hello world", results[0].GetValue().GetStringValue())
		assert.Equal(t, "goodbye world", results[1].GetValue().GetStringValue())
	})
}

func TestGraph_Input_Double_FilterTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_double_filter_gt2.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1.0, 2.5, 0.5, 3.0)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.Equal(t, 2.5, results[0].GetValue().GetDoubleValue())
		assert.Equal(t, 3.0, results[1].GetValue().GetDoubleValue())
	})
}

func TestGraph_Input_Bool_FilterTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// Filter only truthy values.
		graph := loadFlow(t, "input_bool_filter_true.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", true, false, true, false)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.True(t, results[0].GetValue().GetBoolValue())
		assert.True(t, results[1].GetValue().GetBoolValue())
	})
}

// Multiple inputs

func TestGraph_Input_TwoInputsSum(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// Two separate inputs feed a single output directly in the expression.
		graph := loadFlow(t, "input_two_int64_sum.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.a", int64(10))
		feedInput(pubsub, "inputs.b", int64(20))

		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.sumAB")
		require.Len(t, results, 1)
		assert.Equal(t, int64(30), results[0].GetValue().GetInt64Value())
	})
}

func TestGraph_Input_MixedTypes(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// String input and int64 input feeding separate outputs.
		graph := loadFlow(t, "input_mixed_string_int64.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.name", "alice")
		feedInput(pubsub, "inputs.count", int64(5))

		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		nameResults := collectOutputs(ctx, pubsub, "outputs.name")
		require.Len(t, nameResults, 1)
		assert.Equal(t, "alice", nameResults[0].GetValue().GetStringValue())

		countResults := collectOutputs(ctx, pubsub, "outputs.count")
		require.Len(t, countResults, 1)
		assert.Equal(t, int64(5), countResults[0].GetValue().GetInt64Value())
	})
}

// Constant input -- only the first value is used

func TestGraph_Input_Constant(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_int64_constant.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Feed 3 values -- only the first should be delivered.
		feedInput(ps, "inputs.x", int64(10), int64(20), int64(30))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{10}, outputInt64s(collectOutputs(ctx, ps, "outputs.result")))
	})
}

// InputRequestEvent emission

func TestGraph_Input_RequestEventEmitted(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_two_int64_sum.yaml")

		inputReqs := make(chan *flowv1beta2.InputRequestEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.a", int64(1))
		feedInput(ps, "inputs.b", int64(2))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInputRequests(inputReqs)}, extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		// Collect all emitted InputRequestEvents.
		close(inputReqs)
		var ids []string
		for evt := range inputReqs {
			ids = append(ids, evt.GetId())
		}
		sort.Strings(ids)
		assert.Equal(t, []string{"inputs.a", "inputs.b"}, ids)
	})
}
