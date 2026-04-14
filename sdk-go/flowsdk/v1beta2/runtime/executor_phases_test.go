package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// phaseOf extracts the Phase from any StateNode via type switch.
func phaseOf(node executor.StateNode) flowv1beta2.RunSnapshot_Phase {
	switch n := node.(type) {
	case *flowv1beta2.RunSnapshot_InputNode:
		return n.GetPhase()
	case *flowv1beta2.RunSnapshot_GeneratorNode:
		return n.GetPhase()
	case *flowv1beta2.RunSnapshot_VarNode:
		return n.GetPhase()
	case *flowv1beta2.RunSnapshot_ActionNode:
		return n.GetPhase()
	case *flowv1beta2.RunSnapshot_StreamNode:
		return n.GetPhase()
	case *flowv1beta2.RunSnapshot_OutputNode:
		return n.GetPhase()
	case *flowv1beta2.RunSnapshot_InteractionNode:
		return n.GetPhase()
	default:
		return flowv1beta2.RunSnapshot_PHASE_UNSPECIFIED
	}
}

// collectPhases reads FlowEvent messages from the given channel, extracts the
// Phase from each NODE_OUTPUT event, and returns when a terminal phase is seen or
// the context is cancelled. NODE_UPDATE events (transform state) are skipped.
func collectPhases(ctx context.Context, ch <-chan *pubsub.Message) []flowv1beta2.RunSnapshot_Phase {
	var phases []flowv1beta2.RunSnapshot_Phase
	for {
		select {
		case <-ctx.Done():
			return phases
		case msg := <-ch:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			if evt.GetEventType() != flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
				continue
			}
			phase := phaseOf(runtimeNodeFromEvent(evt))
			phases = append(phases, phase)
			switch phase {
			case flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
				flowv1beta2.RunSnapshot_PHASE_ERRORED,
				flowv1beta2.RunSnapshot_PHASE_FAILED,
				flowv1beta2.RunSnapshot_PHASE_CANCELLED:
				return phases
			}
		}
	}
}

// waitForPhase reads FlowEvent messages (both NODE_OUTPUT and NODE_UPDATE) from the
// given channel until the target phase is observed. Returns true if found,
// false on context cancellation.
func waitForPhase(ctx context.Context, ch <-chan *pubsub.Message, target flowv1beta2.RunSnapshot_Phase) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		case msg := <-ch:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			node := runtimeNodeFromEvent(evt)
			if phaseOf(node) == target {
				return true
			}
		}
	}
}

// phaseNames converts Phase values to short strings for readable test output.
func phaseNames(phases []flowv1beta2.RunSnapshot_Phase) []string {
	names := make([]string, len(phases))
	for i, p := range phases {
		switch p {
		case flowv1beta2.RunSnapshot_PHASE_PENDING:
			names[i] = "PENDING"
		case flowv1beta2.RunSnapshot_PHASE_RUNNING:
			names[i] = "RUNNING"
		case flowv1beta2.RunSnapshot_PHASE_SUCCEEDED:
			names[i] = "SUCCEEDED"
		case flowv1beta2.RunSnapshot_PHASE_ERRORED:
			names[i] = "ERRORED"
		case flowv1beta2.RunSnapshot_PHASE_FAILED:
			names[i] = "FAILED"
		case flowv1beta2.RunSnapshot_PHASE_CANCELLED:
			names[i] = "CANCELLED"
		case flowv1beta2.RunSnapshot_PHASE_SUSPENDED:
			names[i] = "SUSPENDED"
		default:
			names[i] = p.String()
		}
	}
	return names
}

// --- Phase tracking tests ---

func TestPhase_InputNode(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_int64_to_output.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.x", 42, 99)
		ctx := testContext(t)

		// Subscribe to internal input topic BEFORE execution (fan-out copy).
		inputCh, err := ps.Subscribe(ctx, testTopics.For("inputs.x"))
		require.NoError(t, err)

		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		phases := collectPhases(ctx, inputCh)
		assert.Equal(t, []string{"RUNNING", "RUNNING", "SUCCEEDED"}, phaseNames(phases),
			"input node: values should be RUNNING, EOF should be SUCCEEDED")
	})
}

func TestPhase_OutputNode(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_int64_to_output.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.x", 42, 99)
		ctx := testContext(t)

		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		// Output topics are buffered (persistent mode) -- subscribe after execution.
		outputCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
		require.NoError(t, err)

		phases := collectPhases(ctx, outputCh)
		assert.Equal(t, []string{"RUNNING", "RUNNING", "SUCCEEDED"}, phaseNames(phases),
			"output node: values should be RUNNING, EOF should be SUCCEEDED")
	})
}

func TestPhase_VarNode(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_var_output.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.x", 7, 8)
		ctx := testContext(t)

		// Subscribe to internal var topic BEFORE execution.
		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.pass"))
		require.NoError(t, err)

		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		phases := collectPhases(ctx, varCh)
		assert.Equal(t, []string{"RUNNING", "RUNNING", "SUCCEEDED"}, phaseNames(phases),
			"var node: values should be RUNNING, EOF should be SUCCEEDED")
	})
}

func TestPhase_GeneratorNode(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_range_step.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)

		// Subscribe to internal generator topic BEFORE execution.
		genCh, err := ps.Subscribe(ctx, testTopics.For("generators.seq"))
		require.NoError(t, err)

		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		// Range 0..10 step 3 produces 4 values: 0, 3, 6, 9.
		phases := collectPhases(ctx, genCh)
		assert.Equal(t, []string{"RUNNING", "RUNNING", "RUNNING", "RUNNING", "SUCCEEDED"}, phaseNames(phases),
			"generator node: each value should be RUNNING, done should be SUCCEEDED")
	})
}

func TestPhase_ActionNode(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_unary_echo.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.msg", 42)
		ctx := testContext(t)

		// Subscribe to internal action topic BEFORE execution.
		actionCh, err := ps.Subscribe(ctx, testTopics.For("actions.echo"))
		require.NoError(t, err)

		err = NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		phases := collectPhases(ctx, actionCh)
		assert.Equal(t, []string{"RUNNING", "SUCCEEDED"}, phaseNames(phases),
			"action node: RPC result should be RUNNING, EOF should be SUCCEEDED")
	})
}

func TestPhase_StreamNode_ServerStream(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "stream_server_stream.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		feedInput(ps, "inputs.count", 1)
		ctx := testContext(t)

		// Subscribe to internal stream topic BEFORE execution.
		streamCh, err := ps.Subscribe(ctx, testTopics.For("streams.numbers"))
		require.NoError(t, err)

		err = NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		// count=1 -> 1 response value, then final EOF.
		phases := collectPhases(ctx, streamCh)
		assert.Equal(t, []string{"RUNNING", "SUCCEEDED"}, phaseNames(phases),
			"stream node: responses should be RUNNING, final close should be SUCCEEDED")
	})
}

func TestPhase_EmptyInput(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "input_int64_to_output.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// No values, just EOF.
		feedInput(ps, "inputs.x")
		ctx := testContext(t)

		inputCh, err := ps.Subscribe(ctx, testTopics.For("inputs.x"))
		require.NoError(t, err)

		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		phases := collectPhases(ctx, inputCh)
		assert.Equal(t, []string{"SUCCEEDED"}, phaseNames(phases),
			"empty input: only EOF with SUCCEEDED phase")
	})
}
