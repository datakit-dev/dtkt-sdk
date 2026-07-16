package shared

import (
	"fmt"
	"slices"
	"sync"
)

// invalidEdges is a map of target -> source prefixes which are not supported
var invalidEdges = map[string][]string{
	ActionPrefix: {ConnectionPrefix, OutputPrefix},
	OutputPrefix: {ConnectionPrefix},
	StreamPrefix: {ConnectionPrefix, OutputPrefix},
	VarPrefix:    {ConnectionPrefix, OutputPrefix},
}

type Edge[T Node] struct {
	src,
	dst T
	buf chan any
	mut sync.Mutex
}

func NewEdge[T Node](source, target T) *Edge[T] {
	return &Edge[T]{
		src: source,
		dst: target,
		buf: make(chan any),
	}
}

func IsValidEdge(sourceIdent, targetIdent string) error {
	sourcePrefix, ok := ParseNodePrefix(sourceIdent)
	if !ok {
		return fmt.Errorf("invalid source node ident: %s", sourceIdent)
	}

	targetPrefix, ok := ParseNodePrefix(targetIdent)
	if !ok {
		return fmt.Errorf("invalid target node ident: %s", targetIdent)
	}

	invalidSources, ok := invalidEdges[targetPrefix]
	if ok && slices.Contains(invalidSources, sourcePrefix) {
		return fmt.Errorf("invalid edge: %s -> %s", sourceIdent, targetIdent)
	}

	return nil
}

func (e *Edge[T]) In() chan<- any {
	e.mut.Lock()
	defer e.mut.Unlock()
	return e.buf
}

func (e *Edge[T]) Out() <-chan any {
	e.mut.Lock()
	defer e.mut.Unlock()
	return e.buf
}

func (e *Edge[T]) Source() T {
	return e.src
}

func (e *Edge[T]) Target() T {
	return e.dst
}
