package shared

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

type (
	Executor[T Node] struct {
		graph *Graph[T]

		// Control flow callbacks
		onInit   OnStart[T]
		onStarts []OnStart[T]
		onEnds   []OnEnd[T]

		// Node flow callbacks
		beforeSends []OnSend[T]
		afterRecvs  []OnRecv[T]

		// Start and end nodes
		starts,
		ends []T

		// Start and end channels
		startsCh,
		endsCh map[string]chan any

		runOnce,
		initOnce sync.Once
		runErr error
	}
	ExecutorOption[T Node] func(*Executor[T])
	OnStart[T Node]        func(context.Context, ...T) (map[string]any, error)
	OnEnd[T Node]          func(context.Context, ...T) error
	OnSend[T Node]         func(context.Context, T, any) error
	OnRecv[T Node]         func(context.Context, T, any) error
	NodeValue[T Node]      struct {
		Node  T
		Value any
	}
)

func NewExecutor[T Node](graph *Graph[T], opts ...ExecutorOption[T]) *Executor[T] {
	e := &Executor[T]{
		graph:    graph,
		startsCh: map[string]chan any{},
		endsCh:   map[string]chan any{},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}

	return e
}

// WithInitCycle is an option for setting the initial values of start nodes
// (without inbound edges) and is ran exactly once before the first start cycle.
func WithInitCycle[T Node](onInit OnStart[T]) ExecutorOption[T] {
	return func(e *Executor[T]) {
		e.onInit = onInit
	}
}

// WithBeforeStart is an option for setting the values of start nodes (without
// inbound edges) and is ran before each cycle.
func WithBeforeStart[T Node](onStart OnStart[T]) ExecutorOption[T] {
	return func(e *Executor[T]) {
		e.onStarts = append(e.onStarts, onStart)
	}
}

// WithAfterEnd is an option for receiving the values of end nodes (without
// outbound edges) and is ran after each cycle.
func WithAfterEnd[T Node](onEnd OnEnd[T]) ExecutorOption[T] {
	return func(e *Executor[T]) {
		e.onEnds = append(e.onEnds, onEnd)
	}
}

// WithBeforeSend is an option for a callback to be called before a node is sent
// its next value.
func WithBeforeSend[T Node](onSend OnSend[T]) ExecutorOption[T] {
	return func(e *Executor[T]) {
		e.beforeSends = append(e.beforeSends, onSend)
	}
}

// WithAfterRecv is an option for a callback to receive a node's value after it
// has been received.
func WithAfterRecv[T Node](onRecv OnRecv[T]) ExecutorOption[T] {
	return func(e *Executor[T]) {
		e.afterRecvs = append(e.afterRecvs, onRecv)
	}
}

func (e *Executor[T]) Run(ctx context.Context, env Env) error {
	e.runOnce.Do(func() {
		var (
			errGroup  errgroup.Group
			nodeFuncs [][2]func() error
		)
		for _, node := range e.graph.Nodes() {
			var (
				inbound        = e.graph.Inbound(node.Ident())
				outbound       = e.graph.Outbound(node.Ident())
				isStart        = len(inbound) == 0
				isEnd          = len(outbound) == 0
				startCh, endCh chan any
			)

			if isStart {
				startCh = make(chan any)
				e.starts = append(e.starts, node)
				e.startsCh[node.Ident()] = startCh
			}

			if isEnd {
				endCh = make(chan any)
				e.ends = append(e.ends, node)
				e.endsCh[node.Ident()] = endCh
			}

			nodeFuncs = append(nodeFuncs, [2]func() error{
				func() error {
					err := node.Exec().Run(ctx, env, node.Pipe())
					if err != nil {
						return fmt.Errorf("%q: %w", node.Ident(), err)
					}
					return nil
				},
				func() error {
					err := e.runNodeFlow(ctx, node, startCh, endCh, inbound, outbound)
					if err != nil {
						return fmt.Errorf("%q: %w", node.Ident(), err)
					}
					return nil
				},
			})
		}

		for _, nodeFunc := range nodeFuncs {
			errGroup.Go(nodeFunc[0])
			errGroup.Go(nodeFunc[1])
		}

		errGroup.Go(func() error {
			return e.runControlFlow(ctx)
		})
		e.runErr = errGroup.Wait()
	})
	return e.runErr
}

func (e *Executor[T]) runControlFlow(ctx context.Context) (err error) {
	for {
		err = e.runStarts(ctx)
		if err != nil {
			return
		}

		err = e.runEnds(ctx)
		if err != nil {
			return
		}
	}
}

func (e *Executor[T]) runStarts(ctx context.Context) (err error) {
	var (
		starts map[string]any
		isInit bool
	)
	if e.onInit != nil {
		e.initOnce.Do(func() {
			isInit = true
			starts, err = e.onInit(ctx, e.starts...)
		})
		if err != nil {
			return
		}
	}

	if starts == nil {
		starts = map[string]any{}
	}

	for _, onStart := range e.onStarts {
		if onStart != nil {
			vals, err := onStart(ctx, e.starts...)
			if err != nil {
				return err
			}

			for ident, val := range vals {
				if _, ok := starts[ident]; ok && isInit {
					continue
				}

				starts[ident] = val
			}
		}
	}

	for ident, startCh := range e.startsCh {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case startCh <- starts[ident]:
		}
	}

	return
}

func (e *Executor[T]) runEnds(ctx context.Context) (err error) {
	var ends []T
	for ident, endCh := range e.endsCh {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-endCh:
			node, ok := e.graph.nodes[ident]
			if ok {
				ends = append(ends, node)
			}
		}
	}

	for _, onEnd := range e.onEnds {
		if onEnd != nil {
			err = onEnd(ctx, ends...)
			if err != nil {
				return err
			}
		}
	}

	return
}

func (e *Executor[T]) runNodeFlow(ctx context.Context, node T, startCh, endCh chan any, inbound, outbound []*Edge[T]) (err error) {
	for {
		var value any
		if startCh != nil {
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			case value = <-startCh:
			}
		} else {
			for _, edge := range inbound {
				select {
				case <-ctx.Done():
					return context.Cause(ctx)
				case <-edge.Out():
				}
			}
		}

		for _, beforeSend := range e.beforeSends {
			if beforeSend != nil {
				err := beforeSend(ctx, node, value)
				if err != nil {
					return err
				}
			}
		}

		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case node.Send() <- value:
		}

		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case value = <-node.Recv():
			for _, afterRecv := range e.afterRecvs {
				if afterRecv != nil {
					err := afterRecv(ctx, node, value)
					if err != nil {
						return err
					}
				}
			}
		}

		if endCh != nil {
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			case endCh <- value:
			}
		} else {
			for _, edge := range outbound {
				select {
				case <-ctx.Done():
					return context.Cause(ctx)
				case edge.In() <- value:
				}
			}
		}
	}
}
