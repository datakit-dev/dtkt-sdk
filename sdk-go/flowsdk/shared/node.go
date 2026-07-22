package shared

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
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
	MetricPrefix     = "metrics"
)

var (
	// validNodePrefixes is a list of valid node id prefixes
	validNodePrefixes = []string{
		ActionPrefix,
		ConnectionPrefix,
		InputPrefix,
		OutputPrefix,
		StreamPrefix,
		VarPrefix,
		MetricPrefix,
	}
)

type (
	Node interface {
		Ident() string
		Send() chan<- any
		Recv() <-chan any
		Run(context.Context) error
		Value() any
		Reset()
	}
	SpecNode interface {
		proto.Message
		GetId() string
	}
	ParsedNode interface {
		Compile(context.Context, Env) error
		Run(context.Context, Env, Pipe) error
	}
	Nodes[T Node]      []T
	NodeParser[T Node] func(target SpecNode, visitor NodeVisitor) (T, error)
	NodeVisitor        func(target string, ast *cel.Ast)
)

func GetNodeIdent(node SpecNode) string {
	return fmt.Sprintf("%s.%s", GetNodePrefix(node), node.GetId())
}

func GetNodePrefix(node SpecNode) string {
	switch node.ProtoReflect().Descriptor().Name() {
	case "Action":
		return ActionPrefix
	case "Connection":
		return ConnectionPrefix
	case "Input":
		return InputPrefix
	case "Metric":
		return MetricPrefix
	case "Output":
		return OutputPrefix
	case "Stream":
		return StreamPrefix
	case "Var":
		return VarPrefix
	}
	return ""
}

func IsNodeIdent(id string) bool {
	return slices.ContainsFunc(validNodePrefixes, func(prefix string) bool {
		return strings.HasPrefix(id, prefix+".")
	}) && len(strings.Split(id, ".")) == 2
}

func IsNodePrefix(prefix string) bool {
	return slices.Contains(validNodePrefixes, prefix)
}

func ParseNodePrefix(ident string) (string, bool) {
	if IsNodeIdent(ident) {
		return strings.Split(ident, ".")[0], true
	}
	return "", false
}

func (n Nodes[T]) Load(ident string) (node T, ok bool) {
	idx := slices.IndexFunc(n, func(n T) bool {
		return n.Ident() == ident
	})
	ok = idx != -1
	if ok {
		node = n[idx]
	}
	return
}

func (n Nodes[T]) Idents() []string {
	return util.SliceMap(n, func(n T) string {
		return n.Ident()
	})
}

func (n Nodes[T]) Range(f func(string, T) bool) {
	for _, node := range n {
		if !f(node.Ident(), node) {
			return
		}
	}
}

func (n Nodes[T]) RangeErr(f func(string, T) error) (err error) {
	for _, node := range n {
		err = f(node.Ident(), node)
		if err != nil {
			return
		}
	}
	return
}

func (n Nodes[T]) Reset() {
	n.Range(func(_ string, n T) bool {
		n.Reset()
		return true
	})
}

func (n Nodes[T]) Values() map[string]any {
	return util.ReduceSliceToMap(n, func(_ int, node T) (string, any, bool) {
		return node.Ident(), node.Value(), true
	})
}

func (f NodeVisitor) ExprVisitor(target string) ExprVisitor {
	return func(expr *cel.Ast) {
		f(target, expr)
	}
}
