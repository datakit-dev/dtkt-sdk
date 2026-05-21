package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/cache"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
	"golang.org/x/sync/errgroup"
)

// wireEdges creates the per-node wiring (input subscriptions and output
// topic) for every node in the graph by subscribing to source-node
// topics for each downstream edge. Edges whose source has cache:true
// are subscribed the same way; the consumer-side recv applies
// last-seen / non-blocking semantics on those refs (see
// nodeRef.recvCached).
func (e *Executor) wireEdges(ctx context.Context, graph *flowv1beta2.Graph) (map[string]*nodeWiring, error) {
	wiring := make(map[string]*nodeWiring, len(graph.GetNodes()))
	cached := make(map[string]bool, len(graph.GetNodes()))
	for _, n := range graph.GetNodes() {
		wiring[n.GetId()] = &nodeWiring{
			inputs: make(map[string]<-chan *pubsub.Message),
			topic:  e.topics.For(n.GetId()),
		}
		if isCachedProducer(n) {
			cached[n.GetId()] = true
		}
	}
	for _, edge := range graph.GetEdges() {
		source := edge.GetSource()
		target := edge.GetTarget()
		ch, err := e.pubsub.Subscribe(ctx, e.topics.For(source))
		if err != nil {
			return nil, fmt.Errorf("subscribing to %s: %w", source, err)
		}
		wiring[target].inputs[source] = ch
		if cached[source] {
			if wiring[target].cachedSources == nil {
				wiring[target].cachedSources = make(map[string]bool)
			}
			wiring[target].cachedSources[source] = true
		}
	}
	return wiring, nil
}

// buildCELEnvAndValidate builds the flow-global shared.Env (one union
// resolver: spec-ordered connectors + platform), then validates request
// trees against proto schemas. `platform` is the SDK platform-types
// resolver (typically api.GlobalResolver() via Executor's
// WithPlatformResolver Option; nil falls through to that default inside
// buildCELEnv).
func buildCELEnvAndValidate(graph *flowv1beta2.Graph, resolved map[string]*rpc.Connector, platform shared.Resolver) (shared.Env, error) {
	ordered := orderedConnectorsFromGraph(graph, resolved)
	celEnv, err := buildCELEnv(ordered, platform)
	if err != nil {
		return nil, fmt.Errorf("building CEL environment: %w", err)
	}
	if len(resolved) > 0 {
		resolvers := make(map[string]shared.Resolver, len(resolved))
		for id, c := range resolved {
			resolvers[id] = c.Resolver
		}
		if result := lintRequestSchemas(graph, resolvers, celEnv); result.HasErrors() {
			return nil, fmt.Errorf("schema validation: %w", result.Errors())
		}
	}
	return celEnv, nil
}

// orderedConnectorsFromGraph returns the resolved connectors in the
// flow-spec declared order, by iterating the graph's Connection nodes
// (which the graph builder preserves in spec order). Connections that
// appear in the graph but have no resolved entry are skipped (the lint
// path surfaces those separately); connections in `resolved` that have
// no graph node (unusual but possible if an action references a
// connection not declared via a Connection node) are appended at the
// end in deterministic name order so they still participate.
func orderedConnectorsFromGraph(graph *flowv1beta2.Graph, resolved map[string]*rpc.Connector) []*rpc.Connector {
	out := make([]*rpc.Connector, 0, len(resolved))
	seen := make(map[string]struct{}, len(resolved))
	for _, n := range graph.GetNodes() {
		if n.WhichType() != flowv1beta2.Node_Connection_case {
			continue
		}
		id := n.GetConnection().GetId()
		if c, ok := resolved[id]; ok {
			out = append(out, c)
			seen[id] = struct{}{}
		}
	}
	if len(out) == len(resolved) {
		return out
	}
	// Append any resolved-but-not-in-graph connectors in deterministic
	// (sorted) order so flow runs are repeatable.
	extras := make([]string, 0, len(resolved)-len(seen))
	for id := range resolved {
		if _, ok := seen[id]; !ok {
			extras = append(extras, id)
		}
	}
	sort.Strings(extras)
	for _, id := range extras {
		out = append(out, resolved[id])
	}
	return out
}

