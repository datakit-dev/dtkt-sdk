package shared

import (
	"fmt"
	"slices"
	"strings"
)

// invalidEdges is a map of target -> source prefixes which are not supported
var invalidEdges = map[string][]string{
	ActionPrefix: {ConnectionPrefix, OutputPrefix, MetricPrefix},
	OutputPrefix: {ConnectionPrefix, MetricPrefix},
	StreamPrefix: {ConnectionPrefix, OutputPrefix, MetricPrefix},
	VarPrefix:    {ConnectionPrefix, OutputPrefix, MetricPrefix},
}

type (
	Edge struct {
		source,
		target string
		buf chan any
	}
	Edges []*Edge
)

func NewEdge(source, target string) (*Edge, error) {
	if !IsExprIdent(source) {
		return nil, fmt.Errorf("invalid source edge: %s", source)
	} else if !IsExprIdent(target) {
		return nil, fmt.Errorf("invalid target edge: %s", target)
	}

	return &Edge{
		source: source,
		target: target,
		buf:    make(chan any),
	}, nil
}

func IsValidEdge(source, target string) error {
	sourcePrefix, ok := ParseExprNodePrefix(source)
	if !ok {
		return fmt.Errorf("invalid source node ident: %s", source)
	}

	targetPrefix, ok := ParseExprNodePrefix(target)
	if !ok {
		return fmt.Errorf("invalid target node ident: %s", target)
	}

	invalidSources, ok := invalidEdges[targetPrefix]
	if ok && slices.Contains(invalidSources, sourcePrefix) {
		return fmt.Errorf("invalid edge: %s -> %s", source, target)
	}

	return nil
}

func ParseExprNodePrefix(id string) (string, bool) {
	if IsExprIdent(id) {
		return strings.Split(id, ".")[0], true
	}
	return "", false
}

func ParseExprNodeIdent(id string) (string, bool) {
	if IsExprIdent(id) {
		return strings.Join(strings.Split(id, ".")[0:2], "."), true
	}
	return "", false
}

func IsExprIdent(id string) bool {
	return slices.ContainsFunc(validNodePrefixes, func(prefix string) bool {
		return strings.HasPrefix(id, prefix+".")
	}) && len(strings.Split(id, ".")) >= 2
}

func (e Edge) In() chan<- any {
	return e.buf
}

func (e Edge) Out() <-chan any {
	return e.buf
}

func (e Edge) SourceNode() (ident string) {
	ident, _ = ParseExprNodeIdent(e.source)
	return
}

func (e Edge) TargetNode() (ident string) {
	ident, _ = ParseExprNodeIdent(e.target)
	return
}

func (e Edge) Source() string {
	return e.source
}

func (e Edge) Target() string {
	return e.target
}

func (e Edge) String() string {
	return fmt.Sprintf("%s->%s", e.source, e.target)
}

func (e Edges) Contains(source, target string) bool {
	return slices.ContainsFunc(e, func(e *Edge) bool {
		return e.source == source && e.target == target
	})
}
