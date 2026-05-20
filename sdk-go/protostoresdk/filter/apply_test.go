package filter

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/resource"
)

// stringPred is a stand-in predicate type so Apply / NameOps can be tested
// without dragging ent into the unit tests. We just record what each
// handler emitted.
type stringPred string

func TestApply_EmptyFilter(t *testing.T) {
	got, err := Apply(" ", Fields[stringPred]{
		"name": func(_ Op, v string) (stringPred, error) { return stringPred(v), nil },
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got != nil {
		t.Fatalf("Apply: got %v, want nil", got)
	}
}

func TestApply_DispatchesToFieldHandler(t *testing.T) {
	preds, err := Apply(`name="foo" AND deployment="X"`, Fields[stringPred]{
		"name":       func(op Op, v string) (stringPred, error) { return stringPred("name " + string(op) + v), nil },
		"deployment": func(op Op, v string) (stringPred, error) { return stringPred("deployment " + string(op) + v), nil },
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	want := []stringPred{"name =foo", "deployment =X"}
	if !reflect.DeepEqual(preds, want) {
		t.Fatalf("Apply\n got: %v\nwant: %v", preds, want)
	}
}

func TestApply_UnknownFieldErrorListsAllowed(t *testing.T) {
	_, err := Apply(`bogus="x"`, Fields[stringPred]{
		"name":        func(Op, string) (stringPred, error) { return "", nil },
		"integration": func(Op, string) (stringPred, error) { return "", nil },
	})
	if err == nil {
		t.Fatal("Apply: want error, got nil")
	}
	for _, s := range []string{"unknown field", `"bogus"`, "integration", "name"} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("err = %q, missing substring %q", err.Error(), s)
		}
	}
}

func TestApply_FieldHandlerErrorPropagates(t *testing.T) {
	want := errors.New("boom")
	_, err := Apply(`name="foo"`, Fields[stringPred]{
		"name": func(Op, string) (stringPred, error) { return "", want },
	})
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want wraps %v", err, want)
	}
}

// fakeName mimics the parts of resource.NameType we need. We can't
// construct a real NameType inline so the test exercises NameOps by
// using a real one - resource.Connection - via a small DSL.
func TestNameOps_SmartMatch(t *testing.T) {
	ops := NameOps[stringPred]{
		Type: resource.Connection,
		Eq:   func(s string) stringPred { return stringPred("eq=" + s) },
		Contains: func(s string) stringPred {
			return stringPred("contains=" + s)
		},
		Parent: func(s string) stringPred { return stringPred("parent=" + s) },
		And: func(ps ...stringPred) stringPred {
			parts := make([]string, 0, len(ps))
			for _, p := range ps {
				parts = append(parts, string(p))
			}
			return stringPred("AND(" + strings.Join(parts, ",") + ")")
		},
	}.Handler()

	cases := []struct {
		clause Clause
		want   stringPred
	}{
		{Clause{"name", OpEqual, "foo"}, "eq=foo"},
		{Clause{"name", OpHas, "foo"}, "contains=foo"},
		{
			Clause{"name", OpEqual, "users/shadi/connections/email"},
			"AND(parent=shadi,eq=email)",
		},
	}
	for _, c := range cases {
		t.Run(c.clause.Field+" "+string(c.clause.Op)+" "+c.clause.Value, func(t *testing.T) {
			got, err := ops(c.clause.Op, c.clause.Value)
			if err != nil {
				t.Fatalf("Handler: %v", err)
			}
			if got != c.want {
				t.Fatalf("Handler\n got: %s\nwant: %s", got, c.want)
			}
		})
	}
}

func TestNameOps_RejectsUnsupportedOp(t *testing.T) {
	ops := NameOps[stringPred]{
		Type: resource.Connection,
		Eq:   func(s string) stringPred { return "eq" },
	}.Handler()
	_, err := ops(Op("<"), "foo")
	if err == nil {
		t.Fatal("Handler: want error on unsupported op, got nil")
	}
	if !strings.Contains(err.Error(), `"<"`) {
		t.Fatalf("err = %q, missing op token", err.Error())
	}
}

func TestNameOps_FullNameParseError(t *testing.T) {
	ops := NameOps[stringPred]{
		Type: resource.Connection,
		Eq:   func(s string) stringPred { return "eq" },
		And:  func(...stringPred) stringPred { return "and" },
		Parent: func(s string) stringPred { return "parent" },
	}.Handler()
	_, err := ops(OpEqual, "users/foo/wrong/email")
	if err == nil {
		t.Fatal("Handler: want error on malformed full name, got nil")
	}
	if !strings.Contains(err.Error(), "invalid name") {
		t.Fatalf("err = %q, want 'invalid name'", err.Error())
	}
}