// runState holds per-execution infrastructure created by setupRunState.
type runState struct {
	genCtx       context.Context
	handlerCtxs  map[string]context.Context
	gracefulStop func()
	operatorStop func()
	cleanup      func() // must be deferred by caller
}

// setupRunState creates the stop/suspend/resume infrastructure, per-node
// contexts, and stores execution state on the Executor. The returned
// runState.cleanup must be deferred by the caller.
func (e *Executor) setupRunState(
	gCtx context.Context,
	runCancel context.CancelFunc,
	graph *flowv1beta2.Graph,
	celEnv shared.Env,
	handlers map[string]executor.NodeHandler,
	nodeProtos map[string]*flowv1beta2.Node,
	inputNodeIDs []string,
	wiring map[string]*nodeWiring,
	handlerPub executor.PubSub,
	strategy flowv1beta2.ErrorStrategy,
) (*runState, error) {
	genCtx, genCancel := context.WithCancel(gCtx)

	// Interactions with no upstream edges have nothing to drain on the
	// EOF cascade -- their handler's act.Resolve returns immediately
	// with empty vars and they sit in promptAndWait forever. Even on
	// graceful stop, these need an explicit stopCh.
	upstreamlessInteractions := make(map[string]bool)
	for id, np := range nodeProtos {
		if np.WhichType() != flowv1beta2.Node_Interaction_case {
			continue
		}
		if w := wiring[id]; w == nil || len(w.inputs) == 0 {
			upstreamlessInteractions[id] = true
		}
	}

	// stopCore implements the parts of stop that are always done,
	// regardless of who initiated it: publish input EOFs, signal stopCh
	// on suspended handlers and generators, and signal stopCh on
	// upstreamless interactions (which can't drain via EOF cascade).
	// Idempotent across repeated calls -- the EOF publish and
	// requestStop are safe to repeat.
	var coreOnce sync.Once
	stopCore := func() {
		coreOnce.Do(func() {
			for _, id := range inputNodeIDs {
				_ = e.pubsub.Publish(e.topics.InputFor(id), pubsub.NewMessage(newEOFValue()))
			}
			e.mu.Lock()
			for id := range e.suspendedNodes {
				if sh, ok := e.handlers[id].(selfStoppable); ok {
					sh.requestStop()
				}
				delete(e.suspendedNodes, id)
			}
			for id, h := range e.handlers {
				np := e.nodeProtos[id]
				isInteraction := np.WhichType() == flowv1beta2.Node_Interaction_case
				if !isGenerator(np) && (!isInteraction || !upstreamlessInteractions[id]) {
					continue
				}
				if sh, ok := h.(selfStoppable); ok {
					sh.requestStop()
				}
			}
			e.mu.Unlock()
		})
	}

	// gracefulStop is what flow_control.stop_when calls. It does the
	// core stop steps (EOF cascade for inputs/var/action/stream/output;
	// stopCh for generators/suspended/upstreamless interactions) and
	// intentionally does NOT signal stopCh on interactions with
	// upstream edges -- those drain via EOF cascade once their current
	// iteration (which may be parked in promptAndWait awaiting an
	// in-flight response) completes. This preserves the "drain buffered,
	// then exit" contract that flow_control.stop_when implies.
	gracefulStop := func() {
		stopCore()
	}

	// operatorStop is what e.Stop() (the user-facing operator command)
	// calls. It does the core stop steps AND additionally signals
	// stopCh on every interaction handler -- including those with
	// upstream edges -- so a parked promptAndWait unblocks immediately.
	// Operator stop means "stop now, don't keep prompting"; without
	// this an interaction parked waiting on user response would never
	// exit (the EOF cascade can't reach it past act.Resolve).
	var aggressiveOnce sync.Once
	operatorStop := func() {
		stopCore()
		aggressiveOnce.Do(func() {
			e.mu.Lock()
			for id, h := range e.handlers {
				np := e.nodeProtos[id]
				if np.WhichType() != flowv1beta2.Node_Interaction_case {
					continue
				}
				if upstreamlessInteractions[id] {
					continue // already handled by stopCore
				}
				if sh, ok := h.(selfStoppable); ok {
					sh.requestStop()
				}
			}
			e.mu.Unlock()
		})
	}

	// Create per-node contexts for StopNode/TerminateNode support.
	nodeCtxs := make(map[string]context.CancelFunc, len(handlers))
	handlerCtxMap := make(map[string]context.Context, len(handlers))
	for id, nodeProto := range nodeProtos {
		var parent context.Context
		if isGenerator(nodeProto) {
			parent = genCtx
		} else {
			parent = gCtx
		}
		nodeCtx, nodeCancel := context.WithCancel(parent)
		nodeCtxs[id] = nodeCancel
		handlerCtxMap[id] = nodeCtx
	}

	// Store per-execution state on the Executor.
	e.mu.Lock()
	e.stopFn = operatorStop
	e.terminateFn = func() {
		e.mu.Lock()
		e.terminated = true
		e.mu.Unlock()
		runCancel()
	}
	e.nodeCtxs = nodeCtxs
	e.nodeProtos = nodeProtos
	e.handlerPub = handlerPub
	e.stoppedNodes = make(map[string]bool)
	e.suspendedNodes = make(map[string]bool)
	e.terminalNodes = make(map[string]bool)
	e.handlers = handlers
	e.mu.Unlock()

	// Wire flow_control, node_control, and self-suspend callbacks.
	for id, h := range handlers {
		np := nodeProtos[id]
		holder, hasHolder := h.(lifecycleHolder)

		fc, err := nodeFlowControl(celEnv, np)
		if err != nil {
			genCancel()
			return nil, fmt.Errorf("compiling flow_control for node %s: %w", id, err)
		}
		if fc != nil && hasHolder {
			holder.setFlowControlCallback(
				makeFlowControlCallback(id, fc, gracefulStop, e.Terminate, e.Suspend),
			)
		}

		nc, err := nodeNodeControl(celEnv, np)
		if err != nil {
			genCancel()
			return nil, fmt.Errorf("compiling node_control for node %s: %w", id, err)
		}
		if nc != nil && hasHolder {
			nodeID := id
			holder.setNodeControlCallback(
				makeNodeControlCallback(nodeID, nc,
					func() { e.StopNode(nodeID) },
					func() { e.TerminateNode(nodeID) },
					func() { e.SuspendNode(nodeID) },
				),
			)
		}

		// RPC handlers (unary, server_stream, client_stream, bidi_stream) may
		// surface *SuspendError from their retry strategy. Give them a
		// bookkeeping callback so they can self-suspend without involving the
		// launchHandlers wrapper. The callback marks suspendedNodes[id]=true
		// and publishes PHASE_SUSPENDED; the handler then parks via its own
		// waitForResume.
		//
		// Idempotent against suspendedNodes: if NC.suspend (via the lifecycle
		// callback) already marked this node suspended in the same iteration,
		// skip the duplicate PHASE_SUSPENDED publish. Avoids two SUSPENDED
		// state events on the wire when retry.suspend and NC.suspend both
		// fire on the same iteration.
		if rs, ok := h.(retrySuspender); ok {
			nodeID := id
			nodeProto := np
			rs.setSelfSuspendCallback(func(suspendErr error) {
				e.mu.Lock()
				if e.suspendedNodes[nodeID] {
					e.mu.Unlock()
					return
				}
				e.suspendedNodes[nodeID] = true
				e.mu.Unlock()
				if err := publishPhaseChange(handlerPub, e.topics.For(nodeID), nodeProto,
					flowv1beta2.RunSnapshot_PHASE_SUSPENDED, suspendErr); err != nil {
					slog.Error("publish phase change failed",
						slog.String("node", nodeID),
						slog.String("phase", "SUSPENDED"),
						slog.Any("err", err))
				}
			})
		}
	}

	cleanup := func() {
		genCancel()
		e.mu.Lock()
		e.clearRunState()
		e.mu.Unlock()
	}

	return &runState{
		genCtx:       genCtx,
		handlerCtxs:  handlerCtxMap,
		gracefulStop: gracefulStop,
		operatorStop: operatorStop,
		cleanup:      cleanup,
	}, nil
}

