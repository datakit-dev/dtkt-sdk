package shared

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

type (
	Executor[T GraphNode] struct {
		graph *Graph[T]

		// Control flow callbacks
		onInit   OnStart[T]
		onStarts []OnStart[T]
		onEnds   []OnEnd[T]

		// Node flow callbacks
		onSends []OnSend[T]
		onRecvs []OnRecv[T]

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
	Pipe                 = PipeOf[any, any]
	PipeOf[I any, O any] interface {
		In() chan<- I
		Out() <-chan O
		Run(context.Context, RunFunc[O, I]) error
	}
	RunFunc[I any, O any]       func(context.Context, I) (O, error)
	OnStart[T GraphNode]        func(context.Context, GraphNodes[T]) (map[string]any, error)
	OnEnd[T GraphNode]          func(context.Context, GraphNodes[T]) error
	OnSend[T GraphNode]         func(context.Context, T, any) error
	OnRecv[T GraphNode]         func(context.Context, T, any) (any, error)
	ExecutorOption[T GraphNode] func(*Executor[T])
)

func NewExecutor[T GraphNode](graph *Graph[T], opts ...ExecutorOption[T]) *Executor[T] {
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

// WithInitStart is an option for setting the initial values of start nodes
// (without inbound edges) and is ran exactly once before the first start cycle.
func WithInitStart[T GraphNode](onInit OnStart[T]) ExecutorOption[T] {
	return func(e *Executor[T]) {
		e.onInit = onInit
	}
}

// WithBeforeStart is an option for setting the values of start nodes (without
// inbound edges) and is ran before each cycle.
func WithBeforeStart[T GraphNode](onStart OnStart[T]) ExecutorOption[T] {
	return func(e *Executor[T]) {
		if onStart != nil {
			e.onStarts = append(e.onStarts, onStart)
		}
	}
}

// WithAfterEnd is an option for receiving the values of end nodes (without
// outbound edges) and is ran after each cycle.
func WithAfterEnd[T GraphNode](onEnd OnEnd[T]) ExecutorOption[T] {
	return func(e *Executor[T]) {
		if onEnd != nil {
			e.onEnds = append(e.onEnds, onEnd)
		}
	}
}

// WithBeforeSend is an option for a callback to be called before a node is sent
// its next value.
func WithBeforeSend[T GraphNode](onSend OnSend[T]) ExecutorOption[T] {
	return func(e *Executor[T]) {
		if onSend != nil {
			e.onSends = append(e.onSends, onSend)
		}
	}
}

// WithAfterRecv is an option for a callback to receive a node's value after it
// has been received.
func WithAfterRecv[T GraphNode](onRecv OnRecv[T]) ExecutorOption[T] {
	return func(e *Executor[T]) {
		if onRecv != nil {
			e.onRecvs = append(e.onRecvs, onRecv)
		}
	}
}

func (e *Executor[T]) Run(ctx context.Context) (err error) {
	e.runOnce.Do(func() {
		var (
			group errgroup.Group
			execs [][2]func() error
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

			execs = append(execs, [2]func() error{
				func() error {
					err := node.Run(ctx)
					if err != nil {
						return fmt.Errorf("%s: %w", node.Ident(), err)
					}
					return nil
				},
				func() error {
					err := e.runNodeFlow(ctx, node, startCh, endCh, inbound, outbound)
					if err != nil {
						return fmt.Errorf("%s: %w", node.Ident(), err)
					}
					return nil
				},
			})
		}
		if err != nil {
			e.runErr = err
			return
		}

		for _, exec := range execs {
			group.Go(exec[0])
			group.Go(exec[1])
		}

		group.Go(func() error {
			return e.runControlFlow(ctx)
		})

		e.runErr = group.Wait()
	})

	return e.runErr
}

func (e *Executor[T]) runControlFlow(ctx context.Context) (err error) {
	for {
		var (
			isInit bool
			starts map[string]any
		)
		if e.onInit != nil {
			e.initOnce.Do(func() {
				isInit = true
				starts, err = e.onInit(ctx, e.starts)
			})
			if err != nil {
				return
			}
		}

		if starts == nil {
			starts = map[string]any{}
		}

		if !isInit {
			for _, onStart := range e.onStarts {
				values, err := onStart(ctx, e.starts)
				if err != nil {
					return err
				}

				for ident, value := range values {
					if _, ok := starts[ident]; ok {
						continue
					}

					starts[ident] = value
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

		var nodes GraphNodes[T]
		for ident, endCh := range e.endsCh {
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			case <-endCh:
				if node, ok := e.graph.Node(ident); ok {
					nodes = append(nodes, node)
				}
			}
		}

		for _, onEnd := range e.onEnds {
			err = onEnd(ctx, nodes)
			if err != nil {
				return err
			}
		}
	}
}

func (e *Executor[T]) runNodeFlow(ctx context.Context, node T, startCh, endCh chan any, inbound, outbound []*Edge) (err error) {
	for {
		var send any
		if startCh != nil {
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			case send = <-startCh:
			}
		} else {
			for _, edge := range inbound {
				select {
				case <-ctx.Done():
					return context.Cause(ctx)
				case <-edge.Out():
					// log.FromCtx(ctx).Info(edge.String(), slog.Any("value", out))
				}
			}
		}

		for _, onSend := range e.onSends {
			err := onSend(ctx, node, send)
			if err != nil {
				return err
			}
		}

		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case node.Send() <- send:
		}

		var recv any
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case recv = <-node.Recv():
			for _, onRecv := range e.onRecvs {
				recv, err = onRecv(ctx, node, recv)
				if err != nil {
					return err
				}
			}
		}

		if endCh != nil {
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			case endCh <- recv:
			}
		} else {
			for _, edge := range outbound {
				select {
				case <-ctx.Done():
					return context.Cause(ctx)
				case edge.In() <- recv:
				}
			}
		}
	}
}
