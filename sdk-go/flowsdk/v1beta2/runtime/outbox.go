package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// txPublisher wraps each Publish call in an outbox transaction.
// For each Publish: opens tx -> writes to outbox via ForwarderPublisher -> commits.
// This is the per-message transactional write pattern from the outbox design.
// It also maintains a running RunSnapshot that is written via the Tx's
// StateWriter after each event, keeping the materialized state in sync.
type txPublisher struct {
	mu         sync.Mutex
	txBeginner outbox.TxBeginner
	snap       *flowv1beta2.RunSnapshot
}

func (tp *txPublisher) Publish(topic string, messages ...*pubsub.Message) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	// Persist the destination topic in message metadata so the outbox
	// relay can publish to the correct topic after reading committed events.
	// This is required by the transactional outbox pattern: the outbox must
	// store both the message and its routing information atomically.
	for _, msg := range messages {
		msg.Metadata[outbox.TopicMetadataKey] = topic
	}

	ctx := context.Background()
	if len(messages) > 0 {
		ctx = messages[0].Context()
	}
	tx, err := tp.txBeginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin outbox tx: %w", err)
	}

	outboxPub := outbox.NewPublisher(tx.Storage())
	if err := outboxPub.Publish(topic, messages...); err != nil {
		_ = tx.Rollback()
		return err
	}
	// Atomically update materialized state within the same transaction.
	sw := tx.StateWriter()
	for _, msg := range messages {
		flowEvent, ok := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
		if !ok {
			_ = tx.Rollback()
			return fmt.Errorf("payload is %T, want *flowv1beta2.RunSnapshot_FlowEvent", msg.Payload)
		}
		outbox.ApplyFlowEvent(tp.snap, flowEvent)
		if err := sw.WriteState(ctx, tp.snap, msg.UUID); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("write state: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit outbox tx: %w", err)
	}
	return nil
}

func (tp *txPublisher) Close() error { return nil }

// outboxPubSub adapts a tx-based Publisher + a Subscriber into a PubSub.
// Handlers publish through the outbox (tx path); subscriptions are pre-wired
// via gochannel in Execute() and not used through this PubSub directly.
type outboxPubSub struct {
	pub pubsub.Publisher
	sub pubsub.Subscriber
}

func (o *outboxPubSub) Publish(topic string, messages ...*pubsub.Message) error {
	return o.pub.Publish(topic, messages...)
}

func (o *outboxPubSub) Subscribe(ctx context.Context, topic string) (<-chan *pubsub.Message, error) {
	return o.sub.Subscribe(ctx, topic)
}

func (o *outboxPubSub) Close() error {
	return errors.Join(o.pub.Close(), o.sub.Close())
}

var _ pubsub.PubSub = (*outboxPubSub)(nil)
