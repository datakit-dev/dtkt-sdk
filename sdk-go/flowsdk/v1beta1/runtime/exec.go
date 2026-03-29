package runtime

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	graphlib "github.com/dominikbraun/graph"
	"golang.org/x/sync/errgroup"
)

type (
	Executor struct {
		runtime *Runtime
		graph   *Graph
		groups  [][]string

		emittedIds,
		triggeredIds,
		eofIds []string

		sendAckCh chan string
		eofCh     chan string
		readyCh   chan struct{}
		proceedCh chan struct{}

		start func() error
		grp   errgroup.Group
	}
)

func NewExecutor(runtime *Runtime, graph *Graph) (*Executor, error) {
	// Use topological ordering to group independent nodes
	// Get all nodes in topological order
	order, err := graphlib.TopologicalSort(graph.Graph)
	if err != nil {
		return nil, fmt.Errorf("topological sort error: %w", err)
	}

	// Track which nodes have been processed
	processed := make(map[string]bool)
	groups := [][]string{}

	// Process nodes in topological order, grouping independent nodes
	// We use the forward/reverse maps that were saved before transitive reduction
	for len(processed) < len(order) {
		nodeIDs := []string{}

		// Find all nodes whose dependencies have been processed
		for _, nodeID := range order {
			if processed[nodeID] {
				continue
			}

			// Check if all dependencies (predecessors) are processed
			// Use the forward map which contains predecessors
			deps := graph.Forward(nodeID)
			allDepsProcessed := true
			for _, depID := range deps {
				if !processed[depID] {
					allDepsProcessed = false
					break
				}
			}

			if allDepsProcessed {
				_, _, err := graph.Vertex(nodeID)
				if err != nil {
					return nil, err
				}

				nodeIDs = append(nodeIDs, nodeID)
			}
		}

		// Mark all nodes in this group as processed AFTER building the group
		// This ensures nodes in the same group don't see each other as already processed
		for _, nodeID := range nodeIDs {
			processed[nodeID] = true
		}

		if len(nodeIDs) > 0 {
			groups = append(groups, nodeIDs)
		} else {
			// Should never happen with a valid DAG
			break
		}
	}

	exec := &Executor{
		runtime: runtime,
		graph:   graph,
		groups:  groups,

		sendAckCh: make(chan string, 1),
		eofCh:     make(chan string, 1),
		readyCh:   make(chan struct{}, 1),
		proceedCh: make(chan struct{}), // unbuffered: exec.Reset() syncs with ack loop
	}

	exec.start = sync.OnceValue(func() error {
		exec.grp.Go(func() error {
			// Kick off the first cycle: trigger nodes with no unmet event deps.
			if err := exec.sendTriggersForReadyNodes(); err != nil {
				return err
			}

			for {
				select {
				case <-runtime.ctx.Done():
					return context.Cause(runtime.ctx)
				case id := <-exec.eofCh:
					if !slices.Contains(exec.eofIds, id) {
						exec.eofIds = append(exec.eofIds, id)
					}
					// Abandon the in-flight partial cycle: a retired stream cannot
					// complete it, and readyCh must not fire with stale ack state.
					exec.emittedIds = exec.emittedIds[:0]
					exec.triggeredIds = exec.triggeredIds[:0]
				case id := <-exec.sendAckCh:
					if !slices.Contains(exec.emittedIds, id) {
						exec.emittedIds = append(exec.emittedIds, id)
					}

					activeCount := len(runtime.sendChs) - len(exec.eofIds)
					cycleComplete := activeCount > 0 && len(exec.emittedIds) == activeCount
					if cycleComplete {
						select {
						case <-runtime.ctx.Done():
							return context.Cause(runtime.ctx)
						case exec.readyCh <- struct{}{}:
						}

						exec.emittedIds = exec.emittedIds[:0]
						exec.triggeredIds = exec.triggeredIds[:0]

						// Wait for exec.Reset() — ensures runtime.Reset() has run before we
						// send the next cycle's triggers (prevents bidiEcho.Recv from reading
						// a stale SUCCESS value from the previous cycle).
						select {
						case <-runtime.ctx.Done():
							return context.Cause(runtime.ctx)
						case <-exec.proceedCh:
						}
					}

					if err := exec.sendTriggersForReadyNodes(); err != nil {
						return err
					}
				}
			}
		})

		return nil
	})

	return exec, nil
}

func (e *Executor) Ready() <-chan struct{} {
	return e.readyCh
}

// Reset resets all node states and signals the ack loop to send the next
// cycle's triggers. It must be called after each readyCh cycle completes.
func (e *Executor) Reset() {
	e.runtime.Reset()
	select {
	case e.proceedCh <- struct{}{}:
	case <-e.runtime.ctx.Done():
	}
}

func (e *Executor) Start() error {
	for id, node := range e.runtime.nodes.ToNativeMap() {
		err := node.startExecution(e)
		if err != nil {
			return fmt.Errorf("%s: start execution: %w", id, err)
		}
	}

	return e.start()
}

