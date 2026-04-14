package outbox

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
)

// SubscriberAdapter reads events from an EventReader and delivers them as a
// pubsub.Subscriber. It polls for new events using cursor-based pagination
// and advances the cursor on ack.
type SubscriberAdapter struct {
	reader       EventReader
	pollInterval time.Duration
	mu           sync.Mutex
	cancel       context.CancelFunc
	draining     bool
}

// NewSubscriber creates a Subscriber that polls the given EventReader for
// new events. pollInterval controls how often the reader is checked when no
// events are available (default 100ms).
func NewSubscriber(r EventReader, pollInterval time.Duration) *SubscriberAdapter {
	if pollInterval <= 0 {
		pollInterval = 100 * time.Millisecond
	}
	return &SubscriberAdapter{
		reader:       r,
		pollInterval: pollInterval,
	}
}

func (sa *SubscriberAdapter) Subscribe(ctx context.Context, _ string) (<-chan *pubsub.Message, error) {
	ctx, cancel := context.WithCancel(ctx)
	sa.mu.Lock()
	sa.cancel = cancel
	sa.mu.Unlock()

	out := make(chan *pubsub.Message)
	go sa.poll(ctx, out)
	return out, nil
}

func (sa *SubscriberAdapter) poll(ctx context.Context, out chan<- *pubsub.Message) {
	defer close(out)
	var afterUID uuid.UUID

	for {
		if ctx.Err() != nil {
			return
		}

		msgs, err := sa.reader.ReadEvents(ctx, afterUID, 1)
		if err != nil || len(msgs) == 0 {
			sa.mu.Lock()
			draining := sa.draining
			sa.mu.Unlock()
			if draining {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(sa.pollInterval):
				continue
			}
		}

		sm := msgs[0]

		// Deliver message with fresh ack/nack channels.
		msg := pubsub.CopyMessage(sm)
		select {
		case <-ctx.Done():
			return
		case out <- msg:
		}

		// Wait for ack or nack.
		select {
		case <-ctx.Done():
			return
		case <-msg.Acked():
			afterUID = sm.UUID
		case <-msg.Nacked():
			select {
			case <-ctx.Done():
				return
			case <-time.After(sa.pollInterval):
			}
		}
	}
}

// CloseWhenDrained signals the subscriber to stop after all unforwarded
// messages have been processed. The output channel is closed once the outbox
// is fully drained, which causes the downstream Forwarder to return.
func (sa *SubscriberAdapter) CloseWhenDrained() {
	sa.mu.Lock()
	sa.draining = true
	sa.mu.Unlock()
}

func (sa *SubscriberAdapter) Close() error {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	if sa.cancel != nil {
		sa.cancel()
	}
	return nil
}

var _ pubsub.Subscriber = (*SubscriberAdapter)(nil)
