package resource

import (
	"regexp"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

const (
	// SegmentPattern defines the valid pattern for name segments.
	SegmentPattern Pattern = "^" + segmentPattern + "?$"
	// ProtoPattern defines the valid pattern of a fully qualified protobuf name.
	ProtoPattern Pattern = "^" + protoPattern + "?$"
	// VariablePattern defines the valid pattern for name segment variables.
	VariablePattern Pattern = "^" + variablePattern + "?$"

	segmentPattern  Pattern = "[a-z](?:[a-z0-9-]{0,61}[a-z0-9])"
	protoPattern    Pattern = "[a-zA-Z_][a-zA-Z0-9_]*(\\.[a-zA-Z_][a-zA-Z0-9_]*)*"
	variablePattern Pattern = "{[a-z][a-z_]+}"
)

var (
	validSegmentRegex  = regexp.MustCompile(SegmentPattern.String())
	validProtoRegex    = regexp.MustCompile(ProtoPattern.String())
	validVariableRegex = regexp.MustCompile(VariablePattern.String())
	patternCache       util.SyncMap[string, *regexp.Regexp]
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

func (p Pattern) getRegex() *regexp.Regexp {
	return p.loadOrStore()
}

func (p Pattern) loadOrStore() *regexp.Regexp {
	regex, ok := patternCache.Load(string(p))
	if !ok {
		regex = regexp.MustCompile(string(p))
		patternCache.Store(string(p), regex)
	}
	return regex
}

func (p Pattern) String() string {
	return string(p)
}
