package entadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
)

type (
	PaginateOptions[T any, R PaginateRecord, O, P ~func(*sql.Selector)] struct {
		IDField   string
		TimeField string
		UUIDV7    bool
		GetUUID   func(R) uuid.UUID
		GetID     func(R) int64
		GetTime   func(R) time.Time
		DefaultPageSize,
		MinPageSize,
		MaxPageSize int32
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

func GetNextPage[T any, R PaginateRecord, P, O ~func(*sql.Selector), Q PaginateQuery[T, R, O, P]](ctx context.Context, req util.PageTokenRequest, query Q) (nextPageToken string, records []R, _ error) {
	return PaginateOptions[T, R, O, P]{}.GetNextPage(ctx, req, query)
}

func (o PaginateOptions[T, R, O, P]) GetNextPage(ctx context.Context, req util.PageTokenRequest, query PaginateQuery[T, R, O, P]) (nextPageToken string, records []R, _ error) {
	if o.UUIDV7 {
		return o.getNextPageUUIDV7(ctx, req, query)
	}
	return o.getNextPage(ctx, req, query)
}

func (o PaginateOptions[T, R, O, P]) getNextPageUUIDV7(ctx context.Context, req util.PageTokenRequest, query PaginateQuery[T, R, O, P]) (nextPageToken string, records []R, _ error) {
	lastUUID, pageSize, err := util.ParsePageTokenRequestUUIDV7(req, o.GetDefaultPageSize(), o.GetMinPageSize(), o.GetMaxPageSize())
	if err != nil {
		return "", nil, err
	}

	var preds []P
	if lastUUID != uuid.Nil {
		preds = append(preds,
			sql.FieldLT(o.GetIDField(), lastUUID),
		)
	}

	query.Where(preds...)
	query.Limit(int(pageSize))
	query.Order(
		sql.OrderByField(o.GetIDField(), sql.OrderDesc()).ToFunc(),
	)

	records, err = query.All(ctx)
	if err != nil {
		return "", nil, err
	}

	if len(records) == int(pageSize) {
		last := records[len(records)-1]

		var lastId uuid.UUID
		if o.GetUUID != nil {
			lastId = o.GetUUID(last)
		} else if getter, ok := any(last).(interface {
			GetID() uuid.UUID
		}); ok {
			lastId = getter.GetID()
		} else {
			idValue, err := last.Value(o.GetIDField())
			if err != nil {
				return "", nil, fmt.Errorf("%s value: %w", o.GetIDField(), err)
			}

			if v, ok := idValue.(uuid.UUID); ok {
				lastId = v
			} else {
				return "", nil, fmt.Errorf("%s invalid", o.GetIDField())
			}
		}

		nextPageToken = util.NextPageTokenUUIDV7(lastId)
	}

	return
}

func (o PaginateOptions[T, R, O, P]) getNextPage(ctx context.Context, req util.PageTokenRequest, query PaginateQuery[T, R, O, P]) (nextPageToken string, records []R, _ error) {
	lastIdx, updateTime, pageSize, err := util.ParsePageTokenRequest(req, o.GetDefaultPageSize(), o.GetMinPageSize(), o.GetMaxPageSize())
	if err != nil {
		return "", nil, err
	}

	var preds []P
	if lastIdx > 0 && !updateTime.IsZero() {
		preds = append(preds,
			sql.OrPredicates[P](
				sql.FieldLT(o.GetIDField(), updateTime),
				sql.AndPredicates[P](
					sql.FieldEQ(o.GetTimeField(), updateTime),
					sql.FieldLT(o.GetIDField(), lastIdx),
				),
			),
		)
	}

	query.Where(preds...)
	query.Limit(int(pageSize))
	query.Order(
		sql.OrderByField(o.GetIDField(), sql.OrderDesc()).ToFunc(),
		sql.OrderByField(o.GetTimeField(), sql.OrderDesc()).ToFunc(),
	)

	records, err = query.All(ctx)
	if err != nil {
		return "", nil, err
	}

	if len(records) == int(pageSize) {
		last := records[len(records)-1]

		var (
			lastId   int64
			lastTime time.Time
		)

		if o.GetID != nil {
			lastId = o.GetID(last)
		} else if getter, ok := any(last).(interface {
			GetID() int64
		}); ok {
			lastId = getter.GetID()
		} else {
			idValue, err := last.Value(o.GetIDField())
			if err != nil {
				return "", nil, fmt.Errorf("%s value: %w", o.GetIDField(), err)
			}

			if v, ok := idValue.(int64); ok {
				lastId = v
			} else {
				return "", nil, fmt.Errorf("%s invalid", o.GetIDField())
			}
		}

		if o.GetTime != nil {
			lastTime = o.GetTime(last)
		} else {
			timeValue, err := last.Value(o.GetTimeField())
			if err != nil {
				return "", nil, fmt.Errorf("%s value: %w", o.GetTimeField(), err)
			}

			if v, ok := timeValue.(*sql.NullTime); ok && v.Valid {
				lastTime = v.Time
			} else {
				return "", nil, fmt.Errorf("%s invalid", o.GetTimeField())
			}
		}

		nextPageToken, err = util.NextPageToken(lastId, lastTime)
		if err != nil {
			return "", nil, fmt.Errorf("next page token: %w", err)
		}
	}

	return
}

func (o PaginateOptions[T, R, O, P]) GetIDField() string {
	if o.IDField == "" {
		return "row_id"
	}
	return o.IDField
}

func (o PaginateOptions[T, R, O, P]) GetTimeField() string {
	if o.TimeField == "" {
		return "updated_at"
	}
	return o.TimeField
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
