package filter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/resource"
)

// FieldHandler maps a single filter clause's op + value to one ent
// predicate (or any user-defined predicate type).
type FieldHandler[P any] func(op Op, value string) (P, error)

// Fields is the per-resource filterable schema: maps a filter field name
// to its predicate builder. Callers populate this once per list handler.
type Fields[P any] map[string]FieldHandler[P]

// Apply parses s and walks each clause, returning the corresponding
// predicates. Empty filter returns nil predicates and no error.
//
// Unknown fields produce an error listing the allowed set so users get
// actionable feedback.
func Apply[P any](s string, fields Fields[P]) ([]P, error) {
	clauses, err := Parse(s)
	if err != nil {
		return nil, err
	}
	var preds []P
	for _, c := range clauses {
		h, ok := fields[c.Field]
		if !ok {
			return nil, fmt.Errorf("filter: unknown field %q (allowed: %s)", c.Field, allowedFields(fields))
		}
		p, err := h(c.Op, c.Value)
		if err != nil {
			return nil, err
		}
		preds = append(preds, p)
	}
	return preds, nil
}

func allowedFields[P any](fields Fields[P]) string {
	names := make([]string, 0, len(fields))
	for n := range fields {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// NameOps bundles the ent predicate constructors needed to map
// `name op value` filter clauses to predicates on a (parent, short-name)
// column pair. Each list handler builds one of these once and reuses it.
//
// Match semantics (op = `=`):
//   - value contains a slash → parsed as a full resource name; predicate
//     ANDs Parent + Eq.
//   - no slashes              → predicate is Eq on the short-name column.
//
// Match semantics (op = `:`): substring (case-insensitive) on the short
// name. AIP-160 says `:` is HAS / contains.
type NameOps[P any] struct {
	Type     resource.NameType
	Eq       func(string) P
	Contains func(string) P
	Parent   func(string) P
	And      func(...P) P
}

// Handler returns a FieldHandler suitable for inclusion in a Fields map
// under the "name" key.
func (n NameOps[P]) Handler() FieldHandler[P] {
	return func(op Op, value string) (P, error) {
		var zero P
		switch op {
		case OpEqual:
			if strings.Contains(value, "/") {
				parsed, err := n.Type.GetName(value)
				if err != nil {
					return zero, fmt.Errorf("filter: invalid name %q: %w", value, err)
				}
				return n.And(n.Parent(parsed.Parent().Short()), n.Eq(parsed.Short())), nil
			}
			return n.Eq(value), nil
		case OpHas:
			return n.Contains(value), nil
		}
		return zero, fmt.Errorf("filter: name supports `=` and `:`, got %q", op)
	}
}
