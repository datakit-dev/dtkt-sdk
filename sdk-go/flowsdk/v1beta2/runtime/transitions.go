package runtime

import flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"

// nodeEvent is an operator-driven lifecycle event.
type nodeEvent int

const (
	eventStop nodeEvent = iota
	eventTerminate
	eventSuspend
	eventResume
)

// String returns a debug-friendly name for an event.
func (e nodeEvent) String() string {
	switch e {
	case eventStop:
		return "stop"
	case eventTerminate:
		return "terminate"
	case eventSuspend:
		return "suspend"
	case eventResume:
		return "resume"
	}
	return "unknown"
}

// nodeTransitions is the canonical (current_phase, event) -> next_phase
// map. Missing entries are intentional no-ops -- the executor must treat
// any (phase, event) pair without an entry as "ignore, do not mutate state,
// do not panic".
//
// The table reflects the validity rules in the lifecycle eval pipeline
// (see docs/flowsdk-v1beta2-cleanup-plan.md):
//
//   - TERMINATE always wins; once in PHASE_CANCELLED there is no path back.
//   - STOP and TERMINATE are both valid on SUSPEND (they choose the
//     terminal phase: SUCCEEDED vs CANCELLED).
//   - SUSPEND requires a running iteration; invalid on STOPPING/CANCELLED.
//   - Terminal phases (SUCCEEDED, FAILED, ERRORED, CANCELLED) accept no
//     events: handlers have exited, the per-execution maps no longer hold
//     anything actionable.
var nodeTransitions = map[transitionKey]flowv1beta2.RunSnapshot_Phase{
	// PENDING: handler hasn't started its first iteration yet, but is
	// already wired up. recv()'s priority check observes events at the
	// next safe point.
	{flowv1beta2.RunSnapshot_PHASE_PENDING, eventStop}:      flowv1beta2.RunSnapshot_PHASE_STOPPING,
	{flowv1beta2.RunSnapshot_PHASE_PENDING, eventTerminate}: flowv1beta2.RunSnapshot_PHASE_CANCELLED,
	{flowv1beta2.RunSnapshot_PHASE_PENDING, eventSuspend}:   flowv1beta2.RunSnapshot_PHASE_SUSPENDED,
	// PENDING + Resume is invalid (nothing to resume).

	// RUNNING: normal operating state. All three lifecycle verbs are valid.
	{flowv1beta2.RunSnapshot_PHASE_RUNNING, eventStop}:      flowv1beta2.RunSnapshot_PHASE_STOPPING,
	{flowv1beta2.RunSnapshot_PHASE_RUNNING, eventTerminate}: flowv1beta2.RunSnapshot_PHASE_CANCELLED,
	{flowv1beta2.RunSnapshot_PHASE_RUNNING, eventSuspend}:   flowv1beta2.RunSnapshot_PHASE_SUSPENDED,
	// RUNNING + Resume is invalid (not suspended).

	// STOPPING: handler is on its way out via input EOF cascade or stopCh.
	// TERMINATE can promote to a hard cancel; SUSPEND/STOP/RESUME are
	// invalid.
	{flowv1beta2.RunSnapshot_PHASE_STOPPING, eventTerminate}: flowv1beta2.RunSnapshot_PHASE_CANCELLED,

	// SUSPENDED: handler is parked in waitForResume.
	// - Stop unparks it and exits with SUCCEEDED.
	// - Terminate cancels ctx and exits with CANCELLED.
	// - Resume continues the loop (next phase = RUNNING).
	// - Suspend is idempotent (no transition).
	{flowv1beta2.RunSnapshot_PHASE_SUSPENDED, eventStop}:      flowv1beta2.RunSnapshot_PHASE_STOPPING,
	{flowv1beta2.RunSnapshot_PHASE_SUSPENDED, eventTerminate}: flowv1beta2.RunSnapshot_PHASE_CANCELLED,
	{flowv1beta2.RunSnapshot_PHASE_SUSPENDED, eventResume}:    flowv1beta2.RunSnapshot_PHASE_RUNNING,
	// SUSPENDED + Suspend is idempotent (no entry; treated as no-op).

	// Terminal phases reject all events. No entries.
	// PHASE_SUCCEEDED, PHASE_FAILED, PHASE_ERRORED, PHASE_CANCELLED.
}

type transitionKey struct {
	from  flowv1beta2.RunSnapshot_Phase
	event nodeEvent
}

// validateNodeTransition returns the next phase for a (current, event)
// pair, and a bool indicating whether the transition is valid. Invalid
// transitions return (current, false) so callers can no-op without
// disturbing state.
func validateNodeTransition(current flowv1beta2.RunSnapshot_Phase, event nodeEvent) (flowv1beta2.RunSnapshot_Phase, bool) {
	next, ok := nodeTransitions[transitionKey{current, event}]
	if !ok {
		return current, false
	}
	return next, true
}
