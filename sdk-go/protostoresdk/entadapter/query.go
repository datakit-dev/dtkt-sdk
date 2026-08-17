package entadapter

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/protostore"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/cel-go/cel"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

var boolType = reflect.TypeFor[bool]()

type (
	PageQuery[Q Query[Q, R, P, O], R Record, M proto.Message, P, O ~func(*sql.Selector)] struct {
		pager Paginator[R, M]
		query Q

		pageSize int32
		lastUUID uuid.UUID
		lastId   int64
		lastTime time.Time

		filter cel.Program
	}
	Query[Q Query[Q, R, P, O], R Record, P, O ~func(*sql.Selector)] interface {
		Where(...P) Q
		Order(...O) Q
		Offset(int) Q
		Limit(int) Q
		Clone() Q
		Count(context.Context) (int, error)
		All(context.Context) ([]R, error)
	}
)

func NewPageQuery[Q Query[Q, R, P, O], R Record, M proto.Message, P, O ~func(*sql.Selector)](pager Paginator[R, M], query Q, req util.PageRequest) (_ *PageQuery[Q, R, M, P, O], err error) {
	var (
		pageSize int32
		lastUUID uuid.UUID
		lastId   int64
		lastTime time.Time
	)

	if pager.isUUID {
		lastUUID, pageSize, err = util.ParsePageTokenRequestUUID(req, pager.GetDefaultPageSize(), pager.GetMinPageSize(), pager.GetMaxPageSize())
		if err != nil {
			return
		}
	} else {
		lastId, lastTime, pageSize, err = util.ParsePageTokenRequest(req, pager.GetDefaultPageSize(), pager.GetMinPageSize(), pager.GetMaxPageSize())
		if err != nil {
			return
		}
	}

	var filter cel.Program
	if req, ok := req.(util.FilterRequest); ok {
		expr := strings.TrimSpace(req.GetFilter())
		if expr != "" {
			types, err := common.NewCELTypes(protostore.Resolver)
			if err != nil {
				return nil, err
			}

			var msg M
			env, err := common.NewCELEnv(
				cel.CustomTypeAdapter(types),
				cel.CustomTypeProvider(types),
				cel.Variable("this", cel.ObjectType(string(msg.ProtoReflect().Descriptor().FullName()))),
			)
			if err != nil {
				return nil, err
			}

			ast, iss := env.Compile(expr)
			if iss.Err() != nil {
				return nil, fmt.Errorf("invalid filter expression: %w", iss.Err())
			}

			filter, err = env.Program(ast)
			if err != nil {
				return nil, err
			}
		}
	}

	return &PageQuery[Q, R, M, P, O]{
		pager: pager,
		query: query,

		pageSize: pageSize,
		lastTime: lastTime,
		lastId:   lastId,
		lastUUID: lastUUID,

		filter: filter,
	}, nil
}

func (q *PageQuery[Q, R, M, P, O]) getNextPage(ctx context.Context) (nextPageToken string, records []R, err error) {
	var hasMore bool

	if q.filter == nil {
		records, err = q.query.Clone().
			Where(q.getPreds()...).
			Order(q.getOrder()...).
			Limit(int(q.pageSize) + 1).
			All(ctx)
		if err != nil {
			return
		}

		if len(records) > int(q.pageSize) {
			records = records[:q.pageSize]
			hasMore = true
		}
	} else {
		var count int
		count, err = q.getCount(ctx)
		if err != nil || count == 0 {
			return "", nil, err
		}

		var (
			limit = max(defaultMaxPageSize, int(q.pageSize))
			batch []R
		)

		for idx := range max(1, (count+limit-1)/limit) {
			offset := idx * limit

			batch, err = q.getFiltered(ctx, limit, offset)
			if err != nil {
				return
			}

			records = append(records, batch...)

			if len(records) > int(q.pageSize) {
				records = records[:q.pageSize]
				hasMore = true
				break
			}
		}
	}

	if hasMore {
		nextPageToken, err = q.getNextPageToken(records)
		if err != nil {
			return
		}
	}

	return
}

func (q *PageQuery[Q, R, M, P, O]) getNextPageToken(records []R) (_ string, _ error) {
	if len(records) == 0 {
		return
	}

	last := records[len(records)-1]
	if q.pager.isUUID {
		id, err := q.getRecordUUID(last)
		if err != nil {
			return
		}

		return util.NextPageTokenUUID(id), nil
	}

	id, err := q.getRecordID(last)
	if err != nil {
		return
	}

	time, err := q.getRecordTime(last)
	if err != nil {
		return
	}

	return util.NextPageToken(id, time)
}