// every connection referenced by Action/Stream calls through the ConnectorProvider.
func (e *Executor) resolveConnections(ctx context.Context, graph *flowv1beta2.Graph) (map[string]*rpc.Connector, error) {
	if e.connectors == nil {
		return nil, nil
	}

	// Phase 1: collect Connection node specs for package/services info.
	connSpecs := make(map[string]*flowv1beta2.Connection)
	for _, n := range graph.GetNodes() {
		if n.WhichType() == flowv1beta2.Node_Connection_case {
			spec := n.GetConnection()
			connSpecs[spec.GetId()] = spec
		}
	}

	// Phase 2: resolve all referenced connections (from calls + spec nodes).
	resolved := make(map[string]*rpc.Connector)
	for _, n := range graph.GetNodes() {
		var connID string
		switch n.WhichType() {
		case flowv1beta2.Node_Action_case:
			if call := n.GetAction().GetCall(); call != nil {
				connID = call.GetConnection()
			}
		case flowv1beta2.Node_Stream_case:
			if call := n.GetStream().GetCall(); call != nil {
				connID = call.GetConnection()
			}
		}
		if connID == "" || resolved[connID] != nil {
			continue
		}
		var pkg rpc.Package
		var services []string
		if spec, ok := connSpecs[connID]; ok {
			if p := spec.GetPackage(); p != nil {
				pkg = p
			}
			services = spec.GetServices()
		}
		c, err := e.connectors.GetConnector(ctx, connID, pkg, services)
		if err != nil {
			return nil, fmt.Errorf("resolving connection %q: %w", connID, err)
		}
		resolved[connID] = c
	}
	return resolved, nil
}

