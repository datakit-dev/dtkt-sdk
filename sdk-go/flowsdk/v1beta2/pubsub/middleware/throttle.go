package middleware

import (
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
)

// Throttle rate-limits handler invocations to at most one per interval.
type Throttle struct {
	// Interval is the minimum duration between handler invocations.
	Interval time.Duration
}

func (t Throttle) Middleware(h pubsub.HandlerFunc) pubsub.HandlerFunc {
	if t.Interval <= 0 {
		return h
	}
	var lastCall time.Time
	return func(msg *pubsub.Message) ([]*pubsub.Message, error) {
		if elapsed := time.Since(lastCall); elapsed < t.Interval {
			time.Sleep(t.Interval - elapsed)
		}
		lastCall = time.Now()
		return h(msg)
	}
}
