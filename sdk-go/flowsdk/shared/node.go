package shared

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/proto"
)

const (
	ActionType     = "Action"
	ConnectionType = "Connection"
	InputType      = "Input"
	OutputType     = "Output"
	StreamType     = "Stream"
	VarType        = "Var"
)

const (
	ActionPrefix     = "actions"
	ConnectionPrefix = "connections"
	InputPrefix      = "inputs"
	OutputPrefix     = "outputs"
	StreamPrefix     = "streams"
	VarPrefix        = "vars"
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
	}
	// invalidEdges is a map of target -> source prefixes which are not supported
	invalidEdges = map[string][]string{
		// connections and inputs contain no expressions
		ActionPrefix: {"connections", "outputs"},
		OutputPrefix: {"connections"},
		StreamPrefix: {"connections", "outputs"},
		VarPrefix:    {"connections", "outputs"},
	}
)

type (
	SpecNode interface {
		proto.Message
		GetId() string
	}
	ExecNode interface {
		Compile(Runtime) error
		Recv() (RecvFunc, bool)
		Send() (SendFunc, bool)
		Eval() (EvalFunc, bool)
		// HasCached returns the cached value and true if this node has caching
		// enabled and a value has already been captured. The executor re-uses this
		// value each cycle instead of waiting for a new external event.
		HasCached() (ref.Val, bool)
	}
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

func IsValidEdge(sourceID, targetID string) error {
	sourcePrefix, ok := ParseNodePrefix(sourceID)
	if !ok {
		return fmt.Errorf("invalid source node id: %s", sourceID)
	}

	targetPrefix, ok := ParseNodePrefix(targetID)
	if !ok {
		return fmt.Errorf("invalid target node id: %s", targetID)
	}

	invalidSources, ok := invalidEdges[targetPrefix]
	if ok && slices.Contains(invalidSources, sourcePrefix) {
		return fmt.Errorf("invalid edge: source[%s] -> target[%s]", sourceID, targetID)
	}

	return nil
}
