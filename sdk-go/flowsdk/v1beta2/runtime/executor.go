package runtime

import (
	"context"
	"errors"
	"sync"
	"time"

	expr "cel.dev/expr"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/cache"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"golang.org/x/sync/errgroup"
)

// ErrTerminated is returned by Execute when the flow is terminated via
// the Terminate method (operator-initiated cancellation).
var ErrTerminated = errors.New("flow terminated by operator")

// nodeWiring holds the subscription channels and output topic for a
// single node. cachedSources marks the subset of input source IDs whose
// producer has cache: true on its spec; consumers subscribe via the
// same pubsub channel as any other dep but apply non-blocking +
// last-seen semantics for those refs (see nodeRef.recvCached).
type nodeWiring struct {
	inputs        map[string]<-chan *pubsub.Message
	cachedSources map[string]bool
	topic         string
}

// hasOnlyCachedDeps reports whether every input dependency of this
// node is a cached source. Such consumers iterate per producer message
// (each cached emit drives one iteration) instead of per streaming
// message; see nodeRef.recvCached.
func (w *nodeWiring) hasOnlyCachedDeps() bool {
	return len(w.cachedSources) > 0 && len(w.inputs) == len(w.cachedSources)
}

// selfSuspendable is implemented by generator handlers that manage their own
// suspend/resume lifecycle (pausing timers in-place rather than having their
// context cancelled).
type selfSuspendable interface {
	suspend()
	resume()
}

// Executor runs a graph using goroutines and PubSub.
type Executor struct {
	pubsub               executor.PubSub
	topics               executor.Topics
	connectors           rpc.ConnectorProvider
	cache                cache.Cache
	outbox               outbox.StatefulOutbox
	subscriber           *outbox.SubscriberAdapter
	defaultInputThrottle *flowv1beta2.Rate
	interactionPrompt    chan<- *flowv1beta2.InteractionRequestEvent
	interactionResponse  <-chan *flowv1beta2.InteractionResponseEvent
	inputRequests        chan<- *flowv1beta2.InputRequestEvent
	errorStrategy        flowv1beta2.ErrorStrategy
	// platformResolver is the SDK platform-types resolver (api.GlobalResolver()
	// by default; overridable via WithPlatformResolver). It is the last member
	// of the flow-global union built by buildCELEnv and carries platform/wkt
	// types (wrappers, struct, Any, timestamp/duration, flow protos) that the
	// declared connectors' descriptor closures do not necessarily include.
	platformResolver shared.Resolver

	// Per-execution state, protected by mu. Set during Execute, nil otherwise.
	mu            sync.Mutex
	stopFn        func()                          // graceful stop: EOF injection + generator cancel
	terminateFn   func()                          // immediate cancel: cancels runCtx
	nodeCtxs      map[string]context.CancelFunc   // per-node context cancellation
	nodeProtos    map[string]*flowv1beta2.Node    // node protos for phase publishing
	handlerPub    pubsub.Publisher                // publisher for terminal phase events
	terminated    bool                            // true if Terminate() was called
	stoppedNodes  map[string]bool                 // nodes stopped by StopNode (→ SUCCEEDED)
	handlers      map[string]executor.NodeHandler // handler reference for suspend routing
	terminalNodes map[string]bool                 // nodes whose handler has exited; operator events become no-ops

	// Suspend/resume state, also protected by mu.
	suspendedNodes map[string]bool // nodes in PHASE_SUSPENDED

	// Cache delivery state. Set during Execute, nil otherwise. Per-node
	// cacheBackend pointers; the executor uses this map for ClearCache
	// lookup. Producer-side captured flag lives on the cacheBackend.
	cacheBackends map[string]*cacheBackend

	// terminalState is the flow-level terminal FlowState (phase + error +
	// timings) the executor produces and publishes at the end of a run.
	// Deliberately NOT cleared by clearRunState: it must remain readable
	// via TerminalState after Execute returns (that is its only contract).
	// Reset to nil at the start of each Execute for reuse safety.
	terminalState *flowv1beta2.RunSnapshot_FlowState
}

type Option func(*Executor)

