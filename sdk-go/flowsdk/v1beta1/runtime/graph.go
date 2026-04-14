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
	"google.golang.org/protobuf/proto"
)

type Graph struct {
	graphlib.Graph[string, *flowv1beta1.Node]
	proto *flowv1beta1.Graph

	groups  [][]string
	forward map[string][]string
	reverse map[string][]string

	errors []error
}

func NewGraph(env *Env) (*Graph, error) {
	graph := &Graph{
		proto: &flowv1beta1.Graph{
			Nodes: env.nodes.Protos(),
		},
		forward: make(map[string][]string),
		reverse: make(map[string][]string),
	}
	graph.Graph = graphlib.NewWithStore(
		GetNodeID,
		graph,
		graphlib.Directed(),
		graphlib.PreventCycles(),
	)

	err := env.nodes.Parse(env, graph.Visit)
	if err != nil {
		return nil, err
	}

	for _, node := range env.nodes {
		err := graph.Graph.AddVertex(node.proto)
		if err != nil {
			return nil, fmt.Errorf("add graph vertex: %w", err)
		}
	}

	if graph.Error() != nil {
		return nil, fmt.Errorf("build graph error: %w", graph.Error())
	}

	err = graph.Build()
	if err != nil {
		return nil, err
	}

	return graph, nil
}

func GraphFromRuntime(run *Runtime) (*Graph, error) {
	env, err := run.env()
	if err != nil {
		return nil, err
	}

	return NewGraph(env)
}

func GraphFromSpec(spec *flowv1beta1.Flow, opts ...Option) (*Graph, error) {
	run := ProtoFromSpec(spec)
	env, err := NewEnv(run, opts...)
	if err != nil {
		return nil, err
	}

	return NewGraph(env)
}

func (g *Graph) Visit(target string, expr *cel.Ast) {
	ast.PreOrderVisit(ast.NavigateAST(expr.NativeRep()), ast.NewExprVisitor(func(expr ast.Expr) {
		switch expr.Kind() {
		case ast.SelectKind:
			// SelectKind represents a field selection expression.
			if expr.AsSelect().Operand().Kind() == ast.IdentKind {
				source := fmt.Sprintf("%s.%s", expr.AsSelect().Operand().AsIdent(), expr.AsSelect().FieldName())
				if shared.IsNodeID(source) {
					if err := shared.IsValidEdge(source, target); err != nil {
						g.AddError(err)
					} else if err = g.Graph.AddEdge(source, target); err != nil {
						if !errors.Is(err, graphlib.ErrEdgeAlreadyExists) {
							g.AddError(err)
						}
					}
				}
			}
		}
	}))
}

func (g *Graph) Build() (err error) {
	err = g.computePreds()
	if err != nil {
		return err
	}

	err = g.computeGroups()
	if err != nil {
		return err
	}

	g.Graph, err = graphlib.TransitiveReduction(g.Graph)

	return
}

func (g *Graph) computePreds() error {
	preds, err := g.Graph.PredecessorMap()
	if err != nil {
		return err
	}

	for targetID, sources := range preds {
		for sourceID := range sources {
			g.forward[targetID] = append(g.forward[targetID], sourceID)
			g.reverse[sourceID] = append(g.reverse[sourceID], targetID)
		}
	}

	return nil
}

func (g *Graph) computeGroups() error {
	// Use topological ordering to group independent nodes
	// Get all nodes in topological order
	order, err := graphlib.TopologicalSort(g.Graph)
	if err != nil {
		return fmt.Errorf("topological sort error: %w", err)
	}

	// Track which nodes have been processed
	processed := make(map[string]bool)

	// Process nodes in topological order, grouping independent nodes
	// We use the forward/reverse maps that were saved before transitive reduction
	for len(processed) < len(order) {
		var nodeIDs []string

		// Find all nodes whose dependencies have been processed
		for _, nodeID := range order {
			if processed[nodeID] {
				continue
			}

			// Check if all dependencies (predecessors) are processed
			// Use the forward map which contains predecessors
			deps := g.Forward(nodeID)
			allDepsProcessed := true
			for _, depID := range deps {
				if !processed[depID] {
					allDepsProcessed = false
					break
				}
			}

			if allDepsProcessed {
				_, _, err := g.Vertex(nodeID)
				if err != nil {
					return err
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
			g.groups = append(g.groups, nodeIDs)
		} else {
			// Should never happen with a valid DAG
			break
		}
	}

	return nil
}

func (g *Graph) Proto() *flowv1beta1.Graph {
	return proto.CloneOf(g.proto)
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
