package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/cache"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"golang.org/x/sync/errgroup"
)

// wireEdges creates the per-node wiring (input subscriptions and output topic)
// for every node in the graph by subscribing to source-node topics for each
// downstream edge.
func (e *Executor) wireEdges(ctx context.Context, graph *flowv1beta2.Graph) (map[string]*nodeWiring, error) {
	wiring := make(map[string]*nodeWiring, len(graph.GetNodes()))
	for _, n := range graph.GetNodes() {
		wiring[n.GetId()] = &nodeWiring{
			inputs: make(map[string]<-chan *pubsub.Message),
			topic:  e.topics.For(n.GetId()),
		}
	}
	for _, edge := range graph.GetEdges() {
		ch, err := e.pubsub.Subscribe(ctx, e.topics.For(edge.GetSource()))
		if err != nil {
			return nil, fmt.Errorf("subscribing to %s: %w", edge.GetSource(), err)
		}
		wiring[edge.GetTarget()].inputs[edge.GetSource()] = ch
	}
	return wiring, nil
}

// buildCELEnvAndValidate builds the shared CEL environment with connection
// resolver types registered, then validates request trees against proto
// schemas.
func buildCELEnvAndValidate(graph *flowv1beta2.Graph, resolved map[string]*rpc.Connector) (shared.Env, error) {
	celEnv, err := buildCELEnv(resolved)
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

// runState holds per-execution infrastructure created by setupRunState.
type runState struct {
	genCtx      context.Context
	parkCtx     context.Context
	handlerCtxs map[string]context.Context
	resumeChans map[string]chan *expr.Value
	performStop func()
	cleanup     func() // must be deferred by caller
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
	handlerPub executor.PubSub,
	strategy flowv1beta2.ErrorStrategy,
) (*runState, error) {
	genCtx, genCancel := context.WithCancel(gCtx)
	parkCtx, parkCancel := context.WithCancel(gCtx)

	var stopOnce sync.Once
	performStop := func() {
		stopOnce.Do(func() {
			for _, id := range inputNodeIDs {
				_ = e.pubsub.Publish(e.topics.InputFor(id), pubsub.NewMessage(newEOFValue()))
			}
			genCancel()
			parkCancel()
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

	// Create resume channels for suspend/resume support.
	resumeChans := make(map[string]chan *expr.Value, len(handlers))
	for id := range handlers {
		resumeChans[id] = make(chan *expr.Value, 1)
	}

	// Store per-execution state on the Executor.
	e.mu.Lock()
	e.stopFn = performStop
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
	e.resumeChans = resumeChans
	e.handlers = handlers
	e.parkCancelFn = parkCancel
	e.mu.Unlock()

	// Wire flow_control callbacks.
	for id, h := range handlers {
		np := nodeProtos[id]
		fc, err := nodeFlowControl(celEnv, np)
		if err != nil {
			genCancel()
			parkCancel()
			return nil, fmt.Errorf("compiling flow_control for node %s: %w", id, err)
		}
		if fc != nil {
			if holder, ok := h.(flowControlHolder); ok {
				cb := makeFlowControlCallback(id, fc, performStop, e.Terminate, e.Suspend)
				holder.setFlowControlCallback(cb)
			}
		}
	}

	cleanup := func() {
		genCancel()
		parkCancel()
		e.mu.Lock()
		e.clearRunState()
		e.mu.Unlock()
	}

	return &runState{
		genCtx:      genCtx,
		parkCtx:     parkCtx,
		handlerCtxs: handlerCtxMap,
		resumeChans: resumeChans,
		performStop: performStop,
		cleanup:     cleanup,
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
		nodeID := n.GetId()
		w := wiring[nodeID]
		h := &interactionHandler{
			id:          nodeID,
			inputs:      w.inputs,
			pubsub:      handlerPub,
			topic:       w.topic,
			prompt:      e.interactionPrompt,
			deliver:     make(chan *expr.Value, 1),
			whenProg:    whenProg,
			transforms:  transforms,
			transformPS: e.pubsub,
			adapter:     celEnv.TypeAdapter(),
		}
		interactionHandlers[nodeID] = h
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
	// Compile input transforms and start pipelines before subscriptions.
	inputTopics := make(map[string]string) // nodeID -> transform input topic
	for _, n := range graph.GetNodes() {
		if n.WhichType() == flowv1beta2.Node_Input_case {
			tp, err := compileTransforms(celEnv, n.GetInput().GetTransforms())
			if err != nil {
				return nil, fmt.Errorf("compiling transforms for input %s: %w", n.GetId(), err)
			}
			if tp != nil {
				nodeID := n.GetId()
				sink := func(_ context.Context, val *expr.Value, eof bool) error {
					phase := flowv1beta2.RunSnapshot_PHASE_RUNNING
					if eof {
						phase = flowv1beta2.RunSnapshot_PHASE_SUCCEEDED
					}
					return publishNode(e.pubsub, e.topics.For(nodeID), flowv1beta2.RunSnapshot_InputNode_builder{
						Id:     nodeID,
						Value:  val,
						Closed: eof,
						Phase:  phase,
					}.Build())
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
		nodeID := n.GetId()
		isConstant := inp.GetConstant()
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
			evt := flowv1beta2.InputRequestEvent_builder{Id: nodeID}.Build()
			select {
			case e.inputRequests <- evt:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		if hasResolution && !isConstant {
			// Throttled/cached/default inputs use an inputHandler with a bridge goroutine.
			rawCh := make(chan *expr.Value, 1)

			var publish func(*expr.Value, bool) error
			if topic, ok := inputTopics[nodeID]; ok {
				publish = func(val *expr.Value, eof bool) error {
					return e.pubsub.Publish(topic, pubsub.NewMessage(val))
				}
			} else {
				publish = func(val *expr.Value, eof bool) error {
					phase := flowv1beta2.RunSnapshot_PHASE_RUNNING
					if eof {
						phase = flowv1beta2.RunSnapshot_PHASE_SUCCEEDED
					}
					return publishNode(e.pubsub, e.topics.For(nodeID), flowv1beta2.RunSnapshot_InputNode_builder{
						Id:     nodeID,
						Value:  val,
						Closed: eof,
						Phase:  phase,
					}.Build())
				}
			}

			h := &inputHandler{
				id:         nodeID,
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
			// Simple, constant, or transform-only inputs: bridge directly.
			g.Go(func() error {
				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case msg, ok := <-inputCh:
						if !ok {
							// Subscription closed; send EOF.
							return e.publishInputEOF(nodeID, inputTopics)
						}
						val := msg.Payload.(*expr.Value)
						msg.Ack()
						if isEOFValue(val) {
							return e.publishInputEOF(nodeID, inputTopics)
						}
						if err := e.publishInputValue(nodeID, val, inputTopics); err != nil {
							return err
						}
						if isConstant {
							// Constant inputs: send EOF after first value.
							return e.publishInputEOF(nodeID, inputTopics)
						}
					}
				}
			})
		}
	}

	return inputNodeIDs, nil
}

// buildHandlers compiles and creates handlers for all non-input,
// non-interaction, non-connection nodes in the graph.
func buildHandlers(
	graph *flowv1beta2.Graph,
	celEnv shared.Env,
	resolved map[string]*rpc.Connector,
	cacheStore cache.Cache,
	wiring map[string]*nodeWiring,
	handlerPub executor.PubSub,
	directPub executor.PubSub,
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
		h, err := newHandler(compiled, n.GetId(), w.inputs, handlerPub, w.topic, directPub, celEnv.TypeAdapter())
		if err != nil {
			return nil, fmt.Errorf("creating handler for node %s: %w", n.GetId(), err)
		}
		handlers[n.GetId()] = h
	}
	return handlers, nil
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
						"node", evt.GetId())
					continue
				}
				val := &expr.Value{Kind: &expr.Value_ObjectValue{ObjectValue: evt.GetValue()}}
				if !h.TryDeliver(evt.GetToken(), val) {
					slog.Warn("dropping interaction response with invalid token",
						"node", evt.GetId(),
						"got", evt.GetToken())
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
	resumeChans map[string]chan *expr.Value
	handlerPub  pubsub.Publisher
	parkCtx     context.Context
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
// interception and suspend/resume support. Returns a launchResults whose
// fields should be checked after g.Wait().
func (e *Executor) launchHandlers(g *errgroup.Group, opts launchOpts) *launchResults {
	res := &launchResults{}

	signalStop := func(nodeID string, err error) {
		res.stopErr = fmt.Errorf("node %s: %w", nodeID, err)
		opts.performStop()
	}

	for id, h := range opts.handlers {
		nodeProto := opts.nodeProtos[id]
		handlerCtx := opts.handlerCtxs[id]
		resumeCh := opts.resumeChans[id]

		g.Go(func() error {
			for {
				runErr := h.Run(handlerCtx)
				if runErr == nil {
					if e.awaitResume(id, nodeProto, &handlerCtx, resumeCh, opts) {
						continue
					}
					return nil
				}

				// SuspendError from retry strategy.
				var suspendErr *SuspendError
				if errors.As(runErr, &suspendErr) {
					if err := publishPhaseChange(opts.handlerPub, e.topics.For(id), nodeProto, flowv1beta2.RunSnapshot_PHASE_SUSPENDED, runErr); err != nil {
						slog.Error("publish phase change failed", slog.String("node", id), slog.String("phase", "SUSPENDED"), slog.Any("err", err))
					}
					e.mu.Lock()
					e.suspendedNodes[id] = true
					e.mu.Unlock()
					if e.parkAndResume(id, nodeProto, &handlerCtx, resumeCh, opts) {
						continue
					}
					return nil
				}

				// Context errors: check for operator-initiated suspend.
				if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
					if e.awaitResume(id, nodeProto, &handlerCtx, resumeCh, opts) {
						continue
					}
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
			}
		})
	}

	return res
}

// awaitResume checks whether a node is suspended and, if so, parks its
// goroutine until resumed or the flow stops. It returns true when the
// handler loop should continue (i.e. the node was resumed).
func (e *Executor) awaitResume(
	id string,
	nodeProto *flowv1beta2.Node,
	handlerCtx *context.Context,
	resumeCh <-chan *expr.Value,
	opts launchOpts,
) bool {
	e.mu.Lock()
	isSuspended := e.suspendedNodes[id]
	e.mu.Unlock()
	if !isSuspended {
		return false
	}
	if err := publishPhaseChange(opts.handlerPub, e.topics.For(id), nodeProto, flowv1beta2.RunSnapshot_PHASE_SUSPENDED, nil); err != nil {
		slog.Error("publish phase change failed", slog.String("node", id), slog.String("phase", "SUSPENDED"), slog.Any("err", err))
	}
	return e.parkAndResume(id, nodeProto, handlerCtx, resumeCh, opts)
}

// parkAndResume parks the current goroutine until the node is resumed or the
// park context is cancelled. Returns true when the handler loop should
// continue (node was resumed).
func (e *Executor) parkAndResume(
	id string,
	nodeProto *flowv1beta2.Node,
	handlerCtx *context.Context,
	resumeCh <-chan *expr.Value,
	opts launchOpts,
) bool {
	select {
	case <-opts.parkCtx.Done():
		return false
	case <-resumeCh:
		e.mu.Lock()
		delete(e.suspendedNodes, id)
		parent := opts.gCtx
		if isGenerator(nodeProto) {
			parent = opts.genCtx
		}
		newCtx, newCancel := context.WithCancel(parent)
		e.nodeCtxs[id] = newCancel
		*handlerCtx = newCtx
		e.mu.Unlock()
		if err := publishPhaseChange(opts.handlerPub, e.topics.For(id), nodeProto, flowv1beta2.RunSnapshot_PHASE_PENDING, nil); err != nil {
			slog.Error("publish phase change failed", slog.String("node", id), slog.String("phase", "PENDING"), slog.Any("err", err))
		}
		return true
	}
}
