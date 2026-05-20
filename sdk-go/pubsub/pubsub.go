package pubsub

import (
	"context"
	"io"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

// Message is a single unit of data flowing between nodes.
type Message struct {
	// UUID uniquely identifies this message (for dedup, outbox tracking, CloudEvents id).
	UUID uuid.UUID

	// Metadata carries correlation IDs, tracing context, and system fields.
	Metadata map[string]string

	// Payload is the protobuf message carried by this message.
	// Between nodes this is *RunSnapshot_Node; between transform steps it is *expr.Value.
	Payload proto.Message

	ctx    context.Context
	ackCh  chan struct{}
	nackCh chan struct{}
	acked  bool
	nacked bool
	mu     sync.Mutex
}

// NewMessage creates a new Message with the given protobuf payload.
func NewMessage(payload proto.Message) *Message {
	return &Message{
		UUID:     uuid.Must(uuid.NewV7()),
		Metadata: make(map[string]string),
		Payload:  payload,
		ctx:      context.Background(),
		ackCh:    make(chan struct{}, 1),
		nackCh:   make(chan struct{}, 1),
	}
}

// CopyMessage creates a copy of the message with fresh ack/nack channels
// but the same UUID, Metadata, Payload, and Context. Used for fan-out and
// nack redelivery.
func CopyMessage(m *Message) *Message {
	metadata := make(map[string]string, len(m.Metadata))
	for k, v := range m.Metadata {
		metadata[k] = v
	}
	return &Message{
		UUID:     m.UUID,
		Metadata: metadata,
		Payload:  m.Payload,
		ctx:      m.ctx,
		ackCh:    make(chan struct{}, 1),
		nackCh:   make(chan struct{}, 1),
	}
}

// Context returns the message's context.
func (m *Message) Context() context.Context {
	return m.ctx
}

// SetContext sets the message's context.
func (m *Message) SetContext(ctx context.Context) {
	m.ctx = ctx
}

// Ack acknowledges successful processing of the message. Idempotent.
func (m *Message) Ack() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.acked || m.nacked {
		return
	}
	m.acked = true
	m.ackCh <- struct{}{}
}

// Nack signals that the message could not be processed and should be redelivered. Idempotent.
func (m *Message) Nack() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.acked || m.nacked {
		return
	}
	m.nacked = true
	m.nackCh <- struct{}{}
}

// Acked returns a channel that is signaled when Ack is called.
func (m *Message) Acked() <-chan struct{} {
	return m.ackCh
}

// Nacked returns a channel that is signaled when Nack is called.
func (m *Message) Nacked() <-chan struct{} {
	return m.nackCh
}

// Publisher sends messages to a named topic.
type Publisher interface {
	// Publish sends one or more messages to the given topic.
	// Implementations must deliver messages in the order they are provided.
	Publish(topic string, messages ...*Message) error

	io.Closer
}

// Subscriber receives messages from a named topic.
type Subscriber interface {
	// Subscribe returns a channel that delivers all messages published to the topic.
	// Each subscriber receives every message (fan-out).
	// The channel is closed when the subscriber is closed or the context is cancelled.
	Subscribe(ctx context.Context, topic string) (<-chan *Message, error)

	io.Closer
}

// PubSub combines Publisher and Subscriber.
type PubSub interface {
	Publisher
	Subscriber
}
