package outbox

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
)

// TopicMetadataKey is the metadata key used to persist the destination topic
// alongside messages in the outbox. The transactional outbox pattern requires
// that both the message and its routing information are stored atomically;
// the Relay reads this key to publish messages to the correct topic.
const TopicMetadataKey = "_outbox_topic"

// Relay is the message relay component of the transactional outbox pattern.
// It reads committed events from a Subscriber (typically a SubscriberAdapter
// polling the outbox) and publishes them to their destination topic on a
// Publisher (typically the in-process PubSub used for inter-node messaging).
//
// Messages must carry their destination topic in Metadata[TopicMetadataKey].
// Messages without this key are dropped with a warning.
//
// The relay provides at-least-once delivery: on publish failure the source
// message is nacked so the SubscriberAdapter retries. On success the source
// is acked and the cursor advances.
type Relay struct {
	sub pubsub.Subscriber
	pub pubsub.Publisher
}

// NewRelay creates a Relay that reads from sub and publishes to pub.
func NewRelay(sub pubsub.Subscriber, pub pubsub.Publisher) *Relay {
	return &Relay{sub: sub, pub: pub}
}

// Run starts the relay loop. It blocks until ctx is cancelled or the
// subscriber channel is closed (e.g. via SubscriberAdapter.CloseWhenDrained).
func (r *Relay) Run(ctx context.Context) error {
	ch, err := r.sub.Subscribe(ctx, "")
	if err != nil {
		return fmt.Errorf("outbox relay subscribe: %w", err)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			topic, hasTopic := msg.Metadata[TopicMetadataKey]
			if !hasTopic {
				slog.Warn("outbox relay: message missing topic metadata, dropping",
					slog.String("uuid", msg.UUID.String()))
				msg.Ack()
				continue
			}
			// Copy the message so the relay has independent ack/nack
			// channels and the topic key is not leaked to consumers.
			relay := pubsub.CopyMessage(msg)
			delete(relay.Metadata, TopicMetadataKey)
			if err := r.pub.Publish(topic, relay); err != nil {
				slog.Error("outbox relay: publish failed",
					slog.String("topic", topic),
					slog.Any("err", err))
				msg.Nack()
				continue
			}
			msg.Ack()
		}
	}
}
