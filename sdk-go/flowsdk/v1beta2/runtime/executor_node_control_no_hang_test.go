package runtime

import (
	"testing"
	"time"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression guard for the NC.suspend_when -> ResumeNode hang fixed by
// unifying the two suspend/resume mechanisms. Before the fix:
//
//   1. NC.suspend_when fires inside the handler's checkLifecycle -> SuspendNode
//      sets suspendedNodes[id]=true and signals mixin.suspendCh.
//   2. Handler's next Resolve does select { suspendCh | inputCh }. If select
//      randomly picks inputCh (EOF marker present in the single-input case),
//      the handler exits via the EOF path WITHOUT going through pauseUntilResume.
//   3. h.Run returns nil. The launchHandlers wrapper's awaitResume reads the
//      stale suspendedNodes[id]=true flag and parks on opts.resumeChans[id]
//      -- a channel ResumeNode never signaled for selfSuspendable handlers.
//   4. Execute hangs forever (until ctx deadline).
//
// The fix unified the path so retry-strategy SuspendError uses the same
// mixin pauseUntilResume as NC/operator suspend; the wrapper no longer
// parks. The 500ms hard cap below catches any reintroduction of a wrapper-
// side park or any new path that leaves Execute blocked after PHASE_SUCCEEDED.
//
// Note: the regular TestNodeControl_*_SuspendThenResume tests cover the
// same path but use the 10s testContext deadline; under regression they
// would silently pass in 10s. THIS test fails fast.
func TestNodeControl_SuspendResume_NoHangAfterSuccess(t *testing.T) {
	parallelByDefault(t)
	g := loadFlow(t, "nc_output_suspend.yaml")
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	feedInput(ps, "inputs.x", int64(7))
	ctx := testContext(t)

	outCh, err := ps.Subscribe(ctx, testTopics.For("outputs.result"))
	require.NoError(t, err)

	exec := NewExecutor(ps, testTopics)
	done := make(chan error, 1)
	go func() { done <- exec.Execute(ctx, g) }()

	require.True(t, waitForPhase(ctx, outCh, flowv1beta2.RunSnapshot_PHASE_SUSPENDED),
		"expected NC-driven PHASE_SUSPENDED")
	exec.ResumeNode("outputs.result", nil)
	require.True(t, waitForPhase(ctx, outCh, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED),
		"expected PHASE_SUCCEEDED after resume")

	err = requireExecuteReturnsBy(t, done, 500*time.Millisecond)
	assert.NoError(t, err, "Execute should return naturally")
}
