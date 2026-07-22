package util

import (
	"context"
	"errors"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"
)

func SliceOf[T any](t ...T) (v []T) {
	return append(v, t...)
}

func AnySlice[T any](s []T) []any {
	var result []any
	for _, v := range s {
		result = append(result, v)
	}
	return result
}

func StringSlice[T ~string](s []T) []string {
	var result []string
	for _, v := range s {
		result = append(result, string(v))
	}
	return result
}

func SliceMap[V, T any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}
	return result
}

// SliceReduce applies a function to each element of the input slice and
// includes the result in the output slice when the function returns true.
func SliceReduce[T, V any](ts []T, fn func(T) (V, bool)) (s []V) {
	for _, t := range ts {
		if v, ok := fn(t); ok {
			s = append(s, v)
		}
	}
	return
}

func SliceReduceError[T, V any](ts []T, fn func(T) (V, bool, error)) (s []V, _ error) {
	for _, t := range ts {
		if v, ok, err := fn(t); err != nil {
			return nil, err
		} else if ok {
			s = append(s, v)
		}
	}
	return
}

func SliceToMap[T, V any, K comparable](slice []T, fn func(int, T) (K, V)) (m map[K]V) {
	m = map[K]V{}
	for idx, v := range slice {
		k, v := fn(idx, v)
		m[k] = v
	}
	return
}

func ReduceSliceToMap[T, V any, K comparable](slice []T, fn func(int, T) (K, V, bool)) (m map[K]V) {
	m = map[K]V{}
	for idx, v := range slice {
		if k, v, ok := fn(idx, v); ok {
			m[k] = v
		}
	}
	return
}

func ReduceSliceToMapErr[T, V any, K comparable](slice []T, fn func(T) (K, V, bool, error)) (m map[K]V, _ error) {
	m = map[K]V{}
	for _, v := range slice {
		k, v, ok, err := fn(v)
		if err != nil {
			return nil, err
		} else if ok {
			m[k] = v
		}
	}
	return
}

func JoinSlices[S ~[]T, T any](ss ...S) (s S) {
	for idx := range ss {
		s = append(s, ss[idx]...)
	}
	return
}

func SliceMapError[T, V any](ts []T, fn func(T) (V, error)) (s []V, _ error) {
	for _, t := range ts {
		if v, err := fn(t); err != nil {
			return nil, err
		} else {
			s = append(s, v)
		}
	}
	return
}

func SliceCount[T any](ts []T, fn func(T) bool) (count int) {
	for _, t := range ts {
		if fn(t) {
			count++
		}
	}
	return
}

func SliceSet[T comparable](slice []T) (set []T) {
	for _, item := range slice {
		if !slices.Contains(set, item) {
			set = append(set, item)
		}
	}
	return
}

func SliceWithout[T comparable](collection []T, elems ...T) []T {
	var result []T
	for _, x := range collection {
		if !slices.Contains(elems, x) {
			result = append(result, x)
		}
	}
	return result
}

func SliceGroups[T comparable](elems ...T) [][]T {
	groups := map[T][]T{}
	for _, a := range elems {
		groups[a] = append(groups[a], a)
	}
	return MapVals(groups)
}

func SliceGroupsFunc[T any](elems []T, eq func(T, T) bool) [][]T {
	var groups [][]T
	for _, a := range elems {
		found := false
		for i, g := range groups {
			if eq(a, g[0]) {
				groups[i] = append(groups[i], a)
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, []T{a})
		}
	}
	return groups
}

func ContextMapParallelErr[T any, R any](ctx context.Context, slice []T, fn func(ctx context.Context, t T) (R, error)) ([]R, error) {
	g, ctx := errgroup.WithContext(ctx)

	results := make([]R, len(slice))
	for idx, item := range slice {
		g.Go(func() error {
			result, err := fn(ctx, item)
			if err != nil {
				return err
			}

			results[idx] = result

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

func ContextMapParallelErrs[T any, R any](ctx context.Context, slice []T, fn func(context.Context, T) (R, error)) (results []R, errs []error) {
	var g sync.WaitGroup

	results = make([]R, len(slice))
	errs = make([]error, len(slice))
	for idx, item := range slice {
		g.Go(func() {
			result, err := fn(ctx, item)
			if err != nil {
				errs[idx] = err
				return
			}

			results[idx] = result
		})
	}
	g.Wait()

	return results, errs
}

func MapParallelErr[T any, R any](slice []T, fn func(T) (R, error)) ([]R, error) {
	results, errs := MapParallelErrs(slice, fn)
	err := errors.Join(errs...)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func MapParallelErrs[T any, R any](slice []T, fn func(T) (R, error)) (results []R, errs []error) {
	var g sync.WaitGroup

	results = make([]R, len(slice))
	errs = make([]error, len(slice))
	for idx, item := range slice {
		g.Go(func() {
			result, err := fn(item)
			if err != nil {
				errs[idx] = err
				return
			}

			results[idx] = result
		})
	}
	g.Wait()

	return results, errs
}
