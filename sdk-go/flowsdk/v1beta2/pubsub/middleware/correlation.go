package middleware

import (
	"github.com/google/uuid"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
)

const CorrelationIDKey = "correlation_id"

// CorrelationID propagates a correlation ID via Message.Metadata.
// If the incoming message has no correlation ID, one is generated.
// All output messages inherit the correlation ID from the input.
func CorrelationID(h pubsub.HandlerFunc) pubsub.HandlerFunc {
	return func(msg *pubsub.Message) ([]*pubsub.Message, error) {
		cid := msg.Metadata[CorrelationIDKey]
		if cid == "" {
			cid = uuid.NewString()
			msg.Metadata[CorrelationIDKey] = cid
		}
		msgs, err := h(msg)
		if err != nil {
			return nil, err
		}
		for _, m := range msgs {
			if m.Metadata == nil {
				m.Metadata = make(map[string]string)
			}
			m.Metadata[CorrelationIDKey] = cid
		}
		return msgs, nil
	}
}
