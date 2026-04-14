package resource

import (
	"fmt"
	"strings"
)

// NameType represents a resource name type.
type NameType string

func IsNameType(input string) (NameType, bool) {
	nt, ok := coreAliases[strings.ToLower(input)]
	return nt, ok
}

// NewName constructs a single Name segment with the given short name for this
// NameType for easy construction of child resource names. For example, given a
// NameType of "flows" and a short name of "my-flow", this would return a Name
// with a single segment of "flows/my-flow".
func (t NameType) New(short string) Name {
	if short == "" {
		return EmptyName()
	}

	parts := strings.Split(short, "/")
	if len(parts) < 1 {
		return EmptyName()
	}

	return Name{{string(t), parts[len(parts)-1]}}
}

// IsName checks if a full resource name matches the expected pattern for the
// given resource type.
func (t NameType) IsName(input string) (ok bool) {
	return t.Pattern().getRegex().MatchString(input)
}

// GetName parses a full resource name string into a Name, validating that it
// matches the expected pattern for the given resource type.
func (t NameType) GetName(input string) (Name, error) {
	regex := t.Pattern().getRegex()
	matches := regex.FindStringSubmatch(input)

	if matches == nil {
		return nil, fmt.Errorf("invalid %s resource name: %q", t, input)
	}

	subexpNames := regex.SubexpNames()

	var name Name
	for i, submatch := range matches {
		if i > 0 && subexpNames[i] != "" && submatch != "" {
			name = append(name, Pair{subexpNames[i], submatch})
		}
	}

	return name, nil
}

func (t NameType) MustGetName(input string) Name {
	name, _ := t.GetName(input)
	return name
}

func (t NameType) Parents() []NameType {
	return coreHierarchy[t]
}

func (t NameType) rawPattern() Pattern {
	var (
		rawPattern   = t.String() + "/(?P<" + t.String() + ">" + corePatterns[t].String() + ")"
		systemParent bool
	)
	if len(coreHierarchy[t]) > 0 {
		var parents []string
		for _, parent := range coreHierarchy[t] {
			if parent == root {
				systemParent = true
				continue
			}
			parents = append(parents, string(parent.rawPattern()))
		}

		if systemParent {
			return Pattern("((?:" + strings.Join(parents, "|") + ")/)?" + rawPattern)
		}

		return Pattern("(?:" + strings.Join(parents, "|") + ")/" + rawPattern)
	}

	return Pattern(rawPattern)
}

func (t NameType) Pattern() Pattern {
	if t == root {
		return ""
	}
	return Pattern("^" + t.rawPattern() + "$")
}

func (t NameType) String() string {
	return string(t)
}

func init() {
	for typ := range coreHierarchy {
		typ.Pattern().loadOrStore()
	}
}
