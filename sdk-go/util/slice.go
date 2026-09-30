package util

import (
	"slices"
)

func SliceOf[T any](t ...T) (v []T) {
	return append(v, t...)
}

func SliceOfFunc[V, T any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}
	return result
}

func SliceOfFuncErr[T, V any](ts []T, fn func(T) (V, error)) (s []V, _ error) {
	for _, t := range ts {
		if v, err := fn(t); err != nil {
			return nil, err
		} else {
			s = append(s, v)
		}
	}
	return
}

func SliceOfAny[T any](s []T) []any {
	var result []any
	for _, v := range s {
		result = append(result, v)
	}
	return result
}

func SliceOfString[T ~string](s []T) []string {
	var result []string
	for _, v := range s {
		result = append(result, string(v))
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

func SliceReduceErr[T, V any](ts []T, fn func(T) (V, bool, error)) (s []V, _ error) {
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

func SliceReduceToMap[T, V any, K comparable](slice []T, fn func(int, T) (K, V, bool)) (m map[K]V) {
	m = map[K]V{}
	for idx, v := range slice {
		if k, v, ok := fn(idx, v); ok {
			m[k] = v
		}
	}
	return
}

func SliceReduceToMapErr[T, V any, K comparable](slice []T, fn func(T) (K, V, bool, error)) (m map[K]V, _ error) {
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

func SliceJoin[S ~[]T, T any](ss ...S) (s S) {
	for idx := range ss {
		s = append(s, ss[idx]...)
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

// SliceIntersect returns a new slice containing elements present in both s1 and s2.
func SliceIntersect[T comparable](slice1, slice2 []T) (diff []T) {
	// Create a map to store the values of slice2.
	set := make(map[T]struct{})

	// Create a slice to store the result.
	// Add the values of slice1 to the set.
	for _, item := range slice1 {
		set[item] = struct{}{}
	}

	// Iterate over the values of slice2.
	for _, item := range slice2 {
		// Check if the item is in the set.
		if _, ok := set[item]; !ok {
			// If the item is not in the set, add it to the result slice.
			diff = append(diff, item)
			delete(set, item)
		}
	}

	return
}

// SliceUnion returns a new slice containing unique elements from both s1 and s2.
func SliceUnion[T comparable](s1, s2 []T) []T {
	// Use an empty struct to save memory in the map
	seen := make(map[T]struct{})
	var result []T

	// Add elements from the first slice
	for _, val := range s1 {
		if _, exists := seen[val]; !exists {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}

	// Add elements from the second slice
	for _, val := range s2 {
		if _, exists := seen[val]; !exists {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}

	return result
}
