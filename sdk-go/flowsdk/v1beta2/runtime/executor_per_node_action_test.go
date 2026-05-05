package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// Per-node action coverage matrix.
//
// For every handler type that can be a target of SuspendNode / StopNode /
// TerminateNode, this file runs three groups of tests:
//
//   1. SuspendNode + ResumeNode + isolation:
//      - target node is suspended → its phase event is published
//      - bookkeeping: ONLY the target is in suspendedNodes
//      - witness topic upstream of target keeps flowing during pause
//        (proves downstream nodes are starved, not directly suspended)
//      - resume restores normal operation
//
//   2. StopNode:
//      - target node receives the stop signal
//      - target transitions to PHASE_SUCCEEDED (graceful stop)
//      - the rest of the flow drains naturally
//
//   3. TerminateNode:
//      - target node receives the terminate signal
//      - target transitions to PHASE_CANCELLED
//      - flow exits cleanly
//
// Each test runs in both direct and outbox pubsub modes via withAndWithoutOutbox.

// nodeFixture describes one handler-type test scenario.
type nodeFixture struct {
	name                 string // table-driven test name
	yaml                 string // testdata fixture
	targetNode           string // node to act on (e.g. "vars.passthrough")
	witnessTopic         string // upstream topic to watch during suspend
	witnessExpectsValues bool   // true: should keep flowing; false: should stop
	mockRPC              bool   // include mockRPCOptions() for action/stream nodes
	interactionResponder bool   // attach an auto-responder for interaction nodes
}

// nodeFixtures is the matrix. Add new handler types here when supported.
var nodeFixtures = []nodeFixture{
	{
		name: "var", yaml: "suspend_resume_var.yaml",
		targetNode: "vars.passthrough", witnessTopic: "generators.seq",
		witnessExpectsValues: true,
	},
	{
		name: "switch", yaml: "suspend_resume_switch.yaml",
		targetNode: "vars.identity", witnessTopic: "generators.seq",
		witnessExpectsValues: true,
	},
	{
		name: "output", yaml: "suspend_resume_var.yaml",
		targetNode: "outputs.result", witnessTopic: "vars.passthrough",
		witnessExpectsValues: true,
	},
	{
		name: "unary_action", yaml: "suspend_resume_unary.yaml",
		targetNode: "actions.echo", witnessTopic: "generators.seq",
		witnessExpectsValues: true, mockRPC: true,
	},
	{
		name: "ticker_generator", yaml: "suspend_resume_ticker.yaml",
		targetNode: "generators.tick", witnessTopic: "generators.tick",
		witnessExpectsValues: false, // self: source quiets when suspended
	},
	{
		name: "range_generator", yaml: "suspend_resume_var.yaml",
		targetNode: "generators.seq", witnessTopic: "generators.seq",
		witnessExpectsValues: false,
	},
	{
		name: "bidi_stream", yaml: "suspend_resume_bidi_stream.yaml",
		targetNode: "streams.echo", witnessTopic: "generators.seq",
		witnessExpectsValues: true, mockRPC: true,
	},
	{
		name: "server_stream", yaml: "suspend_resume_server_stream.yaml",
		targetNode: "streams.numbers", witnessTopic: "generators.seq",
		witnessExpectsValues: true, mockRPC: true,
	},
	{
		name: "client_stream", yaml: "suspend_resume_client_stream.yaml",
		targetNode: "streams.collect", witnessTopic: "generators.seq",
		witnessExpectsValues: true, mockRPC: true,
	},
	{
		name: "interaction", yaml: "suspend_resume_interaction.yaml",
		targetNode: "interactions.confirm", witnessTopic: "generators.seq",
		witnessExpectsValues: true, interactionResponder: true,
	},
}

// nodeTestRig is the shared per-test scaffolding: set up the flow,
// optionally wire up mocks/responders, expose the executor, the pubsub,
// the done channel, and a cleanup func.
type nodeTestRig struct {
	t             *testing.T
	exec          *Executor
	ps            executor.PubSub
	ctx           context.Context
	doneCh        chan error
	cancel        func() // signals interaction responder to exit (no-op otherwise)
	responderDone chan struct{}
	promptCh      chan *flowv1beta2.InteractionRequestEvent
}

