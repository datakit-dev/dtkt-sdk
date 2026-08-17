package runtime

import (
	"context"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	_ "github.com/reugn/go-streams"
)

type Pipe[I any, O any] struct {
	inCh  chan I
	outCh chan O
}

func NewPipe[I any, O any](in chan I, out chan O) *Pipe[I, O] {
	return &Pipe[I, O]{
		inCh:  in,
		outCh: out,
	}
}

func (p *Pipe[I, O]) In() chan<- I {
	return p.inCh
}

func (p *Pipe[I, O]) Out() <-chan O {
	return p.outCh
}

func (p *Pipe[I, O]) Run(ctx context.Context, exec shared.RunFunc[O, I]) (err error) {
	for {
		var (
			in  I
			out O
		)

		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case out = <-p.Out():
		}

		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		default:
			in, err = exec(ctx, out)
			if err != nil {
				return
			}
		}

		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case p.In() <- in:
		}
	}
}
