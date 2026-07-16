package namespace

import (
	"regexp"
)

const (
	// SegmentPattern defines the valid pattern for name segments.
	SegmentPattern Pattern = "[a-z](?:[a-z0-9-]{0,61}[a-z0-9])"
	// ProtoPattern defines the valid pattern of a fully qualified protobuf name.
	ProtoPattern Pattern = "[a-zA-Z_][a-zA-Z0-9_]*(\\.[a-zA-Z_][a-zA-Z0-9_]*)*"
	// VariablePattern defines the valid pattern for name segment variables.
	VariablePattern Pattern = "{[a-z][a-z_]+}"
)

var (
	validSegmentRegex  = regexp.MustCompile("^" + SegmentPattern.String() + "?$")
	validProtoRegex    = regexp.MustCompile("^" + ProtoPattern.String() + "?$")
	validVariableRegex = regexp.MustCompile("^" + VariablePattern.String() + "?$")
)

type Pattern string

func ValidSegmentRegex() *regexp.Regexp {
	return validSegmentRegex
}

func ValidProtoNameRegex() *regexp.Regexp {
	return validProtoRegex
}

func ValidVariableRegex() *regexp.Regexp {
	return validVariableRegex
}

func ValidSegment(segment string) bool {
	return validSegmentRegex.MatchString(segment)
}

func ValidProtoName(proto string) bool {
	return validProtoRegex.MatchString(proto)
}

func ValidVariable(variable string) bool {
	return validVariableRegex.MatchString(variable)
}

func (p Pattern) Strict() string {
	return "^" + string(p) + "?$"
}

func (p Pattern) String() string {
	return string(p)
}

func (p Pattern) getRegex(ns *NS) (regex *regexp.Regexp) {
	strict := p.Strict()
	regex, ok := ns.cache.Load(strict)
	if !ok {
		regex = regexp.MustCompile(strict)
		ns.cache.Store(strict, regex)
	}
	return regex
}