func (e *Executor) sendTriggersForReadyNodes() error {
	for id, recvCh := range e.runtime.recvChs {
		// Required inputs (no default, not nullable) must receive a real external
		// value; a nil trigger would just spin through their recv loop. Inputs with
		// a default or nullable flag are fine: nil → use default path.
		if strings.HasPrefix(id, shared.InputPrefix+".") {
			node, ok := e.runtime.nodes.Load(id)
			if ok {
				if inp, ok := node.RuntimeNode.(*spec.Input); ok && inp.IsRequired() {
					continue
				}
			}
		}

		// Skip streams that have retired via EOF.
		if slices.Contains(e.eofIds, id) {
			continue
		}

		if slices.Contains(e.triggeredIds, id) {
			continue
		}

		// Skip if any event-producing predecessor has not yet emitted.
		// Required inputs are a special case: they are never executor-triggered
		// (no entry in emittedIds), so we check their node state instead.
		// - If a required input has Completed (CurrValue is set via applyValue
		//   at ack time), it does not block the trigger.
		// - If a required input has NOT yet received a value (UNSPECIFIED state),
		//   it DOES block the trigger — the downstream node must wait.
		if slices.ContainsFunc(e.graph.Forward(id), func(depID string) bool {
			_, hasSendCh := e.runtime.sendChs[depID]
			if !hasSendCh || slices.Contains(e.eofIds, depID) {
				return false
			}
			if node, ok := e.runtime.nodes.Load(depID); ok {
				if inp, ok := node.RuntimeNode.(*spec.Input); ok && inp.IsRequired() {
					// Block only if the required input has not yet received a value.
					return !node.Completed()
				}
			}
			return !slices.Contains(e.emittedIds, depID)
		}) {
			continue
		}

		select {
		case <-e.runtime.ctx.Done():
			return context.Cause(e.runtime.ctx)
		case recvCh <- nil:
			e.triggeredIds = append(e.triggeredIds, id)
		}
	}
	return nil
}

func (e *Executor) RunSnapshot(callback func(*flowv1beta1.Runtime) error) (err error) {
	err = e.Start()
	if err != nil {
		return
	}

	var isDone bool
loop:
	for {
		select {
		case <-e.runtime.Context().Done():
			err = context.Cause(e.runtime.Context())
			break loop
		case <-e.readyCh:
			err = e.Eval()
			if err != nil {
				break loop
			}

			err = callback(e.runtime.Proto())
			if err != nil {
				break loop
			}

			err, isDone = IsRuntimeDone(e.runtime)
			if isDone {
				break loop
			}

			e.Reset()
		}
	}

	e.runtime.cancel(err)
	<-e.runtime.ctx.Done()

	return e.grp.Wait()
}

func (e *Executor) RunValues(callback func(map[string]any) error) (err error) {
	err = e.Start()
	if err != nil {
		return
	}

	var (
		values map[string]any
		isDone bool
	)
loop:
	for {
		select {
		case <-e.runtime.Context().Done():
			err = context.Cause(e.runtime.Context())
			break loop
		case <-e.readyCh:
			values, err = e.GetValues()
			if err != nil {
				break loop
			}

			err = callback(values)
			if err != nil {
				break loop
			}

			err, isDone = IsRuntimeDone(e.runtime)
			if isDone {
				break loop
			}

			e.Reset()
		}
	}

	e.runtime.cancel(err)
	<-e.runtime.ctx.Done()

	return e.grp.Wait()
}

func (e *Executor) Eval() error {
	for _, group := range e.groups {
		// Execute all nodes in this group in parallel
		exec, _ := errgroup.WithContext(e.runtime.Context())
		for _, id := range group {
			exec.Go(func() error {
				_, err := e.runtime.GetValue(id)
				return err
			})
		}

		// Wait for all nodes in this group to complete before moving to next group
		if err := exec.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func (e *Executor) GetValues() (map[string]any, error) {
	// Execute groups SEQUENTIALLY to respect dependencies
	// but execute nodes WITHIN each group in parallel

	var values util.SyncMap[string, any]
	for _, group := range e.groups {
		// Execute all nodes in this group in parallel
		exec, _ := errgroup.WithContext(e.runtime.Context())
		for _, id := range group {
			exec.Go(func() error {
				value, err := e.runtime.GetValue(id)
				if err != nil {
					return err
				}
				values.Store(id, value)
				return nil
			})
		}

		// Wait for all nodes in this group to complete before moving to next group
		if err := exec.Wait(); err != nil {
			return nil, err
		}
	}

	return values.ToNativeMap(), nil
}

func (e *Executor) GetValuesReset() (map[string]any, error) {
	// Reset all nodes after evaluation to ensure clean state
	values, err := e.GetValues()
	if err != nil {
		return nil, err
	}

	e.runtime.Reset()

	return values, nil
}

func (e *Executor) LoadGroup(idx int) ([]string, bool) {
	if idx >= 0 && idx < len(e.groups) {
		return e.groups[idx], true
	}
	return nil, false
}

func (e *Executor) RangeGroups(callback func(int, []string) bool) {
	for idx, group := range e.groups {
		if !callback(idx, group) {
			return
		}
	}
}

func (e *Executor) Proto() *flowv1beta1.Groups {
	return &flowv1beta1.Groups{
		Groups: util.SliceMap(e.groups, func(group []string) *flowv1beta1.Groups_Group {
			return &flowv1beta1.Groups_Group{
				Ids: group,
			}
		}),
	}
}
