package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Stream via unary call (echo.Echo)

func TestGraph_Stream_UnaryEcho(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_unary_echo.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", "hello")
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, "hello", results[0].GetValue().GetStringValue())
	})
}

// Stream via server-stream call (random.Numbers sends count=1 random int).

func TestGraph_Stream_ServerStream(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_server_stream.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.count", 1)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
	})
}

// Stream with transforms (map on stream output)

func TestGraph_Stream_WithMapTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_map_transform.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		// echo returns 5 → var map *10 = 50
		assert.Equal(t, int64(50), results[0].GetValue().GetInt64Value())
	})
}

func TestGraph_Stream_WithFilterTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_filter_transform.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 1, 5, 2, 7)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		// echo returns each value → var filter keeps >3: 5, 7
		assert.Equal(t, []int64{5, 7}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Stream_WithTransformChain(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_transform_chain.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// Values 1..6, echo returns each → var: filter evens: 2,4,6 → *10: 20,40,60 → running sum: 20,60,120
		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5, 6)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{20, 60, 120}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Bidi stream (echo.BidiEcho)

func TestGraph_Stream_BidiEcho(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_bidi_echo.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 10, 20, 30)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		// BidiEcho echoes each message back.
		assert.Equal(t, []int64{10, 20, 30}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Client stream (log.Collect)

func TestGraph_Stream_ClientStream(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_client_stream.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.msg", 1, 2, 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		// log.Collect returns "logged N messages"
		assert.Equal(t, "logged 3 messages", results[0].GetValue().GetStringValue())
	})
}

// Stream feeding downstream var

func TestGraph_Stream_ToVar(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_to_var.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		feedInput(pubsub, "inputs.x", 3, 7)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		// echo returns 3,7 → doubled 6,14
		assert.Equal(t, []int64{6, 14}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Stream with `when` guard -- bidi echo only when condition is true

func TestGraph_Stream_BidiWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_bidi_when.yaml")

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

// Stream with `when` guard -- server-stream only fires when true

func TestGraph_Stream_ServerStreamWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_server_when.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// Feed 3 values: 3 (skip), 8 (fire), 1 (skip)
		feedInput(pubsub, "inputs.x", 3, 8, 1)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(8), results[0].GetValue().GetInt64Value())
	})
}

// Stream with `close_request_when` -- bidi closes request side

func TestGraph_Stream_BidiCloseRequestWhen(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_bidi_close_request.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// Feed values: 1, 2, 99 (close trigger) -- only 1 and 2 should echo
		feedInput(pubsub, "inputs.x", 1, 2, 99)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		assert.Equal(t, int64(1), results[0].GetValue().GetInt64Value())
		assert.Equal(t, int64(2), results[1].GetValue().GetInt64Value())
	})
}

// Stream throttle (bidi stream with rate limiting)

func TestGraph_Stream_BidiThrottle(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_bidi_throttle.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		// 5 values at 5 per 500ms = 100ms apart. Should take ~400ms (first fires immediately).
		start := time.Now()
		feedInput(pubsub, "inputs.x", 1, 2, 3, 4, 5)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		elapsed := time.Since(start)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		assert.Equal(t, []int64{1, 2, 3, 4, 5}, outputInt64s(results))
		// 4 throttle waits of ~100ms each
		assert.GreaterOrEqual(t, elapsed, 300*time.Millisecond)
	})
}
