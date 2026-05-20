package orderby

import (
	"reflect"
	"strings"
	"testing"
)

func TestParse_Empty(t *testing.T) {
	for _, in := range []string{"", "  ", "\t"} {
		got, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q) err: %v", in, err)
		}
		if got != nil {
			t.Fatalf("Parse(%q) = %#v, want nil", in, got)
		}
	}
}

func TestParse_Valid(t *testing.T) {
	cases := []struct {
		in   string
		want []Spec
	}{
		{in: `name`, want: []Spec{{Field: "name", Direction: Asc}}},
		{in: `name asc`, want: []Spec{{Field: "name", Direction: Asc}}},
		{in: `name desc`, want: []Spec{{Field: "name", Direction: Desc}}},
		{in: `name  DESC`, want: []Spec{{Field: "name", Direction: Desc}}},
		{
			in: `updated_at desc, id desc`,
			want: []Spec{
				{Field: "updated_at", Direction: Desc},
				{Field: "id", Direction: Desc},
			},
		},
		{
			in: `build.integration asc`,
			want: []Spec{
				{Field: "build.integration", Direction: Asc},
			},
		},
		{
			// extra whitespace around commas is fine
			in: `  name asc ,   id   desc  `,
			want: []Spec{
				{Field: "name", Direction: Asc},
				{Field: "id", Direction: Desc},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := Parse(c.in)
			if err != nil {
				t.Fatalf("Parse(%q) err: %v", c.in, err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Parse(%q)\n got: %#v\nwant: %#v", c.in, got, c.want)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	cases := []struct {
		in       string
		wantPart string
	}{
		{`name foo`, "expected `asc` or `desc`"},
		{`name asc desc`, "malformed term"},
		{`name<`, "invalid field name"},
		{`, name`, "empty term"},
		{`name,`, "empty term"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			_, err := Parse(c.in)
			if err == nil {
				t.Fatalf("Parse(%q) ok, want error containing %q", c.in, c.wantPart)
			}
			if !strings.Contains(err.Error(), c.wantPart) {
				t.Fatalf("Parse(%q) err = %q, want substring %q", c.in, err.Error(), c.wantPart)
			}
		})
	}
}

type orderTerm string

func TestApply_EmptyOrderBy(t *testing.T) {
	got, err := Apply(" ", Fields[orderTerm]{
		"name": func(Direction) orderTerm { return "" },
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got != nil {
		t.Fatalf("Apply: got %v, want nil", got)
	}
}

func TestApply_DispatchesToFieldHandler(t *testing.T) {
	terms, err := Apply(`name desc, id asc`, Fields[orderTerm]{
		"name": func(dir Direction) orderTerm { return orderTerm("name " + string(dir)) },
		"id":   func(dir Direction) orderTerm { return orderTerm("id " + string(dir)) },
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	want := []orderTerm{"name desc", "id asc"}
	if !reflect.DeepEqual(terms, want) {
		t.Fatalf("Apply\n got: %v\nwant: %v", terms, want)
	}
}

func TestApply_UnknownFieldErrorListsAllowed(t *testing.T) {
	_, err := Apply(`bogus desc`, Fields[orderTerm]{
		"name":       func(Direction) orderTerm { return "" },
		"updated_at": func(Direction) orderTerm { return "" },
	})
	if err == nil {
		t.Fatal("Apply: want error, got nil")
	}
	for _, s := range []string{"unknown field", `"bogus"`, "updated_at", "name"} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("err = %q, missing substring %q", err.Error(), s)
		}
	}
}
