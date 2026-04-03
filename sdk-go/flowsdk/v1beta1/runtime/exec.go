package runtime

import (
	"context"
	"fmt"
	"slices"
	"sync"

	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/cel-go/common/types"
	"golang.org/x/sync/errgroup"
)

type Executor struct {
	run   *Runtime
	graph *Graph

	emittedIds,
	triggeredIds,
	eofIds []string

	sendAckCh chan string
	eofCh     chan string
	readyCh   chan struct{}
	proceedCh chan struct{}

	start func() error
	grp   errgroup.Group
	mut   sync.Mutex
}

func NewExecutor(run *Runtime, graph *Graph) (*Executor, error) {
	err := run.nodes.Compile(run, GraphVisitor(graph))
	if err != nil {
		return nil, err
	}

	return &Executor{
		run:   run,
		graph: graph,

		sendAckCh: make(chan string, 1),
		eofCh:     make(chan string, 1),
		readyCh:   make(chan struct{}, 1),
		proceedCh: make(chan struct{}), // unbuffered: exec.Reset() syncs with ack loop
	}, nil
}

func (e *Executor) Ready() <-chan struct{} {
	return e.readyCh
}

// Reset resets all node states and signals the ack loop to send the next
// cycle's triggers. It must be called after each readyCh cycle completes.
func (e *Executor) Reset() {
	e.run.Reset()
	select {
	case e.proceedCh <- struct{}{}:
	case <-e.run.ctx.Done():
	}
}

func (e *Executor) Start() error {
	for _, node := range e.run.nodes.Values() {
		e.startNodeExecution(e.run, node)
	}

	e.mut.Lock()
	if e.start == nil {
		e.start = sync.OnceValue(func() error {
			e.grp.Go(func() error {
				// Kick off the first cycle: trigger nodes with no unmet event deps.
				if err := e.sendTriggersForReadyNodes(); err != nil {
					return err
				}

				for {
					select {
					case <-e.run.ctx.Done():
						return context.Cause(e.run.ctx)
					case id := <-e.eofCh:
						if !slices.Contains(e.eofIds, id) {
							e.eofIds = append(e.eofIds, id)
						}
						// Abandon the in-flight partial cycle: a retired stream cannot
						// complete it, and readyCh must not fire with stale ack state.
						e.emittedIds = e.emittedIds[:0]
						e.triggeredIds = e.triggeredIds[:0]
					case id := <-e.sendAckCh:
						if !slices.Contains(e.emittedIds, id) {
							e.emittedIds = append(e.emittedIds, id)
						}

						activeCount := len(e.run.sendChs) - len(e.eofIds)
						cycleComplete := activeCount > 0 && len(e.emittedIds) == activeCount
						if cycleComplete {
							select {
							case <-e.run.ctx.Done():
								return context.Cause(e.run.ctx)
							case e.readyCh <- struct{}{}:
							}

							e.emittedIds = e.emittedIds[:0]
							e.triggeredIds = e.triggeredIds[:0]

							// Wait for exec.Reset() — ensures runtime.Reset() has run before we
							// send the next cycle's triggers (prevents bidiEcho.Recv from reading
							// a stale SUCCESS value from the previous cycle).
							select {
							case <-e.run.ctx.Done():
								return context.Cause(e.run.ctx)
							case <-e.proceedCh:
							}
						}

						if err := e.sendTriggersForReadyNodes(); err != nil {
							return err
						}
					}
				}
			})
			return nil
		})
	}
	e.mut.Unlock()

	return e.start()
}

func (e *Executor) startNodeExecution(run *Runtime, node *Node) {
	if node.hasRecv {
		e.grp.Go(func() error {
			return node.recv(run, node.recvCh)
		})
	}

	if node.hasSend {
		e.grp.Go(func() error {
			return node.send(run, node.sendCh)
		})

		e.grp.Go(func() error {
			for {
				select {
				case <-run.ctx.Done():
					return context.Cause(run.ctx)
				case value, ok := <-node.sendCh:
					if !ok {
						return fmt.Errorf("%s: event channel closed", node.id)
					}

					// Runtime_EOF is a transport sentinel: the stream closed cleanly.
					// Retire this node without surfacing a spurious output cycle.
					if _, isEOF := value.Value().(*flowv1beta1.Runtime_EOF); isEOF {
						node.applyValue(run, types.NullValue) //nolint:errcheck
						select {
						case <-run.ctx.Done():
							return context.Cause(run.ctx)
						case e.eofCh <- node.id:
						}
						return nil
					}

					// Apply the value immediately so that CurrValue is set on the
					// proto before any downstream stream-recv nodes (e.g. BidiStream)
					// read it via getValue(). The Eval() path checks SUCCESS state and
					// returns the cached value without re-reading n.valueCh.
					if _, err := node.applyValue(run, value); err != nil {
						return err
					}

					select {
					case <-run.ctx.Done():
						return context.Cause(run.ctx)
					case e.sendAckCh <- node.id:
					}
					// Wait for Reset() before accepting the next value. This prevents
					// a second external event from overwriting CurrValue before the
					// current cycle completes and evalReq has read the value.
					select {
					case <-run.ctx.Done():
						return context.Cause(run.ctx)
					case <-node.resetCh:
					}
				}
			}
		})
	}
}

func (e *Executor) sendTriggersForReadyNodes() error {
	for id, recvCh := range e.run.recvChs {
		// Required inputs (no default, not nullable) must receive a real external
		// value; a nil trigger would just spin through their recv loop. Inputs with
		// a default or nullable flag are fine: nil → use default path.
		node, ok := e.run.nodes.Load(id)
		if ok && node.isRequiredInput {
			continue
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
			_, hasSendCh := e.run.sendChs[depID]
			if !hasSendCh || slices.Contains(e.eofIds, depID) {
				return false
			}
			if node, ok := e.run.nodes.Load(depID); ok && node.isRequiredInput {
				// Block only if the required input has not yet received a value.
				return !node.Completed()
			}
			return !slices.Contains(e.emittedIds, depID)
		}) {
			continue
		}

		select {
		case <-e.run.ctx.Done():
			return context.Cause(e.run.ctx)
		case recvCh <- nil:
			e.triggeredIds = append(e.triggeredIds, id)
		}
	}
	return nil
}

func (e *Executor) Eval() error {
	for _, group := range e.graph.groups {
		// Execute all nodes in this group in parallel
		exec, _ := errgroup.WithContext(e.run.Context())
		for _, id := range group {
			exec.Go(func() error {
				_, err := e.run.GetValue(id)
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
	for _, group := range e.graph.groups {
		// Execute all nodes in this group in parallel
		exec, _ := errgroup.WithContext(e.run.Context())
		for _, id := range group {
			exec.Go(func() error {
				value, err := e.run.GetValue(id)
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

	e.run.Reset()

	return values, nil
}

func (e *Executor) LoadGroup(idx int) ([]string, bool) {
	if idx >= 0 && idx < len(e.graph.groups) {
		return e.graph.groups[idx], true
	}
	return nil, false
}

func (e *Executor) RangeGroups(callback func(int, []string) bool) {
	for idx, group := range e.graph.groups {
		if !callback(idx, group) {
			return
		}
	}
}
