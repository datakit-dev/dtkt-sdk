package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
)

// Timeout cancels the handler if it exceeds the deadline.
type Timeout struct {
	// Duration is the maximum time allowed for the handler to complete.
	Duration time.Duration
}

func (t Timeout) Middleware(h pubsub.HandlerFunc) pubsub.HandlerFunc {
	if t.Duration <= 0 {
		return h
	}
	return func(msg *pubsub.Message) ([]*pubsub.Message, error) {
		ctx, cancel := context.WithTimeout(msg.Context(), t.Duration)
		defer cancel()
		msg.SetContext(ctx)

		type result struct {
			msgs []*pubsub.Message
			err  error
		}
		ch := make(chan result, 1)
		go func() {
			msgs, err := h(msg)
			ch <- result{msgs, err}
		}()

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("handler timed out after %s: %w", t.Duration, ctx.Err())
		case r := <-ch:
			return r.msgs, r.err
		}
	}
}
