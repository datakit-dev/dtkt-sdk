package filter

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
		want []Clause
	}{
		{
			in:   `name=foo`,
			want: []Clause{{Field: "name", Op: OpEqual, Value: "foo"}},
		},
		{
			in:   `name:foo`,
			want: []Clause{{Field: "name", Op: OpHas, Value: "foo"}},
		},
		{
			in:   `name="users/foo/connections/email"`,
			want: []Clause{{Field: "name", Op: OpEqual, Value: "users/foo/connections/email"}},
		},
		{
			in:   `name='single quoted'`,
			want: []Clause{{Field: "name", Op: OpEqual, Value: "single quoted"}},
		},
		{
			in: `deployment="users/foo/deployments/email-default" AND integration="users/foo/integrations/email"`,
			want: []Clause{
				{Field: "deployment", Op: OpEqual, Value: "users/foo/deployments/email-default"},
				{Field: "integration", Op: OpEqual, Value: "users/foo/integrations/email"},
			},
		},
		{
			in:   `build.integration="users/foo/integrations/email"`,
			want: []Clause{{Field: "build.integration", Op: OpEqual, Value: "users/foo/integrations/email"}},
		},
		{
			in: `name:foo AND deployment="X"`,
			want: []Clause{
				{Field: "name", Op: OpHas, Value: "foo"},
				{Field: "deployment", Op: OpEqual, Value: "X"},
			},
		},
		{
			// Whitespace tolerance around AND
			in: `name:a   AND   name:b`,
			want: []Clause{
				{Field: "name", Op: OpHas, Value: "a"},
				{Field: "name", Op: OpHas, Value: "b"},
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
		{`=foo`, "expected field name"},
		{`name`, "expected operator after field"},
		{`name foo`, "expected `=` or `:`"},
		{`name<foo`, "expected `=` or `:`"},
		{`name=`, "expected value after"},
		{`name="unterminated`, "unterminated string"},
		{`name=foo OR name=bar`, "expected `AND`"},
		{`name=foo AND `, "trailing AND"},
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
