package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Range generator: bounded integer sequence

func TestGraph_Generator_Range(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_range.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{1, 2, 3, 4, 5}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Generator_RangeWithStep(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_range_step.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		// 0, 3, 6, 9
		assert.Equal(t, []int64{0, 3, 6, 9}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

func TestGraph_Generator_RangeWithRate(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_range_rate.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// 5 values at 5 per 100ms = 20ms apart. Should take ~80ms (first emits immediately).
		start := time.Now()
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		elapsed := time.Since(start)
		require.NoError(t, err)

		assert.Equal(t, []int64{1, 2, 3, 4, 5}, outputInt64s(collectOutputs(ctx, ps, "outputs.result")))
		// Should take at least 60ms (4 waits of ~20ms each, with some slack).
		assert.GreaterOrEqual(t, elapsed, 60*time.Millisecond)
	})
}

// Ticker generator: stop_when eval_count >= 5 produces exactly 5 ticks.

func TestGraph_Generator_Ticker(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_ticker.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		// Ticker without a value expression emits the monotonic eval_count.
		// FC.stop_when fires at eval_count >= 5, so we expect exactly
		// [1,2,3,4,5].
		results := outputInt64s(collectOutputs(ctx, pubsub, "outputs.result"))
		assert.Equal(t, []int64{1, 2, 3, 4, 5}, results,
			"ticker default value is the monotonic eval_count")
	})
}

func TestGraph_Generator_TickerWithValueExpr(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_ticker_value_expr.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, pubsub, "outputs.result"))
		// value = count * 100; stop at eval_count >= 5.
		assert.Equal(t, []int64{100, 200, 300, 400, 500}, results)
	})
}

// Ticker generator with initial delay -- the first emission is gated by `delay`.
// Verifies the field is honored, not just decoded.

func TestGraph_Generator_TickerDelay(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_ticker_delay.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		ctx := testContext(t)
		start := time.Now()
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		elapsed := time.Since(start)
		require.NoError(t, err)

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 2)
		// 50ms delay + 50ms interval = at least ~100ms total before stop.
		assert.GreaterOrEqual(t, elapsed, 80*time.Millisecond,
			"first tick must wait for delay before emitting")
	})
}

// Range generator feeding downstream var chain

func TestGraph_Generator_RangeToVar(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_range_to_var.yaml")

		pubsub := newPubSub()
		defer pubsub.Close() //nolint:errcheck

		ctx := testContext(t)
		err := NewExecutor(pubsub, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		assert.Equal(t, []int64{2, 4, 6, 8, 10}, outputInt64s(collectOutputs(ctx, pubsub, "outputs.result")))
	})
}

// Cron generator

func TestGraph_Generator_Cron(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		// @every 1s is the minimum resolution for cron.
		graph := loadFlow(t, "gen_cron.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// stop_when at eval_count >= 2; exactly 2 cron ticks.
		assert.Equal(t, []int64{1, 2}, results)
	})
}

func TestGraph_Generator_CronWithValueExpr(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_cron_value_expr.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		// value = count * 100; stop at eval_count >= 2.
		assert.Equal(t, []int64{100, 200}, results)
	})
}

// Cron generator with invalid expression should fail at startup.

func TestGraph_Generator_CronInvalidExpression(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_cron_invalid.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing cron expression")
	})
}

