package runtime

import (
	"context"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/proto"
)

type (
	Graph struct {
		*shared.Graph[*Node]
		env   *Env
		proto *flowv1beta1.Graph
		opts  []GraphOption
	}
	Pipe struct {
		in, out chan any
	}
)

func NewGraph(env *Env, opts ...Option) (*Graph, error) {
	graph := &Graph{
		env: env,
		proto: &flowv1beta1.Graph{
			Nodes: util.SliceMap(env.nodes.Values(), func(n *Node) *flowv1beta1.Graph_Node {
				return new(flowv1beta1.Graph_Node{
					Ident: n.Ident(),
				})
			}),
		},
	}

	graph.opts = []GraphOption{
		shared.WithDAG[*Node](),
	}
	graph.applyOptions(opts...)

	graph.Graph = shared.NewGraph(graph.opts...)

	err := graph.Graph.Build(func(node *Node, visitor shared.NodeVisitor) error {
		return node.Parse(env, visitor)
	}, env.nodes.Values()...)
	if err != nil {
		return nil, err
	}

	edges := graph.Edges()
	graph.proto.Edges = make([]*flowv1beta1.Graph_Edge, len(edges))
	for idx, edge := range edges {
		graph.proto.Edges[idx] = &flowv1beta1.Graph_Edge{
			Source: edge.Source().Ident(),
			Target: edge.Target().Ident(),
		}
	}

	return graph, nil
}

func NewGraphFromSpec(spec *flowv1beta1.Flow, opts ...Option) (*Graph, error) {
	env, err := NewEnv(ProtoFromSpec(spec), opts...)
	if err != nil {
		return nil, err
	}
	return NewGraph(env, opts...)
}

func NewPipe(in, out chan any) *Pipe {
	return &Pipe{
		in:  in,
		out: out,
	}
}

func (p *Pipe) In() chan<- any {
	return p.in
}

func (p *Pipe) Out() <-chan any {
	return p.out
}

func (p *Pipe) Run(ctx context.Context, exec shared.Exec) (err error) {
	var value any
	for {
		select {
		case <-ctx.Done():
			return nil
		case value = <-p.Out():
		}

		value, err = exec(ctx, value)
		if err != nil {
			return
		}

		select {
		case <-ctx.Done():
			return nil
		case p.In() <- value:
		}
	}
}

func (g *Graph) Env() *Env {
	return g.env
}

func (g *Graph) Proto() *flowv1beta1.Graph {
	return proto.CloneOf(g.proto)
}

func (g *Graph) Execute(ctx context.Context, opts ...ExecutorOption) error {
	for _, node := range g.Nodes() {
		err := node.Compile(ctx, g.env)
		if err != nil {
			return err
		}
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	opts = append([]ExecutorOption{
		shared.WithBeforeSend(func(_ context.Context, node *Node, _ any) (_ error) {
			node.setPending()
			return
		}),
		shared.WithAfterRecv(func(ctx context.Context, node *Node, value any) (err error) {
			_, err = node.setValue(ctx, g.env, value)
			return
		}),
		shared.WithAfterEnd(func(ctx context.Context, _ ...*Node) (err error) {
			done, ok := g.env.IsDone()
			if ok {
				cancel(done)
				return done
			}

			g.env.nodes.reset()

			return
		}),
	}, opts...)

	err := shared.NewExecutor(g.Graph, opts...).Run(ctx, g.env)
	if err != nil {
		cancel(err)
		return context.Cause(ctx)
	}

	return nil
}

func (g *Graph) applyOptions(opts ...Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}
}
