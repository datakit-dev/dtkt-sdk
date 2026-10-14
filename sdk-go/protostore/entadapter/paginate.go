package entadapter

import (
	"context"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
)

const (
	defaultPageSize    = 10
	defaultMinPageSize = 1
	defaultMaxPageSize = 100
)

type (
	Paginator[R Record, M proto.Message] struct {
		defPageSize,
		minPageSize,
		maxPageSize int32

		toProto ToProtoFunc[R, M]

		idField string
		getID   GetIDFunc[R]
		getUUID GetUUIDFunc[R]
		isUUID  bool

		timeField string
		getTime   GetTimeFunc[R]
	}
	PaginatorOption[R Record, M proto.Message] func(*Paginator[R, M])

	Record interface {
		Value(string) (ent.Value, error)
	}

	GetUUIDFunc[R Record] func(R) uuid.UUID
	GetIDFunc[R Record]   func(R) int64
	GetTimeFunc[R Record] func(R) time.Time
	GetRecordID           interface {
		GetID() int64
	}
	GetRecordUUID interface {
		GetID() uuid.UUID
	}

	ToProtoFunc[R Record, M proto.Message] func(context.Context, R) (M, error)
)

func NewPaginator[R Record, M proto.Message](toProto ToProtoFunc[R, M], opts ...PaginatorOption[R, M]) Paginator[R, M] {
	pager := Paginator[R, M]{
		toProto: toProto,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&pager)
		}
	}

	return pager
}

func WithPaginatorDefaultPageSize[R Record, M proto.Message](defPageSize int32) PaginatorOption[R, M] {
	return func(p *Paginator[R, M]) {
		p.defPageSize = defPageSize
	}
}

func WithPaginatorMinPageSize[R Record, M proto.Message](minPageSize int32) PaginatorOption[R, M] {
	return func(p *Paginator[R, M]) {
		p.minPageSize = minPageSize
	}
}
func WithPaginatorMaxPageSize[R Record, M proto.Message](maxPageSize int32) PaginatorOption[R, M] {
	return func(p *Paginator[R, M]) {
		p.maxPageSize = maxPageSize
	}
}

func WithPaginatorUUIDField[R Record, M proto.Message](idField string, getUUID GetUUIDFunc[R]) PaginatorOption[R, M] {
	return func(p *Paginator[R, M]) {
		p.idField = idField
		p.getUUID = getUUID
		p.isUUID = true
	}
}

func WithPaginatorIDField[R Record, M proto.Message](idField string, getID GetIDFunc[R]) PaginatorOption[R, M] {
	return func(p *Paginator[R, M]) {
		p.idField = idField
		p.getID = getID
	}
}

func WithPaginatorTimeField[R Record, M proto.Message](timeField string, getTime GetTimeFunc[R]) PaginatorOption[R, M] {
	return func(p *Paginator[R, M]) {
		p.timeField = timeField
		p.getTime = getTime
	}
}

func (p Paginator[R, M]) GetNextPage[Q Query[Q, R, P, O], P, O ~func(*sql.Selector)](ctx context.Context, req util.PageRequest, q Q) (_ string, _ []R, err error) {
	query, err := NewPageQuery(p, q, req)
	if err != nil {
		return
	}

	return query.getNextPage(ctx)
}

func (p Paginator[R, M]) GetIDField() string {
	if p.idField == "" {
		return "id"
	}
	return p.idField
}

func (p Paginator[R, M]) GetTimeField() string {
	if p.timeField == "" {
		return "updated_at"
	}
	return p.timeField
}

func (p Paginator[R, M]) GetDefaultPageSize() int32 {
	if p.defPageSize == 0 {
		return defaultPageSize
	}
	return p.defPageSize
}

func (p Paginator[R, M]) GetMinPageSize() int32 {
	if p.minPageSize <= 0 {
		return defaultMinPageSize
	}
	return p.minPageSize
}

func (p Paginator[R, M]) GetMaxPageSize() int32 {
	if p.maxPageSize == 0 {
		return defaultMaxPageSize
	}
	return p.maxPageSize
}
