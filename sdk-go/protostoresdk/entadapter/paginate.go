package entadapter

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/orderby"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

// ErrInvalidArgument tags pager errors that map to INVALID_ARGUMENT
// at the RPC boundary (unknown sort field, malformed page_token,
// parent/filter/order_by mismatch with cursor, etc.). Callers map
// this to connect.CodeInvalidArgument in their error converters
// (e.g. db.ToConnectError) so AIP-160 / AIP-158 violations surface
// with the correct status.
var ErrInvalidArgument = errors.New("invalid argument")

type (
	// PaginateOptions configures a paginated List endpoint backed by
	// ent. All resources are expected to have a UUID v7 primary key
	// column; that column is the default sort (id DESC, newest-first)
	// when the request's order_by is empty, and the stable tiebreaker
	// appended to any user-specified order_by chain.
	PaginateOptions[T any, R PaginateRecord, O, P ~func(*sql.Selector)] struct {
		// IDField is the ent column name of the UUID v7 primary key.
		IDField string

		// GetID returns the row's UUID v7 id. Used to build the next
		// page cursor. Optional: if nil, the pager falls back to
		// (1) the `interface{ GetID() uuid.UUID }` method if the
		// record satisfies it (record types that expose a
		// `GetID() uuid.UUID` accessor do; plain ent records do
		// not), then (2) `record.Value(IDField)` reflection.
		GetID func(R) uuid.UUID

		DefaultPageSize, MinPageSize, MaxPageSize int32

		// OrderFields is the per-resource whitelist of sortable fields.
		// When the request's order_by is non-empty the pager parses it
		// against this whitelist; unknown fields produce
		// INVALID_ARGUMENT per AIP-160. Optional: if nil, passing a
		// non-empty order_by returns an error.
		OrderFields orderby.Fields[O]

		// OrderValue extracts the value of a sortable field from a
		// record, used to build the next page's cursor. Must support
		// every field name registered in OrderFields. ent's generated
		// `record.Value()` only returns dynamically-selected columns
		// (Aggregate / GroupBy / Select modifiers) - it does NOT
		// expose struct fields like `Name` / `UpdatedAt` / `ID`. So
		// when OrderFields is set, this callback is required: just
		// switch on the field name and return the matching struct
		// field (e.g. `r.Name`, `r.UpdatedAt`, `r.ID`).
		OrderValue func(R, string) (any, error)
	}

	// PageParams is the consistency envelope for a paginated List
	// request. Per AIP-158, every paginated request must carry the
	// same values for these params (page_size MAY change; nothing
	// else). The pager encodes them into the cursor at page
	// boundaries and validates them on every follow-up request.
	//
	// Resources with no order_by capability (longrunning operations)
	// leave OrderBy empty.
	PageParams struct {
		Parent  string
		Filter  string
		OrderBy string
	}

	PaginateQuery[T any, R PaginateRecord, O, P ~func(*sql.Selector)] interface {
		Where(...P) T
		Limit(int) T
		Order(...O) T
		All(context.Context) ([]R, error)
	}
	PaginateRecord interface {
		Value(string) (ent.Value, error)
	}
)

// GetNextPage is a convenience wrapper for callers that don't need
// per-resource pager config (e.g. REST endpoints with no order_by /
// filter surface). It delegates to a zero-value PaginateOptions,
// which means: IDField defaults to "id", GetID is nil so the pager
// falls back to the record's `GetID() uuid.UUID` method or
// `record.Value("id")` reflection, and OrderFields is nil (so passing
// a non-empty params.OrderBy will error).
func GetNextPage[T any, R PaginateRecord, P, O ~func(*sql.Selector), Q PaginateQuery[T, R, O, P]](
	ctx context.Context,
	req util.PageTokenRequest,
	params PageParams,
	query Q,
) (string, []R, error) {
	return PaginateOptions[T, R, O, P]{}.GetNextPage(ctx, req, params, query)
}

