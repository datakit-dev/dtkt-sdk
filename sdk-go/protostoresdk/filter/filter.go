// Package filter implements a small subset of the AIP-160 filter
// expression grammar - just enough for ent-backed list endpoints to
// accept filters like `field=value` and `field:value` joined by AND.
//
// Grammar (subset):
//
//	expression ::= clause (" AND " clause)*
//	clause     ::= field op value
//	op         ::= "=" | ":"
//	field      ::= identifier ("." identifier)*
//	value      ::= quoted-string | bare-token
//
// `=` is exact equality; `:` is the AIP-160 HAS operator (substring
// containment for strings). OR, NOT, parentheses, comparison operators,
// and function calls are intentionally not supported - this is a
// pragmatic subset chosen so the parser stays trivial. Extending later
// (e.g. adding OR or `startsWith()`) only adds productions and doesn't
// require changes to existing callers.
package filter

import (
	"fmt"
	"strings"
)

// NameFilter builds an AIP-160 filter expression that selects resources
// whose name HAS the given substring (the `:` operator). Returns nil
// when value is empty so callers can pass it directly to a request's
// Filter field without producing a parse error server-side.
//
// Used for free-text shortcuts like tab completion and short-name
// resolution where the user types a partial name and we want anything
// matching it.
func NameFilter(value string) *string {
	if value == "" {
		return nil
	}
	s := fmt.Sprintf("name:%q", value)
	return &s
}

// NameEqualFilter builds an AIP-160 filter expression for exact name
// equality (`name="value"`). The daemon's name handler accepts either
// a fully-qualified resource name (parses to parent + short) or a bare
// short name. Returns nil when value is empty.
func NameEqualFilter(value string) *string {
	if value == "" {
		return nil
	}
	s := fmt.Sprintf("name=%q", value)
	return &s
}

// Op identifies the comparison operator on a clause.
type Op string

const (
	OpEqual Op = "="
	OpHas   Op = ":"
)

// Clause is one filter term: `field op value`.
type Clause struct {
	Field string
	Op    Op
	Value string
}

// Parse parses a filter expression. Returns nil clauses for the empty
// string (which means "no filter" by AIP convention).
func Parse(s string) ([]Clause, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	var clauses []Clause
	for {
		c, rest, err := parseClause(s)
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, c)

		rest = strings.TrimLeft(rest, " ")
		if rest == "" {
			return clauses, nil
		}
		// Bare `AND` (no clause following) and `AND ` followed only by
		// whitespace both indicate a trailing operator with no operand.
		if rest == "AND" {
			return nil, fmt.Errorf("filter: trailing AND with no clause")
		}
		const andPrefix = "AND "
		if !strings.HasPrefix(rest, andPrefix) {
			return nil, fmt.Errorf("filter: expected `AND` between clauses, got %q", rest)
		}
		s = strings.TrimLeft(rest[len(andPrefix):], " ")
		if s == "" {
			return nil, fmt.Errorf("filter: trailing AND with no clause")
		}
	}
}

func parseClause(s string) (Clause, string, error) {
	// Field: identifier characters until we hit an operator.
	i := 0
	for i < len(s) && isFieldByte(s[i]) {
		i++
	}
	if i == 0 {
		return Clause{}, "", fmt.Errorf("filter: expected field name in %q", s)
	}
	field := s[:i]

	if i >= len(s) {
		return Clause{}, "", fmt.Errorf("filter: expected operator after field %q", field)
	}

	var op Op
	switch s[i] {
	case '=':
		op = OpEqual
	case ':':
		op = OpHas
	default:
		return Clause{}, "", fmt.Errorf("filter: expected `=` or `:` after field %q, got %q", field, string(s[i]))
	}
	i++

	// Value: quoted or bare. AIP-160 doesn't define escape sequences,
	// so neither do we; closing quote ends the string.
	if i >= len(s) {
		return Clause{}, "", fmt.Errorf("filter: expected value after `%s%s`", field, op)
	}

	var value, rest string
	switch s[i] {
	case '"', '\'':
		quote := s[i]
		i++
		end := strings.IndexByte(s[i:], quote)
		if end < 0 {
			return Clause{}, "", fmt.Errorf("filter: unterminated string starting at %q", s[i-1:])
		}
		value = s[i : i+end]
		rest = s[i+end+1:]
	default:
		end := strings.IndexByte(s[i:], ' ')
		if end < 0 {
			value = s[i:]
			rest = ""
		} else {
			value = s[i : i+end]
			rest = s[i+end:]
		}
	}

	return Clause{Field: field, Op: op, Value: value}, rest, nil
}

func isFieldByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z':
		return true
	case b >= 'A' && b <= 'Z':
		return true
	case b >= '0' && b <= '9':
		return true
	case b == '_' || b == '.':
		return true
	}
	return false
}
