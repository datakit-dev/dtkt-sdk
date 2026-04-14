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
// graph is running. The executor should return nil (clean shutdown).
func TestGraph_Error_ContextCancellation(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "error_context_cancellation.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()

		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		// Ticker-based graphs exit cleanly when context is cancelled.
		require.NoError(t, err)

		results := collectOutputs(testContext(t), ps, "outputs.result")
		// Should have gotten at least 1 tick before cancellation.
		assert.GreaterOrEqual(t, len(results), 1)
	})
}
