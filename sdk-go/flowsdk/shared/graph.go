package shared

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	graphlib "github.com/dominikbraun/graph"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"
)

type (
	Graph[T Node] struct {
		graphlib.Graph[string, T]

		hash   graphlib.Hash[string, T]
		traits graphlib.Traits
		store  graphlib.Store[string, T]

		nodes Nodes[T]
		edges Edges

		groups  [][]string
		forward map[string][]string
		reverse map[string][]string

		errors []error
	}
	GraphOption[T Node] func(*Graph[T])
	GraphTrait          func(*graphlib.Traits)
)

func NewGraph[T Node](opts ...GraphOption[T]) *Graph[T] {
	graph := &Graph[T]{
		forward: map[string][]string{},
		reverse: map[string][]string{},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(graph)
		}
	}

	if graph.hash == nil {
		graph.hash = func(node T) string {
			return node.Ident()
		}
	}

	if graph.store != nil {
		graph.Graph = graphlib.NewWithStore(graph.hash, graph.store, func(traits *graphlib.Traits) {
			*traits = graph.traits
		})
	} else {
		graph.Graph = graphlib.New(graph.hash, func(traits *graphlib.Traits) {
			*traits = graph.traits
		})
	}

	return graph
}

func WithNodeHash[T Node](hash graphlib.Hash[string, T]) GraphOption[T] {
	return func(g *Graph[T]) {
		g.hash = hash
	}
}

func WithNodes[T Node](nodes []T) GraphOption[T] {
	return func(g *Graph[T]) {
		g.nodes = nodes
	}
}

func WithEdges[T Node](edges []*Edge) GraphOption[T] {
	return func(g *Graph[T]) {
		g.edges = edges
	}
}

func WithDAG[T Node]() GraphOption[T] {
	return func(g *Graph[T]) {
		graphlib.Directed()(&g.traits)
		graphlib.PreventCycles()(&g.traits)
	}
}

func WithTraits[T Node](traits ...GraphTrait) GraphOption[T] {
	return func(g *Graph[T]) {
		for _, trait := range traits {
			if trait != nil {
				trait(&g.traits)
			}
		}
	}
}

func WithStore[T Node](store graphlib.Store[string, T]) GraphOption[T] {
	return func(g *Graph[T]) {
		g.store = store
	}
}

func (g *Graph[T]) Parse(nodes []SpecNode, parser NodeParser[T]) (err error) {
	for _, node := range nodes {
		node, err := parser(node, g.Visitor)
		if err != nil {
			return fmt.Errorf("parse node: %w", err)
		}

		err = g.AddVertex(node)
		if err != nil {
			return fmt.Errorf("add vertex: %w", err)
		}

		g.nodes = append(g.nodes, node)
	}

	err = errors.Join(g.errors...)
	if err != nil {
		return err
	}

	for _, edge := range g.edges {
		err = g.AddEdge(edge.SourceNode(), edge.TargetNode())
		if err != nil {
			return err
		}
	}

	err = g.computePreds()
	if err != nil {
		return fmt.Errorf("compute preds: %w", err)
	}

	err = g.computeGroups()
	if err != nil {
		return fmt.Errorf("compute groups: %w", err)
	}

	if g.traits.IsAcyclic && g.traits.IsDirected {
		g.Graph, err = graphlib.TransitiveReduction(g.Graph)
		if err != nil {
			return fmt.Errorf("transitive reduction: %w", err)
		}
	}

	return
}

func (g *Graph[T]) Visitor(target string, expr *cel.Ast) {
	ast.PreOrderVisit(
		ast.NavigateAST(expr.NativeRep()),
		ast.NewExprVisitor(func(expr ast.Expr) {
			switch expr.Kind() {
			case ast.SelectKind:
				if expr.AsSelect().Operand().Kind() == ast.IdentKind {
					source := expr.AsSelect().Operand().AsIdent() + "." + expr.AsSelect().FieldName()
					if IsExprIdent(source) {
						err := IsValidEdge(source, target)
						if err != nil {
							g.addError(err)
						} else {
							g.addEdge(source, target)
						}
					}
				}
			}
		}),
	)
}

func (g *Graph[T]) addEdge(source, target string) {
	if !g.edges.Contains(source, target) {
		if edge, ok := NewEdge(source, target); ok {
			g.edges = append(g.edges, edge)
		}
	}
}

func (g *Graph[T]) Nodes() Nodes[T] {
	return g.nodes
}

func (g *Graph[T]) Edges() Edges {
	return g.edges
}

func (g *Graph[T]) Node(ident string) (node T, ok bool) {
	node, err := g.Vertex(ident)
	ok = err == nil
	return
}

func (g *Graph[T]) Edge(source, target string) (edge *Edge, ok bool) {
	for _, edge = range g.edges {
		// if strings.HasPrefix(edge.source, source) && strings.HasPrefix(edge.target, target) {
		if edge.source == source && edge.target == target {
			return
		}
	}
	return
}

func (g *Graph[T]) Inbound(target string) (edges []*Edge) {
	for _, edge := range g.edges {
		// if strings.HasPrefix(edge.target, target) {
		if edge.target == target {
			edges = append(edges, edge)
		}
	}
	return
}

func (g *Graph[T]) Outbound(source string) (edges []*Edge) {
	for _, edge := range g.edges {
		// if strings.HasPrefix(edge.source, source) {
		if edge.source == source {
			edges = append(edges, edge)
		}
	}
	return
}

func (g *Graph[T]) Groups() (groups [][]T, _ error) {
	groups = make([][]T, len(g.groups))
	for idx, ids := range g.groups {
		group := make([]T, len(ids))
		for idx, id := range ids {
			node, err := g.Vertex(id)
			if err != nil {
				return nil, fmt.Errorf("groups: %w", err)
			}
			group[idx] = node
		}
		groups[idx] = group
	}
	return
}

// Forward returns the predecessors (dependencies) of the given target nodes.
// If no targets are specified, returns all nodes that have outgoing edges.
func (g *Graph[T]) Forward(targets ...string) (sources []string) {
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
func (g *Graph[T]) Reverse(sources ...string) (targets []string) {
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
func (g *Graph[T]) Starts() (nodes []T, _ error) {
	for source := range g.reverse {
		if len(g.forward[source]) == 0 {
			node, err := g.Vertex(source)
			if err != nil {
				return nil, fmt.Errorf("starts: %w", err)
			}

			nodes = append(nodes, node)
		}
	}
	return
}

// Ends returns all nodes that have no dependents (leaf nodes).
func (g *Graph[T]) Ends() (nodes []T, _ error) {
	for target := range g.forward {
		if len(g.reverse[target]) == 0 {
			node, err := g.Vertex(target)
			if err != nil {
				return nil, fmt.Errorf("ends: %w", err)
			}

			nodes = append(nodes, node)
		}
	}
	return
}

func (g *Graph[T]) addError(err error) {
	if err != nil {
		g.errors = append(g.errors, err)
	}
}

func (g *Graph[T]) computePreds() error {
	preds, err := g.Graph.PredecessorMap()
	if err != nil {
		return err
	}

	for target, sources := range preds {
		for source := range sources {
			g.forward[target] = append(g.forward[target], source)
			g.reverse[source] = append(g.reverse[source], target)
		}
	}

	return nil
}

func (g *Graph[T]) computeGroups() error {
	if !g.traits.IsDirected {
		return nil
	}

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
				_, err := g.Vertex(nodeID)
				if err != nil {
					return fmt.Errorf("compute groups: %w", err)
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
