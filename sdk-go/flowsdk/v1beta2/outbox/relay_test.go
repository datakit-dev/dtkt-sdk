package outbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox"
	outboxmem "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox/memory"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub/memory"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

func TestRelay_ForwardsToCorrectTopic(t *testing.T) {
	store := outboxmem.New()
	ps := memory.New()
	defer func() { _ = ps.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Subscribe to the destination topic on the in-process pubsub.
	destCh, err := ps.Subscribe(ctx, "node.myvar")
	if err != nil {
		t.Fatal(err)
	}

	// Store a message in the outbox with the topic metadata set.
	msg := pubsub.NewMessage(testNode("myvar"))
	msg.Metadata[outbox.TopicMetadataKey] = "node.myvar"
	msg.Metadata["user_key"] = "user_val"
	if err := store.Store(ctx, msg); err != nil {
		t.Fatal(err)
	}

	// Start the relay.
	sub := outbox.NewSubscriber(store, 5*time.Millisecond)
	relay := outbox.NewRelay(sub, ps)
	relayErr := make(chan error, 1)
	go func() {
		relayErr <- relay.Run(ctx)
	}()

	// Verify the message arrives at the destination topic.
	select {
	case received := <-destCh:
		node := received.Payload.(*flowv1beta2.RunSnapshot_VarNode)
		if node.GetId() != "myvar" {
			t.Errorf("node ID = %q, want %q", node.GetId(), "myvar")
		}
		if received.Metadata["user_key"] != "user_val" {
			t.Error("user metadata not preserved")
		}
		if _, exists := received.Metadata[outbox.TopicMetadataKey]; exists {
			t.Error("topic metadata key should be removed from relayed message")
		}
		received.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for relayed message")
	}

	cancel()
	<-relayErr
}

func TestRelay_DropsMessagesWithoutTopic(t *testing.T) {
	store := outboxmem.New()
	ps := memory.New()
	defer func() { _ = ps.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Store a message WITHOUT topic metadata.
	msg := pubsub.NewMessage(testNode("n1"))
	if err := store.Store(ctx, msg); err != nil {
		t.Fatal(err)
	}

	// Store a second message WITH topic metadata.
	msg2 := pubsub.NewMessage(testNode("n2"))
	msg2.Metadata[outbox.TopicMetadataKey] = "dest"
	if err := store.Store(ctx, msg2); err != nil {
		t.Fatal(err)
	}

	// Subscribe to the destination.
	destCh, err := ps.Subscribe(ctx, "dest")
	if err != nil {
		t.Fatal(err)
	}

	// Start relay.
	sub := outbox.NewSubscriber(store, 5*time.Millisecond)
	relay := outbox.NewRelay(sub, ps)
	relayErr := make(chan error, 1)
	go func() {
		relayErr <- relay.Run(ctx)
	}()

	// The first message (no topic) should be dropped; the second should arrive.
	select {
	case received := <-destCh:
		node := received.Payload.(*flowv1beta2.RunSnapshot_VarNode)
		if node.GetId() != "n2" {
			t.Errorf("expected n2 (first message should be dropped), got %q", node.GetId())
		}
		received.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second message")
	}

	cancel()
	<-relayErr
}

func TestRelay_MultipleTopics(t *testing.T) {
	store := outboxmem.New()
	ps := memory.New()
	defer func() { _ = ps.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Subscribe to two different destination topics.
	ch1, err := ps.Subscribe(ctx, "node.a")
	if err != nil {
		t.Fatal(err)
	}
	ch2, err := ps.Subscribe(ctx, "output.b")
	if err != nil {
		t.Fatal(err)
	}

	// Store messages destined for different topics.
	msg1 := pubsub.NewMessage(testNode("a"))
	msg1.Metadata[outbox.TopicMetadataKey] = "node.a"
	if err := store.Store(ctx, msg1); err != nil {
		t.Fatal(err)
	}

	msg2 := pubsub.NewMessage(testNode("b"))
	msg2.Metadata[outbox.TopicMetadataKey] = "output.b"
	if err := store.Store(ctx, msg2); err != nil {
		t.Fatal(err)
	}

	// Start relay.
	sub := outbox.NewSubscriber(store, 5*time.Millisecond)
	relay := outbox.NewRelay(sub, ps)
	relayErr := make(chan error, 1)
	go func() {
		relayErr <- relay.Run(ctx)
	}()

	// Verify each message arrives at its correct topic.
	select {
	case received := <-ch1:
		if received.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "a" {
			t.Errorf("topic node.a: got node %q, want %q", received.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "a")
		}
		received.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out on topic node.a")
	}

	select {
	case received := <-ch2:
		if received.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "b" {
			t.Errorf("topic output.b: got node %q, want %q", received.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "b")
		}
		received.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out on topic output.b")
	}

	cancel()
	<-relayErr
}

func TestRelay_StopsOnChannelClose(t *testing.T) {
	store := outboxmem.New()
	ps := memory.New()
	defer func() { _ = ps.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Store one message so the relay has work.
	msg := pubsub.NewMessage(testNode("n1"))
	msg.Metadata[outbox.TopicMetadataKey] = "dest"
	if err := store.Store(ctx, msg); err != nil {
		t.Fatal(err)
	}

	destCh, err := ps.Subscribe(ctx, "dest")
	if err != nil {
		t.Fatal(err)
	}

	sub := outbox.NewSubscriber(store, 5*time.Millisecond)
	relay := outbox.NewRelay(sub, ps)
	relayErr := make(chan error, 1)
	go func() {
		relayErr <- relay.Run(ctx)
	}()

	// Consume the relayed message.
	select {
	case received := <-destCh:
		received.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	// Signal drain and wait for relay to finish.
	sub.CloseWhenDrained()

	select {
	case err := <-relayErr:
		if err != nil {
			t.Errorf("relay returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("relay did not stop after drain")
	}
}

func TestRelay_PreservesOrdering(t *testing.T) {
	store := outboxmem.New()
	ps := memory.New(memory.WithPersistent())
	defer func() { _ = ps.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Store messages in order.
	for i := 0; i < 5; i++ {
		msg := pubsub.NewMessage(testNode(string(rune('a' + i))))
		msg.Metadata[outbox.TopicMetadataKey] = "dest"
		if err := store.Store(ctx, msg); err != nil {
			t.Fatal(err)
		}
	}

	// Subscribe after writing to verify relay delivers in order.
	destCh, err := ps.Subscribe(ctx, "dest")
	if err != nil {
		t.Fatal(err)
	}

	sub := outbox.NewSubscriber(store, 5*time.Millisecond)
	relay := outbox.NewRelay(sub, ps)
	relayErr := make(chan error, 1)
	go func() {
		relayErr <- relay.Run(ctx)
	}()

	expected := []string{"a", "b", "c", "d", "e"}
	for i, want := range expected {
		select {
		case received := <-destCh:
			got := received.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId()
			if got != want {
				t.Errorf("message %d: got %q, want %q", i, got, want)
			}
			received.Ack()
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for message %d (%q)", i, want)
		}
	}

	cancel()
	<-relayErr
}
