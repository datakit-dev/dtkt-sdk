package forwarder

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
)

const forwarderTopic = "_forwarder"

// envelope wraps a message with its destination topic for relay through the forwarder.
type envelope struct {
	DestTopic string `json:"dest_topic"`
}

const envelopeKey = "_forwarder_dest"

// Publisher is a Publisher decorator that wraps messages in envelopes with the
// destination topic and publishes them to a single forwarder topic. The actual
// delivery to the real topic is done by the Forwarder daemon.
type Publisher struct {
	wrapped pubsub.Publisher
}

// NewPublisher creates a ForwarderPublisher that publishes enveloped messages
// to the forwarder topic on the given publisher.
func NewPublisher(pub pubsub.Publisher) *Publisher {
	return &Publisher{wrapped: pub}
}

func (fp *Publisher) Publish(topic string, messages ...*pubsub.Message) error {
	for _, msg := range messages {
		env := envelope{DestTopic: topic}
		data, err := json.Marshal(env)
		if err != nil {
			return fmt.Errorf("marshaling forwarder envelope: %w", err)
		}
		msg.Metadata[envelopeKey] = string(data)
		if err := fp.wrapped.Publish(forwarderTopic, msg); err != nil {
			return fmt.Errorf("publishing to forwarder topic: %w", err)
		}
	}
	return nil
}

func (fp *Publisher) Close() error {
	return fp.wrapped.Close()
}

// Forwarder is a background daemon that subscribes to the forwarder topic,
// unwraps envelopes, and publishes messages to their real destination topics.
// It composes any Subscriber (input) + any Publisher (output).
type Forwarder struct {
	sub pubsub.Subscriber
	pub pubsub.Publisher
}

// NewForwarder creates a Forwarder that reads enveloped messages from sub and
// publishes them to their destination topic on pub.
func NewForwarder(sub pubsub.Subscriber, pub pubsub.Publisher) *Forwarder {
	return &Forwarder{sub: sub, pub: pub}
}

// Run starts the forwarder loop. It blocks until the context is cancelled.
func (f *Forwarder) Run(ctx context.Context) error {
	ch, err := f.sub.Subscribe(ctx, forwarderTopic)
	if err != nil {
		return fmt.Errorf("subscribing to forwarder topic: %w", err)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			destData, exists := msg.Metadata[envelopeKey]
			if !exists {
				slog.Warn("forwarder: message missing envelope metadata, dropping")
				msg.Ack()
				continue
			}
			var env envelope
			if err := json.Unmarshal([]byte(destData), &env); err != nil {
				slog.Error("forwarder: invalid envelope", slog.Any("err", err))
				msg.Ack()
				continue
			}
			// Copy the message so the relay has independent ack/nack channels.
			relay := pubsub.CopyMessage(msg)
			delete(relay.Metadata, envelopeKey)
			if err := f.pub.Publish(env.DestTopic, relay); err != nil {
				slog.Error("forwarder: publish failed", slog.String("topic", env.DestTopic), slog.Any("err", err))
				msg.Nack()
				continue
			}
			msg.Ack()
		}
	}
}
