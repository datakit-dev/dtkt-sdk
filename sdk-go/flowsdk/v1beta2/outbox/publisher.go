package outbox

import "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"

// PublisherAdapter wraps an outbox Storage as a pubsub.Publisher.
// Each Publish call stores messages in the outbox via Storage.Store.
// The topic parameter from pubsub.Publisher.Publish is intentionally ignored;
// the FlowEvent payload contains all routing information.
type PublisherAdapter struct {
	storage Storage
}

// NewPublisher creates a Publisher that writes messages to the given outbox Storage.
func NewPublisher(s Storage) *PublisherAdapter {
	return &PublisherAdapter{storage: s}
}

func (p *PublisherAdapter) Publish(_ string, messages ...*pubsub.Message) error {
	for _, msg := range messages {
		if err := p.storage.Store(msg.Context(), msg); err != nil {
			return err
		}
	}
	return nil
}

func (p *PublisherAdapter) Close() error { return nil }

var _ pubsub.Publisher = (*PublisherAdapter)(nil)