// buildInteractionHandlers compiles and constructs interaction handlers for all
// Interaction nodes in the graph. Returns the handlers keyed by node ID.
func (e *Executor) buildInteractionHandlers(
	graph *flowv1beta2.Graph,
	celEnv shared.Env,
	wiring map[string]*nodeWiring,
	handlerPub executor.PubSub,
) (map[string]*interactionHandler, error) {
	interactionHandlers := make(map[string]*interactionHandler)
	for _, n := range graph.GetNodes() {
		if n.WhichType() != flowv1beta2.Node_Interaction_case {
			continue
		}
		if e.interactionPrompt == nil {
			return nil, fmt.Errorf("interaction node %s requires WithInteractions option", n.GetId())
		}
		inter := n.GetInteraction()
		transforms, err := compileTransforms(celEnv, inter.GetTransforms())
		if err != nil {
			return nil, fmt.Errorf("compiling transforms for %s: %w", n.GetId(), err)
		}
		var whenProg cel.Program
		if w := inter.GetWhen(); w != "" {
			whenProg, err = compileCEL(celEnv, w)
			if err != nil {
				return nil, fmt.Errorf("compiling when for %s: %w", n.GetId(), err)
			}
		}
		// Compile per-input title/description CEL once. Programs run
		// against activation vars at prompt time; resolved values are
		// shipped as Interaction.Input clones on the
		// InteractionRequestEvent so responders never see raw CEL.
		formInputs, err := compileInteractionInputs(celEnv, inter.GetInputs())
		if err != nil {
			return nil, fmt.Errorf("compiling input CEL for %s: %w", n.GetId(), err)
		}
		nodeID := n.GetId()     // Format B (e.g. "interactions.confirm") - cross-category routing
		bareID := inter.GetId() // Format A (e.g. "confirm") - per-event protobuf id field contract
		w := wiring[nodeID]
		cb := &cacheBackend{
			cacheDeps: cacheDeps{
				cachedSources: w.cachedSources,
				allCached:     w.hasOnlyCachedDeps(),
			},
		}
		e.cacheBackends[nodeID] = cb
		h := &interactionHandler{
			id:          bareID,
			inputs:      w.inputs,
			pubsub:      handlerPub,
			topic:       w.topic,
			prompt:      e.interactionPrompt,
			deliver:     make(chan *expr.Value, 1),
			whenProg:    whenProg,
			transforms:  transforms,
			transformPS: e.pubsub,
			env:         celEnv,
			cache:       cb,
			formInputs:  formInputs,
		}
		h.initSuspendable()
		h.initStoppable()
		// Keyed by bareID because incoming InteractionResponseEvent.id is the
		// bare-id form per its protobuf pattern.
		interactionHandlers[bareID] = h
	}
	return interactionHandlers, nil
}

