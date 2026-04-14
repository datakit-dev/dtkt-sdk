package executor

import (
	"context"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// PubSub combines Publisher and Subscriber for inter-node messaging.
// Implementations: GoChannel (in-memory), Kafka, NATS, Redis (Valkey), SQL, etc.
type PubSub = pubsub.PubSub

// Topics computes PubSub topic names scoped to a specific flow run.
// The prefix ensures topic isolation when multiple flows share a PubSub
// backend such as Valkey.
type Topics struct {
	prefix string
}

// NewTopics returns a Topics scoped to the given flow-run name.
// Example prefix: "users/shadi/flowruns/ticker-tdlulm"
func NewTopics(prefix string) Topics {
	return Topics{prefix: prefix}
}

// For returns the node-to-node topic for a given node ID.
func (t Topics) For(nodeID string) string {
	return t.prefix + ":" + nodeID
}

// InputFor returns the external input topic for a given node ID.
// Callers publish input values to this topic; the executor subscribes to it.
func (t Topics) InputFor(nodeID string) string {
	return t.prefix + ":" + nodeID + ":input"
}

// Flow returns the flow-level state change topic (the prefix itself).
func (t Topics) Flow() string {
	return t.prefix
}

// Executor runs a graph to completion.
type Executor interface {
	Execute(ctx context.Context, graph *flowv1beta2.Graph) error
}
