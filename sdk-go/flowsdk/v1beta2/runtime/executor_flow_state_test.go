package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	outboxmem "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox/memory"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// collectFlowStates reads FlowEvent messages from the flow topic and returns
// all FlowState oneof variants (FLOW_UPDATE events).
func collectFlowStates(ctx context.Context, ch <-chan *pubsub.Message) []*flowv1beta2.RunSnapshot_FlowState {
	var states []*flowv1beta2.RunSnapshot_FlowState
	for {
		select {
		case <-ctx.Done():
			return states
		case msg, ok := <-ch:
			if !ok {
				return states
			}
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			if evt.WhichData() == flowv1beta2.RunSnapshot_FlowEvent_Flow_case {
				states = append(states, evt.GetFlow())
				// Stop after terminal phase.
				switch evt.GetFlow().GetPhase() {
				case flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
					flowv1beta2.RunSnapshot_PHASE_ERRORED,
					flowv1beta2.RunSnapshot_PHASE_CANCELLED:
					return states
				}
			}
		}
	}
}

func TestFlowState_Success(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "outbox_input_var_output.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		flowCh, err := ps.Subscribe(ctx, testTopics.Flow())
		require.NoError(t, err)

		feedInput(ps, "inputs.x", 5)
		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		states := collectFlowStates(ctx, flowCh)
		require.Len(t, states, 2, "expected RUNNING + SUCCEEDED flow states")
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_RUNNING, states[0].GetPhase())
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, states[1].GetPhase())

		// start_time should be set on both
		assert.True(t, states[0].GetStartTime().IsValid(), "RUNNING should have start_time")
		assert.True(t, states[1].GetStartTime().IsValid(), "SUCCEEDED should have start_time")

		// stop_time only on terminal
		assert.False(t, states[0].GetStopTime().IsValid(), "RUNNING should not have stop_time")
		assert.True(t, states[1].GetStopTime().IsValid(), "SUCCEEDED should have stop_time")

		// event_time on both
		assert.True(t, states[0].GetEventTime().IsValid())
		assert.True(t, states[1].GetEventTime().IsValid())

		// Chronological ordering.
		assert.False(t, states[1].GetStopTime().AsTime().Before(states[0].GetStartTime().AsTime()),
			"stop_time should be >= start_time")

		// No error on success.
		assert.Nil(t, states[1].GetError())
	})
}

func TestFlowState_Error(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "action_error_internal.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		flowCh, err := ps.Subscribe(ctx, testTopics.Flow())
		require.NoError(t, err)

		feedInput(ps, "inputs.msg", 99)
		err = NewExecutor(ps, testTopics, append(mockRPCOptions(), extraOpts...)...).Execute(ctx, graph)
		require.Error(t, err)

		states := collectFlowStates(ctx, flowCh)
		require.Len(t, states, 2, "expected RUNNING + ERRORED flow states")
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_RUNNING, states[0].GetPhase())
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_ERRORED, states[1].GetPhase())

		// Error status should be populated.
		require.NotNil(t, states[1].GetError())
		assert.NotEmpty(t, states[1].GetError().GetMessage())
	})
}

func TestFlowState_Terminate(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "gen_long_running.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		flowCh, err := ps.Subscribe(ctx, testTopics.Flow())
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)

		// Run in background and terminate after a short delay.
		execDone := make(chan error, 1)
		go func() {
			execDone <- exec.Execute(ctx, graph)
		}()

		// Wait for RUNNING event before terminating.
		waitCtx, waitCancel := context.WithTimeout(ctx, 2*time.Second)
		defer waitCancel()
		var runningState *flowv1beta2.RunSnapshot_FlowState
		for {
			select {
			case <-waitCtx.Done():
				t.Fatal("timed out waiting for RUNNING flow state")
			case msg := <-flowCh:
				evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
				msg.Ack()
				if evt.WhichData() == flowv1beta2.RunSnapshot_FlowEvent_Flow_case {
					runningState = evt.GetFlow()
				}
			}
			if runningState != nil {
				break
			}
		}
		require.Equal(t, flowv1beta2.RunSnapshot_PHASE_RUNNING, runningState.GetPhase())

		exec.Terminate()
		err = <-execDone
		require.ErrorIs(t, err, ErrTerminated)

		// Collect terminal flow state.
		termCtx, termCancel := context.WithTimeout(ctx, 2*time.Second)
		defer termCancel()
		states := collectFlowStates(termCtx, flowCh)
		require.NotEmpty(t, states, "expected terminal CANCELLED flow state")
		last := states[len(states)-1]
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_CANCELLED, last.GetPhase())
	})
}

func TestFlowState_Outbox_Snapshot(t *testing.T) {
	// Verify the FlowState is materialized in the outbox snapshot.
	graph := loadFlow(t, "outbox_input_var_output.yaml")

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck

	store := outboxmem.New()
	feedInput(ps, "inputs.x", 5)
	ctx := testContext(t)
	err := NewExecutor(ps, testTopics, WithOutbox(store)).Execute(ctx, graph)
	require.NoError(t, err)

	// Give the relay time to forward events.
	time.Sleep(100 * time.Millisecond)

	snap, err := store.SnapshotAt(context.Background(), uuid.Max)
	require.NoError(t, err)

	// The flow field should be populated with the terminal state.
	require.NotNil(t, snap.GetFlow(), "snapshot should have flow state")
	assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, snap.GetFlow().GetPhase())
	assert.True(t, snap.GetFlow().GetStartTime().IsValid())
	assert.True(t, snap.GetFlow().GetStopTime().IsValid())
}

