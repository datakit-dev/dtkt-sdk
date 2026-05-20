package entadapter

import (
	"strings"
	"testing"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/orderby"
)

// Stubs for the type parameters in tests that don't exercise the
// SQL builder. They satisfy PaginateRecord without pulling in a real
// ent client.

type (
	stubRec   struct{}
	stubOrder func(*sql.Selector)
	stubPred  func(*sql.Selector)
)

func (stubRec) Value(string) (ent.Value, error) { return nil, nil }

// TestCursorRoundTrip verifies that a PageCursor proto round-trips
// through base64+protobuf encoding losslessly.
func TestCursorRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	uid := uuid.Must(uuid.NewV7())

	original := &sharedv1beta1.PageCursor{
		Parent:  "users/foo",
		Filter:  `name:bar`,
		OrderBy: "updated_at desc, name",
		Values: []*sharedv1beta1.PageCursor_CursorValue{
			{V: &sharedv1beta1.PageCursor_CursorValue_T{T: timestamppb.New(now)}},
			{V: &sharedv1beta1.PageCursor_CursorValue_S{S: "alpha"}},
			{V: &sharedv1beta1.PageCursor_CursorValue_Uuid{Uuid: uid.String()}},
		},
	}

	encoded, err := encodeCursor(original)
	if err != nil {
		t.Fatalf("encodeCursor: %v", err)
	}
	if encoded == "" {
		t.Fatal("encodeCursor returned empty string")
	}

	decoded, err := decodeCursor(encoded)
	if err != nil {
		t.Fatalf("decodeCursor: %v", err)
	}
	if decoded.GetParent() != original.GetParent() {
		t.Errorf("parent: got %q, want %q", decoded.GetParent(), original.GetParent())
	}
	if decoded.GetFilter() != original.GetFilter() {
		t.Errorf("filter: got %q, want %q", decoded.GetFilter(), original.GetFilter())
	}
	if decoded.GetOrderBy() != original.GetOrderBy() {
		t.Errorf("order_by: got %q, want %q", decoded.GetOrderBy(), original.GetOrderBy())
	}
	if got, want := len(decoded.GetValues()), len(original.GetValues()); got != want {
		t.Fatalf("values len: got %d, want %d", got, want)
	}
}

func TestDecodeCursor_Empty(t *testing.T) {
	got, err := decodeCursor("")
	if err != nil {
		t.Fatalf("decodeCursor(\"\"): %v", err)
	}
	if got != nil {
		t.Errorf("decodeCursor(\"\") = %v, want nil", got)
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	cases := []string{
		"not-base64-!@#",
		"@@@@@",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			_, err := decodeCursor(in)
			if err == nil {
				t.Fatalf("decodeCursor(%q) ok, want error", in)
			}
		})
	}
}

// TestValidateCursor covers the AIP-158 consistency check: parent,
// filter, and order_by from the request must all match the cursor's
// stored values.
func TestValidateCursor(t *testing.T) {
	mkCursor := func() *sharedv1beta1.PageCursor {
		return &sharedv1beta1.PageCursor{
			Parent: "p", Filter: "f", OrderBy: "o",
		}
	}
	match := PageParams{Parent: "p", Filter: "f", OrderBy: "o"}

	if err := validateCursor(nil, match); err != nil {
		t.Errorf("nil cursor: got err %v, want nil (first-page request)", err)
	}
	if err := validateCursor(mkCursor(), match); err != nil {
		t.Errorf("matching params: got err %v, want nil", err)
	}

	cases := []struct {
		name     string
		params   PageParams
		wantPart string
	}{
		{"parent mismatch", PageParams{Parent: "P", Filter: "f", OrderBy: "o"}, "parent mismatch"},
		{"filter mismatch", PageParams{Parent: "p", Filter: "F", OrderBy: "o"}, "filter mismatch"},
		{"order_by mismatch", PageParams{Parent: "p", Filter: "f", OrderBy: "O"}, "order_by mismatch"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateCursor(mkCursor(), c.params)
			if err == nil {
				t.Fatalf("validateCursor(%v) ok, want error containing %q", c.params, c.wantPart)
			}
			if !strings.Contains(err.Error(), c.wantPart) {
				t.Errorf("err = %q, want substring %q", err.Error(), c.wantPart)
			}
		})
	}
}

func TestToCursorValueRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	uid := uuid.Must(uuid.NewV7())

	cases := []struct {
		name  string
		input any
	}{
		{"string", "hello"},
		{"int64", int64(42)},
		{"int32 widens to int64", int32(42)},
		{"int widens to int64", int(42)},
		{"float64", 3.14},
		{"float32 widens to float64", float32(3.14)},
		{"bool", true},
		{"bytes", []byte{1, 2, 3}},
		{"time", now},
		{"uuid", uid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cv, err := toCursorValue(c.input)
			if err != nil {
				t.Fatalf("toCursorValue(%v): %v", c.input, err)
			}
			if valueOf(cv) == nil {
				t.Fatalf("valueOf(toCursorValue(%v)) is nil", c.input)
			}
		})
	}
}

func TestToCursorValue_Nil(t *testing.T) {
	_, err := toCursorValue(nil)
	if err == nil {
		t.Fatal("toCursorValue(nil) ok, want error")
	}
}

func TestToCursorValue_Unsupported(t *testing.T) {
	type wrapper struct{ X string }
	_, err := toCursorValue(wrapper{X: "y"})
	if err == nil {
		t.Fatal("toCursorValue(struct) ok, want error")
	}
}

func TestBuildLexPredicate_LengthMismatch(t *testing.T) {
	_, err := buildLexPredicate[stubPred]("id",
		[]orderby.Spec{{Field: "name", Direction: orderby.Asc}},
		[]*sharedv1beta1.PageCursor_CursorValue{
			{V: &sharedv1beta1.PageCursor_CursorValue_S{S: "alpha"}},
		})
	if err == nil {
		t.Fatal("buildLexPredicate with mismatched lengths returned no error")
	}
	if !strings.Contains(err.Error(), "cursor has") {
		t.Errorf("err = %q, want substring %q", err.Error(), "cursor has")
	}
}

func TestPaginateOptionsDefaults(t *testing.T) {
	var o PaginateOptions[any, stubRec, stubOrder, stubPred]
	if got, want := o.GetIDField(), "id"; got != want {
		t.Errorf("GetIDField: %q, want %q", got, want)
	}
	if got, want := o.GetDefaultPageSize(), int32(10); got != want {
		t.Errorf("GetDefaultPageSize: %d, want %d", got, want)
	}
	if got, want := o.GetMinPageSize(), int32(1); got != want {
		t.Errorf("GetMinPageSize: %d, want %d", got, want)
	}
	if got, want := o.GetMaxPageSize(), int32(100); got != want {
		t.Errorf("GetMaxPageSize: %d, want %d", got, want)
	}
}

func TestResolveOrderBy_Unsupported(t *testing.T) {
	var o PaginateOptions[any, stubRec, stubOrder, stubPred]
	_, _, err := o.resolveOrderBy("name asc")
	if err == nil {
		t.Fatal("resolveOrderBy with nil OrderFields returned no error")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("err = %q, want substring %q", err.Error(), "not supported")
	}
}

func TestResolveOrderBy_UnknownField(t *testing.T) {
	o := PaginateOptions[any, stubRec, stubOrder, stubPred]{
		OrderFields: orderby.Fields[stubOrder]{
			"name": func(orderby.Direction) stubOrder { return nil },
		},
	}
	_, _, err := o.resolveOrderBy("bogus desc")
	if err == nil {
		t.Fatal("resolveOrderBy with unknown field returned no error")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("err = %q, want substring %q", err.Error(), "unknown field")
	}
}

func TestResolveOrderBy_Empty(t *testing.T) {
	var o PaginateOptions[any, stubRec, stubOrder, stubPred]
	specs, terms, err := o.resolveOrderBy("")
	if err != nil {
		t.Fatalf("resolveOrderBy(\"\"): %v", err)
	}
	if specs != nil || terms != nil {
		t.Errorf("resolveOrderBy(\"\"): got specs=%v terms=%v, want nil", specs, terms)
	}
}

func TestResolveOrderBy_HappyPath(t *testing.T) {
	called := []string{}
	o := PaginateOptions[any, stubRec, stubOrder, stubPred]{
		OrderFields: orderby.Fields[stubOrder]{
			"name": func(d orderby.Direction) stubOrder {
				called = append(called, "name "+string(d))
				return nil
			},
			"updated_at": func(d orderby.Direction) stubOrder {
				called = append(called, "updated_at "+string(d))
				return nil
			},
		},
	}
	specs, terms, err := o.resolveOrderBy("updated_at desc, name asc")
	if err != nil {
		t.Fatalf("resolveOrderBy: %v", err)
	}
	if got, want := len(specs), 2; got != want {
		t.Fatalf("specs len: %d, want %d", got, want)
	}
	if got, want := len(terms), 2; got != want {
		t.Fatalf("terms len: %d, want %d", got, want)
	}
	if got, want := called, []string{"updated_at desc", "name asc"}; !equalStrings(got, want) {
		t.Errorf("call order: got %v, want %v", got, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
