package shared

import (
	"fmt"
	"slices"
	"strings"

	"cel.dev/expr"
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
	// validNodePrefixes is a list of valid node type prefixes.
	validNodePrefixes = []string{
		ActionPrefix,
		ConnectionPrefix,
		InputPrefix,
		OutputPrefix,
		StreamPrefix,
		VarPrefix,
	}
	// invalidEdges is a map of invalid target to source node prefixes
	invalidEdges = map[string][]string{
		ConnectionPrefix: {}, // connections contain no expressions
		InputPrefix:      {}, // inputs contain no expressions
		VarPrefix:        {"connections", "outputs"},
		ActionPrefix:     {"connections", "outputs"},
		StreamPrefix:     {"connections", "outputs"},
		OutputPrefix:     {"connections"},
	}
)

type (
	EvalNode interface {
		proto.Message
		GetId() string
		GetCallCount() uint64
		GetCurrValue() *expr.Value
		GetPrevValue() *expr.Value
	}
	SpecNode interface {
		proto.Message
		GetId() string
	}
)

func IsNodeID(id string) bool {
	return slices.ContainsFunc(validNodePrefixes, func(prefix string) bool {
		return strings.HasPrefix(id, prefix+".")
	}) && len(strings.Split(id, ".")) == 2
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
	if sourcePrefix, ok := ParseNodePrefix(sourceID); !ok {
		return fmt.Errorf("invalid node id: %s", sourceID)
	} else if targetPrefix, ok := ParseNodePrefix(targetID); !ok {
		return fmt.Errorf("invalid node id: %s", targetID)
	} else if invalidSources, ok := invalidEdges[targetPrefix]; ok && len(invalidSources) > 0 {
		if slices.Contains(invalidSources, sourcePrefix) {
			return fmt.Errorf("invalid reference: %s -> %s", sourceID, targetID)
		}
	}
	return nil
}
