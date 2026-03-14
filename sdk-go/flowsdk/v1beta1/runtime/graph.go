package runtime

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	graphlib "github.com/dominikbraun/graph"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"
)

type Graph struct {
	graphlib.Graph[string, *flowv1beta1.Node]
	proto   *flowv1beta1.Graph
	errors  []error
	forward map[string][]string
	reverse map[string][]string
}

func NewGraph(run *Runtime) (*Graph, error) {
	graph := &Graph{
		proto:   &flowv1beta1.Graph{},
		forward: make(map[string][]string),
		reverse: make(map[string][]string),
	}
	graph.Graph = graphlib.NewWithStore(
		GetNodeID,
		graph,
		graphlib.Directed(),
		graphlib.PreventCycles(),
	)

	var err error
	run.nodes.Range(func(id string, node *Node) bool {
		err = graph.Graph.AddVertex(node.proto)
		return err == nil
	})
	if err != nil {
		return nil, fmt.Errorf("graph vertex addition error: %w", err)
	}

	err = run.Parse(GraphVisitor(graph))
	if err != nil {
		return nil, fmt.Errorf("graph parsing error: %w", err)
	}

	if graph.Error() != nil {
		return nil, fmt.Errorf("graph traversal error: %w", graph.Error())
	} else if err = graph.Build(); err != nil {
		return nil, fmt.Errorf("graph build error: %w", err)
	}

	return graph, nil
}

func GraphFromProto(proto *flowv1beta1.Graph) (*Graph, error) {
	graph := &Graph{
		proto:   proto,
		forward: make(map[string][]string),
		reverse: make(map[string][]string),
	}
	graph.Graph = graphlib.NewWithStore(
		GetNodeID,
		graph,
		graphlib.Directed(),
		graphlib.PreventCycles(),
	)

	if err := graph.Build(); err != nil {
		return nil, fmt.Errorf("failed to build graph: %w", err)
	}

	return graph, nil
}

func GraphVisitor(graph *Graph) shared.ParseNodeFunc {
	return func(target string, expr *cel.Ast) {
		ast.PreOrderVisit(ast.NavigateAST(expr.NativeRep()), ast.NewExprVisitor(func(expr ast.Expr) {
			switch expr.Kind() {
			case ast.SelectKind:
				// SelectKind represents a field selection expression.
				if expr.AsSelect().Operand().Kind() == ast.IdentKind {
					source := fmt.Sprintf("%s.%s", expr.AsSelect().Operand().AsIdent(), expr.AsSelect().FieldName())
					if shared.IsNodeID(source) {
						if err := shared.IsValidEdge(source, target); err != nil {
							graph.AddError(err)
						} else if err = graph.Graph.AddEdge(source, target); err != nil {
							if !errors.Is(err, graphlib.ErrEdgeAlreadyExists) {
								graph.AddError(err)
							}
						}
					}
				}
			}
		}))
	}
}

func (g *Graph) Proto() *flowv1beta1.Graph {
	return g.proto
}

// Build constructs the forward/reverse dependency maps and applies transitive reduction.
// This must be called after all vertices and edges are added to prepare the graph for grouping.
func (g *Graph) Build() error {
	preds, err := g.PredecessorMap()
	if err != nil {
		return err
	}

	for targetID, sources := range preds {
		for sourceID := range sources {
			g.forward[targetID] = append(g.forward[targetID], sourceID)
			g.reverse[sourceID] = append(g.reverse[sourceID], targetID)
		}
	}

	graph, err := graphlib.TransitiveReduction(g.Graph)
	if err != nil {
		return err
	}

	g.Graph = graph

	return nil
}

// AddVertex should add the given vertex with the given hash value and vertex properties to the
// graph. If the vertex already exists, it is up to you whether ErrVertexAlreadyExists or no
// error should be returned.
func (g *Graph) AddVertex(id string, node *flowv1beta1.Node, _ graphlib.VertexProperties) error {
	if !slices.ContainsFunc(g.proto.Nodes, func(node *flowv1beta1.Node) bool {
		return node.GetId() == id
	}) {
		g.proto.Nodes = append(g.proto.Nodes, node)
	}
	return nil
}

// Vertex should return the vertex and vertex properties with the given hash value. If the
// vertex doesn't exist, ErrVertexNotFound should be returned.
func (g *Graph) Vertex(id string) (*flowv1beta1.Node, graphlib.VertexProperties, error) {
	idx := slices.IndexFunc(g.proto.Nodes, func(node *flowv1beta1.Node) bool {
		return node.GetId() == id
	})
	if idx >= 0 {
		return g.proto.Nodes[idx], graphlib.VertexProperties{}, nil
	}
	return nil, graphlib.VertexProperties{}, graphlib.ErrVertexNotFound
}

// RemoveVertex should remove the vertex with the given hash value. If the vertex doesn't
// exist, ErrVertexNotFound should be returned. If the vertex has edges to other vertices,
// ErrVertexHasEdges should be returned.
func (g *Graph) RemoveVertex(id string) error {
	idx := slices.IndexFunc(g.proto.Nodes, func(node *flowv1beta1.Node) bool {
		return node.GetId() == id
	})
	if idx == -1 {
		return graphlib.ErrVertexNotFound
	}

	if slices.ContainsFunc(g.proto.Edges, func(edge *flowv1beta1.Edge) bool {
		return edge.GetSource() == id || edge.GetTarget() == id
	}) {
		return graphlib.ErrVertexHasEdges
	}

	g.proto.Nodes = append(g.proto.Nodes[:idx], g.proto.Nodes[idx+1:]...)
	return nil
}

