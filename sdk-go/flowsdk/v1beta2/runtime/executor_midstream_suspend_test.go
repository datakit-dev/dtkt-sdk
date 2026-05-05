package runtime

import (
	"testing"
	"time"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mid-stream suspend tests: a generator emits values over time; a
// suspend_when fires partway through; some values are processed before
// the suspend lands, the rest stay queued; resume drains the remainder.
//
// These complement the "feed-all-inputs-then-suspend" suspend tests
// (TestFlowControl_*_SuspendWhen / TestNodeControl_*_SuspendWhen) by
// exercising the realistic case suspend was designed for: pausing a
// long-running flow while more work is still pending.

// FC.suspend_when fires mid-stream on a range generator. Verifies:
//  1. Some values arrive at the output before suspend lands.
//  2. The flow reaches PHASE_SUSPENDED.
//  3. After resume, the remaining values arrive.
//  4. The flow exits cleanly when the generator drains.
func TestFlowControl_Var_SuspendWhen_MidStream(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "fc_var_suspend_midstream.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		// Suspend lands when generators.seq.value == 3 (third tick). The
		// var publishes 2, 4 before the third evaluation triggers suspend.
		// Some early values are guaranteed; PHASE_SUSPENDED is the marker
		// we wait for.
		require.True(t, waitForPhase(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
			"expected var to reach PHASE_SUSPENDED mid-stream")

		// Behavioral verification: while suspended, no new NODE_OUTPUT
		// arrives even though the generator is still ticking (the var
		// handler is paused via the suspendable mixin).
		assertNoOutputDuring(t, varCh, 200*time.Millisecond)

		// Resume the flow. The generator continues and the var processes
		// the rest. Range generator emits 1..10, so we expect remaining
		// values 4..10 (doubled to 8..20) to drain after resume.
		exec.Resume()

		// Wait for the natural completion. Range exhausts at 10; var sees
		// EOF and exits SUCCEEDED.
		require.True(t, waitForPhase(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED),
			"expected var to reach PHASE_SUCCEEDED after resume drains the generator")

		err = requireExecuteReturnsBy(t, done, 1*time.Second)
		require.NoError(t, err, "Execute should return naturally after generator drains")

		// Output should have all 10 doubled values; suspend just delays
		// them, doesn't drop any.
		results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
		assert.Equal(t, []int64{2, 4, 6, 8, 10, 12, 14, 16, 18, 20}, results,
			"all generator values must be processed; suspend pauses but does not drop")
	})
}

// NC.suspend_when on an Output mid-stream. Verifies the per-node suspend
// scope: only the output is paused; the generator continues firing (its
// values queue in the output's input channel until resume).
//
// We use a single subscription on the output topic for everything (phase
// observation, no-output-during, value collection) -- subscribing twice
// to the same topic would split the events between two channels.
func TestNodeControl_Output_SuspendWhen_MidStream(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		g := loadFlow(t, "nc_output_suspend_midstream.yaml")
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		ctx := testContext(t)
		outCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
		require.NoError(t, err)

		exec := NewExecutor(ps, testTopics, extraOpts...)
		done := make(chan error, 1)
		go func() { done <- exec.Execute(ctx, g) }()

		// Drain the output topic into a slice. Stops on PHASE_SUCCEEDED.
		// Tracks the order: collected values, observation of SUSPENDED,
		// observation of SUCCEEDED. Resume is fired when SUSPENDED is seen.
		var collected []int64
		sawSuspended := false
		valuesBeforeSuspend := -1
		resumeOnce := false
		deadline := time.After(2 * time.Second)
	collect:
		for {
			select {
			case <-deadline:
				t.Fatalf("timed out collecting outputs; collected=%v sawSuspended=%v", collected, sawSuspended)
			case msg := <-outCh:
				evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
				msg.Ack()
				node := runtimeNodeFromEvent(evt)
				if phaseOf(node) == flowv1beta2.RunSnapshot_PHASE_SUSPENDED {
					sawSuspended = true
					valuesBeforeSuspend = len(collected)
					if !resumeOnce {
						resumeOnce = true
						exec.ResumeNode("outputs.result", nil)
					}
					continue
				}
				if phaseOf(node) == flowv1beta2.RunSnapshot_PHASE_SUCCEEDED {
					break collect
				}
				if evt.GetEventType() != flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
					continue
				}
				if isEOFValue(node.GetValue()) {
					continue
				}
				collected = append(collected, node.GetValue().GetInt64Value())
			}
		}

		err = requireExecuteReturnsBy(t, done, 1*time.Second)
		require.NoError(t, err, "Execute should return naturally")

		assert.True(t, sawSuspended, "expected output to reach PHASE_SUSPENDED mid-stream")
		assert.Greater(t, valuesBeforeSuspend, 0,
			"expected at least one value to publish before suspend lands (mid-stream behavior)")
		assert.Less(t, valuesBeforeSuspend, 10,
			"expected suspend to land before all 10 values published (mid-stream)")
		assert.Equal(t, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, collected,
			"all generator values must reach the output; per-node suspend pauses but does not drop")
	})
}
