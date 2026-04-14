package pubsubtest

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// Factory creates a fresh PubSub instance for each test.
type Factory func(t *testing.T) pubsub.PubSub

// Options configures which conformance tests to run.
type Options struct {
	// Timeout is the maximum time to wait for a message. Defaults to 5s.
	Timeout time.Duration

	// SupportsPublishBeforeSubscribe indicates the backend persists messages
	// so a subscriber created after publish still receives them.
	SupportsPublishBeforeSubscribe bool
}

func getTimeout(opts Options) time.Duration {
	if opts.Timeout > 0 {
		return opts.Timeout
	}
	return 5 * time.Second
}

func testNode(id string) *flowv1beta2.RunSnapshot_VarNode {
	n := &flowv1beta2.RunSnapshot_VarNode{}
	n.SetId(id)
	return n
}

func uniqueTopic(t *testing.T) string {
	t.Helper()
	// Replace characters invalid in Kafka topic names.
	name := strings.NewReplacer("/", "-", " ", "_").Replace(t.Name())
	return fmt.Sprintf("conformance-%s-%d", name, time.Now().UnixNano())
}

// Run executes the full pubsub conformance test suite against the given factory.
func Run(t *testing.T, factory Factory, opts Options) {
	t.Helper()
	to := getTimeout(opts)

	t.Run("Basic", func(t *testing.T) {
		ps := factory(t)
		defer func() { _ = ps.Close() }()
		topic := uniqueTopic(t)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := ps.Subscribe(ctx, topic)
		if err != nil {
			t.Fatal(err)
		}

		msg := pubsub.NewMessage(testNode("n1"))
		if err := ps.Publish(topic, msg); err != nil {
			t.Fatal(err)
		}

		select {
		case received := <-ch:
			node, ok := received.Payload.(*flowv1beta2.RunSnapshot_VarNode)
			if !ok {
				t.Fatalf("payload type %T, want *RunSnapshot_VarNode", received.Payload)
			}
			if node.GetId() != "n1" {
				t.Errorf("got node ID %q, want %q", node.GetId(), "n1")
			}
			received.Ack()
		case <-time.After(to):
			t.Fatal("timed out waiting for message")
		}
	})

	t.Run("UUID", func(t *testing.T) {
		ps := factory(t)
		defer func() { _ = ps.Close() }()
		topic := uniqueTopic(t)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := ps.Subscribe(ctx, topic)
		if err != nil {
			t.Fatal(err)
		}

		msg := pubsub.NewMessage(testNode("n1"))
		if msg.UUID == (uuid.UUID{}) {
			t.Fatal("NewMessage should assign a UUID")
		}
		if err := ps.Publish(topic, msg); err != nil {
			t.Fatal(err)
		}

		select {
		case received := <-ch:
			if received.UUID != msg.UUID {
				t.Errorf("UUID mismatch: got %v, want %v", received.UUID, msg.UUID)
			}
			received.Ack()
		case <-time.After(to):
			t.Fatal("timed out")
		}
	})

	t.Run("AckBeforeNext", func(t *testing.T) {
		ps := factory(t)
		defer func() { _ = ps.Close() }()
		topic := uniqueTopic(t)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := ps.Subscribe(ctx, topic)
		if err != nil {
			t.Fatal(err)
		}

		msg1 := pubsub.NewMessage(testNode("n1"))
		msg2 := pubsub.NewMessage(testNode("n2"))
		if err := ps.Publish(topic, msg1, msg2); err != nil {
			t.Fatal(err)
		}

		var first *pubsub.Message
		select {
		case first = <-ch:
			node := first.Payload.(*flowv1beta2.RunSnapshot_VarNode)
			if node.GetId() != "n1" {
				t.Fatalf("expected n1, got %s", node.GetId())
			}
		case <-time.After(to):
			t.Fatal("timed out waiting for first message")
		}

		select {
		case <-ch:
			t.Fatal("received second message before acking first")
		case <-time.After(200 * time.Millisecond):
		}

		first.Ack()

		select {
		case second := <-ch:
			node := second.Payload.(*flowv1beta2.RunSnapshot_VarNode)
			if node.GetId() != "n2" {
				t.Fatalf("expected n2, got %s", node.GetId())
			}
			second.Ack()
		case <-time.After(to):
			t.Fatal("timed out waiting for second message after ack")
		}
	})

	t.Run("Metadata", func(t *testing.T) {
		ps := factory(t)
		defer func() { _ = ps.Close() }()
		topic := uniqueTopic(t)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := ps.Subscribe(ctx, topic)
		if err != nil {
			t.Fatal(err)
		}

		msg := pubsub.NewMessage(testNode("n1"))
		msg.Metadata["trace_id"] = "abc-123"
		msg.Metadata["flow_id"] = "flow-42"
		if err := ps.Publish(topic, msg); err != nil {
			t.Fatal(err)
		}

		select {
		case received := <-ch:
			if received.Metadata["trace_id"] != "abc-123" {
				t.Errorf("trace_id: got %q, want %q", received.Metadata["trace_id"], "abc-123")
			}
			if received.Metadata["flow_id"] != "flow-42" {
				t.Errorf("flow_id: got %q, want %q", received.Metadata["flow_id"], "flow-42")
			}
			received.Ack()
		case <-time.After(to):
			t.Fatal("timed out")
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		ps := factory(t)
		defer func() { _ = ps.Close() }()
		topic := uniqueTopic(t)

		ctx, cancel := context.WithCancel(context.Background())
		ch, err := ps.Subscribe(ctx, topic)
		if err != nil {
			t.Fatal(err)
		}

		cancel()

		select {
		case _, ok := <-ch:
			if ok {
				t.Error("expected channel to be closed after context cancel")
			}
		case <-time.After(to):
			t.Fatal("timed out waiting for channel close")
		}
	})

	t.Run("MultipleMessages", func(t *testing.T) {
		ps := factory(t)
		defer func() { _ = ps.Close() }()
		topic := uniqueTopic(t)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := ps.Subscribe(ctx, topic)
		if err != nil {
			t.Fatal(err)
		}

		const count = 5
		for i := range count {
			msg := pubsub.NewMessage(testNode(fmt.Sprintf("n%d", i)))
			if err := ps.Publish(topic, msg); err != nil {
				t.Fatal(err)
			}
		}

		seen := make(map[string]bool, count)
		for i := range count {
			select {
			case received := <-ch:
				node := received.Payload.(*flowv1beta2.RunSnapshot_VarNode)
				seen[node.GetId()] = true
				received.Ack()
			case <-time.After(to):
				t.Fatalf("timed out waiting for message %d", i)
			}
		}

		for i := range count {
			id := fmt.Sprintf("n%d", i)
			if !seen[id] {
				t.Errorf("missing message %q", id)
			}
		}
	})

	t.Run("PublishToNoSubscribers", func(t *testing.T) {
		ps := factory(t)
		defer func() { _ = ps.Close() }()
		topic := uniqueTopic(t)

		msg := pubsub.NewMessage(testNode("n1"))
		if err := ps.Publish(topic, msg); err != nil {
			t.Fatal(err)
		}
	})

	if opts.SupportsPublishBeforeSubscribe {
		t.Run("PublishBeforeSubscribe", func(t *testing.T) {
			ps := factory(t)
			defer func() { _ = ps.Close() }()
			topic := uniqueTopic(t)

			msg := pubsub.NewMessage(testNode("n1"))
			if err := ps.Publish(topic, msg); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ch, err := ps.Subscribe(ctx, topic)
			if err != nil {
				t.Fatal(err)
			}

			select {
			case received := <-ch:
				node := received.Payload.(*flowv1beta2.RunSnapshot_VarNode)
				if node.GetId() != "n1" {
					t.Errorf("got %q, want %q", node.GetId(), "n1")
				}
				received.Ack()
			case <-time.After(to):
				t.Fatal("timed out")
			}
		})
	}
}