func (q *PageQuery[Q, R, M, P, O]) getRecordUUID(record R) (id uuid.UUID, err error) {
	if q.pager.getUUID != nil {
		id = q.pager.getUUID(record)
	} else if getter, ok := any(record).(GetRecordUUID); ok {
		id = getter.GetID()
	} else {
		idValue, err := record.Value(q.pager.GetIDField())
		if err != nil {
			return uuid.Nil, fmt.Errorf("%s value: %w", q.pager.GetIDField(), err)
		}

		if v, ok := idValue.(uuid.UUID); ok {
			id = v
		} else {
			return uuid.Nil, fmt.Errorf("%s invalid", q.pager.GetIDField())
		}
	}
	return
}

func (q *PageQuery[Q, R, M, P, O]) getRecordTime(record R) (t time.Time, err error) {
	if q.pager.getTime != nil {
		t = q.pager.getTime(record)
	} else {
		var timeValue ent.Value
		timeValue, err = record.Value(q.pager.GetTimeField())
		if err != nil {
			return
		}

		if v, ok := timeValue.(*sql.NullTime); ok && v.Valid {
			t = v.Time
		} else {
			err = fmt.Errorf("%s invalid", q.pager.GetTimeField())
			return
		}
	}
	return
}

func (q *PageQuery[Q, R, M, P, O]) getRecordID(record R) (id int64, err error) {
	if q.pager.getID != nil {
		id = q.pager.getID(record)
	} else if getter, ok := any(record).(GetRecordID); ok {
		id = getter.GetID()
	} else {
		var idValue ent.Value
		idValue, err = record.Value(q.pager.GetIDField())
		if err != nil {
			return
		}

		if v, ok := idValue.(int64); ok {
			id = v
		} else {
			err = fmt.Errorf("%s invalid", q.pager.GetIDField())
			return
		}
	}
	return
}

func (q *PageQuery[Q, R, M, P, O]) getPreds() (preds []P) {
	if q.pager.isUUID {
		if q.lastUUID != uuid.Nil {
			preds = append(preds,
				sql.FieldLT(q.pager.GetIDField(), q.lastUUID),
			)
		}
	} else {
		if q.lastId > 0 && !q.lastTime.IsZero() {
			preds = append(preds,
				sql.OrPredicates[P](
					sql.FieldLT(q.pager.GetTimeField(), q.lastTime),
					sql.AndPredicates[P](
						sql.FieldLT(q.pager.GetIDField(), q.lastId),
						sql.FieldEQ(q.pager.GetTimeField(), q.lastTime),
					),
				),
			)
		}
	}

	return
}

func (q *PageQuery[Q, R, M, P, O]) getOrder() (order []O) {
	if q.pager.isUUID {
		return []O{
			sql.OrderByField(q.pager.GetIDField(), sql.OrderDesc()).ToFunc(),
		}
	}

	return []O{
		sql.OrderByField(q.pager.GetTimeField(), sql.OrderDesc()).ToFunc(),
		sql.OrderByField(q.pager.GetIDField(), sql.OrderDesc()).ToFunc(),
	}
}

func (q *PageQuery[Q, R, M, P, O]) getCount(ctx context.Context) (int, error) {
	return q.query.Clone().
		Where(q.getPreds()...).
		Order(q.getOrder()...).
		Count(ctx)
}

func (q *PageQuery[Q, R, M, P, O]) getFiltered(ctx context.Context, limit, offset int) ([]R, error) {
	records, err := q.query.Clone().
		Where(q.getPreds()...).
		Order(q.getOrder()...).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 || q.filter == nil {
		return records, nil
	}

	return util.SliceReduceError(records, func(record R) (_ R, ok bool, err error) {
		ok, err = q.checkRecord(ctx, record)
		if !ok || err != nil {
			return
		}
		return record, true, nil
	})
}

func (q *PageQuery[Q, R, M, P, O]) checkRecord(ctx context.Context, record R) (_ bool, err error) {
	proto, err := q.pager.toProto(ctx, record)
	if err != nil {
		return
	}

	value, _, err := q.filter.ContextEval(ctx, map[string]any{
		"this": proto,
	})
	if err != nil {
		return
	} else if value == nil {
		return
	}

	ok, err := value.ConvertToNative(boolType)
	if err != nil {
		return
	}

	return ok.(bool), nil
}
