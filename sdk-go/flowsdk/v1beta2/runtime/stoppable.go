package runtime

import "errors"

// errOperatorStopped is returned by nodeRef.recv when the handler's stopCh
// fires before a real input message arrives. It signals to the caller that
// the iteration should terminate cleanly (the lifecycle wrapper will publish
// PHASE_SUCCEEDED on natural exit) rather than continue.
//
// This is a sentinel error, never wrapped. It's caught explicitly with
// errors.Is so it never bubbles up past the handler's main loop.
var errOperatorStopped = errors.New("operator stopped")

// stoppableMixin gives a handler the ability to exit its main loop in
// response to an operator-driven graceful StopNode without ctx cancellation.
//
// The execution model:
//   - exec.StopNode(id) calls h.requestStop() which signals stopCh
//   - The handler's loop notices errOperatorStopped at its next pause point
//     (or via <-h.StopChan() in throttle/wait selects) and exits cleanly
//   - Any in-flight operation (RPC, stream send, prompt wait) completes
//     naturally before the handler returns, because we DO NOT cancel ctx
//   - The lifecycle wrapper publishes PHASE_SUCCEEDED on the natural exit
//
// Crucially: stop is one-way. There is no "resume from stop" - once a
// node is stopped, it's done. ctx cancellation is reserved for "abort
// in-flight work right now" (TerminateNode), which is distinct from
// graceful stop.
type stoppableMixin struct {
	stopCh chan struct{}
}

// initStoppable initializes the channel with capacity 1 so requestStop()
// is non-blocking.
func (m *stoppableMixin) initStoppable() {
	m.stopCh = make(chan struct{}, 1)
}

func (m *stoppableMixin) requestStop() {
	select {
	case m.stopCh <- struct{}{}:
	default:
	}
}

// StopChan returns the receive end of the stop signal so the handler's
// main loop and nodeRef.recv can include it in a select.
func (m *stoppableMixin) StopChan() <-chan struct{} {
	return m.stopCh
}

// selfStoppable is implemented by handlers that support graceful stop.
// Executor.StopNode dispatches via this interface.
type selfStoppable interface {
	requestStop()
	StopChan() <-chan struct{}
}