// setupInputBridges compiles input transforms, starts transform pipelines,
// subscribes to external input topics, and launches bridge goroutines that
// forward values into the graph. Returns the list of input node IDs (needed
// by the stop infrastructure to inject EOFs).
func (e *Executor) setupInputBridges(
	ctx context.Context,
	g *errgroup.Group,
	graph *flowv1beta2.Graph,
	celEnv shared.Env,
) ([]string, error) {
	// First pass: create the cacheBackend for every input so the
	// transform-pipeline sinks (set up in the next pass) can capture it.
	// For cache:true inputs with transforms, markCaptured must fire in
	// the sink (post-transforms) so the cached value matches what
	// consumers actually see.
	inputCBs := make(map[string]*cacheBackend, len(graph.GetNodes()))
	for _, n := range graph.GetNodes() {
		if n.WhichType() != flowv1beta2.Node_Input_case {
			continue
		}
		nodeID := n.GetId()
		cache := n.GetInput().GetCache()
		cb := &cacheBackend{cacheCapture: cacheCapture{enabled: cache}}
		inputCBs[nodeID] = cb
		if cache {
			e.cacheBackends[nodeID] = cb
		}
	}

	// Second pass: compile transforms and start pipelines. The sink
	// publishes post-transform values to consumers AND, for cache:true
	// inputs, marks captured on the first such publish.
	inputTopics := make(map[string]string) // nodeID -> transform input topic
	for _, n := range graph.GetNodes() {
		if n.WhichType() == flowv1beta2.Node_Input_case {
			tp, err := compileTransforms(celEnv, n.GetInput().GetTransforms())
			if err != nil {
				return nil, fmt.Errorf("compiling transforms for input %s: %w", n.GetId(), err)
			}
			if tp != nil {
				nodeID := n.GetId()            // Format B - topic routing
				bareID := n.GetInput().GetId() // Format A - per-event protobuf id field contract
				cb := inputCBs[nodeID]
				sink := func(_ context.Context, val *expr.Value, eof bool) error {
					// cache:true: only the FIRST post-transform value
					// reaches consumers. The bridge may have fed several
					// inputs before captured flips; drop subsequent sink
					// emissions silently so consumers see exactly one value.
					if !eof && cb.isCaptured() {
						return nil
					}
					phase := flowv1beta2.RunSnapshot_PHASE_RUNNING
					if eof {
						phase = flowv1beta2.RunSnapshot_PHASE_SUCCEEDED
					}
					if err := publishNode(e.pubsub, e.topics.For(nodeID), flowv1beta2.RunSnapshot_InputNode_builder{
						Id:     bareID,
						Value:  val,
						Closed: eof,
						Phase:  phase,
					}.Build()); err != nil {
						return err
					}
					if !eof {
						cb.markCaptured()
					}
					return nil
				}
				topic, err := tp.Start(ctx, g, e.pubsub, e.topics.For(nodeID), sink, nil)
				if err != nil {
					return nil, fmt.Errorf("starting input pipeline for %s: %w", nodeID, err)
				}
				inputTopics[nodeID] = topic
			}
		}
	}

	// Collect input node IDs (needed by STOP error strategy to inject EOFs).
	var inputNodeIDs []string
	for _, n := range graph.GetNodes() {
		if n.WhichType() == flowv1beta2.Node_Input_case {
			inputNodeIDs = append(inputNodeIDs, n.GetId())
		}
	}

	// Subscribe to external input topics and start per-input bridge goroutines.
	for _, n := range graph.GetNodes() {
		if n.WhichType() != flowv1beta2.Node_Input_case {
			continue
		}
		inp := n.GetInput()
		nodeID := n.GetId()   // Format B - topic routing, handler map key
		bareID := inp.GetId() // Format A - per-event protobuf id field contract
		throttle := rateToDuration(inp.GetThrottle())
		defVal := inputTypeDefault(inp)
		cache := inp.GetCache()

		// Inject throttle when cache or default is set but no explicit throttle.
		// Use the caller-provided default if set, otherwise fall back to the
		// built-in minimum so the default/cache fallback can always fire.
		if throttle == 0 && (cache || defVal != nil) {
			if e.defaultInputThrottle != nil {
				throttle = rateToDuration(e.defaultInputThrottle)
			} else {
				throttle = minInputThrottle
			}
		}

		hasResolution := throttle > 0 || cache || defVal != nil

		// Subscribe to the external input topic.
		inputCh, err := e.pubsub.Subscribe(ctx, e.topics.InputFor(nodeID))
		if err != nil {
			return nil, fmt.Errorf("subscribing to input topic for %s: %w", nodeID, err)
		}

		// Notify external consumers that this input is ready for values.
		if e.inputRequests != nil {
			evt := flowv1beta2.InputRequestEvent_builder{Id: bareID}.Build()
			select {
			case e.inputRequests <- evt:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		if hasResolution {
			// Throttled/cached/default inputs use an inputHandler with a bridge goroutine.
			rawCh := make(chan *expr.Value, 1)

			cb := inputCBs[nodeID]
			hasTransforms := false

			// rawPublish is the actual delivery: to the transform input
			// topic if the input has transforms, otherwise direct to the
			// consumer-facing topic with a full InputNode message.
			var rawPublish func(*expr.Value, bool) error
			if topic, ok := inputTopics[nodeID]; ok {
				hasTransforms = true
				rawPublish = func(val *expr.Value, _ bool) error {
					return e.pubsub.Publish(topic, pubsub.NewMessage(val))
				}
			} else {
				rawPublish = func(val *expr.Value, eof bool) error {
					phase := flowv1beta2.RunSnapshot_PHASE_RUNNING
					if eof {
						phase = flowv1beta2.RunSnapshot_PHASE_SUCCEEDED
					}
					return publishNode(e.pubsub, e.topics.For(nodeID), flowv1beta2.RunSnapshot_InputNode_builder{
						Id:     bareID,
						Value:  val,
						Closed: eof,
						Phase:  phase,
					}.Build())
				}
			}

			// publish wraps rawPublish with cache:true drain-and-skip.
			// markCaptured fires here only when the input has no transforms
			// (rawPublish goes directly to consumers). With transforms,
			// the pipeline's sink owns markCaptured so the cached value
			// matches what consumers actually see post-transforms.
			publish := func(val *expr.Value, eof bool) error {
				if !eof && cb.isCaptured() {
					return nil
				}
				if err := rawPublish(val, eof); err != nil {
					return err
				}
				if !eof && !hasTransforms {
					cb.markCaptured()
				}
				return nil
			}

			h := &inputHandler{
				id:         bareID,
				raw:        rawCh,
				publish:    publish,
				throttle:   throttle,
				cache:      cache,
				defaultVal: defVal,
			}
			g.Go(func() error { return h.Run(ctx) })

			// Bridge: read from PubSub subscription, forward to inputHandler's rawCh.
			g.Go(func() error {
				defer close(rawCh)
				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case msg, ok := <-inputCh:
						if !ok {
							return nil
						}
						val := msg.Payload.(*expr.Value)
						msg.Ack()
						if isEOFValue(val) {
							return nil
						}
						select {
						case rawCh <- val:
						case <-ctx.Done():
							return ctx.Err()
						}
					}
				}
			})
		} else {
			// Simple or transform-only inputs: bridge directly.
			g.Go(func() error {
				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case msg, ok := <-inputCh:
						if !ok {
							// Subscription closed; send EOF.
							return e.publishInputEOF(nodeID, bareID, inputTopics)
						}
						val := msg.Payload.(*expr.Value)
						msg.Ack()
						if isEOFValue(val) {
							return e.publishInputEOF(nodeID, bareID, inputTopics)
						}
						if err := e.publishInputValue(nodeID, bareID, val, inputTopics); err != nil {
							return err
						}
					}
				}
			})
		}
	}

	return inputNodeIDs, nil
}

