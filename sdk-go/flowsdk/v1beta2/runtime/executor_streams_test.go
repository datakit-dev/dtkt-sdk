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
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// random.Numbers mock emits N int64s in [0, 100). count=3 means 3
		// emissions in order from the server stream.
		feedInput(pubsub, "inputs.count", 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 3, "expected 3 server-stream emissions for count=3")
		for i, r := range results {
			v := r.GetValue().GetInt64Value()
			assert.GreaterOrEqual(t, v, int64(0), "result[%d] must be int64 in [0,100)", i)
			assert.Less(t, v, int64(100), "result[%d] must be int64 in [0,100)", i)
		}
	})
}

// Stream with transforms (map on stream output)

func TestGraph_Stream_WithMapTransform(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_map_transform.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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

// Stream throttle (bidi stream with rate limiting)

func TestGraph_Stream_BidiThrottle(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_bidi_throttle.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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

// --- Stream error paths ---

// Server stream that sends 2 messages then aborts mid-flight. With the default
// TERMINATE strategy, the executor returns the abort error.
func TestGraph_Stream_Error_Aborted(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_aborted.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(pubsub, "inputs.msg", 1)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "stream aborted mid-flight")
	})
}

// Server stream that closes immediately with no messages: flow completes
// successfully and produces no outputs.
func TestGraph_Stream_Error_Empty(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_empty.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(pubsub, "inputs.msg", 1)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Empty(t, collectOutputs(ctx, pubsub, "outputs.result"))
	})
}

// Server stream that opens but never sends or closes. Terminate() cancels the
// per-handler context; Execute returns ErrTerminated.
func TestGraph_Stream_Error_Idle_Terminate(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_idle.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(pubsub, "inputs.msg", 1)
		ctx := testContext(t)
		exec := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...)

		done := make(chan error, 1)
		go func() {
			done <- exec.Execute(ctx, graph)
		}()

		// Wait for the stream to open and idle, then terminate.
		time.Sleep(100 * time.Millisecond)
		exec.Terminate()

		// Bound the post-Terminate wait. A regression that leaves the idle
		// stream's recv blocked would otherwise fall through to testContext's
		// 10s deadline instead of failing fast.
		err := requireExecuteReturnsBy(t, done, 500*time.Millisecond)
		assert.ErrorIs(t, err, ErrTerminated)
	})
}

// Bidi stream that accepts one message then returns DeadlineExceeded.
func TestGraph_Stream_Error_BidiDeadline(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_bidi_deadline.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(pubsub, "inputs.msg", 1)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "deadline exceeded")
	})
}

// Client stream that accepts 2 messages then fails with InvalidArgument.
func TestGraph_Stream_Error_ClientStreamInvalid(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_client_invalid.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(pubsub, "inputs.msg", 1, 2, 3)
		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid payload")
	})
}