// GetNextPage executes one page of the request.
//
// Behavior:
//   - First request (page_token empty): build the sort chain from
//     params.OrderBy (plus an id DESC tiebreaker), run the query,
//     issue a cursor encoding (params, last-row sort values) if a
//     full page came back.
//   - Subsequent requests: decode the cursor, return
//     INVALID_ARGUMENT if any of params.Parent / params.Filter /
//     params.OrderBy differ from what the cursor was issued under,
//     then apply the composite lex-comparison WHERE predicate built
//     from the cursor's sort values.
//
// The "default sort" (params.OrderBy empty) is a special case of the
// composite path where the sort chain is just id DESC and the
// cursor's values slice has a single entry (the id).
func (o PaginateOptions[T, R, O, P]) GetNextPage(
	ctx context.Context,
	req util.PageTokenRequest,
	params PageParams,
	query PaginateQuery[T, R, O, P],
) (nextPageToken string, records []R, _ error) {
	// Resolve the user-supplied order_by (if any) against the
	// per-resource whitelist; build the ORDER BY chain.
	specs, terms, err := o.resolveOrderBy(params.OrderBy)
	if err != nil {
		return "", nil, err
	}
	terms = append(terms, O(sql.OrderByField(o.GetIDField(), sql.OrderDesc()).ToFunc()))

	// Decode + validate cursor.
	cursor, err := decodeCursor(req.GetPageToken())
	if err != nil {
		return "", nil, fmt.Errorf("%w: invalid page_token: %s", ErrInvalidArgument, err.Error())
	}
	if err := validateCursor(cursor, params); err != nil {
		return "", nil, err
	}

	pageSize := util.GetPageSizeRequest(req, o.GetDefaultPageSize(), o.GetMinPageSize(), o.GetMaxPageSize())

	// Apply the lex-comparison predicate built from the cursor's
	// stored values (sort keys + id tiebreaker).
	if cursor != nil && len(cursor.GetValues()) > 0 {
		pred, err := buildLexPredicate[P](o.GetIDField(), specs, cursor.GetValues())
		if err != nil {
			return "", nil, fmt.Errorf("%w: invalid page_token: %s", ErrInvalidArgument, err.Error())
		}
		query.Where(pred)
	}

	query.Limit(int(pageSize))
	query.Order(terms...)

	records, err = query.All(ctx)
	if err != nil {
		return "", nil, err
	}

	if len(records) == int(pageSize) {
		last := records[len(records)-1]
		lastUUID, err := o.lastUUID(last)
		if err != nil {
			return "", nil, err
		}
		next, err := o.buildCursor(last, specs, lastUUID, params)
		if err != nil {
			return "", nil, fmt.Errorf("encode next page token: %w", err)
		}
		nextPageToken, err = encodeCursor(next)
		if err != nil {
			return "", nil, fmt.Errorf("encode next page token: %w", err)
		}
	}

	return nextPageToken, records, nil
}

// resolveOrderBy parses the order_by expression against the
// per-resource whitelist and returns both the parsed specs (for
// cursor encoding / lex predicate) and the ent ORDER BY chain.
func (o PaginateOptions[T, R, O, P]) resolveOrderBy(orderByExpr string) ([]orderby.Spec, []O, error) {
	if orderByExpr == "" {
		return nil, nil, nil
	}
	if o.OrderFields == nil {
		return nil, nil, fmt.Errorf("%w: order_by not supported on this resource", ErrInvalidArgument)
	}
	specs, err := orderby.Parse(orderByExpr)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrInvalidArgument, err.Error())
	}
	terms := make([]O, 0, len(specs))
	for _, spec := range specs {
		handler, ok := o.OrderFields[spec.Field]
		if !ok {
			return nil, nil, fmt.Errorf("%w: orderby: unknown field %q", ErrInvalidArgument, spec.Field)
		}
		terms = append(terms, handler(spec.Direction))
	}
	return specs, terms, nil
}

