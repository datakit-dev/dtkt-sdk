package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 5.2 Error path tests

// TestGraph_VarEvalError exercises a CEL expression that compiles but fails
// at runtime (division by zero). The executor should return an error.
func TestGraph_Error_VarEval(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "error_var_eval.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.x", 42)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "division by zero")
	})
}

// TestGraph_Action_MissingMethod verifies that referencing a method that
// doesn't exist in the registry returns an error at handler creation time.
func TestGraph_Action_MissingMethod(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "error_action_missing_method.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.x", 1)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestGraph_Action_NoMethods verifies that having an action node without
// providing any connectors to the executor returns an error.
func TestGraph_Action_NoMethods(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "error_action_no_methods.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// No WithConnectors -- executor has nil connectors map.
		feedInput(ps, "inputs.x", 1)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no connector")
	})
}

// TestGraph_ContextCancellation cancels the context while a ticker-driven
// graph is running. The executor should exit cleanly (no error) and the
// number of ticks must be bounded by the cancellation deadline -- a runtime
// that ignored cancellation would emit far more ticks than fit in the 150ms
// window. We assert both the lower bound (>=1, ticker did fire) and an
// upper bound (<=15, well above the ~3 ticks expected at 50ms interval but
// catching a "ran forever" regression).
func TestGraph_Error_ContextCancellation(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "error_context_cancellation.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		elapsed := time.Since(start)
		// Ticker-based graphs exit cleanly when context is cancelled.
		require.NoError(t, err)
		// Execute must return promptly after cancellation -- not run to
		// some natural completion that happens to also be quick.
		assert.Less(t, elapsed, 1*time.Second,
			"Execute should return promptly after ctx cancel; took %v", elapsed)

		results := collectOutputs(testContext(t), ps, "outputs.result")
		// At least one tick before cancellation (proves the ticker started).
		assert.GreaterOrEqual(t, len(results), 1)
		// Bounded by the cancel deadline. At 50ms interval over 150ms we
		// expect ~3 ticks; allow generous headroom for scheduler jitter
		// but catch "ignored ctx cancel and ran forever".
		assert.LessOrEqual(t, len(results), 15,
			"too many ticks for 150ms cancel window; ctx cancel may not have propagated")
	})
}
