package runtime

import (
	"testing"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
)

// Pure-data unit tests for the transition table. Each row is one entry
// from nodeTransitions; missing combinations are explicitly asserted as
// no-ops via the !ok return.

func TestValidateNodeTransition_ValidEntries(t *testing.T) {
	parallelByDefault(t)
	cases := []struct {
		name    string
		current flowv1beta2.RunSnapshot_Phase
		event   nodeEvent
		want    flowv1beta2.RunSnapshot_Phase
	}{
		// PENDING
		{"pending+stop", flowv1beta2.RunSnapshot_PHASE_PENDING, eventStop, flowv1beta2.RunSnapshot_PHASE_STOPPING},
		{"pending+terminate", flowv1beta2.RunSnapshot_PHASE_PENDING, eventTerminate, flowv1beta2.RunSnapshot_PHASE_CANCELLED},
		{"pending+suspend", flowv1beta2.RunSnapshot_PHASE_PENDING, eventSuspend, flowv1beta2.RunSnapshot_PHASE_SUSPENDED},

		// RUNNING
		{"running+stop", flowv1beta2.RunSnapshot_PHASE_RUNNING, eventStop, flowv1beta2.RunSnapshot_PHASE_STOPPING},
		{"running+terminate", flowv1beta2.RunSnapshot_PHASE_RUNNING, eventTerminate, flowv1beta2.RunSnapshot_PHASE_CANCELLED},
		{"running+suspend", flowv1beta2.RunSnapshot_PHASE_RUNNING, eventSuspend, flowv1beta2.RunSnapshot_PHASE_SUSPENDED},

		// STOPPING (only Terminate promotes)
		{"stopping+terminate", flowv1beta2.RunSnapshot_PHASE_STOPPING, eventTerminate, flowv1beta2.RunSnapshot_PHASE_CANCELLED},

		// SUSPENDED
		{"suspended+stop", flowv1beta2.RunSnapshot_PHASE_SUSPENDED, eventStop, flowv1beta2.RunSnapshot_PHASE_STOPPING},
		{"suspended+terminate", flowv1beta2.RunSnapshot_PHASE_SUSPENDED, eventTerminate, flowv1beta2.RunSnapshot_PHASE_CANCELLED},
		{"suspended+resume", flowv1beta2.RunSnapshot_PHASE_SUSPENDED, eventResume, flowv1beta2.RunSnapshot_PHASE_RUNNING},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			parallelByDefault(t)
			got, ok := validateNodeTransition(tc.current, tc.event)
			assert.True(t, ok, "expected (%v, %v) to be a valid transition", tc.current, tc.event)
			assert.Equal(t, tc.want, got, "transition (%v, %v) -> %v", tc.current, tc.event, tc.want)
		})
	}
}

// Invalid transitions return (current, false). The runtime relies on this
// to no-op safely without mutating state.
func TestValidateNodeTransition_InvalidEntries(t *testing.T) {
	parallelByDefault(t)
	cases := []struct {
		name    string
		current flowv1beta2.RunSnapshot_Phase
		event   nodeEvent
	}{
		// PENDING + Resume (nothing to resume)
		{"pending+resume", flowv1beta2.RunSnapshot_PHASE_PENDING, eventResume},

		// RUNNING + Resume (not suspended)
		{"running+resume", flowv1beta2.RunSnapshot_PHASE_RUNNING, eventResume},

		// STOPPING: only Terminate is valid
		{"stopping+stop", flowv1beta2.RunSnapshot_PHASE_STOPPING, eventStop},
		{"stopping+suspend", flowv1beta2.RunSnapshot_PHASE_STOPPING, eventSuspend},
		{"stopping+resume", flowv1beta2.RunSnapshot_PHASE_STOPPING, eventResume},

		// SUSPENDED + Suspend is idempotent (no entry; treated as no-op)
		{"suspended+suspend", flowv1beta2.RunSnapshot_PHASE_SUSPENDED, eventSuspend},

		// Terminal phases: no events valid.
		{"succeeded+stop", flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, eventStop},
		{"succeeded+terminate", flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, eventTerminate},
		{"succeeded+suspend", flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, eventSuspend},
		{"succeeded+resume", flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, eventResume},
		{"cancelled+stop", flowv1beta2.RunSnapshot_PHASE_CANCELLED, eventStop},
		{"cancelled+terminate", flowv1beta2.RunSnapshot_PHASE_CANCELLED, eventTerminate},
		{"cancelled+suspend", flowv1beta2.RunSnapshot_PHASE_CANCELLED, eventSuspend},
		{"cancelled+resume", flowv1beta2.RunSnapshot_PHASE_CANCELLED, eventResume},
		{"errored+stop", flowv1beta2.RunSnapshot_PHASE_ERRORED, eventStop},
		{"errored+terminate", flowv1beta2.RunSnapshot_PHASE_ERRORED, eventTerminate},
		{"errored+suspend", flowv1beta2.RunSnapshot_PHASE_ERRORED, eventSuspend},
		{"errored+resume", flowv1beta2.RunSnapshot_PHASE_ERRORED, eventResume},
		{"failed+stop", flowv1beta2.RunSnapshot_PHASE_FAILED, eventStop},
		{"failed+terminate", flowv1beta2.RunSnapshot_PHASE_FAILED, eventTerminate},
		{"failed+suspend", flowv1beta2.RunSnapshot_PHASE_FAILED, eventSuspend},
		{"failed+resume", flowv1beta2.RunSnapshot_PHASE_FAILED, eventResume},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			parallelByDefault(t)
			got, ok := validateNodeTransition(tc.current, tc.event)
			assert.False(t, ok, "expected (%v, %v) to be invalid", tc.current, tc.event)
			assert.Equal(t, tc.current, got, "invalid transition must return current phase unchanged")
		})
	}
}

// Invariant: every entry in the transition table has a non-NONE next
// phase. A NONE entry would indicate a bug (transitions don't go to
// "no phase").
func TestNodeTransitions_NoNonePhaseTargets(t *testing.T) {
	parallelByDefault(t)
	for k, v := range nodeTransitions {
		assert.NotEqual(t, flowv1beta2.RunSnapshot_PHASE_UNSPECIFIED, v,
			"transition %v -> PHASE_UNSPECIFIED is not a valid target", k)
	}
}
