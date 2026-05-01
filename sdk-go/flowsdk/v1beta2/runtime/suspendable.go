package runtime

import (
	"context"
	"errors"
)

// errOperatorSuspended is returned by nodeRef.recv when the handler's
// suspendCh fires before a real input message arrives. It signals to the
// caller that the iteration should pause via the suspendable mixin's
// pauseUntilResume rather than exit.
//
// This is a sentinel error, never wrapped. It's caught explicitly with
// errors.Is so it never bubbles up past the handler's main loop.
var errOperatorSuspended = errors.New("operator suspended")

// suspendableMixin gives a handler the ability to pause its main loop in
// response to operator-driven Suspend/Resume without exiting its goroutine.
//
// The execution model:
//   - exec.Suspend() calls h.suspend() which signals suspendCh
//   - The handler's loop notices errOperatorSuspended at its next pause
//     point and calls pauseUntilResume(ctx)
//   - exec.Resume() calls h.resume() which signals resumeCh
//   - pauseUntilResume returns; the loop continues with the same goroutine
//
// Crucially: ctx cancellation is reserved for "stop forever". Suspend
// never cancels ctx. This eliminates the entire class of "consumed but
// not published" data-loss bugs that plagued the previous design where
// ctx cancellation served as both stop and suspend.
type suspendableMixin struct {
	suspendCh chan struct{}
	resumeCh  chan struct{}
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

// pauseUntilResume blocks until resume is signalled or ctx is cancelled.
// Returns true if resumed (caller should continue the loop), false if
// the context was cancelled (caller should exit).
func (m *suspendableMixin) pauseUntilResume(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-m.resumeCh:
		return true
	}
}

