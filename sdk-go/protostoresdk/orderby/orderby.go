// Package orderby implements a small parser for the AIP-132 order_by
// expression grammar - the comma-separated list of field-with-optional-
// direction tokens that List requests use to sort results.
//
// Grammar (AIP-132 subset):
//
//	expression ::= spec ("," spec)*
//	spec       ::= field [ws "desc" | ws "asc"]
//	field      ::= identifier ("." identifier)*
//
// The default direction is ASC. Whitespace around commas is tolerated.
// Nested field paths (e.g. `build.integration`) are accepted by the
// parser; whether a given field is actually sortable is decided by the
// per-handler Fields map at Apply time.
//
// The package is generic over the predicate type so it can be reused
// across any ent-backed store - each caller plugs in the ent
// OrderOption (or any equivalent) and a per-field handler that emits
// one.
package orderby

import (
	"fmt"
	"sort"
	"strings"
)

// Direction is the sort direction for a single order_by spec.
type Direction string

const (
	Asc  Direction = "asc"
	Desc Direction = "desc"
)

// Spec is one order_by term: a field plus a direction.
type Spec struct {
	Field     string
	Direction Direction
}

// Parse parses an order_by expression. Returns nil specs for the empty
// string (which means "no order_by" by AIP convention - the server
// picks the default).
func Parse(s string) ([]Spec, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var specs []Spec
	for _, raw := range strings.Split(s, ",") {
		spec, err := parseSpec(strings.TrimSpace(raw))
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

func parseSpec(s string) (Spec, error) {
	if s == "" {
		return Spec{}, fmt.Errorf("orderby: empty term")
	}
	parts := strings.Fields(s)
	switch len(parts) {
	case 1:
		if !validField(parts[0]) {
			return Spec{}, fmt.Errorf("orderby: invalid field name %q", parts[0])
		}
		return Spec{Field: parts[0], Direction: Asc}, nil
	case 2:
		if !validField(parts[0]) {
			return Spec{}, fmt.Errorf("orderby: invalid field name %q", parts[0])
		}
		switch strings.ToLower(parts[1]) {
		case "asc":
			return Spec{Field: parts[0], Direction: Asc}, nil
		case "desc":
			return Spec{Field: parts[0], Direction: Desc}, nil
		default:
			return Spec{}, fmt.Errorf("orderby: expected `asc` or `desc` after field %q, got %q", parts[0], parts[1])
		}
	default:
		return Spec{}, fmt.Errorf("orderby: malformed term %q", s)
	}
}

func validField(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		switch {
		case b >= 'a' && b <= 'z':
		case b >= 'A' && b <= 'Z':
		case b >= '0' && b <= '9':
		case b == '_' || b == '.':
		default:
			return false
		}
	}
	return true
}

// FieldHandler maps a single order_by Spec to one ent OrderOption (or
// any user-defined order term type). The caller-supplied handler
// translates the abstract Direction into the ent term that flips
// asc/desc on the backing column.
type FieldHandler[O any] func(dir Direction) O

// Fields is the per-resource sortable schema: maps an order_by field
// name to its order-term builder. Callers populate this once per list
// handler. Unknown fields are rejected at Apply time.
type Fields[O any] map[string]FieldHandler[O]

// Apply parses s and walks each spec, returning the corresponding
// order terms. Empty order_by returns nil terms and no error so the
// caller can fall back to its default sort.
//
// Unknown fields produce an error listing the allowed set so users get
// actionable feedback.
func Apply[O any](s string, fields Fields[O]) ([]O, error) {
	specs, err := Parse(s)
	if err != nil {
		return nil, err
	}
	var terms []O
	for _, spec := range specs {
		h, ok := fields[spec.Field]
		if !ok {
			return nil, fmt.Errorf("orderby: unknown field %q (allowed: %s)", spec.Field, allowedFields(fields))
		}
		terms = append(terms, h(spec.Direction))
	}
	return terms, nil
}

func allowedFields[O any](fields Fields[O]) string {
	names := make([]string, 0, len(fields))
	for n := range fields {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
