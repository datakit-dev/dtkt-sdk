package shared

import (
	"context"
	"slices"
	"strings"

	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/proto"
)

const (
	ActionPrefix     = "actions"
	ConnectionPrefix = "connections"
	InputPrefix      = "inputs"
	OutputPrefix     = "outputs"
	StreamPrefix     = "streams"
	VarPrefix        = "vars"
)

// NodeVarsType is the CEL type used for node variable bindings (e.g.
// vars.x, streams.y, and self), each of which resolves to a map exposing
// value, prev, count, start_time, and finish_time.
var NodeVarsType = cel.MapType(cel.StringType, cel.DynType)

var (
	// validNodePrefixes is a list of valid node id prefixes
	validNodePrefixes = []string{
		ActionPrefix,
		ConnectionPrefix,
		InputPrefix,
		OutputPrefix,
		StreamPrefix,
		VarPrefix,
	}
)

type (
	Node interface {
		Ident() string
		Spec() SpecNode
		Exec() ExecNode
		Pipe() Pipe
		Send() chan<- any
		Recv() <-chan any
	}
	SpecNode interface {
		proto.Message
		GetId() string
	}
	ExecNode interface {
		Compile(context.Context, Env) error
		Run(context.Context, Env, Pipe) error
	}
	NodeParser[T Node] func(T, NodeVisitor) error
	NodeVisitor        func(id string, expr *cel.Ast)
)

func IsNodeID(id string) bool {
	return slices.ContainsFunc(validNodePrefixes, func(prefix string) bool {
		return strings.HasPrefix(id, prefix+".")
	}) && len(strings.Split(id, ".")) == 2
}

func IsNodePrefix(prefix string) bool {
	return slices.Contains(validNodePrefixes, prefix)
}

func ParseNodePrefix(ident string) (string, bool) {
	if IsNodeID(ident) {
		return strings.Split(ident, ".")[0], true
	}
	return "", false
}

func ParseNodePrefixAndID(ident string) (string, string, bool) {
	if IsNodeID(ident) {
		parts := strings.Split(ident, ".")
		return parts[0], parts[1], true
	}
	return "", "", false
}
