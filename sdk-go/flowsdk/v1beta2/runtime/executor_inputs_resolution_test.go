package runtime

import (
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
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
		ps.Publish(topic, pubsub.NewMessage(val)) //nolint:errcheck // test fixture feed to in-memory pubsub; a real failure surfaces as a downstream assertion
	}
	go func() {
		time.Sleep(closeAfter)
		ps.Publish(topic, pubsub.NewMessage(newEOFValue())) //nolint:errcheck // test fixture feed to in-memory pubsub; a real failure surfaces as a downstream assertion
	}()
}

// Input cache: pushed value is delivered to consumers via the snapshot
// (read inline). When the consumer has only the cached dep (oneShot),
// it fires exactly once with the latest cached value.

func TestGraph_Input_Cache(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_cache.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

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

// Cache branch of the throttle-injection fallback: cache:true with NO
// explicit throttle declared. When an input has a cache (or default) but
// no throttle, the runtime injects a minimum throttle so the resolution
// loop can pulse. Guards that such an input doesn't error at startup and
// delivers a pushed value once to the all-cached consumer.
//
// NOTE: coverage/smoke only, NOT a behavioral discriminator. With a value
// pushed, `cache` already makes the input resolvable (hasResolution), so
// the value is delivered whether or not the throttle is injected, and the
// cache dedup masks any re-emission -- this test passes even if the cache
// half of the injection conditional is removed. The pulse effect of the
// injection is proven non-vacuously by the default-only sibling
// (TestGraph_Input_DefaultOnlyThrottleFallback), which feeds no value and
// asserts the default re-emits repeatedly.

func TestGraph_Input_DefaultThrottleFallback(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_default_throttle.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(7))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1,
			"all-cached consumer should fire exactly once with the captured value")
		assert.Equal(t, int64(7), results[0].GetValue().GetInt64Value(),
			"captured value must be the pushed value, proving the default-throttle path "+
				"didn't drop or corrupt the cache")
	})
}

// Default-throttle fallback (no-cache branch): a default-bearing input with
// NO explicit throttle and NO cache. The cache-branch sister test above
// covers (cache || default) with cache set. This one covers the default-only
// half of the same conditional at executor_setup.go:568-574: even without
// cache, a declared default must still trigger the fallback throttle
// injection so the handler can re-emit the default on each pulse.
//
// Non-vacuous shape: feed ZERO values; hold the input subscription open for
// 130ms via the EOF-after delay; assert that multiple emissions of the
// default value (99) reach the output. If the fallback DIDN'T inject (i.e.
// the conditional were `if throttle == 0 && cache` without the `|| defVal !=
// nil` half), throttle would stay at 0, the input handler would never pulse,
// and zero outputs would surface. The test would then fail at
// `require.GreaterOrEqual(len(results), 2)`.

func TestGraph_Input_DefaultOnlyThrottleFallback(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_default_only_throttle.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Feed NO values; hold the subscription open for 130ms so the
		// 10ms injected throttle gets ~13 pulses before EOF closes the
		// run. A minimum of 2 emissions proves the pulse fired more than
		// once -- "fired exactly once at iteration 0" would be consistent
		// with several non-fallback explanations (e.g. a stray
		// one-off emit), but repeated firing isolates the throttle
		// behavior.
		timedFeedInput(ps, "inputs.x", 130*time.Millisecond)

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.GreaterOrEqual(t, len(results), 2,
			"default-only fallback must inject a throttle so the default fires "+
				"repeatedly during the open window; zero/one outputs means the "+
				"defVal!=nil branch of executor_setup.go:568-574 isn't firing")
		for i, r := range results {
			assert.Equal(t, int64(99), r.GetValue().GetInt64Value(),
				"emission %d must be the declared default (99)", i)
		}
	})
}

// WithDefaultInputThrottle override: when the caller supplies a custom rate
// via WithDefaultInputThrottle, the runtime must use it instead of the
// built-in minInputThrottle (10ms) at executor_setup.go:568-574.
//
// Non-vacuous shape: reuse input_default_only_throttle.yaml (no explicit
// throttle, default=99) and override with a SLOW rate (100ms per event).
// Hold the input subscription open for 130ms. Two ranges differ by an
// order of magnitude:
//   - if the override is honored: 100ms/event over 130ms emits at most 2.
//   - if the override is dropped (10ms minInputThrottle wins): ~13
//     emissions over 130ms.
//
// Assertion `LessOrEqual(len(results), 2)` fails loudly under the
// drop-the-override regression (would see ~13). `NotEmpty` rules out the
// nothing-fires alternative (e.g. the option being installed but
// rateToDuration returning 0 from a malformed rate).

func TestGraph_Input_WithDefaultInputThrottleOverride(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_default_only_throttle.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		timedFeedInput(ps, "inputs.x", 130*time.Millisecond)

		// 1 event per 100ms = 100ms throttle window. Over 130ms only
		// one or two pulses fit (boundary depends on whether time=0
		// counts as the first pulse).
		slowRate := flowv1beta2.Rate_builder{
			Count:    1,
			Interval: durationpb.New(100 * time.Millisecond),
		}.Build()
		opts := append([]Option{WithDefaultInputThrottle(slowRate)}, extraOpts...)

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.NotEmpty(t, results,
			"the override rate must still pulse at least once during the 130ms window")
		require.LessOrEqual(t, len(results), 2,
			"WithDefaultInputThrottle(100ms) must throttle to <=2 emissions in 130ms; "+
				"a higher count (~13) means the override is being ignored and the 10ms "+
				"minInputThrottle is in effect")
		for i, r := range results {
			assert.Equal(t, int64(99), r.GetValue().GetInt64Value(),
				"emission %d must be the declared default", i)
		}
	})
}
