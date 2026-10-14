package entadapter

import (
	"context"
	"strings"
	"testing"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/stretchr/testify/require"
)

const (
	mockIDField   = "rowid"
	mockTimeField = "updated_at"
)

type (
	mockRecord struct {
		id        int64
		updatedAt time.Time
		name      string
	}
	mockPredicate func(*sql.Selector)
	mockOrder     func(*sql.Selector)
	mockQuery     struct {
		records []*mockRecord
		offset  int
		limit   int
		stats   *mockQueryStats
	}
	mockQueryStats struct {
		countCalls int
		allCalls   int
		lastLimit  int
		wherePreds int
	}
	mockPageRequest struct {
		pageSize  int32
		pageToken string
		filter    string
	}
)

func (r *mockRecord) Value(field string) (ent.Value, error) {
	switch field {
	case mockIDField:
		return r.id, nil
	case mockTimeField:
		return r.updatedAt, nil
	default:
		return nil, nil
	}
}

func (q *mockQuery) Where(predicates ...mockPredicate) *mockQuery {
	if q.stats != nil {
		q.stats.wherePreds += len(predicates)
	}
	return q
}

func (q *mockQuery) Order(...mockOrder) *mockQuery {
	return q
}

func (q *mockQuery) Offset(offset int) *mockQuery {
	q.offset = offset
	return q
}

func (q *mockQuery) Limit(limit int) *mockQuery {
	q.limit = limit
	return q
}

func (q *mockQuery) Clone() *mockQuery {
	clone := *q
	return &clone
}

func (q *mockQuery) Count(context.Context) (int, error) {
	if q.stats != nil {
		q.stats.countCalls++
	}
	return len(q.records), nil
}

func (q *mockQuery) All(context.Context) ([]*mockRecord, error) {
	if q.stats != nil {
		q.stats.allCalls++
		q.stats.lastLimit = q.limit
	}
	start := min(q.offset, len(q.records))
	end := len(q.records)
	if q.limit > 0 {
		end = min(start+q.limit, end)
	}
	return q.records[start:end], nil
}

func (r mockPageRequest) GetPageSize() int32 {
	return r.pageSize
}

func (r mockPageRequest) GetPageToken() string {
	return r.pageToken
}

func (r mockPageRequest) GetFilter() string {
	return r.filter
}

func newMockPaginator() Paginator[*mockRecord, *corev1.Flow] {
	return NewPaginator(
		func(_ context.Context, record *mockRecord) (*corev1.Flow, error) {
			return &corev1.Flow{Name: record.name}, nil
		},
		WithPaginatorIDField[*mockRecord, *corev1.Flow](mockIDField, func(record *mockRecord) int64 {
			return record.id
		}),
		WithPaginatorTimeField[*mockRecord, *corev1.Flow](mockTimeField, func(record *mockRecord) time.Time {
			return record.updatedAt
		}),
	)
}

func TestPageQueryCursorMatchesOrder(t *testing.T) {
	cursorTime := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	token, err := util.NextPageToken(42, cursorTime)
	require.NoError(t, err)

	query, err := NewPageQuery(newMockPaginator(), &mockQuery{}, mockPageRequest{
		pageSize:  10,
		pageToken: token,
	})
	require.NoError(t, err)

	predicates := query.getPreds()
	require.Len(t, predicates, 1)
	selector := sql.Select().From(sql.Table("records"))
	predicates[0](selector)
	statement, _ := selector.Query()
	require.Contains(t, statement, "`updated_at` < ?")

	orders := query.getOrder()
	require.Len(t, orders, 2)
	selector = sql.Select().From(sql.Table("records"))
	for _, order := range orders {
		order(selector)
	}
	statement, _ = selector.Query()
	require.Less(t, strings.Index(statement, "`updated_at`"), strings.Index(statement, "`rowid`"))
}

func TestPageQueryAppliesLivePageToken(t *testing.T) {
	stats := new(mockQueryStats)
	query, err := NewPageQuery(newMockPaginator(), &mockQuery{
		records: newMockRecords(1),
		stats:   stats,
	}, mockPageRequest{
		pageSize:  10,
		pageToken: "CMAFEgwI3O6J1AYQv6vLxQI=",
	})
	require.NoError(t, err)
	require.Equal(t, int64(704), query.lastId)
	require.Equal(t, time.Unix(1786935132, 682808767).UTC(), query.lastTime)

	_, _, err = query.getNextPage(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, stats.wherePreds)
}

func TestPageQueryFilterIncludesBatchBoundary(t *testing.T) {
	records := newMockRecords(201)
	records[100].name = "match"

	_, page, err := newMockPaginator().GetNextPage(context.Background(), mockPageRequest{
		pageSize: 1,
		filter:   `this.name == "match"`,
	}, &mockQuery{records: records})
	require.NoError(t, err)
	require.Equal(t, []*mockRecord{records[100]}, page)
}

func TestPageQueryFilterIncludesFinalPartialBatch(t *testing.T) {
	records := newMockRecords(101)
	records[100].name = "match"

	_, page, err := newMockPaginator().GetNextPage(context.Background(), mockPageRequest{
		pageSize: 1,
		filter:   `this.name == "match"`,
	}, &mockQuery{records: records})
	require.NoError(t, err)
	require.Equal(t, []*mockRecord{records[100]}, page)
}

func TestPageQueryUnfilteredOverfetchesWithoutCounting(t *testing.T) {
	t.Run("exact final page", func(t *testing.T) {
		stats := new(mockQueryStats)
		token, page, err := newMockPaginator().GetNextPage(context.Background(), mockPageRequest{
			pageSize: 10,
		}, &mockQuery{
			records: newMockRecords(10),
			stats:   stats,
		})
		require.NoError(t, err)
		require.Len(t, page, 10)
		require.Empty(t, token)
		require.Zero(t, stats.countCalls)
		require.Equal(t, 1, stats.allCalls)
		require.Equal(t, 11, stats.lastLimit)
	})

	t.Run("additional page", func(t *testing.T) {
		stats := new(mockQueryStats)
		token, page, err := newMockPaginator().GetNextPage(context.Background(), mockPageRequest{
			pageSize: 10,
		}, &mockQuery{
			records: newMockRecords(11),
			stats:   stats,
		})
		require.NoError(t, err)
		require.Len(t, page, 10)
		require.NotEmpty(t, token)
		require.Zero(t, stats.countCalls)
		require.Equal(t, 1, stats.allCalls)
		require.Equal(t, 11, stats.lastLimit)
	})
}

func newMockRecords(count int) []*mockRecord {
	records := make([]*mockRecord, count)
	for idx := range records {
		records[idx] = &mockRecord{
			id:        int64(count - idx),
			updatedAt: time.Unix(int64(count-idx), 0),
		}
	}
	return records
}
