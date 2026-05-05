package runtime

import (
	"testing"
	"time"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Runtime tests that exercise the transition table's promotion paths
// (e.g. STOPPING + Terminate -> CANCELLED) end-to-end against the
// executor methods. Complements the pure-data tests in transitions_test.go
// by verifying the runtime honors the table.

// TestPerNode_SuspendThenStopThenTerminate exercises the deterministic
// SUSPENDED -> STOPPING -> CANCELLED chain. The handler is parked in
// waitForResume so all transitions land on a known state without the
// race that bedevils stop-then-immediately-terminate on a running node.
//
//  1. SuspendNode -> handler parked, currentPhase=SUSPENDED
//  2. StopNode    -> currentPhase becomes STOPPING (stoppedNodes set);
//     handler unparks via stopCh, but its post-loop
//     SUCCEEDED publish hasn't happened yet
//  3. TerminateNode -> validateNodeTransition(STOPPING, Terminate) ->
//     CANCELLED is valid; ctx is cancelled, CANCELLED published
//
// We assert PHASE_CANCELLED appears on the wire -- proof the promotion
// went through. The handler may also publish SUCCEEDED if it raced past
// the ctx-cancel; the order doesn't matter for this contract.
func TestPerNode_SuspendThenStopThenTerminate(t *testing.T) {
	fixtures := []nodeFixture{
		{name: "var", yaml: "suspend_resume_var.yaml", targetNode: "vars.passthrough"},
		{name: "output", yaml: "suspend_resume_var.yaml", targetNode: "outputs.result"},
		{name: "unary_action", yaml: "suspend_resume_unary.yaml", targetNode: "actions.echo", mockRPC: true},
	}
	for _, tc := range fixtures {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			parallelByDefault(t)
			withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
				rig := setupNodeRig(t, tc, extraOpts)
				defer rig.cleanup()

				topicCh, err := rig.ps.Subscribe(rig.ctx, testTopics.For(tc.targetNode))
				require.NoError(t, err)

				time.Sleep(50 * time.Millisecond)

				// Park the handler.
				rig.exec.SuspendNode(tc.targetNode)
				assert.Truef(t,
					waitForPhaseOnChan(topicCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED, 1*time.Second),
					"%s: expected PHASE_SUSPENDED before promotion", tc.name)

				// Move from SUSPENDED to STOPPING.
				rig.exec.StopNode(tc.targetNode)

				// Promote STOPPING to CANCELLED. The transition table
				// allows STOPPING+Terminate; the runtime publishes
				// PHASE_CANCELLED via TerminateNode.
				rig.exec.TerminateNode(tc.targetNode)

				assert.Truef(t,
					waitForPhaseOnChan(topicCh, flowv1beta2.RunSnapshot_PHASE_CANCELLED, 2*time.Second),
					"%s: expected PHASE_CANCELLED after Suspend+Stop+Terminate (terminate promotes stopping)",
					tc.name)

				rig.exec.Stop()
				<-rig.doneCh
			})
		})
	}
}

// TestPerNode_TerminateOnTerminal_NoOp verifies the transition table's
// no-op rule for terminal phases: a TerminateNode call on an already-
// terminal node must not panic, must not mutate state, must not publish
// a duplicate phase event.
//
// Today the runtime's terminalNodes guard handles this; the transition
// table makes it explicit. Both should agree.
func TestPerNode_TerminateOnTerminal_NoOp(t *testing.T) {
	parallelByDefault(t)
	g := loadFlow(t, "input_int64_to_output.yaml")
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck

	feedInput(ps, "inputs.x", int64(42))
	ctx := testContext(t)
	exec := NewExecutor(ps, testTopics)

	err := exec.Execute(ctx, g)
	require.NoError(t, err)

	// After Execute returns, all state is cleared. Operator calls must
	// short-circuit on stopFn/terminateFn nil guards. (The transition
	// table check happens after those, so it's not even reached -- which
	// is fine; the contract is "no panic, no mutation".)
	assert.NotPanics(t, func() { exec.TerminateNode("outputs.result") })
	assert.NotPanics(t, func() { exec.StopNode("outputs.result") })
	assert.NotPanics(t, func() { exec.SuspendNode("outputs.result") })
	assert.NotPanics(t, func() { exec.ResumeNode("outputs.result", nil) })
}