// buildHandlers compiles and creates handlers for all non-input,
// non-interaction, non-connection nodes in the graph. registry collects
// the per-node cacheBackend pointers so the executor can find them later
// (e.g. for ClearCache).
func buildHandlers(
	graph *flowv1beta2.Graph,
	celEnv shared.Env,
	resolved map[string]*rpc.Connector,
	cacheStore cache.Cache,
	wiring map[string]*nodeWiring,
	handlerPub executor.PubSub,
	directPub executor.PubSub,
	registry map[string]*cacheBackend,
) (map[string]executor.NodeHandler, error) {
	handlers := make(map[string]executor.NodeHandler, len(graph.GetNodes()))
	for _, n := range graph.GetNodes() {
		switch n.WhichType() {
		case flowv1beta2.Node_Input_case, flowv1beta2.Node_Interaction_case, flowv1beta2.Node_Connection_case:
			continue // handled separately
		}
		compiled, err := compileNode(celEnv, n, resolved, cacheStore)
		if err != nil {
			return nil, fmt.Errorf("compiling node %s: %w", n.GetId(), err)
		}
		w := wiring[n.GetId()]
		cb := &cacheBackend{
			cacheCapture: cacheCapture{enabled: isCachedProducer(n)},
			cacheDeps: cacheDeps{
				cachedSources: w.cachedSources,
				allCached:     w.hasOnlyCachedDeps(),
			},
		}
		registry[n.GetId()] = cb
		// bareID (Format A) is the spec id stored in handler.id for use in
		// protobuf event/snapshot fields whose validator is the bare-id
		// pattern. n.GetId() is fully-qualified (Format B) and stays as the
		// cross-category handler map key in `handlers` below.
		h, err := newHandler(compiled, bareNodeID(n), w.inputs, handlerPub, w.topic, directPub, celEnv, cb)
		if err != nil {
			return nil, fmt.Errorf("creating handler for node %s: %w", n.GetId(), err)
		}
		handlers[n.GetId()] = h
	}
	return handlers, nil
}