// validateCursor enforces AIP-158: subsequent paginated requests must
// carry the same parent, filter, and order_by as the request that
// issued the page_token. Differing values produce INVALID_ARGUMENT.
//
// First-page requests pass `cursor == nil` and skip validation.
func validateCursor(cursor *sharedv1beta1.PageCursor, params PageParams) error {
	if cursor == nil {
		return nil
	}
	if cursor.GetParent() != params.Parent {
		return fmt.Errorf("%w: page_token parent mismatch: token=%q, request=%q", ErrInvalidArgument, cursor.GetParent(), params.Parent)
	}
	if cursor.GetFilter() != params.Filter {
		return fmt.Errorf("%w: page_token filter mismatch: token=%q, request=%q", ErrInvalidArgument, cursor.GetFilter(), params.Filter)
	}
	if cursor.GetOrderBy() != params.OrderBy {
		return fmt.Errorf("%w: page_token order_by mismatch: token=%q, request=%q", ErrInvalidArgument, cursor.GetOrderBy(), params.OrderBy)
	}
	return nil
}

func (o PaginateOptions[T, R, O, P]) lastUUID(last R) (uuid.UUID, error) {
	if o.GetID != nil {
		return o.GetID(last), nil
	}
	if getter, ok := any(last).(interface{ GetID() uuid.UUID }); ok {
		return getter.GetID(), nil
	}
	v, err := last.Value(o.GetIDField())
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s value: %w", o.GetIDField(), err)
	}
	u, ok := v.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("%s: expected uuid.UUID, got %T", o.GetIDField(), v)
	}
	return u, nil
}

// buildLexPredicate constructs the cursor's lex-comparison predicate.
//
// For ORDER BY a DESC, b ASC with cursor values (A, B, id=I) it
// returns:
//
//	(a < A) OR (a = A AND b > B) OR (a = A AND b = B AND id < I)
//
// Each clause i is `AND(EQ(spec[0..i]), CMP(spec[i]))`. CMP is < when
// the spec direction is DESC (or for the trailing id, which is
// always DESC), and > when ASC. Clauses are OR'd. With n sort keys
// there are n+1 clauses (one per key plus the id tiebreaker).
//
// When specs is empty (default sort) cursorValues has a single entry
// (the id), and the function returns the trivial `id < I` predicate.
func buildLexPredicate[P ~func(*sql.Selector)](
	idField string,
	specs []orderby.Spec,
	cursorValues []*sharedv1beta1.PageCursor_CursorValue,
) (P, error) {
	n := len(specs)
	if len(cursorValues) != n+1 {
		return nil, fmt.Errorf("cursor has %d values for %d sort keys (expected %d)", len(cursorValues), n, n+1)
	}

	clauses := make([]P, 0, n+1)
	for i := 0; i <= n; i++ {
		ands := make([]P, 0, i+1)
		for j := 0; j < i; j++ {
			ands = append(ands, sql.FieldEQ(specs[j].Field, valueOf(cursorValues[j])))
		}
		var (
			field string
			lt    bool
		)
		if i == n {
			field = idField
			lt = true // id is always DESC
		} else {
			field = specs[i].Field
			lt = specs[i].Direction == orderby.Desc
		}
		if lt {
			ands = append(ands, sql.FieldLT(field, valueOf(cursorValues[i])))
		} else {
			ands = append(ands, sql.FieldGT(field, valueOf(cursorValues[i])))
		}
		clauses = append(clauses, sql.AndPredicates[P](ands...))
	}
	return sql.OrPredicates[P](clauses...), nil
}

// buildCursor reads sort-key values from the last row using the
// caller-supplied OrderValue getter (struct-field access; ent's
// record.Value only works for dynamically-selected columns), then
// encodes them into a PageCursor proto with the active request
// params and the UUID v7 id tiebreaker.
func (o PaginateOptions[T, R, O, P]) buildCursor(
	last R,
	specs []orderby.Spec,
	lastUUID uuid.UUID,
	params PageParams,
) (*sharedv1beta1.PageCursor, error) {
	values := make([]*sharedv1beta1.PageCursor_CursorValue, 0, len(specs)+1)
	for _, spec := range specs {
		v, err := o.readOrderValue(last, spec.Field)
		if err != nil {
			return nil, fmt.Errorf("read field %q for cursor: %w", spec.Field, err)
		}
		cv, err := toCursorValue(v)
		if err != nil {
			return nil, fmt.Errorf("encode field %q for cursor: %w", spec.Field, err)
		}
		values = append(values, cv)
	}
	values = append(values, &sharedv1beta1.PageCursor_CursorValue{
		V: &sharedv1beta1.PageCursor_CursorValue_Uuid{Uuid: lastUUID.String()},
	})
	return &sharedv1beta1.PageCursor{
		Parent:  params.Parent,
		Filter:  params.Filter,
		OrderBy: params.OrderBy,
		Values:  values,
	}, nil
}

