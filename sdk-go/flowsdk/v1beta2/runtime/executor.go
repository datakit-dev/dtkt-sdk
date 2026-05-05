package runtime

import (
	"context"
	"errors"
	"sync"
	"time"

	expr "cel.dev/expr"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/cache"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"golang.org/x/sync/errgroup"
)

// ErrTerminated is returned by Execute when the flow is terminated via
// the Terminate method (operator-initiated cancellation).
var ErrTerminated = errors.New("flow terminated by operator")

// nodeWiring holds the subscription channels and output topic for a single node.
type nodeWiring struct {
	inputs map[string]<-chan *pubsub.Message
	topic  string
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
	outbox               outbox.Outbox
	defaultInputThrottle *flowv1beta2.Rate
	interactionPrompt    chan<- *flowv1beta2.InteractionRequestEvent
	interactionResponse  <-chan *flowv1beta2.InteractionResponseEvent
	inputRequests        chan<- *flowv1beta2.InputRequestEvent
	errorStrategy        flowv1beta2.ErrorStrategy

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
}

type Option func(*Executor)

func WithConnectors(conns rpc.ConnectorProvider) Option {
	return func(e *Executor) {
		e.connectors = conns
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

func WithOutbox(o outbox.Outbox) Option {
	return func(e *Executor) {
		e.outbox = o
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
}

func (e *Executor) Execute(ctx context.Context, graph *flowv1beta2.Graph) error {
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
		outboxSub = outbox.NewSubscriber(e.outbox, 10*time.Millisecond)
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
	celEnv, err := buildCELEnvAndValidate(graph, resolved)
	if err != nil {
		return err
	}

	handlers, err := buildHandlers(graph, celEnv, resolved, e.cache, wiring, handlerPub, e.pubsub)
	if err != nil {
		return err
	}

	// Build interaction handlers.
	interactionHandlers, err := e.buildInteractionHandlers(graph, celEnv, wiring, handlerPub)
	if err != nil {
		return err
	}
	for id, h := range interactionHandlers {
		handlers[id] = h
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
	rs, err := e.setupRunState(gCtx, runCancel, graph, celEnv, handlers, nodeProtos, inputNodeIDs, handlerPub, strategy)
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
		performStop: rs.performStop,
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
	if pubErr := publishFlowState(handlerPub, e.topics.Flow(), terminalState.Build()); pubErr != nil && err == nil {
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
func (e *Executor) publishInputValue(nodeID string, val *expr.Value, inputTopics map[string]string) error {
	if topic, ok := inputTopics[nodeID]; ok {
		return e.pubsub.Publish(topic, pubsub.NewMessage(val))
	}
	return publishNode(e.pubsub, e.topics.For(nodeID), flowv1beta2.RunSnapshot_InputNode_builder{
		Id:    nodeID,
		Value: val,
		Phase: flowv1beta2.RunSnapshot_PHASE_RUNNING,
	}.Build())
}

// publishInputEOF publishes an EOF to the appropriate input destination.
func (e *Executor) publishInputEOF(nodeID string, inputTopics map[string]string) error {
	if topic, ok := inputTopics[nodeID]; ok {
		return e.pubsub.Publish(topic, pubsub.NewMessage(newEOFValue()))
	}
	return publishNode(e.pubsub, e.topics.For(nodeID), flowv1beta2.RunSnapshot_InputNode_builder{
		Id:     nodeID,
		Value:  newEOFValue(),
		Closed: true,
		Phase:  flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
	}.Build())
}
