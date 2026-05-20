package runtime

import (
	"testing"
	"time"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Step N: state-machine matrix coverage.
//
// Operator events (StopNode, TerminateNode, SuspendNode, ResumeNode) on
// nodes in terminal phases (SUCCEEDED, ERRORED, FAILED, CANCELLED) must
// be clean no-ops -- no panic, no stale flags. The runtime tracks
// terminalNodes per execution; once a handler exits, all subsequent
// operator commands targeting that node short-circuit.
//
// Pre-fix bug: SuspendNode on a SUCCEEDED node set suspendedNodes[id]=true
// with no handler ever to clear it, leaving stale state visible to
// observers (and breaking the suspendedNodes-set invariants).
//
// Each test runs a flow to completion, then fires the operator event on
// the now-terminal node and asserts:
//   1. exec.suspendedNodes is empty (no stale flags)
//   2. The flow exits without error or hang.

// runFlowToCompletion runs the simplest possible flow (input → output)
// and returns the executor after it has fully drained, ready for terminal-
// phase operator-event probing.
func runFlowToCompletion(t *testing.T, fixtureName, inputID string, value any) *Executor {
	t.Helper()
	g := loadFlow(t, fixtureName)
	ps := newPubSub()
	t.Cleanup(func() { _ = ps.Close() })

	feedInput(ps, inputID, value)
	ctx := testContext(t)
	exec := NewExecutor(ps, testTopics)
	err := exec.Execute(ctx, g)
	require.NoError(t, err)
	return exec
}

// TestStateMachine_OperatorEventsOnTerminal_NoStale: after a flow runs to
// completion, every operator event on a now-terminal node must leave
// suspendedNodes empty. Validates the terminalNodes guard in
// Stop/Terminate/Suspend/SuspendNode/ResumeNode.
//
// Note: by the time Execute returns, clearRunState has run via deferred
// cleanup, so all per-execution state is nil. The realistic scenario is
// firing operator events DURING the run (before Execute returns) on a
// node that has just exited. The TestStateMachine_*_DuringRun tests
// below cover that path.
func TestStateMachine_OperatorEventsAfterExecute_AreNoOp(t *testing.T) {
	parallelByDefault(t)
	exec := runFlowToCompletion(t, "input_int64_to_output.yaml", "inputs.x", int64(42))

	// Per-execution state is cleared after Execute returns. Operator events
	// must short-circuit cleanly (early-return on stopFn/terminateFn nil
	// guards). Asserting NoPanic.
	assert.NotPanics(t, func() { exec.StopNode("outputs.result") })
	assert.NotPanics(t, func() { exec.TerminateNode("outputs.result") })
	assert.NotPanics(t, func() { exec.SuspendNode("outputs.result") })
	assert.NotPanics(t, func() { exec.ResumeNode("outputs.result", nil) })
	assert.NotPanics(t, func() { exec.Stop() })
	assert.NotPanics(t, func() { exec.Terminate() })
	assert.NotPanics(t, func() { exec.Suspend() })
	assert.NotPanics(t, func() { exec.Resume() })
}

// TestStateMachine_SuspendNode_OnTerminal_DuringRun: a single-node flow
// finishes its work; while Execute is still draining, fire SuspendNode
// on the now-terminal node and assert it produces no stale state.
//
// Implementation: use a flow with one input and one output. After the
// output handler returns (terminal), launchHandlers' deferred handler
// marks terminalNodes[id]=true. We probe by spawning a goroutine that
// races SuspendNode against execute completion. Behavioral assertions:
//
//  1. Execute returns nil (the spam probe doesn't break the flow).
//  2. The output value 42 reaches outputs.result (proving the flow
//     completed normally despite spam).
//  3. After Execute returns, exec.suspendedNodes is empty -- proves
//     either the SuspendNode happened-before-terminal path cleared
//     the flag on handler exit, OR the terminalNodes guard rejected
//     the post-terminal SuspendNode without writing the flag.
func TestStateMachine_SuspendNode_OnTerminal_DuringRun(t *testing.T) {
	parallelByDefault(t)
	g := loadFlow(t, "input_int64_to_output.yaml")
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	feedInput(ps, "inputs.x", int64(42))
	ctx := testContext(t)
	exec := NewExecutor(ps, testTopics)

	// Spam SuspendNode in a tight loop while the flow runs. Once the
	// node terminates, terminalNodes guard kicks in and SuspendNode is
	// a no-op.
	stopProbe := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopProbe:
				return
			default:
				exec.SuspendNode("outputs.result")
				time.Sleep(time.Millisecond)
			}
		}
	}()

	err := exec.Execute(ctx, g)
	close(stopProbe)
	require.NoError(t, err)

	// Output value reached outputs.result (flow completed despite spam).
	results := outputInt64s(collectOutputs(ctx, ps, "outputs.result"))
	assert.Equal(t, []int64{42}, results,
		"flow must complete despite SuspendNode spam; got %v", results)

	// After Execute returns, clearRunState nilled the maps. Verify the
	// state-clearing path actually ran and no leftover flags persist.
	exec.mu.Lock()
	suspendedLen := len(exec.suspendedNodes)
	exec.mu.Unlock()
	assert.Equal(t, 0, suspendedLen,
		"suspendedNodes must be empty/nil after Execute returns; had %d", suspendedLen)
}

