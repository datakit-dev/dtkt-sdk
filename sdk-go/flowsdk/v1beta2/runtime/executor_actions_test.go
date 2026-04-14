package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// Basic action: unary call (echo.Echo)

func TestGraph_Action_UnaryEcho(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_unary_echo.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 42)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
	})
}

// Action with multiple input values

func TestGraph_Action_MultipleValues(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_multiple_values.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 10, 20, 30)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{10, 20, 30}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Action with transforms (map on action result)

func TestGraph_Action_WithMapTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_map_transform.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 3, 7)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		// echo returns 3,7 → var map *100 → 300,700
		assert.Equal(t, []int64{300, 700}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Action_WithFilterTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_filter_transform.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 8, 3, 10)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		// echo returns 1,8,3,10 → var filter >5 → 8,10
		assert.Equal(t, []int64{8, 10}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Action_WithTransformChain(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// Action → var(filter evens → map *5 → running sum) → output
		graph := loadFlow(t, "action_transform_chain.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 1,2,3,4 → echo → var: filter evens: 2,4 → *5: 10,20 → running sum: 10,30
		feedInput(pubsub, "inputs.x", 1, 2, 3, 4)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{10, 30}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Action feeding downstream var

func TestGraph_Action_ToVar(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_to_var.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 4, 9)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		// echo returns 4,9 → var tripled: 12,27
		assert.Equal(t, []int64{12, 27}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Action with string input

func TestGraph_Action_StringEcho(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_string_echo.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", "hello", "world")
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.Equal(t, "hello", results[0].GetValue().GetStringValue())
		assert.Equal(t, "world", results[1].GetValue().GetStringValue())
	})
}

// Action with `when` guard -- only fires when condition is true

func TestGraph_Action_When(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_when.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// Feed 3 values: 2 (skip), 10 (fire), 1 (skip)
		feedInput(pubsub, "inputs.x", 2, 10, 1)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(10), results[0].GetValue().GetInt64Value())
	})
}

// Action throttle: rate-limits how often the action fires

func TestGraph_Action_Throttle(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_throttle.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// 5 values at 5 per 500ms = 100ms apart. Should take ~400ms (first fires immediately).
		start := time.Now()
		feedInput(ps, "inputs.x", 1, 2, 3, 4, 5)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		elapsed := time.Since(start)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		assert.Equal(t, []int64{1, 2, 3, 4, 5}, outputInt64s(results))
		// 4 throttle waits of ~100ms each
		assert.GreaterOrEqual(t, elapsed, 300*time.Millisecond)
	})
}

// Memoize: same input produces cached response, RPC called once per unique input.

func TestGraph_Action_MemoizeSameInput(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		var callCount int
		client := newMockRPCClient()
		client.RegisterUnary("echo.Echo", func(_ context.Context, req proto.Message) (proto.Message, error) {
			callCount++
			return req, nil
		})

		graph := loadFlow(t, "action_memoize.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Send 42 three times -- only one RPC should fire.
		feedInput(ps, "inputs.x", 42, 42, 42)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithConnectors(rpc.Connectors{"echo": {Client: client, Resolver: client}})}, extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		assert.Equal(t, []int64{42, 42, 42}, outputInt64s(results))
		assert.Equal(t, 1, callCount, "RPC should be called only once for identical inputs")
	})
}

func TestGraph_Action_MemoizeDifferentInputs(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		var callCount int
		client := newMockRPCClient()
		client.RegisterUnary("echo.Echo", func(_ context.Context, req proto.Message) (proto.Message, error) {
			callCount++
			return req, nil
		})

		graph := loadFlow(t, "action_memoize.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// 1, 2, 1, 3, 2 -- three unique values, so 3 RPCs.
		feedInput(ps, "inputs.x", 1, 2, 1, 3, 2)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithConnectors(rpc.Connectors{"echo": {Client: client, Resolver: client}})}, extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		assert.Equal(t, []int64{1, 2, 1, 3, 2}, outputInt64s(results))
		assert.Equal(t, 3, callCount, "RPC should be called once per unique input value")
	})
}
