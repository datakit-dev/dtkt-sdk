package util

import (
	"context"
	"errors"
	"sync"

	"golang.org/x/sync/errgroup"
)

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