// setupNodeRig builds the rig for a given fixture and option set. Caller
// MUST defer rig.cleanup().
func setupNodeRig(t *testing.T, tc nodeFixture, extraOpts []Option) *nodeTestRig {
	t.Helper()
	graph := loadFlow(t, tc.yaml)
	ps := newPubSub()

	opts := append([]Option(nil), extraOpts...)
	if tc.mockRPC {
		opts = append(opts, mockRPCOptions()...)
	}

	rig := &nodeTestRig{t: t, ps: ps}
	if tc.interactionResponder {
		rig.promptCh = make(chan *flowv1beta2.InteractionRequestEvent, 64)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 64)
		rig.responderDone = make(chan struct{})
		opts = append(opts, WithInteractions(rig.promptCh, response))
		go func() {
			defer close(rig.responderDone)
			var counter int64
			for req := range rig.promptCh {
				counter++
				anyVal, _ := common.WrapProtoAny(counter)
				select {
				case response <- &flowv1beta2.InteractionResponseEvent{
					Id:    req.GetId(),
					Token: req.GetToken(),
					Value: anyVal,
				}:
				case <-time.After(2 * time.Second):
					return
				}
			}
		}()
	}

	rig.exec = NewExecutor(ps, testTopics, opts...)
	rig.ctx = testContext(t)
	rig.doneCh = make(chan error, 1)
	go func() {
		rig.doneCh <- rig.exec.Execute(rig.ctx, graph)
	}()
	return rig
}

// cleanup tears down: closes pubsub, signals interaction responder.
func (r *nodeTestRig) cleanup() {
	_ = r.ps.Close()
	if r.promptCh != nil {
		close(r.promptCh)
		<-r.responderDone
	}
}

// suspendedNodeSet returns a snapshot of the executor's suspendedNodes
// map. Used to assert exactly which nodes are paused.
func (r *nodeTestRig) suspendedNodeSet() map[string]bool {
	r.exec.mu.Lock()
	defer r.exec.mu.Unlock()
	out := make(map[string]bool, len(r.exec.suspendedNodes))
	for id := range r.exec.suspendedNodes {
		out[id] = true
	}
	return out
}

// countNodeOutputsOver subscribes to a topic and returns the count of
// EVENT_TYPE_NODE_OUTPUT messages received within the given window.
// Used as the witness assertion: "this topic produced N values during
// the suspend window."
func countNodeOutputsOver(t *testing.T, ps executor.PubSub, ctx context.Context, topic string, window time.Duration) int {
	t.Helper()
	ch, err := ps.Subscribe(ctx, testTopics.For(topic))
	require.NoError(t, err)
	count := 0
	deadline := time.After(window)
	for {
		select {
		case msg := <-ch:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			if evt.GetEventType() == flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
				count++
			}
		case <-deadline:
			return count
		}
	}
}

