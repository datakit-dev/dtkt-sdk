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

		results := collectOutputs(ctx, pubsub, "outputs.result")
		require.Len(t, results, 5)
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

// CEL EOF() function

func TestCEL_EOF_Function(t *testing.T) {
	env, err := buildCELEnv(nil)
	require.NoError(t, err)
	prog, err := compileCEL(env, "= EOF()")
	require.NoError(t, err, "EOF() should compile")

	result, err := evalCEL(prog, nil)
	require.NoError(t, err, "EOF() should evaluate")

	val, err := refValToExpr(result)
	require.NoError(t, err, "EOF() result should convert to expr.Value")
	assert.True(t, isEOFValue(val), "EOF() result should be recognized as EOF")
}

func TestCEL_EOF_Conditional(t *testing.T) {
	env, err := buildCELEnv(nil)
	require.NoError(t, err)
	prog, err := compileCEL(env, "= this.count > 2 ? EOF() : this.count")
	require.NoError(t, err)

	// count=1: should return 1
	result, err := evalCEL(prog, map[string]any{"this": map[string]any{"count": int64(1)}})
	require.NoError(t, err)
	val, err := refValToExpr(result)
	require.NoError(t, err)
	assert.False(t, isEOFValue(val))
	assert.Equal(t, int64(1), val.GetInt64Value())

	// count=3: should return EOF
	result, err = evalCEL(prog, map[string]any{"this": map[string]any{"count": int64(3)}})
	require.NoError(t, err)
	val, err = refValToExpr(result)
	require.NoError(t, err)
	assert.True(t, isEOFValue(val))
}

// CEL EOF() function