func WithConnectors(conns rpc.ConnectorProvider) Option {
	return func(e *Executor) {
		e.connectors = conns
	}
}

// WithPlatformResolver overrides the platform-types resolver that forms
// the last member of the flow-global type-resolution union (after the
// run's declared connectors, in spec order). When unset, buildCELEnv
// defaults to api.GlobalResolver() - the SDK's named platform-types
// resolver (the same default v1beta1 uses in flowsdk/v1beta1/runtime/
// env.go). Override for tests that need to isolate from the SDK's
// global proto set, or for future per-runtime-boundary deployments
// (e.g. a node running in its own pod that wants to scope the platform
// layer to exactly the protos it needs).
func WithPlatformResolver(r shared.Resolver) Option {
	return func(e *Executor) {
		e.platformResolver = r
	}
}

// WithDefaultInputThrottle overrides the built-in minimum throttle injected
// for Inputs that have cache or a type-level default set but no explicit
// throttle. When unset, minInputThrottle (10ms) is used.
func WithDefaultInputThrottle(rate *flowv1beta2.Rate) Option {
	return func(e *Executor) {
		e.defaultInputThrottle = rate
	}
}

func WithCache(c cache.Cache) Option {
	return func(e *Executor) {
		e.cache = c
	}
}

func WithOutbox(o outbox.StatefulOutbox) Option {
	return func(e *Executor) {
		e.outbox = o
	}
}

// WithSubscriber injects an externally-owned SubscriberAdapter that the
// outbox relay will read from. Callers that want to nudge the poll loop
// from outside the executor (e.g. an ent commit hook calling Wake) should
// construct the adapter, pass it here, and retain a reference. When unset,
// Execute creates its own adapter internally with a fixed poll interval.
func WithSubscriber(s *outbox.SubscriberAdapter) Option {
	return func(e *Executor) {
		e.subscriber = s
	}
}

// WithInteractions configures the executor to support interaction nodes.
// The prompt channel receives InteractionRequestEvents (id + token) emitted
// when an interaction node needs external input. The response channel provides
// InteractionResponseEvents (id + token + value + actor) from external sources.
func WithInteractions(
	prompt chan<- *flowv1beta2.InteractionRequestEvent,
	response <-chan *flowv1beta2.InteractionResponseEvent,
) Option {
	return func(e *Executor) {
		e.interactionPrompt = prompt
		e.interactionResponse = response
	}
}

// WithInputRequests configures a channel that receives InputRequestEvents
// when the executor is ready to accept values for each input node.
func WithInputRequests(ch chan<- *flowv1beta2.InputRequestEvent) Option {
	return func(e *Executor) {
		e.inputRequests = ch
	}
}

// WithErrorStrategy sets the flow-level error strategy. When a node enters
// PHASE_ERRORED, this strategy determines whether the flow terminates
// immediately (TERMINATE, default), drains gracefully (STOP), or continues
// running independent paths (CONTINUE).
func WithErrorStrategy(s flowv1beta2.ErrorStrategy) Option {
	return func(e *Executor) {
		e.errorStrategy = s
	}
}

