package shared

import (
	"errors"
	"maps"
	"slices"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	graphlib "github.com/dominikbraun/graph"
)

type (
	Graph[T any] struct {
		graphlib.Graph[string, T]
		errors  []error
		forward map[string][]string
		reverse map[string][]string
	}
	GraphOption func(*graphlib.Traits)
)

func NewGraphWithStore[T any](hash graphlib.Hash[string, T], store graphlib.Store[string, T], opts ...GraphOption) *Graph[T] {
	opts = append([]GraphOption{
		graphlib.Directed(),
		graphlib.PreventCycles(),
	}, opts...)

	var graph graphlib.Graph[string, T]
	if store != nil {
		graph = graphlib.NewWithStore(hash, store, util.SliceMap(opts, func(o GraphOption) func(*graphlib.Traits) {
			return o
		})...)
	} else {
		graph = graphlib.New(hash, util.SliceMap(opts, func(o GraphOption) func(*graphlib.Traits) {
			return o
		})...)
	}

	return &Graph[T]{
		Graph:   graph,
		forward: map[string][]string{},
		reverse: map[string][]string{},
	}
}

func (g *Graph[T]) Build() error {
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

func (g *Graph[T]) Error() error {
	if len(g.errors) > 0 {
		return errors.Join(g.errors...)
	}
	return nil
}

func (g *Graph[T]) AddError(err error) {
	if err != nil {
		g.errors = append(g.errors, err)
	}
}

func (g *Graph[T]) Starts() (ids []string) {
	for source := range g.reverse {
		if len(g.forward[source]) == 0 {
			ids = append(ids, source)
		}
	}
	return
}

func (g *Graph[T]) Ends() (ids []string) {
	for target := range g.forward {
		if len(g.reverse[target]) == 0 {
			ids = append(ids, target)
		}
	}
	return
}

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
