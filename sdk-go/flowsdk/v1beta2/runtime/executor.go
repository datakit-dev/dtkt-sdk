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
	mu           sync.Mutex
	stopFn       func()                        // graceful stop: EOF injection + generator cancel
	terminateFn  func()                        // immediate cancel: cancels runCtx
	nodeCtxs     map[string]context.CancelFunc // per-node context cancellation
	nodeProtos   map[string]*flowv1beta2.Node  // node protos for phase publishing
	handlerPub   pubsub.Publisher              // publisher for terminal phase events
	terminated   bool                          // true if Terminate() was called
	stoppedNodes map[string]bool               // nodes stopped by StopNode (→ SUCCEEDED)
	handlers     map[string]executor.NodeHandler // handler reference for suspend routing

	// Suspend/resume state, also protected by mu.
	suspendedNodes map[string]bool             // nodes in PHASE_SUSPENDED
	resumeChans    map[string]chan *expr.Value  // resume channels per non-generator handler
	parkCancelFn   context.CancelFunc          // wakes suspended goroutines on stop
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

// StopNode initiates a graceful stop of a single node. The effect depends
// on the node type:
//   - Input: publishes EOF to the input topic
//   - Generator: cancels the generator's context (stops firing)
//   - Other: cancels the node's context (finishes current operation, then stops)
//
// The stopped node transitions to PHASE_SUCCEEDED. Does not trigger
// error_strategy. No-op if the flow is not running or the node ID is unknown.
func (e *Executor) StopNode(nodeID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopFn == nil {
		return // not running
	}

	// For input nodes, inject EOF to close the input.
	if np, ok := e.nodeProtos[nodeID]; ok && np.WhichType() == flowv1beta2.Node_Input_case {
		_ = e.pubsub.Publish(e.topics.InputFor(nodeID), pubsub.NewMessage(newEOFValue()))
		return
	}

	// Track this node as operator-stopped so the handler wrapper publishes
	// PHASE_SUCCEEDED instead of silently swallowing the context error.
	if e.stoppedNodes == nil {
		e.stoppedNodes = make(map[string]bool)
	}
	e.stoppedNodes[nodeID] = true

	// Publish PHASE_SUCCEEDED for the stopped node.
	if np, ok := e.nodeProtos[nodeID]; ok {
		_ = publishTerminalPhase(e.handlerPub, e.topics.For(nodeID), np,
			flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, nil)
	}

	// Cancel the node's context.
	if cancel, ok := e.nodeCtxs[nodeID]; ok {
		cancel()
	}
}

// TerminateNode cancels a single node immediately. The node transitions to
// PHASE_CANCELLED. Does not trigger error_strategy. No-op if the flow is not
// running or the node ID is unknown.
func (e *Executor) TerminateNode(nodeID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.terminateFn == nil {
		return // not running
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

// Suspend suspends all running handler nodes. Generators pause their
// timer/iteration in-place. Non-generators have their context cancelled
// and park until resumed. Input nodes are not affected.
// Safe to call concurrently. No-op if the flow is not running.
func (e *Executor) Suspend() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopFn == nil {
		return
	}
	for id := range e.resumeChans {
		if e.suspendedNodes[id] {
			continue
		}
		e.suspendedNodes[id] = true
		// Generators handle suspend internally (pause timer, stay alive).
		if h, ok := e.handlers[id]; ok {
			if sh, ok := h.(selfSuspendable); ok {
				sh.suspend()
				continue
			}
		}
		// Non-generators: cancel context; awaitResume will park them.
		if cancel, ok := e.nodeCtxs[id]; ok {
			cancel()
		}
	}
}

// SuspendNode suspends a single handler node. Generators pause internally;
// non-generators have their context cancelled. Input nodes are not affected.
// No-op if the flow is not running or the node ID is unknown.
func (e *Executor) SuspendNode(nodeID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopFn == nil {
		return
	}
	if _, ok := e.resumeChans[nodeID]; !ok {
		return
	}
	if e.suspendedNodes[nodeID] {
		return
	}
	e.suspendedNodes[nodeID] = true
	if h, ok := e.handlers[nodeID]; ok {
		if sh, ok := h.(selfSuspendable); ok {
			sh.suspend()
			return
		}
	}
	if cancel, ok := e.nodeCtxs[nodeID]; ok {
		cancel()
	}
}

// Resume resumes all suspended nodes. Generators are signalled to restart
// their timer/iteration. Non-generators receive a resume signal via their
// parkAndResume channel.
// Safe to call concurrently. No-op if no nodes are suspended.
func (e *Executor) Resume() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for id := range e.suspendedNodes {
		if h, ok := e.handlers[id]; ok {
			if sh, ok := h.(selfSuspendable); ok {
				sh.resume()
				delete(e.suspendedNodes, id)
				continue
			}
		}
		if ch, ok := e.resumeChans[id]; ok {
			select {
			case ch <- nil:
			default:
			}
		}
	}
}

// ResumeNode resumes a single suspended node. Generators are signalled
// internally. Non-generators receive a resume signal via parkAndResume.
// No-op if the node is not suspended.
func (e *Executor) ResumeNode(nodeID string, val *expr.Value) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.suspendedNodes[nodeID] {
		return
	}
	if h, ok := e.handlers[nodeID]; ok {
		if sh, ok := h.(selfSuspendable); ok {
			sh.resume()
			delete(e.suspendedNodes, nodeID)
			return
		}
	}
	if ch, ok := e.resumeChans[nodeID]; ok {
		select {
		case ch <- val:
		default:
		}
	}
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
	e.resumeChans = nil
	e.parkCancelFn = nil
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

	// Determine effective error strategy.
	strategy := e.errorStrategy
	if strategy == flowv1beta2.ErrorStrategy_ERROR_STRATEGY_UNSPECIFIED {
		strategy = flowv1beta2.ErrorStrategy_ERROR_STRATEGY_TERMINATE
	}

	// Setup run state: contexts, stop/resume infrastructure, flow_control wiring.
	rs, err := e.setupRunState(gCtx, runCancel, graph, celEnv, handlers, nodeProtos, inputNodeIDs, handlerPub, strategy)
	if err != nil {
		return err
	}
	defer rs.cleanup()

	// Launch all handlers with error interception and suspend/resume support.
	res := e.launchHandlers(g, launchOpts{
		handlers:    handlers,
		nodeProtos:  nodeProtos,
		handlerCtxs: rs.handlerCtxs,
		resumeChans: rs.resumeChans,
		handlerPub:  handlerPub,
		parkCtx:     rs.parkCtx,
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

	// If Terminate() was called, return ErrTerminated
	// (handlers swallow context.Canceled, so err is nil).
	if err == nil {
		e.mu.Lock()
		wasTerminated := e.terminated
		e.mu.Unlock()
		if wasTerminated {
			err = ErrTerminated
		}
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
