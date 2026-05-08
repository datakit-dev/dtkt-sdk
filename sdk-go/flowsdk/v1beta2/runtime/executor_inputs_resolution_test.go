package runtime

import (
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// timedFeedInput publishes values to the input's PubSub topic, then publishes
// EOF after closeAfter. This keeps the input subscription open so the
// inputHandler can re-emit cached/default values on throttle timeouts.
func timedFeedInput(ps executor.PubSub, nodeID string, closeAfter time.Duration, values ...any) {
	topic := testTopics.InputFor(nodeID)
	for _, v := range values {
		val, _ := nativeToExpr(v)
		ps.Publish(topic, pubsub.NewMessage(val)) //nolint:errcheck
	}
	go func() {
		time.Sleep(closeAfter)
		ps.Publish(topic, pubsub.NewMessage(newEOFValue())) //nolint:errcheck
	}()
}

// Input cache: pushed value is delivered to consumers via the snapshot
// (read inline). When the consumer has only the cached dep (oneShot),
// it fires exactly once with the latest cached value.

func TestGraph_Input_Cache(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_cache.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.x", int64(42))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1, "all-cached-deps consumer fires exactly once")
		assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
	})
}

// Type default: use the type's default when no value and no cache

func TestGraph_Input_TypeDefault(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_type_default.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Feed no values; the inputHandler should fall back to default=99 on every
		// throttle timeout.
		timedFeedInput(ps, "inputs.x", 130*time.Millisecond)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 2, "expected 2 outputs from type default")
		for _, r := range results {
			assert.Equal(t, int64(99), r.GetValue().GetInt64Value())
		}
	})
}

// Input cache + default: a pushed value updates the cached value
// (default seeds the initial cached value; subsequent pushes replace
// it). The all-cached consumer fires once per producer pulse, so we may
// see 1 or 2 outputs depending on race between the default pre-pulse
// drain and the bridge's emit:
//   - If bridge emit lands first: 1 output (42, since cachedValues
//     was already updated and the pre-pulse drained reads the new
//     value).
//   - If consumer drains pre-pulse first: 2 outputs (99 then 42).
// In both cases, the LAST output is 42, demonstrating cache priority
// over default.

func TestGraph_Input_CachePriorityOverDefault(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_cache_over_default.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.x", int64(42))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.NotEmpty(t, results, "all-cached consumer should fire at least once")
		last := results[len(results)-1].GetValue().GetInt64Value()
		assert.Equal(t, int64(42), last, "last value must be the pushed cache, demonstrating priority over default")
	})
}

// intPtr returns a pointer to the given int64 value for use in proto optional fields.
func intPtr(v int64) *int64 { return &v }