// readOrderValue extracts the value of a sortable field from a
// record. When the caller registered an OrderValue callback, that is
// used (the typical path for ent rows - struct fields aren't visible
// via ent's record.Value). Otherwise falls back to record.Value(),
// which only works for dynamically-selected columns.
func (o PaginateOptions[T, R, O, P]) readOrderValue(r R, name string) (any, error) {
	if o.OrderValue != nil {
		return o.OrderValue(r, name)
	}
	return r.Value(name)
}

func toCursorValue(v any) (*sharedv1beta1.PageCursor_CursorValue, error) {
	cv := &sharedv1beta1.PageCursor_CursorValue{}
	switch x := v.(type) {
	case nil:
		return nil, fmt.Errorf("cannot encode nil for cursor")
	case string:
		cv.V = &sharedv1beta1.PageCursor_CursorValue_S{S: x}
	case int:
		cv.V = &sharedv1beta1.PageCursor_CursorValue_I{I: int64(x)}
	case int32:
		cv.V = &sharedv1beta1.PageCursor_CursorValue_I{I: int64(x)}
	case int64:
		cv.V = &sharedv1beta1.PageCursor_CursorValue_I{I: x}
	case float32:
		cv.V = &sharedv1beta1.PageCursor_CursorValue_F{F: float64(x)}
	case float64:
		cv.V = &sharedv1beta1.PageCursor_CursorValue_F{F: x}
	case bool:
		cv.V = &sharedv1beta1.PageCursor_CursorValue_B{B: x}
	case []byte:
		cv.V = &sharedv1beta1.PageCursor_CursorValue_Bytes{Bytes: x}
	case time.Time:
		cv.V = &sharedv1beta1.PageCursor_CursorValue_T{T: timestamppb.New(x)}
	case uuid.UUID:
		cv.V = &sharedv1beta1.PageCursor_CursorValue_Uuid{Uuid: x.String()}
	default:
		return nil, fmt.Errorf("cannot encode %T for cursor", v)
	}
	return cv, nil
}

func valueOf(cv *sharedv1beta1.PageCursor_CursorValue) any {
	switch x := cv.GetV().(type) {
	case *sharedv1beta1.PageCursor_CursorValue_S:
		return x.S
	case *sharedv1beta1.PageCursor_CursorValue_I:
		return x.I
	case *sharedv1beta1.PageCursor_CursorValue_F:
		return x.F
	case *sharedv1beta1.PageCursor_CursorValue_B:
		return x.B
	case *sharedv1beta1.PageCursor_CursorValue_Bytes:
		return x.Bytes
	case *sharedv1beta1.PageCursor_CursorValue_T:
		return x.T.AsTime()
	case *sharedv1beta1.PageCursor_CursorValue_Uuid:
		u, _ := uuid.Parse(x.Uuid)
		return u
	}
	return nil
}

func encodeCursor(c *sharedv1beta1.PageCursor) (string, error) {
	b, err := proto.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func decodeCursor(s string) (*sharedv1beta1.PageCursor, error) {
	if s == "" {
		return nil, nil
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	c := &sharedv1beta1.PageCursor{}
	if err := proto.Unmarshal(b, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (o PaginateOptions[T, R, O, P]) GetIDField() string {
	if o.IDField == "" {
		return "id"
	}
	return o.IDField
}

func (o PaginateOptions[T, R, O, P]) GetDefaultPageSize() int32 {
	if o.DefaultPageSize == 0 {
		return 10
	}
	return o.DefaultPageSize
}

func (o PaginateOptions[T, R, O, P]) GetMinPageSize() int32 {
	if o.MinPageSize == 0 {
		return 1
	}
	return o.MinPageSize
}

func (o PaginateOptions[T, R, O, P]) GetMaxPageSize() int32 {
	if o.MaxPageSize == 0 {
		return 100
	}
	return o.MaxPageSize
}
