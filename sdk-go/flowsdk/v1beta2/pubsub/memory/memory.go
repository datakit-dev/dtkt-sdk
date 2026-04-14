package memory

import (
	"context"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
)

// PubSub is an in-memory publish/subscribe implementation.
// Each subscriber receives a copy of every message (fan-out).
// Subscribers must ack the current message before receiving the next one.
// Nacked messages are re-enqueued at the tail of the subscriber's buffer.
type PubSub struct {
	mu         sync.RWMutex
	topics     map[string][]*subscriber
	buffer     map[string][]*pubsub.Message // persistent mode: buffered messages for topics with no subscribers
	persistent bool
	closed     bool
}

// Option configures a PubSub instance.
type Option func(*PubSub)

// WithPersistent enables persistent mode. Messages published to topics with
// no subscribers are buffered and delivered to the first subscriber that
// arrives. This is useful when publishers and subscribers start independently
// (e.g., pre-publishing input values before the executor subscribes).
func WithPersistent() Option {
	return func(ps *PubSub) {
		ps.persistent = true
		ps.buffer = make(map[string][]*pubsub.Message)
	}
}

type subscriber struct {
	// msgs is the queue of pending messages for this subscriber.
	msgs []*pubsub.Message
	mu   sync.Mutex
	// notify signals the delivery goroutine that new messages are available.
	notify chan struct{}
	// out is the channel the consumer reads from.
	out chan *pubsub.Message
	ctx context.Context
}

func newSubscriber(ctx context.Context) *subscriber {
	return &subscriber{
		notify: make(chan struct{}, 1),
		out:    make(chan *pubsub.Message),
		ctx:    ctx,
	}
}

// enqueue adds a message to the subscriber's queue and signals the delivery goroutine.
func (s *subscriber) enqueue(msg *pubsub.Message) {
	s.mu.Lock()
	s.msgs = append(s.msgs, msg)
	s.mu.Unlock()
	// Non-blocking signal -- if already signaled, the goroutine will drain the queue.
	select {
	case s.notify <- struct{}{}:
	default:
	}
}

// run is the delivery goroutine. It delivers one message at a time and waits
// for ack/nack before delivering the next.
func (s *subscriber) run() {
	defer close(s.out)
	for {
		// Wait for messages in the queue.
		select {
		case <-s.ctx.Done():
			return
		case <-s.notify:
		}

		// Drain the queue one message at a time with ack-before-next gating.
		for {
			s.mu.Lock()
			if len(s.msgs) == 0 {
				s.mu.Unlock()
				break
			}
			msg := s.msgs[0]
			s.msgs = s.msgs[1:]
			s.mu.Unlock()

			// Deliver the message to the consumer.
			select {
			case <-s.ctx.Done():
				return
			case s.out <- msg:
			}

			// Wait for ack or nack before delivering the next message.
			select {
			case <-s.ctx.Done():
				return
			case <-msg.Acked():
				// Message processed successfully, continue to next.
			case <-msg.Nacked():
				// Create a fresh copy for redelivery (reset ack/nack state).
				s.enqueue(pubsub.CopyMessage(msg))
			}
		}
	}
}

// New creates a new in-memory PubSub.
func New(opts ...Option) *PubSub {
	ps := &PubSub{
		topics: make(map[string][]*subscriber),
	}
	for _, opt := range opts {
		opt(ps)
	}
	return ps
}

func (ps *PubSub) Publish(topic string, messages ...*pubsub.Message) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.closed {
		return nil
	}
	subs := ps.topics[topic]
	for _, msg := range messages {
		if len(subs) == 0 && ps.persistent {
			ps.buffer[topic] = append(ps.buffer[topic], msg)
			continue
		}
		for i, s := range subs {
			if s.ctx.Err() != nil {
				continue
			}
			if i == 0 {
				// First subscriber gets the original message.
				s.enqueue(msg)
			} else {
				// Fan-out: each additional subscriber gets a copy with
				// its own ack/nack channels but the same UUID and payload.
				s.enqueue(pubsub.CopyMessage(msg))
			}
		}
	}
	return nil
}

func (ps *PubSub) Subscribe(ctx context.Context, topic string) (<-chan *pubsub.Message, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.closed {
		ch := make(chan *pubsub.Message)
		close(ch)
		return ch, nil
	}
	s := newSubscriber(ctx)
	ps.topics[topic] = append(ps.topics[topic], s)

	// Deliver buffered messages from persistent mode to the first subscriber.
	if ps.persistent {
		if buffered, ok := ps.buffer[topic]; ok {
			for _, msg := range buffered {
				s.enqueue(msg)
			}
			delete(ps.buffer, topic)
		}
	}

	// Start the delivery goroutine.
	go s.run()

	// Cleanup goroutine: remove subscriber when context is cancelled.
	go func() {
		<-ctx.Done()
		ps.mu.Lock()
		defer ps.mu.Unlock()
		subs := ps.topics[topic]
		for i, sub := range subs {
			if sub == s {
				ps.topics[topic] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}()

	return s.out, nil
}

func (ps *PubSub) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.closed = true
	ps.topics = make(map[string][]*subscriber)
	return nil
}