// bareNodeID returns the bare/simple node id (Format A) from a Node by
// reading the id off the typed inner spec. The bare id is the form
// required by the buf-validate pattern on every per-node id field in
// state.proto / events.proto (`^[a-zA-Z][a-zA-Z0-9_]*$`).
//
// Use n.GetId() (Format B, e.g. "vars.x") for graph-level operations
// (topic routing, cross-category handler map, edges) where category
// disambiguation is required.
func bareNodeID(n *flowv1beta2.Node) string {
	switch n.WhichType() {
	case flowv1beta2.Node_Connection_case:
		return n.GetConnection().GetId()
	case flowv1beta2.Node_Input_case:
		return n.GetInput().GetId()
	case flowv1beta2.Node_Generator_case:
		return n.GetGenerator().GetId()
	case flowv1beta2.Node_Var_case:
		return n.GetVar().GetId()
	case flowv1beta2.Node_Action_case:
		return n.GetAction().GetId()
	case flowv1beta2.Node_Stream_case:
		return n.GetStream().GetId()
	case flowv1beta2.Node_Interaction_case:
		return n.GetInteraction().GetId()
	case flowv1beta2.Node_Output_case:
		return n.GetOutput().GetId()
	}
	return ""
}

// startInteractionDemux routes InteractionResponseEvents from the external
// response channel to the correct interaction handler. It runs outside the
// errgroup because it must outlive individual handlers. The context
// cancellation from errgroup.Wait signals the demux to shut down.
func startInteractionDemux(
	ctx context.Context,
	responseCh <-chan *flowv1beta2.InteractionResponseEvent,
	handlers map[string]*interactionHandler,
) {
	if responseCh == nil {
		return
	}
	go func() {
		defer func() {
			for _, h := range handlers {
				h.Close()
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-responseCh:
				if !ok {
					return
				}
				h, found := handlers[evt.GetId()]
				if !found {
					slog.Warn("dropping interaction response for unknown node",
						slog.String("node", evt.GetId()))
					continue
				}
				val := h.responseValue(evt.GetValue())
				if !h.TryDeliver(evt.GetToken(), val) {
					slog.Warn("dropping interaction response with invalid token",
						slog.String("node", evt.GetId()),
						slog.String("got", evt.GetToken()))
				}
			}
		}
	}()
}