// TestStateMachine_ResumeNode_OnNeverSuspended_IsNoOp: ResumeNode targeting
// a node that was never suspended must be a clean no-op (regression guard:
// pre-fix SuspendNode on already-terminal node could leave a flag that a
// stray ResumeNode would then attempt to clear).
func TestStateMachine_ResumeNode_OnNeverSuspended_IsNoOp(t *testing.T) {
	parallelByDefault(t)
	g := loadFlow(t, "input_int64_to_output.yaml")
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	feedInput(ps, "inputs.x", int64(42))
	ctx := testContext(t)
	exec := NewExecutor(ps, testTopics)
	done := make(chan error, 1)
	go func() { done <- exec.Execute(ctx, g) }()

	// ResumeNode on a never-suspended node mid-flight: must early-return
	// via suspendedNodes check, no panic, no spurious wake-up.
	assert.NotPanics(t, func() { exec.ResumeNode("outputs.result", nil) })

	err := requireExecuteReturnsBy(t, done, 500*time.Millisecond)
	require.NoError(t, err, "flow should complete normally")
}

// TestStateMachine_StopNode_OnUnknownNode_IsNoOp: operator event with an
// unknown node ID must early-return cleanly.
func TestStateMachine_StopNode_OnUnknownNode_IsNoOp(t *testing.T) {
	parallelByDefault(t)
	g := loadFlow(t, "input_int64_to_output.yaml")
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	feedInput(ps, "inputs.x", int64(42))
	ctx := testContext(t)
	exec := NewExecutor(ps, testTopics)
	done := make(chan error, 1)
	go func() { done <- exec.Execute(ctx, g) }()

	assert.NotPanics(t, func() { exec.StopNode("does.not.exist") })
	assert.NotPanics(t, func() { exec.TerminateNode("does.not.exist") })
	assert.NotPanics(t, func() { exec.SuspendNode("does.not.exist") })
	assert.NotPanics(t, func() { exec.ResumeNode("does.not.exist", nil) })

	err := requireExecuteReturnsBy(t, done, 500*time.Millisecond)
	require.NoError(t, err)
}

// TestStateMachine_FlowEventsBeforeExecute_AreNoOp: operator events fired
// before Execute() runs (no per-execution state set yet) must early-return
// via the stopFn/terminateFn nil guards.
func TestStateMachine_FlowEventsBeforeExecute_AreNoOp(t *testing.T) {
	parallelByDefault(t)
	exec := NewExecutor(newPubSub(), testTopics)

	assert.NotPanics(t, func() { exec.Stop() })
	assert.NotPanics(t, func() { exec.Terminate() })
	assert.NotPanics(t, func() { exec.Suspend() })
	assert.NotPanics(t, func() { exec.Resume() })
	assert.NotPanics(t, func() { exec.StopNode("any.node") })
	assert.NotPanics(t, func() { exec.TerminateNode("any.node") })
	assert.NotPanics(t, func() { exec.SuspendNode("any.node") })
	assert.NotPanics(t, func() { exec.ResumeNode("any.node", nil) })
}

// --- PENDING state coverage ---
//
// Before a handler's first Resolve iteration runs, the recv() priority
// check ensures suspendCh / stopCh / ctx.Done preempt input. So an
// operator event fired DURING the initial setup window must be observed
// at iter 1's first recv. The tests below verify this contract -- a node
// that has been registered but not yet processed any messages still
// honors operator commands.

