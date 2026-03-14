package common

import (
	"context"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/itchyny/gojq"
)

type JSONMap map[string]any

func GetJSONValue[T any](json JSONMap, path string) (v T, ok bool) {
	if val, err := json.Value(path); err != nil {
		return
	} else if val == nil {
		return
	} else {
		v, ok = val.(T)
	}
	return
}

func JSONValueMust[T any](j JSONMap, key string) T {
	if val, ok := GetJSONValue[T](j, key); ok {
		return val
	}
	panic("property: JSONValueMust: key not found")
}

func (j JSONMap) Keys() []string {
	var keys []string
	for k := range j {
		keys = append(keys, k)
	}
	return keys
}

func (j JSONMap) QueryContext(ctx context.Context, expr string) ([]any, error) {
	q, err := gojq.Parse(expr)
	if err != nil {
		return nil, err
	}

	var vals []any
	iter := q.RunWithContext(ctx, map[string]any(j))
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			if err, ok := err.(*gojq.HaltError); ok && err.Value() == nil {
				break
			}
			return nil, err
		}
		vals = append(vals, v)
	}
	return vals, nil
}

func (j JSONMap) Query(expr string) ([]any, error) {
	return j.QueryContext(context.Background(), expr)
}

func (j JSONMap) ValueContext(ctx context.Context, expr string) (any, error) {
	vals, err := j.QueryContext(ctx, expr)
	if err != nil {
		return nil, err
	}
	if len(vals) == 0 {
		return nil, nil
	}
	for _, v := range vals {
		return v, nil
	}
	return nil, nil
}

func (j JSONMap) Value(expr string) (any, error) {
	return j.ValueContext(context.Background(), expr)
}

func (j JSONMap) Exists(expr string) (bool, error) {
	vals, err := j.Query(expr)
	if err != nil {
		return false, err
	}
	if len(vals) == 0 {
		return false, nil
	}
	for _, v := range vals {
		if v != nil {
			return true, nil
		}
	}
	return false, nil
}

func (j JSONMap) Equals(expr string, val any) (bool, error) {
	vals, err := j.Query(expr)
	if err != nil {
		return false, err
	}
	if len(vals) == 0 {
		return false, nil
	}
	for _, v := range vals {
		return val == v, nil
	}
	return false, nil
}

func (j JSONMap) IsNull(expr string) (bool, error) {
	return j.Equals(expr, nil)
}

func (j JSONMap) IsFalse(expr string) (bool, error) {
	return j.Equals(expr, false)
}

func (j JSONMap) IsTrue(expr string) (bool, error) {
	return j.Equals(expr, true)
}

func (j JSONMap) Merge(other JSONMap) JSONMap {
	return util.MergeMaps(j, other)
}
