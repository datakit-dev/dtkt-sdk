package runtime

import (
	"context"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/impl"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

type (
	Executor struct {
		graph    *Graph
		control  *impl.Control
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

	control, err := impl.NewControl(graph.env, graph.proto.GetSpec().GetControl())
	if err != nil {
		return nil, err
	}

	return &Executor{
		graph:   graph,
		control: control,
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

func (e *Executor) Run(ctx context.Context, opts ...ExecutorOption) error {
	opts = append([]ExecutorOption{
		WithBeforeSend(func(_ context.Context, node *Node, value any) (_ error) {
			node.setPending()
			return
		}),
		WithAfterRecv(func(_ context.Context, node *Node, value any) (any, error) {
			return node.setValue(value)
		}),
		WithAfterEnd(func(context.Context, shared.GraphNodes[*Node]) (_ error) {
			e.setSnapshot()
			return
		}),
	}, opts...)

	return shared.NewExecutor(e.graph.Graph, opts...).Run(ctx)
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

func (e *Executor) ShouldStart(ctx context.Context) (bool, error) {
	return e.control.ShouldStart(ctx, e.graph.env)
}

func (e *Executor) ShouldStop(ctx context.Context) (bool, error) {
	return e.control.ShouldStop(ctx, e.graph.env)
}

func (e *Executor) IsDone() (done *Done, ok bool) {
	e.mut.RLock()
	defer e.mut.RUnlock()

	for _, value := range e.snapshot.values {
		if !ok {
			done, ok = IsDone(value)
			if ok {
				return
			}
		}
	}

	return
}

func (e *Executor) setSnapshot() {
	e.mut.Lock()
	defer e.mut.Unlock()

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
