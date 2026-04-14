package runtime

import (
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
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

// Input cache: reuse last received value when throttle expires

func TestGraph_Input_Cache(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_cache.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Feed one value; keep channel open for 50ms so the inputHandler can
		// re-emit the cached value on subsequent throttle timeouts.
		timedFeedInput(ps, "inputs.x", 80*time.Millisecond, int64(42))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 2, "expected 1 fresh + 1 cached output")
		for _, r := range results {
			assert.Equal(t, int64(42), r.GetValue().GetInt64Value())
		}
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

// Cache takes priority over type default

func TestGraph_Input_CachePriorityOverDefault(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_cache_over_default.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Feed value 42; subsequent timeouts should use cache (42), not default (99).
		timedFeedInput(ps, "inputs.x", 80*time.Millisecond, int64(42))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 2, "expected 1 fresh + 1 cached output")
		for _, r := range results {
			assert.Equal(t, int64(42), r.GetValue().GetInt64Value(),
				"cache should take priority over type default")
		}
	})
}

// Default throttle injection: cache without explicit throttle

func TestGraph_Input_DefaultThrottleInjection(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_default_throttle.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		defaultThrottle := &flowv1beta2.Rate{Count: 1, Interval: durationpb.New(50 * time.Millisecond)}
		timedFeedInput(ps, "inputs.x", 80*time.Millisecond, int64(42))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithDefaultInputThrottle(defaultThrottle)}, extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 2, "expected 1 fresh + 1 cached, proving default throttle was injected")
		for _, r := range results {
			assert.Equal(t, int64(42), r.GetValue().GetInt64Value())
		}
	})
}

// intPtr returns a pointer to the given int64 value for use in proto optional fields.
func intPtr(v int64) *int64 { return &v }