// ListVertices should return all vertices in the graph in a slice.
func (g *Graph) ListVertices() ([]string, error) {
	return util.SliceMap(g.proto.Nodes, GetNodeID), nil
}

// VertexCount should return the number of vertices in the graph. This should be equal to the
// length of the slice returned by ListVertices.
func (g *Graph) VertexCount() (int, error) {
	return len(g.proto.Nodes), nil
}

// AddEdge should add an edge between the vertices with the given source and target hashes.
//
// If either vertex doesn't exit, ErrVertexNotFound should be returned for the respective
// vertex. If the edge already exists, ErrEdgeAlreadyExists should be returned.
func (g *Graph) AddEdge(source, target string, edge graphlib.Edge[string]) error {
	_, _, err := g.Vertex(source)
	if err != nil {
		return err
	}

	_, _, err = g.Vertex(target)
	if err != nil {
		return err
	}

	if slices.ContainsFunc(g.proto.Edges, func(e *flowv1beta1.Edge) bool {
		return e.GetSource() == source && e.GetTarget() == target
	}) {
		return graphlib.ErrEdgeAlreadyExists
	}

	g.proto.Edges = append(g.proto.Edges, &flowv1beta1.Edge{
		Source: source,
		Target: target,
	})

	return nil
}

// UpdateEdge should update the edge between the given vertices with the data of the given
// Edge instance. If the edge doesn't exist, ErrEdgeNotFound should be returned.
func (g *Graph) UpdateEdge(source, target string, edge graphlib.Edge[string]) error {
	idx := slices.IndexFunc(g.proto.Edges, func(e *flowv1beta1.Edge) bool {
		return e.GetSource() == source && e.GetTarget() == target
	})
	if idx >= 0 {
		g.proto.Edges[idx] = &flowv1beta1.Edge{
			Source: edge.Source,
			Target: edge.Target,
		}
		return nil
	}
	return graphlib.ErrEdgeNotFound
}

// RemoveEdge should remove the edge between the vertices with the given source and target
// hashes.
//
// If either vertex doesn't exist, it is up to you whether ErrVertexNotFound or no error should
// be returned. If the edge doesn't exist, it is up to you whether ErrEdgeNotFound or no error
// should be returned.
func (g *Graph) RemoveEdge(source, target string) error {
	idx := slices.IndexFunc(g.proto.Edges, func(e *flowv1beta1.Edge) bool {
		return e.GetSource() == source && e.GetTarget() == target
	})
	if idx >= 0 {
		g.proto.Edges = append(g.proto.Edges[:idx], g.proto.Edges[idx+1:]...)
		return nil
	}
	return graphlib.ErrEdgeNotFound
}

// Edge should return the edge joining the vertices with the given hash values. It should
// exclusively look for an edge between the source and the target vertex, not vice versa. The
// graph implementation does this for undirected graphs itself.
//
// Note that unlike Graph.Edge, this function is supposed to return an Edge[K], i.e. an edge
// that only contains the vertex hashes instead of the vertices themselves.
//
// If the edge doesn't exist, ErrEdgeNotFound should be returned.
func (g *Graph) Edge(source, target string) (_ graphlib.Edge[string], err error) {
	idx := slices.IndexFunc(g.proto.Edges, func(e *flowv1beta1.Edge) bool {
		return e.GetSource() == source && e.GetTarget() == target
	})
	if idx >= 0 {
		edge := g.proto.Edges[idx]
		return graphlib.Edge[string]{
			Source: edge.GetSource(),
			Target: edge.GetTarget(),
		}, nil
	}

	err = graphlib.ErrEdgeNotFound
	return
}

// ListEdges should return all edges in the graph in a slice.
func (g *Graph) ListEdges() ([]graphlib.Edge[string], error) {
	return util.SliceMap(g.proto.Edges, func(edge *flowv1beta1.Edge) graphlib.Edge[string] {
		return graphlib.Edge[string]{
			Source: edge.GetSource(),
			Target: edge.GetTarget(),
		}
	}), nil
}

// Error returns any accumulated errors from graph operations.
func (g *Graph) Error() error {
	if len(g.errors) > 0 {
		return errors.Join(g.errors...)
	}
	return nil
}

// AddError adds an error to the graph's error list.
func (g *Graph) AddError(err error) {
	if err != nil {
		g.errors = append(g.errors, err)
	}
}

// Forward returns the predecessors (dependencies) of the given target nodes.
// If no targets are specified, returns all nodes that have outgoing edges.
func (g *Graph) Forward(targets ...string) (sources []string) {
	if len(targets) == 0 {
		targets = slices.Collect(maps.Keys(g.reverse))
	}

	for _, target := range targets {
		for _, source := range g.forward[target] {
			if !slices.Contains(sources, source) {
				sources = append(sources, source)
			}
		}
	}
	return
}

// Reverse returns the successors (dependents) of the given source nodes.
// If no sources are specified, returns all nodes that have incoming edges.
func (g *Graph) Reverse(sources ...string) (targets []string) {
	if len(sources) == 0 {
		sources = slices.Collect(maps.Keys(g.forward))
	}

	for _, source := range sources {
		for _, target := range g.reverse[source] {
			if !slices.Contains(targets, target) {
				targets = append(targets, target)
			}
		}
	}
	return
}

// Starts returns all nodes that have no dependencies (root nodes).
func (g *Graph) Starts() (ids []string) {
	for source := range g.reverse {
		if len(g.forward[source]) == 0 {
			ids = append(ids, source)
		}
	}
	return
}

// Ends returns all nodes that have no dependents (leaf nodes).
func (g *Graph) Ends() (ids []string) {
	for target := range g.forward {
		if len(g.reverse[target]) == 0 {
			ids = append(ids, target)
		}
	}
	return
}