// waitForPhaseOnChan reads from a pre-subscribed channel and waits
// until an event with the given phase arrives. Used to verify
// PHASE_SUSPENDED / PHASE_SUCCEEDED / PHASE_CANCELLED transitions.
// The caller MUST subscribe BEFORE the action that publishes the
// phase event, since direct (non-persistent) pubsub drops messages
// with no subscriber.
func waitForPhaseOnChan(ch <-chan *pubsub.Message, phase flowv1beta2.RunSnapshot_Phase, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for {
		select {
		case msg := <-ch:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			node := runtimeNodeFromEvent(evt)
			if phaseOf(node) == phase {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

// -- Test 1: SuspendNode + isolation -----------------------------------------

// TestPerNode_SuspendNode_Isolation runs SuspendNode on each handler type and
// verifies the unified-suspend contract:
//  1. PHASE_SUSPENDED is published on the target node's topic.
//  2. Only the target node is in suspendedNodes.
//  3. The witness topic continues to receive values during the pause
//     (or doesn't, for the "self == source" cases).
//  4. ResumeNode unblocks normal operation; flow stops cleanly.
func TestPerNode_SuspendNode_Isolation(t *testing.T) {
	for _, tc := range nodeFixtures {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			parallelByDefault(t)
			withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
				rig := setupNodeRig(t, tc, extraOpts)
				defer rig.cleanup()

				// Let the flow warm up so the target is actively running.
				time.Sleep(50 * time.Millisecond)

				// Subscribe to witness topic BEFORE suspending so we capture
				// post-suspend values cleanly.
				witnessCh, err := rig.ps.Subscribe(rig.ctx, testTopics.For(tc.witnessTopic))
				require.NoError(t, err)

				rig.exec.SuspendNode(tc.targetNode)

				// Bookkeeping invariant: ONLY the target node is in suspendedNodes.
				suspended := rig.suspendedNodeSet()
				require.Truef(t, suspended[tc.targetNode],
					"%s should be in suspendedNodes after SuspendNode", tc.targetNode)
				require.Lenf(t, suspended, 1,
					"%s: only target should be suspended; got %v "+
						"(downstream nodes should be starved, not suspended)",
					tc.name, suspended)

				// Drain witness channel during a 250ms pause window. Count
				// only NODE_OUTPUT events (skipping NODE_UPDATE phase events).
				const pauseWindow = 250 * time.Millisecond
				witnessCount := 0
				deadline := time.After(pauseWindow)
			drain:
				for {
					select {
					case msg := <-witnessCh:
						evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
						msg.Ack()
						if evt.GetEventType() == flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
							witnessCount++
						}
					case <-deadline:
						break drain
					}
				}

				if tc.witnessExpectsValues {
					require.Greaterf(t, witnessCount, 0,
						"%s: witness %s should receive values while %s is suspended; got %d",
						tc.name, tc.witnessTopic, tc.targetNode, witnessCount)
				} else {
					// Suspended source - allow up to 1 in-flight value
					// (outbox relay may have one queued).
					require.LessOrEqualf(t, witnessCount, 1,
						"%s: witness %s (= target) expected ≤1 in-flight values during suspend; got %d",
						tc.name, tc.witnessTopic, witnessCount)
				}

				rig.exec.ResumeNode(tc.targetNode, nil)

				// After resume, suspendedNodes must no longer contain target.
				suspended = rig.suspendedNodeSet()
				require.Falsef(t, suspended[tc.targetNode],
					"%s should be removed from suspendedNodes after ResumeNode", tc.targetNode)

				rig.exec.Stop()
				require.NoError(t, <-rig.doneCh)
			})
		})
	}
}

// -- Test 2: StopNode --------------------------------------------------------

// TestPerNode_StopNode runs StopNode on each handler type and verifies:
//  1. The target node transitions to PHASE_SUCCEEDED on its topic.
//  2. The flow drains and exits without hanging.
//
// StopNode is graceful: in-flight work completes, the rest of the flow
// shuts down naturally as EOFs cascade.
func TestPerNode_StopNode(t *testing.T) {
	for _, tc := range nodeFixtures {
		tc := tc
		// StopNode on a stream/interaction in the middle of an in-flight
		// operation has nuanced semantics that aren't worth over-asserting
		// here. Skip those - they're covered by other targeted tests.
		t.Run(tc.name, func(t *testing.T) {
			parallelByDefault(t)
			withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
				rig := setupNodeRig(t, tc, extraOpts)
				defer rig.cleanup()

				// Subscribe to the target's topic BEFORE issuing StopNode.
				// Direct pubsub drops messages with no subscriber; the
				// PHASE_SUCCEEDED event is published synchronously inside
				// StopNode, so we'd miss it without an active subscriber.
				topicCh, err := rig.ps.Subscribe(rig.ctx, testTopics.For(tc.targetNode))
				require.NoError(t, err)

				time.Sleep(50 * time.Millisecond)
				rig.exec.StopNode(tc.targetNode)

				assert.Truef(t,
					waitForPhaseOnChan(topicCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, 5*time.Second),
					"%s: expected PHASE_SUCCEEDED on %s after StopNode",
					tc.name, tc.targetNode)

				rig.exec.Stop()
				require.NoError(t, <-rig.doneCh)
			})
		})
	}
}