// nc_pending_var_suspend.yaml is loaded but the test never feeds the input,
// so the var stays in PENDING. We then fire SuspendNode externally and
// expect PHASE_SUSPENDED (the var transitions PENDING -> SUSPENDED at its
// first recv attempt).
func TestStateMachine_OperatorSuspend_OnPending(t *testing.T) {
	parallelByDefault(t)
	g := loadFlow(t, "nc_var_stop.yaml") // simple input -> var -> output, no NC firing without input
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	ctx := testContext(t)
	varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
	require.NoError(t, err)

	exec := NewExecutor(ps, testTopics)
	done := make(chan error, 1)
	go func() { done <- exec.Execute(ctx, g) }()

	// Don't feed input -- var sits in its first Resolve waiting on inputCh.
	// Allow setup to finish, then suspend.
	time.Sleep(50 * time.Millisecond)
	exec.SuspendNode("vars.doubled")

	require.True(t, waitForPhase(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
		"PENDING + Suspend must transition to SUSPENDED at the next safe recv")

	exec.Terminate()
	assert.ErrorIs(t, <-done, ErrTerminated)
}

// PENDING + Stop: the recv() priority check returns errOperatorStopped
// from the handler's first Resolve, the handler exits cleanly with
// PHASE_SUCCEEDED via post-loop publish.
func TestStateMachine_OperatorStop_OnPending(t *testing.T) {
	parallelByDefault(t)
	g := loadFlow(t, "nc_var_stop.yaml")
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	ctx := testContext(t)
	varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
	require.NoError(t, err)

	exec := NewExecutor(ps, testTopics)
	done := make(chan error, 1)
	go func() { done <- exec.Execute(ctx, g) }()

	time.Sleep(50 * time.Millisecond)
	exec.StopNode("vars.doubled")

	require.True(t, waitForPhase(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED),
		"PENDING + Stop must transition to SUCCEEDED via graceful exit")

	exec.Stop()
	require.NoError(t, <-done, "flow should drain and exit cleanly")
}

// PENDING + Terminate: nodeCtxs[id] is cancelled; the handler's first
// Resolve sees ctx.Done immediately (priority check) and exits with
// ctx.Err. TerminateNode publishes PHASE_CANCELLED state event.
func TestStateMachine_OperatorTerminate_OnPending(t *testing.T) {
	parallelByDefault(t)
	g := loadFlow(t, "nc_var_stop.yaml")
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	ctx := testContext(t)
	varCh, err := ps.Subscribe(ctx, testTopics.For("vars.doubled"))
	require.NoError(t, err)

	exec := NewExecutor(ps, testTopics)
	done := make(chan error, 1)
	go func() { done <- exec.Execute(ctx, g) }()

	time.Sleep(50 * time.Millisecond)
	exec.TerminateNode("vars.doubled")

	require.True(t, waitForPhase(ctx, varCh, flowv1beta2.RunSnapshot_PHASE_CANCELLED),
		"PENDING + Terminate must publish PHASE_CANCELLED state event")

	exec.Terminate()
	assert.ErrorIs(t, <-done, ErrTerminated)
}

// TestStateMachine_TerminalNodesFlag_SetOnHandlerExit: verifies the
// terminalNodes guard actually short-circuits operator events on exited
// nodes. Strategy:
//  1. Use a flow with two nodes: one that exits quickly (output for x=42)
//     and one that holds open (a long-running ticker). This keeps Execute
//     running so we can probe per-execution state.
//  2. After the output exits, fire SuspendNode on it. The terminalNodes
//     guard must reject it: suspendedNodes["outputs.result"] must NOT
//     be set even momentarily.
//  3. Tear down with Terminate, verify Execute returns ErrTerminated.
func TestStateMachine_TerminalNodesFlag_SetOnHandlerExit(t *testing.T) {
	parallelByDefault(t)
	g := loadFlow(t, "state_machine_terminal_probe.yaml")
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	feedInput(ps, "inputs.x", int64(42))
	ctx := testContext(t)
	outCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
	require.NoError(t, err)

	exec := NewExecutor(ps, testTopics)
	done := make(chan error, 1)
	go func() { done <- exec.Execute(ctx, g) }()

	// Wait for outputs.result to reach SUCCEEDED (the input-driven path
	// exits while the ticker keeps Execute alive). At that point the
	// output's terminalNodes flag is set.
	require.True(t, waitForPhase(ctx, outCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED),
		"output node must reach SUCCEEDED before we probe SuspendNode")

	// Fire SuspendNode on the now-terminal node. terminalNodes guard
	// must reject it -- no entry in suspendedNodes.
	exec.SuspendNode("outputs.result")
	exec.mu.Lock()
	_, isSuspended := exec.suspendedNodes["outputs.result"]
	exec.mu.Unlock()
	assert.False(t, isSuspended,
		"terminalNodes guard must reject SuspendNode on exited node; suspendedNodes had outputs.result")

	// Cleanup: terminate to unblock the long-running ticker.
	exec.Terminate()
	err = requireExecuteReturnsBy(t, done, 1*time.Second)
	assert.ErrorIs(t, err, ErrTerminated)
}
