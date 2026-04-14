package executor

import "context"

// NodeHandler is a runnable unit that processes messages.
type NodeHandler interface {
	Run(ctx context.Context) error
}
