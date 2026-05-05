package runtime

import (
	"context"
	"errors"
)

// errOperatorSuspended is returned by nodeRef.recv when the handler's
// suspendCh fires before a real input message arrives. It signals to the
// caller that the iteration should pause via the suspendable mixin's
// waitForResume rather than exit.
//
// This is a sentinel error, never wrapped. It's caught explicitly with
// errors.Is so it never bubbles up past the handler's main loop.
var errOperatorSuspended = errors.New("operator suspended")

// suspendResult is what waitForResume returns. The handler branches on
// it to decide what the suspended handler should do when it wakes:
// continue the loop, exit cleanly with PHASE_SUCCEEDED, or exit with
// PHASE_CANCELLED.
type suspendResult int

const (
	// suspendResumed: ResumeNode was called; continue the main loop.
	suspendResumed suspendResult = iota
	// suspendStopped: StopNode was called while suspended; exit the main
	// loop cleanly. The handler's post-loop publish emits PHASE_SUCCEEDED.
	suspendStopped
	// suspendCancelled: ctx was cancelled (TerminateNode or flow-wide
	// terminate). The handler returns ctx.Err(); the lifecycle wrapper
	// emits PHASE_CANCELLED.
	suspendCancelled
)

// suspendableMixin gives a handler the ability to pause its main loop in
// response to operator-driven Suspend/Resume without exiting its goroutine.
//
// The execution model:
//   - exec.Suspend() calls h.suspend() which signals suspendCh
//   - The handler's loop notices errOperatorSuspended at its next pause
//     point and calls waitForResume(ctx, h.StopChan())
//   - exec.Resume() calls h.resume() which signals resumeCh; waitForResume
//     returns suspendResumed and the loop continues with the same goroutine
//   - exec.StopNode() calls h.requestStop() which signals stopCh (from
//     stoppableMixin); a suspended handler in waitForResume returns suspendStopped, the
//     handler breaks out of its loop, and the post-loop publish emits
//     PHASE_SUCCEEDED. Stop is one-way -- there is no "resume from stop"
//   - ctx cancellation (TerminateNode / flow terminate) returns
//     suspendCancelled; the handler returns ctx.Err() and the lifecycle
//     wrapper publishes PHASE_CANCELLED
//
// Crucially: ctx cancellation is reserved for "terminate forever". Suspend
// never cancels ctx. This eliminates the entire class of "consumed but
// not published" data-loss bugs that plagued the previous design where
// ctx cancellation served as both stop and suspend.
type suspendableMixin struct {
	suspendCh chan struct{}
	resumeCh  chan struct{}
	// selfSuspendFn is set by the executor at handler-wire time. RPC handlers
	// invoke it after `executeWithRetry` returns *SuspendError -- the handler
	// is already inside its own loop and does NOT need a suspendCh signal,
	// but it does need executor-side bookkeeping (mark `suspendedNodes[id]`,
	// publish PHASE_SUSPENDED on the node's topic) before parking via
	// waitForResume. Nil for handlers without retry-strategy support.
	selfSuspendFn func(error)
}

// initSuspendable initializes the channels with capacity 1 so suspend()
// and resume() are non-blocking.
func (m *suspendableMixin) initSuspendable() {
	m.suspendCh = make(chan struct{}, 1)
	m.resumeCh = make(chan struct{}, 1)
}

func (m *suspendableMixin) suspend() {
	select {
	case m.suspendCh <- struct{}{}:
	default:
	}
}

func (m *suspendableMixin) resume() {
	select {
	case m.resumeCh <- struct{}{}:
	default:
	}
}

// SuspendChan returns the receive end of the suspend signal so the
// handler's main loop and nodeRef.recv can include it in a select.
func (m *suspendableMixin) SuspendChan() <-chan struct{} {
	return m.suspendCh
}

// waitForResume blocks until one of: a stop signal arrives on stopCh, a
// resume signal arrives on resumeCh, or ctx is cancelled. Returns the
// suspendResult so the caller can branch on which intent fired.
//
// stopCh comes from the handler's stoppableMixin (h.StopChan()). Pass
// nil only if the handler has no stoppable surface -- but every handler
// in this package embeds stoppableMixin, so in practice always pass it.
func (m *suspendableMixin) waitForResume(ctx context.Context, stopCh <-chan struct{}) suspendResult {
	select {
	case <-ctx.Done():
		return suspendCancelled
	case <-stopCh:
		return suspendStopped
	case <-m.resumeCh:
		return suspendResumed
	}
}

// setSelfSuspendCallback installs the executor-side bookkeeping closure
// invoked by RPC handlers when they catch *SuspendError from the retry
// strategy. The callback marks the node as suspended and publishes
// PHASE_SUSPENDED; the handler is responsible for then calling
// waitForResume to actually park.
func (m *suspendableMixin) setSelfSuspendCallback(fn func(error)) {
	m.selfSuspendFn = fn
}

// selfSuspend invokes the installed bookkeeping callback. No-op when
// unwired (the handler may also be on a path that never sees SuspendError).
func (m *suspendableMixin) selfSuspend(err error) {
	if m.selfSuspendFn != nil {
		m.selfSuspendFn(err)
	}
}

// retrySuspender is implemented by handlers that may surface
// *SuspendError from a retry strategy and therefore need an executor-
// installed bookkeeping callback. Currently the four RPC handlers.
type retrySuspender interface {
	setSelfSuspendCallback(fn func(error))
}