func NewExecutor(pubsub executor.PubSub, topics executor.Topics, opts ...Option) *Executor {
	e := &Executor{
		pubsub: pubsub,
		topics: topics,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Stop initiates a graceful drain of the running flow. Input EOFs are
// published, generators stop, and the pipeline drains naturally. Nodes
// finish their current work and transition to PHASE_SUCCEEDED.
// Safe to call concurrently. No-op if the flow is not running.
func (e *Executor) Stop() {
	e.mu.Lock()
	fn := e.stopFn
	e.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// Terminate cancels the flow execution immediately. Running nodes get
// context cancelled and transition to PHASE_CANCELLED. Execute returns
// ErrTerminated. Safe to call concurrently. No-op if the flow is not running.
func (e *Executor) Terminate() {
	e.mu.Lock()
	fn := e.terminateFn
	e.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// currentPhase derives the current logical phase of a node from the
// executor's tracking maps. Used by operator methods to consult the
// transition table before mutating state.
//
// Must be called with e.mu held.
//
// We don't distinguish PENDING from RUNNING because the executor doesn't
// track when a handler's first iteration actually runs -- the transition
// table treats both the same for all operator events anyway (Stop/
// Terminate/Suspend valid on both; Resume invalid on both).
//
// All terminal phases (SUCCEEDED/CANCELLED/ERRORED/FAILED) collapse to
// SUCCEEDED here because the transition table has no entries for any
// terminal phase -- any event is invalid -- so the specific phase doesn't
// affect validity decisions.
func (e *Executor) currentPhase(nodeID string) flowv1beta2.RunSnapshot_Phase {
	if e.terminalNodes[nodeID] {
		return flowv1beta2.RunSnapshot_PHASE_SUCCEEDED
	}
	if e.suspendedNodes[nodeID] {
		return flowv1beta2.RunSnapshot_PHASE_SUSPENDED
	}
	if e.stoppedNodes[nodeID] {
		return flowv1beta2.RunSnapshot_PHASE_STOPPING
	}
	return flowv1beta2.RunSnapshot_PHASE_RUNNING
}

// StopNode initiates a graceful stop of a single node. The node transitions
// to PHASE_STOPPING immediately, drains its in-flight work per-type, and
// then transitions to PHASE_SUCCEEDED on natural exit. Per-type drain:
//   - Input: publishes EOF on the input topic; downstream subscribers
//     drain to PHASE_SUCCEEDED.
//   - Generator (ticker/cron/range): signals stopCh via the stoppable
//     mixin; the handler exits with PHASE_SUCCEEDED at its next safe point.
//   - Stream / unary action / interaction / var / output / switch: signals
//     the stoppable mixin so the handler exits its main loop at a safe
//     point. In-flight RPCs / streams / prompts complete naturally; ctx
//     is NOT cancelled (that's TerminateNode's job).
//
// Does not trigger error_strategy. No-op if the flow is not running, the
// node ID is unknown, or the (currentPhase, eventStop) transition is
// invalid per the transition table (validateNodeTransition).
func (e *Executor) StopNode(nodeID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopFn == nil {
		return // not running
	}
	np, ok := e.nodeProtos[nodeID]
	if !ok {
		return // unknown node
	}

	// Consult the transition table. Invalid (phase, event) -> no-op.
	if _, ok := validateNodeTransition(e.currentPhase(nodeID), eventStop); !ok {
		return
	}

	// Input nodes: inject EOF on the input topic. The bridge goroutine
	// observes the EOF, propagates downstream, and the input node's
	// handler publishes PHASE_SUCCEEDED on exit.
	if np.WhichType() == flowv1beta2.Node_Input_case {
		_ = e.pubsub.Publish(e.topics.InputFor(nodeID), pubsub.NewMessage(newEOFValue()))
		return
	}

	// Publish PHASE_STOPPING (transient state) so observers see the
	// drain in progress. PHASE_SUCCEEDED is published by the handler
	// itself on natural exit (post-loop EOF/SUCCEEDED publish).
	_ = publishPhaseChange(e.handlerPub, e.topics.For(nodeID), np,
		flowv1beta2.RunSnapshot_PHASE_STOPPING, nil)

	// Mark stoppedNodes so currentPhase reports STOPPING. This makes
	// subsequent transitions consult the table correctly: e.g.
	// STOPPING+Stop is idempotent (no-op), STOPPING+Suspend is invalid.
	// terminalNodes will overwrite this in the lifecycle wrapper's defer
	// when the handler exits.
	e.stoppedNodes[nodeID] = true

	// All long-lived handlers implement selfStoppable. Stop signals stopCh;
	// the handler exits cleanly with PHASE_SUCCEEDED via its post-loop
	// publish. Suspended handlers also see stopCh, so a suspended-then-
	// stopped handler exits gracefully.
	if sh, ok := e.handlers[nodeID].(selfStoppable); ok {
		sh.requestStop()
		delete(e.suspendedNodes, nodeID)
	}
}

// TerminateNode cancels a single node immediately. The node transitions to
// PHASE_CANCELLED. Does not trigger error_strategy. No-op if the flow is
// not running, the node ID is unknown, or the (currentPhase, eventTerminate)
// transition is invalid per the transition table.
func (e *Executor) TerminateNode(nodeID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.terminateFn == nil {
		return // not running
	}
	if _, ok := validateNodeTransition(e.currentPhase(nodeID), eventTerminate); !ok {
		return
	}

	if cancel, ok := e.nodeCtxs[nodeID]; ok {
		cancel()
		// Publish PHASE_CANCELLED for the terminated node.
		if np, ok := e.nodeProtos[nodeID]; ok {
			_ = publishTerminalPhase(e.handlerPub, e.topics.For(nodeID), np,
				flowv1beta2.RunSnapshot_PHASE_CANCELLED, nil)
		}
	}
}

// Suspend pauses every running handler in place. Every handler implements
// selfSuspendable, so we never cancel handler contexts on suspend - that
// reservation belongs to "stop forever". Each handler honors the suspend
// signal at a safe pause point in its loop (between iterations, never
// mid-consume), which preserves goroutine lifetimes and eliminates the
// consumed-but-not-published race that ctx-cancel-on-suspend allowed.
// In-flight external operations (RPC calls, stream sends, prompt waits)
// are NEVER aborted; they complete naturally before the handler pauses.
// Safe to call concurrently. No-op if the flow is not running.
func (e *Executor) Suspend() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopFn == nil {
		return
	}
	for id := range e.handlers {
		if e.suspendedNodes[id] || e.terminalNodes[id] {
			continue // skip already-suspended and terminal nodes
		}
		e.suspendedNodes[id] = true
		if sh, ok := e.handlers[id].(selfSuspendable); ok {
			sh.suspend()
		}
		e.publishSuspendedPhase(id)
	}
}

// SuspendNode suspends a single handler. See Suspend. No-op if the flow
// is not running, the node ID is unknown, or the (currentPhase,
// eventSuspend) transition is invalid (e.g. node is already SUSPENDED,
// STOPPING, or terminal).
func (e *Executor) SuspendNode(nodeID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopFn == nil {
		return
	}
	if _, ok := e.handlers[nodeID]; !ok {
		return
	}
	if _, ok := validateNodeTransition(e.currentPhase(nodeID), eventSuspend); !ok {
		return // invalid transition (idempotent on SUSPENDED, no-op on terminal/stopping)
	}
	e.suspendedNodes[nodeID] = true
	if sh, ok := e.handlers[nodeID].(selfSuspendable); ok {
		sh.suspend()
	}
	e.publishSuspendedPhase(nodeID)
}

// publishSuspendedPhase emits a PHASE_SUSPENDED state event on the node's
// topic so attached clients can observe the operator's suspend intent
// immediately. The handler may not actually be paused yet (e.g. blocked
// in an RPC call), but the operator's intent is the source of truth for
// the visible phase. Must be called with e.mu held.
func (e *Executor) publishSuspendedPhase(nodeID string) {
	np, ok := e.nodeProtos[nodeID]
	if !ok {
		return
	}
	_ = publishPhaseChange(e.handlerPub, e.topics.For(nodeID), np,
		flowv1beta2.RunSnapshot_PHASE_SUSPENDED, nil)
}

// Resume signals every suspended handler to continue, and clears the
// suspendedNodes entry for each. Resume is the single writer that flips
// nodes back to "running" - handlers no longer participate in the
// bookkeeping. This eliminates the race where a Suspend/Resume/Suspend
// triple could silently no-op the second Suspend.
// Safe to call concurrently. No-op if no nodes are suspended.
func (e *Executor) Resume() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for id := range e.suspendedNodes {
		if sh, ok := e.handlers[id].(selfSuspendable); ok {
			sh.resume()
		}
		delete(e.suspendedNodes, id)
	}
}

// ResumeNode resumes a single suspended node. Signals the handler's mixin
// resumeCh so its waitForResume returns and clears the suspendedNodes
// flag. The `val` parameter is currently unused -- kept on the signature
// for future "resume with corrected input" support; today every handler
// resumes from where it was suspended and reads the next input naturally.
//
// All current handlers implement selfSuspendable. No-op if the
// (currentPhase, eventResume) transition is invalid (i.e. node is not
// currently SUSPENDED).
func (e *Executor) ResumeNode(nodeID string, _ *expr.Value) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := validateNodeTransition(e.currentPhase(nodeID), eventResume); !ok {
		return
	}
	if sh, ok := e.handlers[nodeID].(selfSuspendable); ok {
		sh.resume()
	}
	delete(e.suspendedNodes, nodeID)
}

// ClearCache resets the captured flag on a cache:true producer so the
// next upstream event will be processed and re-emitted. The current
// cached value remains until the next emit replaces it -- consumers
// reading inline see the prior value during the gap. No-op when the
// flow is not running, the node is unknown, or the node does not have
// cache:true on its spec.
func (e *Executor) ClearCache(nodeID string) {
	e.mu.Lock()
	cb, ok := e.cacheBackends[nodeID]
	e.mu.Unlock()
	if !ok || cb == nil || !cb.enabled {
		return
	}
	cb.clearCapture()
}

// clearRunState clears all per-execution state. Must be called with mu held.
func (e *Executor) clearRunState() {
	e.stopFn = nil
	e.terminateFn = nil
	e.nodeCtxs = nil
	e.nodeProtos = nil
	e.handlerPub = nil
	e.terminated = false
	e.stoppedNodes = nil
	e.handlers = nil
	e.suspendedNodes = nil
	e.terminalNodes = nil
	e.cacheBackends = nil
}

// TerminalState returns the flow-level terminal FlowState the executor
// produced for the most recent run (phase, error, start/stop/event
// times) - the same object published on the flow topic. The bool is
// false until Execute has published the terminal (i.e. only valid after
// Execute returns); callers must not rely on it mid-run.
func (e *Executor) TerminalState() (*flowv1beta2.RunSnapshot_FlowState, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.terminalState == nil {
		return nil, false
	}
	return e.terminalState, true
}

func (e *Executor) Execute(ctx context.Context, graph *flowv1beta2.Graph) error {
	// Reset terminal state so TerminalState reports "not done" until this
	// run publishes its own terminal (reuse safety; single-use in prod).
	e.mu.Lock()
	e.terminalState = nil
	e.mu.Unlock()

	// Wrap context so Terminate() can cancel execution independently.
	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	g, gCtx := errgroup.WithContext(runCtx)

	// Wire nodes: subscribe to source topics for each downstream edge
	wiring, err := e.wireEdges(gCtx, graph)
	if err != nil {
		return err
	}

	// Determine handler publisher: outbox if configured, direct pubsub if not.
	var handlerPub executor.PubSub
	var outboxSub *outbox.SubscriberAdapter
	var fwdDone chan error
	if e.outbox != nil {
		handlerPub = &outboxPubSub{
			pub: &txPublisher{
				txBeginner: e.outbox,
				snap:       &flowv1beta2.RunSnapshot{},
			},
			sub: e.pubsub,
		}

		// Outbox relay: reads committed messages from the outbox and
		// publishes them to the gochannel for downstream subscribers.
		// This is the "message relay" from the transactional outbox
		// pattern: it uses its own context (not the caller's or the
		// errgroup's) so it can continue draining after context
		// cancellation. Its lifetime is controlled by CloseWhenDrained
		// (normal) or the deferred relayCancel (early exit).
		relayCtx, relayCancel := context.WithCancel(context.Background())
		if e.subscriber != nil {
			outboxSub = e.subscriber
		} else {
			outboxSub = outbox.NewSubscriber(e.outbox, 10*time.Millisecond)
		}
		relay := outbox.NewRelay(outboxSub, e.pubsub)
		fwdDone = make(chan error, 1)
		go func() {
			fwdDone <- relay.Run(relayCtx)
		}()
		// Ensure the relay is stopped if Execute returns early.
		defer func() {
			relayCancel()
			if outboxSub != nil {
				_ = outboxSub.Close()
				<-fwdDone
			}
		}()
	} else {
		handlerPub = e.pubsub
	}

	// Build handlers for non-input, non-interaction nodes
	nodeProtos := make(map[string]*flowv1beta2.Node, len(graph.GetNodes()))
	// Populate nodeProtos for ALL nodes first (needed by StopNode/TerminateNode).
	for _, n := range graph.GetNodes() {
		nodeProtos[n.GetId()] = n
	}

	// Resolve connections: collect Connection node specs and resolve all
	// connections referenced by Action/Stream calls.
	resolved, err := e.resolveConnections(gCtx, graph)
	if err != nil {
		return err
	}

	// Build CEL environment and validate request schemas.
	celEnv, err := buildCELEnvAndValidate(graph, resolved, e.platformResolver)
	if err != nil {
		return err
	}

	// Cache delivery: registry of per-node cacheBackend pointers,
	// populated by buildHandlers / setupInputBridges as they construct
	// handlers. ClearCache looks up cacheBackends[nodeID] to reset the
	// captured flag.
	e.mu.Lock()
	e.cacheBackends = make(map[string]*cacheBackend)
	e.mu.Unlock()

	handlers, err := buildHandlers(graph, celEnv, resolved, e.cache, wiring, handlerPub, e.pubsub, e.cacheBackends)
	if err != nil {
		return err
	}

	// Build interaction handlers.
	interactionHandlers, err := e.buildInteractionHandlers(graph, celEnv, wiring, handlerPub)
	if err != nil {
		return err
	}
	// interactionHandlers is keyed by the bare spec id (Format A) so the
	// demux can look up by InteractionResponseEvent.id directly. Re-key into
	// the cross-category handlers map by the fully-qualified node id
	// (Format B) which is what every other operator-API path expects.
	for _, h := range interactionHandlers {
		handlers["interactions."+h.id] = h
	}

	// Setup input bridges: compile transforms, start pipelines, subscribe to
	// external input topics, and launch bridge goroutines.
	inputNodeIDs, err := e.setupInputBridges(gCtx, g, graph, celEnv)
	if err != nil {
		return err
	}

	// Interaction response demux: route InteractionResponseEvents to handlers.
	// Token validation and delivery are atomic inside TryDeliver.
	//
	// Route interaction responses from the external channel to handlers.
	startInteractionDemux(gCtx, e.interactionResponse, interactionHandlers)

	// Determine effective error strategy. Precedence: option override
	// (WithErrorStrategy) wins over the spec field carried on the graph.
	strategy := e.errorStrategy
	if strategy == flowv1beta2.ErrorStrategy_ERROR_STRATEGY_UNSPECIFIED {
		strategy = graph.GetErrorStrategy()
	}
	if strategy == flowv1beta2.ErrorStrategy_ERROR_STRATEGY_UNSPECIFIED {
		strategy = flowv1beta2.ErrorStrategy_ERROR_STRATEGY_TERMINATE
	}

	// Setup run state: contexts, stop/resume infrastructure, flow_control wiring.
	rs, err := e.setupRunState(gCtx, runCancel, graph, celEnv, handlers, nodeProtos, inputNodeIDs, wiring, handlerPub, strategy)
	if err != nil {
		return err
	}
	defer rs.cleanup()

	// Launch all handlers. Suspend/resume is fully handled inside each handler
	// via the suspendableMixin -- this wrapper only intercepts terminal errors
	// and applies flow-level error_strategy.
	res := e.launchHandlers(g, launchOpts{
		handlers:    handlers,
		nodeProtos:  nodeProtos,
		handlerCtxs: rs.handlerCtxs,
		handlerPub:  handlerPub,
		gCtx:        gCtx,
		genCtx:      rs.genCtx,
		strategy:    strategy,
		performStop: rs.operatorStop,
	})

	// Emit flow-level RUNNING event.
	startTime := time.Now()
	if pubErr := publishFlowState(handlerPub, e.topics.Flow(), flowv1beta2.RunSnapshot_FlowState_builder{
		Phase:     flowv1beta2.RunSnapshot_PHASE_RUNNING,
		StartTime: timestamppb.New(startTime),
		EventTime: timestamppb.New(startTime),
	}.Build()); pubErr != nil {
		return pubErr
	}

	err = g.Wait()

	// For STOP mode, return the stored error after the pipeline has drained.
	if err == nil && res.stopErr != nil {
		err = res.stopErr
	}

	// For CONTINUE mode, return collected node errors after all nodes complete.
	res.mu.Lock()
	resContinueErr := res.continueErr
	res.mu.Unlock()
	if err == nil && resContinueErr != nil {
		err = resContinueErr
	}

	// If Terminate() was called, surface ErrTerminated. Most handlers exit
	// with nil (context.Canceled is swallowed by the launchHandlers wrapper),
	// but the input-bridge goroutines return ctx.Err() directly on cancel
	// and may race ahead of handler exits, leaving g.Wait with
	// context.Canceled. In either case the flow-level signal we want to
	// expose is ErrTerminated, so override.
	e.mu.Lock()
	wasTerminated := e.terminated
	e.mu.Unlock()
	if wasTerminated && (err == nil || errors.Is(err, context.Canceled)) {
		err = ErrTerminated
	}

	// Emit flow-level terminal event before draining the outbox.
	stopTime := time.Now()
	terminalState := flowv1beta2.RunSnapshot_FlowState_builder{
		StartTime: timestamppb.New(startTime),
		StopTime:  timestamppb.New(stopTime),
		EventTime: timestamppb.New(stopTime),
	}
	switch {
	case errors.Is(err, ErrTerminated):
		terminalState.Phase = flowv1beta2.RunSnapshot_PHASE_CANCELLED
	case err != nil:
		terminalState.Phase = flowv1beta2.RunSnapshot_PHASE_ERRORED
		terminalState.Error = grpcStatusProto(err)
	default:
		terminalState.Phase = flowv1beta2.RunSnapshot_PHASE_SUCCEEDED
	}
	builtTerminal := terminalState.Build()
	e.mu.Lock()
	e.terminalState = builtTerminal
	e.mu.Unlock()
	if pubErr := publishFlowState(handlerPub, e.topics.Flow(), builtTerminal); pubErr != nil && err == nil {
		err = pubErr
	}

	// After all handlers complete, drain the outbox: signal the subscriber
	// to stop once no unrelayed messages remain, then wait for the
	// relay to finish publishing. This guarantees every outbox message
	// reaches gochannel subscribers before Execute returns.
	if outboxSub != nil {
		outboxSub.CloseWhenDrained()
		if fwdErr := <-fwdDone; err == nil && fwdErr != nil {
			err = fwdErr
		}
		outboxSub = nil // prevent deferred Close from double-stopping
	}

	return err
}

// publishInputValue publishes a value to the appropriate input destination.
// nodeID is the fully-qualified node id (Format B, e.g. "inputs.x") used
// for topic routing; bareID is the bare spec id (Format A, e.g. "x") used
// in the InputNode protobuf id field whose validator pattern is bare-only.
func (e *Executor) publishInputValue(nodeID, bareID string, val *expr.Value, inputTopics map[string]string) error {
	if topic, ok := inputTopics[nodeID]; ok {
		return e.pubsub.Publish(topic, pubsub.NewMessage(val))
	}
	return publishNode(e.pubsub, e.topics.For(nodeID), flowv1beta2.RunSnapshot_InputNode_builder{
		Id:    bareID,
		Value: val,
		Phase: flowv1beta2.RunSnapshot_PHASE_RUNNING,
	}.Build())
}

// publishInputEOF publishes an EOF to the appropriate input destination.
// See publishInputValue for the nodeID/bareID distinction.
func (e *Executor) publishInputEOF(nodeID, bareID string, inputTopics map[string]string) error {
	if topic, ok := inputTopics[nodeID]; ok {
		return e.pubsub.Publish(topic, pubsub.NewMessage(newEOFValue()))
	}
	return publishNode(e.pubsub, e.topics.For(nodeID), flowv1beta2.RunSnapshot_InputNode_builder{
		Id:     bareID,
		Value:  newEOFValue(),
		Closed: true,
		Phase:  flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
	}.Build())
}
