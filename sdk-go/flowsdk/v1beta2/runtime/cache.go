package runtime

import (
	"context"
	"sync/atomic"

	"github.com/google/cel-go/common/types"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// cacheCapture is the producer-side state for a cache:true node. It
// tracks the captured flag (drain-and-skip after first emit, reset by
// ClearCache) and is a no-op when the producer's spec doesn't have
// cache:true (enabled=false).
type cacheCapture struct {
	enabled  bool
	captured atomic.Bool
}

// isCaptured reports whether the producer has captured a value. The
// handler checks this before its eval/publish path; if true, the
// iteration is drain-and-skip (read upstream but don't produce).
// Non-cached producers always return false.
func (c *cacheCapture) isCaptured() bool {
	return c.enabled && c.captured.Load()
}

// markCaptured records that the producer has emitted its captured
// value. Called by the handler immediately after a successful publish.
// Subsequent isCaptured calls return true until ClearCache resets it.
// No-op for non-cached producers.
func (c *cacheCapture) markCaptured() {
	if !c.enabled {
		return
	}
	c.captured.Store(true)
}

// clearCapture resets captured=false so the next upstream event will
// be processed and re-emitted. Called by Executor.ClearCache.
func (c *cacheCapture) clearCapture() {
	c.captured.Store(false)
}

// cacheDeps is the consumer-side state for a node that subscribes to
// cache:true upstream sources. cachedSources is the subset of input
// source IDs whose producer has cache:true; the activation builder
// uses this to switch nodeRef recv strategy. allCached drives
// per-iteration blocking semantics. cachedMem holds per-source
// last-seen state across iterations on the same handler.
type cacheDeps struct {
	cachedSources map[string]bool
	allCached     bool
	cachedMem     map[string]*cachedRefState
}

// newActivation builds an activation that mixes streaming and cached
// deps. When no cached deps exist, falls through to the streaming-only
// path. Lazily allocates per-source last-seen state on first call;
// reused across iterations.
func (c *cacheDeps) newActivation(
	ctx context.Context,
	inputs map[string]<-chan *pubsub.Message,
	adapter types.Adapter,
	suspendCh, stopCh <-chan struct{},
) *activation {
	if len(c.cachedSources) == 0 {
		return newActivationFromChannelsInterruptible(ctx, inputs, adapter, suspendCh, stopCh)
	}
	if c.cachedMem == nil {
		c.cachedMem = make(map[string]*cachedRefState, len(c.cachedSources))
		for id := range c.cachedSources {
			c.cachedMem[id] = &cachedRefState{}
		}
	}
	return newActivationFromMixedDeps(
		ctx, inputs, c.cachedSources, c.allCached, c.cachedMem,
		adapter, suspendCh, stopCh)
}

// cacheBackend is a per-handler view onto cache:true semantics. It
// composes the producer-side state (cacheCapture) and consumer-side
// state (cacheDeps); each is independent. Methods inherited via
// embedding so handler call sites read naturally:
//
//	h.cache.isCaptured()             // producer side
//	h.cache.markCaptured()           // producer side
//	h.cache.newActivation(...)       // consumer side
//
// cacheBackend is shared by pointer between the executor and the
// handler so ClearCache can reset the captured flag from outside the
// handler goroutine.
type cacheBackend struct {
	cacheCapture
	cacheDeps
}

// isCachedProducer reports whether the given node has cache: true on
// its Input/Var/Action spec. Stream nodes are not cache producers
// (Stream proto has no `cache` field); they can be consumers of cached
// deps but never produce cached values.
func isCachedProducer(node *flowv1beta2.Node) bool {
	switch node.WhichType() {
	case flowv1beta2.Node_Input_case:
		return node.GetInput().GetCache()
	case flowv1beta2.Node_Var_case:
		return node.GetVar().GetCache()
	case flowv1beta2.Node_Action_case:
		return node.GetAction().GetCache()
	}
	return false
}
