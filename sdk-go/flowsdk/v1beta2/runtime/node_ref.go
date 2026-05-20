package runtime

import (
	"context"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// nodeRef holds a reference to a subscription channel for one upstream
// dependency. The channel is the same for streaming and cached deps;
// the difference is the recv strategy.
//
// Streaming refs (isCached=false): block on ch for one message per
// iteration; closed channel or EOF value sets chanClosed=true.
//
// Cached refs (isCached=true): the producer's `cache: true` spec
// guarantees it emits at most once between captures (drain-and-skip).
// The lastSeen pointer holds per-source state across iterations:
//
//   - First recv on a cached ref: blocking. Same as streaming. The
//     received node is stored in lastSeen for reuse.
//   - Mixed-deps subsequent recv (allCached=false): non-blocking drain.
//     Consumes any pending messages, refreshing lastSeen with the
//     latest non-EOF NODE_OUTPUT. Returns lastSeen if no fresh value
//     is pending. A closed channel is ignored when we have lastSeen
//     (consumer keeps using last value while streaming deps drive
//     iteration).
//   - All-cached subsequent recv (allCached=true): blocking like the
//     first recv. The producer's EOF (channel close) propagates via
//     chanClosed so AnyEOF lets the consumer exit cleanly.
type nodeRef struct {
	ch         <-chan *pubsub.Message
	ctx        context.Context
	suspendCh  <-chan struct{} // optional: handler's suspend signal
	stopCh     <-chan struct{} // optional: handler's graceful-stop signal
	node       executor.StateNode
	chanClosed bool // channel itself was closed (not EOF value)
	consumed   bool

	// Cached-dep fields. When isCached is true, the recv strategy
	// differs (see above doc). lastSeen is required when isCached.
	isCached  bool
	allCached bool            // consumer-side: this consumer has only cached deps
	lastSeen  *cachedRefState // shared per-source last-received node
}

// cachedRefState holds the last node successfully received from a
// cached source, shared across iterations on the same handler. node ==
// nil means the consumer hasn't yet received a first value.
type cachedRefState struct {
	node executor.StateNode
}

// nodeFromMessage acks the message and returns the StateNode if it
// carries a NODE_OUTPUT event, or nil if it's a NODE_UPDATE that should
// be skipped. Centralizes the recv-side handling shared across recv,
// recvCachedBlocking, and refreshLastSeen.
func nodeFromMessage(msg *pubsub.Message) executor.StateNode {
	event := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
	msg.Ack()
	if event.GetEventType() == flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_UPDATE {
		return nil
	}
	return runtimeNodeFromEvent(event)
}

func (nr *nodeRef) recv() error {
	if nr.consumed {
		return nil
	}
	if nr.isCached {
		return nr.recvCached()
	}
	if nr.ch == nil {
		return nil
	}
	for {
		// Priority check: control signals (ctx cancel, operator suspend,
		// operator stop) take precedence over input messages. Without this,
		// Go's select randomization can pick a buffered input (e.g. an EOF
		// marker) over an already-signaled suspendCh, causing the handler
		// to exit naturally instead of suspending -- a race that masks
		// suspend semantics. errOperatorSuspended/errOperatorStopped are
		// returned WITHOUT consuming a message, so the next recv (after
		// resume, for suspend) picks up the same buffered message.
		//
		// stopCh (StopNode -- per-node graceful stop) wins over input by
		// design: StopNode means "exit at the next safe point", not
		// "drain everything first". For drain-and-exit semantics, FC.stop
		// uses input EOF cascade instead of stopCh.
		select {
		case <-nr.ctx.Done():
			return nr.ctx.Err()
		case <-nr.suspendCh:
			return errOperatorSuspended
		case <-nr.stopCh:
			return errOperatorStopped
		default:
		}
		select {
		case <-nr.ctx.Done():
			return nr.ctx.Err()
		case <-nr.suspendCh:
			return errOperatorSuspended
		case <-nr.stopCh:
			return errOperatorStopped
		case msg, ok := <-nr.ch:
			if !ok {
				nr.chanClosed = true
				nr.consumed = true
				return nil
			}
			node := nodeFromMessage(msg)
			if node == nil {
				continue // NODE_UPDATE: skip, wait for next NODE_OUTPUT
			}
			nr.node = node
			nr.consumed = true
			return nil
		}
	}
}

// recvCached is the cached-dep variant of recv. See nodeRef doc for
// the strategy. Mode (blocking vs non-blocking drain) is decided by
// allCached and whether we already have a lastSeen.
//
// nr.lastSeen MUST be non-nil for cached refs -- the activation
// builder allocates one per cached source. recvCachedDrain assumes
// this invariant; recvCachedBlocking handles both the no-prior-value
// and have-prior-value cases via the hasLast flag.
func (nr *nodeRef) recvCached() error {
	hasLast := nr.lastSeen != nil && nr.lastSeen.node != nil
	if nr.allCached || !hasLast {
		return nr.recvCachedBlocking(hasLast)
	}
	return nr.recvCachedDrain()
}

// recvCachedBlocking is the blocking branch: first recv on a mixed-deps
// consumer, or every recv on an all-cached consumer. NODE_UPDATEs are
// skipped via a loop (no recursion) so a flood of update events can't
// blow the stack.
func (nr *nodeRef) recvCachedBlocking(hasLast bool) error {
	for {
		select {
		case <-nr.ctx.Done():
			return nr.ctx.Err()
		case <-nr.suspendCh:
			return errOperatorSuspended
		case <-nr.stopCh:
			return errOperatorStopped
		default:
		}
		select {
		case <-nr.ctx.Done():
			return nr.ctx.Err()
		case <-nr.suspendCh:
			return errOperatorSuspended
		case <-nr.stopCh:
			return errOperatorStopped
		case msg, ok := <-nr.ch:
			if !ok {
				// Producer terminated.
				if nr.allCached || !hasLast {
					// No fallback: signal EOF so consumer exits.
					nr.chanClosed = true
				} else {
					// Mixed-deps with prior last value: keep using it.
					nr.node = nr.lastSeen.node
				}
				nr.consumed = true
				return nil
			}
			node := nodeFromMessage(msg)
			if node == nil {
				continue // NODE_UPDATE: skip and re-block
			}
			nr.node = node
			nr.lastSeen.node = node
			nr.consumed = true
			return nil
		}
	}
}

// recvCachedDrain is the non-blocking branch for mixed-deps subsequent
// reads. The consumer's iteration is driven by streaming deps, so this
// just refreshes lastSeen with any newly-arrived value and returns it.
func (nr *nodeRef) recvCachedDrain() error {
	nr.refreshLastSeen()
	nr.node = nr.lastSeen.node
	nr.consumed = true
	return nil
}

// refreshLastSeen non-blockingly drains any pending messages and
// updates lastSeen with the most recent non-EOF NODE_OUTPUT.
// NODE_UPDATEs and EOFs are drained but don't replace lastSeen --
// they're noise from a cached consumer's perspective. Returns when the
// channel is empty or closed; lastSeen always reflects the freshest
// available value.
func (nr *nodeRef) refreshLastSeen() {
	for {
		select {
		case msg, ok := <-nr.ch:
			if !ok {
				return // channel closed; nothing more to drain
			}
			fresh := nodeFromMessage(msg)
			if fresh == nil || isEOFValue(fresh.GetValue()) {
				continue
			}
			nr.lastSeen.node = fresh
		default:
			return // no pending messages
		}
	}
}