// -- Test 3: TerminateNode ---------------------------------------------------

// TestPerNode_TerminateNode runs TerminateNode on each handler type and
// verifies:
//  1. The target node transitions to PHASE_CANCELLED on its topic.
//  2. The flow exits without hanging (Terminate may or may not
//     propagate to a flow-level Cancel depending on the strategy;
//     we just verify the target node is cancelled and the flow
//     exits via Stop or naturally).
func TestPerNode_TerminateNode(t *testing.T) {
	for _, tc := range nodeFixtures {
		tc := tc
		// TerminateNode on streams/interactions has the same caveat as
		// StopNode (in-flight semantics); skip for now.
		t.Run(tc.name, func(t *testing.T) {
			parallelByDefault(t)
			withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
				rig := setupNodeRig(t, tc, extraOpts)
				defer rig.cleanup()

				// Subscribe BEFORE the action - see StopNode test rationale.
				topicCh, err := rig.ps.Subscribe(rig.ctx, testTopics.For(tc.targetNode))
				require.NoError(t, err)

				time.Sleep(50 * time.Millisecond)
				rig.exec.TerminateNode(tc.targetNode)

				assert.Truef(t,
					waitForPhaseOnChan(topicCh, flowv1beta2.RunSnapshot_PHASE_CANCELLED, 5*time.Second),
					"%s: expected PHASE_CANCELLED on %s after TerminateNode",
					tc.name, tc.targetNode)

				rig.exec.Stop()
				// Don't require NoError: TerminateNode may yield ErrTerminated
				// at the flow level depending on the error strategy.
				<-rig.doneCh
			})
		})
	}
}

// -- Test 4: combined action sequences --------------------------------------

// TestPerNode_SuspendThenStop verifies that a suspended node can still
// be stopped - the SuspendNode followed by StopNode sequence resolves
// without deadlock and the target node ends in PHASE_SUCCEEDED.
func TestPerNode_SuspendThenStop(t *testing.T) {
	for _, tc := range nodeFixtures {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			parallelByDefault(t)
			withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
				rig := setupNodeRig(t, tc, extraOpts)
				defer rig.cleanup()

				topicCh, err := rig.ps.Subscribe(rig.ctx, testTopics.For(tc.targetNode))
				require.NoError(t, err)

				time.Sleep(50 * time.Millisecond)

				rig.exec.SuspendNode(tc.targetNode)
				time.Sleep(50 * time.Millisecond)
				rig.exec.StopNode(tc.targetNode)

				assert.Truef(t,
					waitForPhaseOnChan(topicCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, 5*time.Second),
					"%s: expected PHASE_SUCCEEDED on %s after Suspend+StopNode",
					tc.name, tc.targetNode)

				rig.exec.Stop()
				<-rig.doneCh
			})
		})
	}
}

// TestPerNode_SuspendThenTerminate verifies that a suspended node can
// be terminated - the SuspendNode followed by TerminateNode sequence
// resolves cleanly and the target node ends in PHASE_CANCELLED.
func TestPerNode_SuspendThenTerminate(t *testing.T) {
	for _, tc := range nodeFixtures {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			parallelByDefault(t)
			withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
				rig := setupNodeRig(t, tc, extraOpts)
				defer rig.cleanup()

				topicCh, err := rig.ps.Subscribe(rig.ctx, testTopics.For(tc.targetNode))
				require.NoError(t, err)

				time.Sleep(50 * time.Millisecond)

				rig.exec.SuspendNode(tc.targetNode)
				time.Sleep(50 * time.Millisecond)
				rig.exec.TerminateNode(tc.targetNode)

				assert.Truef(t,
					waitForPhaseOnChan(topicCh, flowv1beta2.RunSnapshot_PHASE_CANCELLED, 5*time.Second),
					"%s: expected PHASE_CANCELLED on %s after Suspend+TerminateNode",
					tc.name, tc.targetNode)

				rig.exec.Stop()
				<-rig.doneCh
			})
		})
	}
}
