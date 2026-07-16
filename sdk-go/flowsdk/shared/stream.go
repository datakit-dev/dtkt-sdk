package shared

import (
	"context"

	streams "github.com/reugn/go-streams"
)

type (
	Pipe interface {
		streams.Inlet
		streams.Outlet
		Run(context.Context, Exec) error
	}
	Exec   func(context.Context, any) (any, error)
	Future interface {
		Value(context.Context) (any, bool, error)
	}
	FutureFunc func(context.Context) (any, bool, error)
)

func (f FutureFunc) Value(ctx context.Context) (any, bool, error) {
	return f(ctx)
}