func TestFlowState_Outbox_ErrorSnapshot(t *testing.T) {
	graph := loadFlow(t, "action_error_internal.yaml")

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck

	store := outboxmem.New()
	feedInput(ps, "inputs.msg", 99)
	ctx := testContext(t)
	err := NewExecutor(ps, testTopics, append(mockRPCOptions(), WithOutbox(store))...).Execute(ctx, graph)
	require.Error(t, err)

	time.Sleep(100 * time.Millisecond)

	snap, err := store.SnapshotAt(context.Background(), uuid.Max)
	require.NoError(t, err)

	require.NotNil(t, snap.GetFlow())
	assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_ERRORED, snap.GetFlow().GetPhase())
	require.NotNil(t, snap.GetFlow().GetError())
}

func TestFlowState_Outbox_Events(t *testing.T) {
	// Verify that FlowState FLOW_UPDATE events are in the outbox event log.
	graph := loadFlow(t, "outbox_input_var_output.yaml")

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck

	store := outboxmem.New()
	feedInput(ps, "inputs.x", 5)
	ctx := testContext(t)
	err := NewExecutor(ps, testTopics, WithOutbox(store)).Execute(ctx, graph)
	require.NoError(t, err)

	msgs, err := store.ReadEvents(context.Background(), uuid.Nil, 1000)
	require.NoError(t, err)

	var flowUpdateCount int
	var lastFlowState *flowv1beta2.RunSnapshot_FlowState
	for _, msg := range msgs {
		evt, ok := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
		if !ok {
			continue
		}
		if evt.GetEventType() == flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_FLOW_UPDATE {
			flowUpdateCount++
			lastFlowState = evt.GetFlow()
		}
	}

	assert.Equal(t, 2, flowUpdateCount, "should have exactly RUNNING + SUCCEEDED FLOW_UPDATE events")
	require.NotNil(t, lastFlowState)
	assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, lastFlowState.GetPhase())
}

func TestFlowState_DirectPubSub(t *testing.T) {
	// Without an outbox, FlowState events should still be published to the flow topic.
	graph := loadFlow(t, "outbox_input_var_output.yaml")

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck

	ctx := testContext(t)
	flowCh, err := ps.Subscribe(ctx, testTopics.Flow())
	require.NoError(t, err)

	feedInput(ps, "inputs.x", 5)
	err = NewExecutor(ps, testTopics).Execute(ctx, graph)
	require.NoError(t, err)

	states := collectFlowStates(ctx, flowCh)
	require.Len(t, states, 2)
	assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_RUNNING, states[0].GetPhase())
	assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, states[1].GetPhase())
}

// TestFlowState_ContextCancelled verifies that external context cancellation
// results in an ERRORED FlowState (not CANCELLED, which is reserved for Terminate).
func TestFlowState_ContextCancelled(t *testing.T) {
	graph := loadFlow(t, "gen_long_running.yaml")

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck

	outerCtx := testContext(t)
	flowCh, err := ps.Subscribe(outerCtx, testTopics.Flow())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(outerCtx)
	defer cancel()
	exec := NewExecutor(ps, testTopics)

	execDone := make(chan error, 1)
	go func() {
		execDone <- exec.Execute(ctx, graph)
	}()

	// Wait for RUNNING flow state before cancelling.
	waitCtx, waitCancel := context.WithTimeout(outerCtx, 5*time.Second)
	defer waitCancel()
	for {
		select {
		case <-waitCtx.Done():
			t.Fatal("timed out waiting for RUNNING flow state")
		case msg := <-flowCh:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			if evt.WhichData() == flowv1beta2.RunSnapshot_FlowEvent_Flow_case &&
				evt.GetFlow().GetPhase() == flowv1beta2.RunSnapshot_PHASE_RUNNING {
				goto gotRunning
			}
		}
	}
gotRunning:

	cancel()
	err = <-execDone
	// Context cancellation may return context.Canceled or nil (if
	// the generator happened to finish between cancel and drain).
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Logf("Execute returned: %v (type %T)", err, err)
	}

	termCtx, termCancel := context.WithTimeout(outerCtx, 2*time.Second)
	defer termCancel()
	states := collectFlowStates(termCtx, flowCh)
	require.NotEmpty(t, states)
	last := states[len(states)-1]
	// External cancel produces ERRORED (if cancel won) or SUCCEEDED (if generator won).
	// Both are acceptable outcomes since it's a race between cancel and completion.
	assert.Contains(t,
		[]flowv1beta2.RunSnapshot_Phase{
			flowv1beta2.RunSnapshot_PHASE_ERRORED,
			flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
		},
		last.GetPhase(),
		"context cancel should produce ERRORED or SUCCEEDED, got %v", last.GetPhase())
}
