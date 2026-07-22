package runtime

import (
	"context"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

type (
	Executor struct {
		graph    *Graph
		snapshot Snapshot

		mut sync.RWMutex
	}
	Snapshot struct {
		proto  *flowv1beta1.Env
		values map[string]any
	}
)

func NewExecutor(ctx context.Context, graph *Graph) (*Executor, error) {
	err := graph.Compile(ctx)
	if err != nil {
		return nil, err
	}

	return &Executor{
		graph: graph,
		snapshot: Snapshot{
			proto:  graph.env.Proto(),
			values: graph.Nodes().Values(),
		},
	}, nil
}

func ExecutorFromSpec(ctx context.Context, spec *flowv1beta1.Flow, opts ...Option) (*Executor, error) {
	graph, err := ParseGraph(spec, opts...)
	if err != nil {
		return nil, err
	}

	return NewExecutor(ctx, graph)
}

func (e *Executor) Graph() *Graph {
	return e.graph
}

func (e *Executor) Env() *Env {
	return e.graph.env
}

func (e *Executor) Snapshot() Snapshot {
	e.mut.RLock()
	defer e.mut.RUnlock()

	return e.snapshot
}

func (s Snapshot) Proto() *flowv1beta1.Env {
	return s.proto
}

func (s Snapshot) Values() map[string]any {
	return s.values
}

func (e *Executor) Run(ctx context.Context, cancel context.CancelCauseFunc, opts ...ExecutorOption) error {
	defer cancel(nil)

	// Prepend options needed to run before provided options.
	opts = append([]ExecutorOption{
		shared.WithBeforeSend(func(_ context.Context, node *Node, value any) (_ error) {
			node.setPending()
			return
		}),
		shared.WithAfterRecv(func(_ context.Context, node *Node, value any) (any, error) {
			return node.setValue(value)
		}),
		shared.WithAfterEnd(func(context.Context, shared.Nodes[*Node]) (err error) {
			e.setSnapshot()
			return
		}),
	}, opts...)

	// Append options needed to run after provided options.
	opts = append(opts,
		shared.WithAfterEnd(func(ctx context.Context, nodes shared.Nodes[*Node]) (err error) {
			done, ok := e.IsDone()
			if ok {
				cancel(done)
				return done
			}

			nodes.Reset()

			return
		}),
	)

	err := shared.NewExecutor(e.graph.Graph, opts...).Run(ctx)
	if err != nil {
		cancel(err)
		return context.Cause(ctx)
	}

	return nil
}

func (e *Executor) IsDone() (done *Done, ok bool) {
	e.graph.Nodes().Range(func(_ string, node *Node) bool {
		switch value := node.curr.(type) {
		case *flowv1beta1.Done:
			// Prefer error Dones: keep searching if we already have a non-error.
			if !ok || value.GetIsError() {
				done = NewDone(value)
				ok = true
			}
			if value.GetIsError() {
				return false // error found — stop immediately
			}
		}
		return true
	})
	return
}

func (e *Executor) setSnapshot() {
	// e.mut.Lock()
	// defer e.mut.Unlock()

	e.snapshot.proto = e.Env().Proto()
	e.snapshot.values = e.graph.Nodes().Values()
}

func (e *Executor) applyOptions(opts ...Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
}
