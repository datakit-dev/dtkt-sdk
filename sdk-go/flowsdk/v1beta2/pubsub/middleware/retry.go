package middleware

import (
	"fmt"
	"math"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
)

// Retry re-runs the handler on error with exponential backoff.
type Retry struct {
	// MaxRetries is the maximum number of retry attempts. 0 means no retries.
	MaxRetries int
	// InitialInterval is the initial backoff duration.
	InitialInterval time.Duration
	// BackoffFactor multiplies the interval on each retry.
	BackoffFactor float64
	// MaxInterval caps the backoff duration.
	MaxInterval time.Duration
}

func (r Retry) Middleware(h pubsub.HandlerFunc) pubsub.HandlerFunc {
	if r.MaxRetries <= 0 {
		return h
	}
	if r.InitialInterval == 0 {
		r.InitialInterval = time.Second
	}
	if r.BackoffFactor == 0 {
		r.BackoffFactor = 2.0
	}
	if r.MaxInterval == 0 {
		r.MaxInterval = 30 * time.Second
	}
	return func(msg *pubsub.Message) ([]*pubsub.Message, error) {
		var lastErr error
		interval := r.InitialInterval
		for attempt := 0; attempt <= r.MaxRetries; attempt++ {
			msgs, err := h(msg)
			if err == nil {
				return msgs, nil
			}
			lastErr = err
			if attempt < r.MaxRetries {
				time.Sleep(interval)
				interval = time.Duration(math.Min(
					float64(interval)*r.BackoffFactor,
					float64(r.MaxInterval),
				))
			}
		}
		return nil, fmt.Errorf("handler failed after %d retries: %w", r.MaxRetries, lastErr)
	}
}
