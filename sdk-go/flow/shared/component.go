package shared

import (
	"context"
	"fmt"
	"maps"
	"slices"

	graphlib "github.com/dominikbraun/graph"
	"golang.org/x/sync/errgroup"
)

// Components partitions g into its weakly-connected components: the
// maximal sets of nodes reachable from one another when edge direction is
// ignored. A fully connected graph (the common case) has exactly one
// component, in which case Components returns []*Graph[T]{g} unchanged —
// no rebuild, no behavior change.
//
// When there is more than one component, each returned *Graph[T] is an
// independent graph containing only that component's nodes and edges (with
// the same hash/traits/store as g), fully usable on its own — including with
// NewExecutor — since it goes through the same vertex/edge/preds/groups
// construction as Graph.Parse.
func Components[T GraphNode](g *Graph[T]) ([]*Graph[T], error) {
	nodes := g.Nodes()
	if len(nodes) == 0 {
		return nil, nil
	}

	uf := newUnionFind(nodes)
	for _, edge := range g.Edges() {
		uf.union(edge.source, edge.target)
	}

	members := map[string][]T{}
	for _, node := range nodes {
		root := uf.find(node.Ident())
		members[root] = append(members[root], node)
	}

	if len(members) <= 1 {
		return []*Graph[T]{g}, nil
	}

	roots := slices.Sorted(maps.Keys(members))

	components := make([]*Graph[T], 0, len(roots))
	for _, root := range roots {
		subNodes := members[root]

		idents := make(map[string]struct{}, len(subNodes))
		for _, node := range subNodes {
			idents[node.Ident()] = struct{}{}
		}

		var subEdges Edges
		for _, edge := range g.Edges() {
			if _, ok := idents[edge.source]; ok {
				subEdges = append(subEdges, edge)
			}
		}

		component, err := newComponentGraph(g, subNodes, subEdges)
		if err != nil {
			return nil, fmt.Errorf("component %q: %w", root, err)
		}

		components = append(components, component)
	}

	return components, nil
}

// RunPartitioned runs g to completion, partitioning it into its
// weakly-connected components (see Components) and running one independent
// Executor per component concurrently. A fully connected graph (the common
// case, exactly one component) runs identically to calling
// NewExecutor(g, opts...).Run(ctx) directly — RunPartitioned is a strict,
// no-op-for-the-common-case refinement, safe to use unconditionally.
//
// If any component's executor returns an error, RunPartitioned cancels the
// remaining components' contexts and returns that error (via
// errgroup.Group's first-error-wins semantics).
func RunPartitioned[T GraphNode](ctx context.Context, g *Graph[T], opts ...ExecutorOption[T]) error {
	components, err := Components(g)
	if err != nil {
		return fmt.Errorf("components: %w", err)
	}

	if len(components) == 0 {
		return nil
	}

	if len(components) == 1 {
		return NewExecutor(components[0], opts...).Run(ctx)
	}

	group, gctx := errgroup.WithContext(ctx)
	for _, component := range components {
		group.Go(func() error {
			return NewExecutor(component, opts...).Run(gctx)
		})
	}

	return group.Wait()
}

// newComponentGraph builds a standalone Graph[T] for a subset of parent's
// nodes/edges, mirroring the construction Graph.Parse performs (vertices,
// edges, predecessor maps, topological groups) so the resulting graph is
// fully usable, not just a partial view.
func newComponentGraph[T GraphNode](parent *Graph[T], nodes GraphNodes[T], edges Edges) (*Graph[T], error) {
	g := &Graph[T]{
		hasher:  parent.hasher,
		traits:  parent.traits,
		store:   parent.store,
		nodes:   nodes,
		edges:   edges,
		forward: map[string][]string{},
		reverse: map[string][]string{},
	}

	if g.store != nil {
		g.Graph = graphlib.NewWithStore(g.hasher, g.store, func(traits *graphlib.Traits) {
			*traits = g.traits
		})
	} else {
		g.Graph = graphlib.New(g.hasher, func(traits *graphlib.Traits) {
			*traits = g.traits
		})
	}

	for _, node := range nodes {
		if err := g.AddVertex(node); err != nil {
			return nil, fmt.Errorf("add vertex: %w", err)
		}
	}

	for _, edge := range edges {
		if err := g.AddEdge(edge.source, edge.target); err != nil {
			return nil, fmt.Errorf("add edge: %w", err)
		}
	}

	if err := g.computePreds(); err != nil {
		return nil, fmt.Errorf("compute preds: %w", err)
	}

	if err := g.computeGroups(); err != nil {
		return nil, fmt.Errorf("compute groups: %w", err)
	}

	return g, nil
}

// unionFind is a minimal union-find (disjoint-set) over node identifiers,
// used by Components to compute weakly-connected components without
// depending on graph direction.
type unionFind[T GraphNode] struct {
	parent map[string]string
}

func newUnionFind[T GraphNode](nodes GraphNodes[T]) *unionFind[T] {
	parent := make(map[string]string, len(nodes))
	for _, node := range nodes {
		parent[node.Ident()] = node.Ident()
	}
	return &unionFind[T]{parent: parent}
}

func (u *unionFind[T]) find(id string) string {
	for u.parent[id] != id {
		u.parent[id] = u.parent[u.parent[id]] // path halving
		id = u.parent[id]
	}
	return id
}

func (u *unionFind[T]) union(a, b string) {
	ra, rb := u.find(a), u.find(b)
	if ra != rb {
		u.parent[ra] = rb
	}
}
