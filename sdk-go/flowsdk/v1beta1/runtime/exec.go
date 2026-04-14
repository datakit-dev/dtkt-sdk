package runtime

import (
	"context"
	"fmt"
	"slices"
	"sync"

	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/cel-go/common/types/ref"
	"golang.org/x/sync/errgroup"
)

type Executor struct {
	run   *Runtime
	graph *Graph

	emittedIds,
	triggeredIds,
	eofIds []string

	// Internal node accounting — populated during startNodeExecution, independent
	// of the runtime's channel maps which may be accessed externally (e.g. to
	// inject values). Using executor-owned sets ensures activeCount and trigger
	// iteration are not affected by external channel creation.
	sendNodeIds []string
	recvNodeChs map[string]chan any

	sendAckCh chan string
	eofCh     chan string
	readyCh   chan struct{}
	proceedCh chan struct{}

	start func() error
	group errgroup.Group
	mut   sync.Mutex
}

func NewExecutor(run *Runtime, graph *Graph) (*Executor, error) {
	err := run.nodes.Compile(run, graph.Visit)
	if err != nil {
		return nil, err
	}

	return &Executor{
		run:   run,
		graph: graph,

		recvNodeChs: make(map[string]chan any),

		sendAckCh: make(chan string, 1),
		eofCh:     make(chan string, 1),
		readyCh:   make(chan struct{}, 1),
		proceedCh: make(chan struct{}), // unbuffered: exec.Reset() syncs with ack loop
	}, nil
}

func (e *Executor) Ready() <-chan struct{} {
	return e.readyCh
}

func (e *Executor) Start() error {
	for _, node := range e.run.nodes.Values() {
		e.startNodeExecution(e.run, node)
	}

	e.mut.Lock()
	if e.start == nil {
		e.start = sync.OnceValue(func() error {
			e.group.Go(func() error {
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

						activeCount := len(e.sendNodeIds) - len(e.eofIds)
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

func (e *Executor) Eval() error {
	for _, group := range e.graph.groups {
		if len(group) == 1 {
			_, err := e.run.GetValue(group[0])
			if err != nil {
				return err
			}
		} else {
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
	}
	return nil
}

func (e *Executor) GetValues() (map[string]any, error) {
	// Execute groups SEQUENTIALLY to respect dependencies
	// but execute nodes WITHIN each group in parallel

	var values util.SyncMap[string, any]
	for _, group := range e.graph.groups {
		if len(group) == 1 {
			value, err := e.run.GetValue(group[0])
			if err != nil {
				return nil, err
			}

			values.Store(group[0], value)
		} else {
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
	}

	return values.ToNativeMap(), nil
}

// Reset resets all node states and signals the ack loop to send the next
// cycle's triggers. It must be called after each readyCh cycle completes.
func (e *Executor) Reset() {
	e.run.nodes.Range(func(id string, node *Node) bool {
		node.Reset()
		return true
	})

	select {
	case e.proceedCh <- struct{}{}:
	case <-e.run.ctx.Done():
	}
}

func (e *Executor) startNodeExecution(run *Runtime, node *Node) {
	if node.hasRecv {
		e.recvNodeChs[node.id] = node.recvCh
		e.group.Go(func() error {
			err := node.recv(run, node.recvCh)
			if err != nil {
				run.Cancel(err)
			}
			return err
		})
	}

	if node.hasSend {
		e.sendNodeIds = append(e.sendNodeIds, node.id)
		e.group.Go(func() error {
			err := node.send(run, node.sendCh)
			if err != nil {
				run.Cancel(err)
			}
			return err
		})

		e.group.Go(func() error {
			env, err := run.env()
			if err != nil {
				run.Cancel(err)
				return err
			}

			for {
				var (
					value ref.Val
					isEOF bool
				)

				if cached, ok := node.hasCached(); ok {
					// Cached inputs re-use their last value each cycle without
					// re-reading sendCh, so they ack even when no new external
					// event arrives.
					value = cached
				} else {
					select {
					case <-run.ctx.Done():
						return context.Cause(run.ctx)
					case v, ok := <-node.sendCh:
						if !ok {
							return fmt.Errorf("%s: send channel closed", node.id)
						}

						// Runtime_EOF is a transport sentinel: the stream closed cleanly.
						// Retire this node without surfacing a spurious output cycle.
						_, isEOF = v.Value().(*flowv1beta1.Runtime_EOF)
						value = v
					}
				}

				// Apply the value immediately so that CurrValue is set on the
				// proto before any downstream stream-recv nodes (e.g. BidiStream)
				// read it via getValue(). The Eval() path checks SUCCESS state and
				// returns the cached value without re-reading n.valueCh.
				if _, err := node.applyValue(env, value); err != nil {
					run.Cancel(err)
					return err
				}

				if isEOF {
					select {
					case <-run.ctx.Done():
						return context.Cause(run.ctx)
					case e.eofCh <- node.id:
					}
					// The send goroutine has exited and sendCh will never produce
					// another value; returning here prevents the goroutine from
					// blocking on <-node.sendCh indefinitely.
					return nil
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
		})
	}
}

func (e *Executor) sendTriggersForReadyNodes() error {
	for id, recvCh := range e.recvNodeChs {
		if node, ok := e.run.nodes.Load(id); ok {
			// Required inputs (no default, not nullable) must receive a real external
			// value; a nil trigger would just spin through their recv loop. Inputs with
			// a default or nullable flag are fine: nil → use default path.
			if node.isRequired() {
				continue
			}

			// Skip cache-enabled inputs that already have a captured value — the ack
			// goroutine re-emits the cached value each cycle without reading sendCh, so
			// sending a nil trigger would pump a value through the Recv→valueCh→Send→
			// sendCh pipeline that nobody ever drains, eventually deadlocking.
			if _, cached := node.hasCached(); cached {
				continue
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
			if !slices.Contains(e.sendNodeIds, depID) || slices.Contains(e.eofIds, depID) {
				return false
			}

			if node, ok := e.run.nodes.Load(depID); ok {
				// A cached predecessor always has its value ready and will ack
				// immediately via the hasCached fast path — never blocks a trigger.
				if _, cached := node.hasCached(); cached {
					return false
				}
				// A required input blocks until it has received its first value.
				if node.isRequired() {
					return !node.Completed()
				}
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
