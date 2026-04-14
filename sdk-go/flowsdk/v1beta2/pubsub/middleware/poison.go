package middleware

import (
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
)

// PoisonQueue routes messages to a dead-letter topic after MaxRetries failures.
type PoisonQueue struct {
	// MaxRetries is the number of handler attempts before routing to the poison topic.
	MaxRetries int
	// Publisher publishes to the dead-letter topic.
	Publisher pubsub.Publisher
	// Topic is the dead-letter topic name.
	Topic string
}

func (pq PoisonQueue) Middleware(h pubsub.HandlerFunc) pubsub.HandlerFunc {
	if pq.MaxRetries <= 0 {
		pq.MaxRetries = 1
	}
	return func(msg *pubsub.Message) ([]*pubsub.Message, error) {
		var lastErr error
		for attempt := 0; attempt <= pq.MaxRetries; attempt++ {
			msgs, err := h(msg)
			if err == nil {
				return msgs, nil
			}
			lastErr = err
		}
		// Route to poison queue.
		msg.Metadata["poison_reason"] = lastErr.Error()
		if err := pq.Publisher.Publish(pq.Topic, msg); err != nil {
			return nil, fmt.Errorf("publishing to poison queue: %w", err)
		}
		return nil, nil
	}
}