// launchOpts holds the parameters for launchHandlers.
type launchOpts struct {
	handlers    map[string]executor.NodeHandler
	nodeProtos  map[string]*flowv1beta2.Node
	handlerCtxs map[string]context.Context
	handlerPub  pubsub.Publisher
	gCtx        context.Context
	genCtx      context.Context
	strategy    flowv1beta2.ErrorStrategy
	performStop func()
}

// launchResults holds shared error state written by handler goroutines
// and read after errgroup.Wait().
type launchResults struct {
	mu          sync.Mutex
	stopErr     error
	continueErr error
}

// launchHandlers launches all node handlers in the errgroup with error
// interception. Suspend/resume is handled entirely inside each handler via
// the suspendableMixin: NC and operator SuspendNode signal via mixin.suspendCh,
// retry-strategy SuspendError is caught by the handler itself which then
// calls selfSuspend + waitForResume. The wrapper here only deals with
// post-Run terminal disposition (clean exit, errored, terminated) and
// flow-level error_strategy.
func (e *Executor) launchHandlers(g *errgroup.Group, opts launchOpts) *launchResults {
	res := &launchResults{}

	signalStop := func(nodeID string, err error) {
		res.stopErr = fmt.Errorf("node %s: %w", nodeID, err)
		opts.performStop()
	}

	for id, h := range opts.handlers {
		nodeProto := opts.nodeProtos[id]
		handlerCtx := opts.handlerCtxs[id]

		g.Go(func() error {
			defer func() {
				// Mark this node as terminal once its handler returns. Operator
				// commands (StopNode/TerminateNode/SuspendNode) check this and
				// no-op for terminal nodes -- prevents stale flag accumulation
				// (e.g. SuspendNode on an already-SUCCEEDED node previously
				// left suspendedNodes[id]=true with no handler to ever clear).
				e.mu.Lock()
				if e.terminalNodes != nil {
					e.terminalNodes[id] = true
				}
				// A node that exits is no longer suspended either.
				delete(e.suspendedNodes, id)
				e.mu.Unlock()
			}()

			runErr := h.Run(handlerCtx)
			if runErr == nil {
				return nil
			}

			// Context errors: handler exited because something cancelled its
			// context (per-node Terminate, parent ctx cancel, etc.). Quiet exit.
			if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
				return nil
			}

			// Publish PHASE_ERRORED for the failing node.
			if err := publishTerminalPhase(opts.handlerPub, e.topics.For(id), nodeProto, flowv1beta2.RunSnapshot_PHASE_ERRORED, runErr); err != nil {
				slog.Error("publish terminal phase failed", slog.String("node", id), slog.String("phase", "ERRORED"), slog.Any("err", err))
			}

			var termErr *TerminateError
			if errors.As(runErr, &termErr) {
				return fmt.Errorf("node %s: %w", id, runErr)
			}

			switch opts.strategy {
			case flowv1beta2.ErrorStrategy_ERROR_STRATEGY_CONTINUE:
				nodeErr := fmt.Errorf("node %s: %w", id, runErr)
				res.mu.Lock()
				res.continueErr = errors.Join(res.continueErr, nodeErr)
				res.mu.Unlock()
				return nil
			case flowv1beta2.ErrorStrategy_ERROR_STRATEGY_STOP:
				signalStop(id, runErr)
				return nil
			default: // TERMINATE
				return fmt.Errorf("node %s: %w", id, runErr)
			}
		})
	}

	return res
}
