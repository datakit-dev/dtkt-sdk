package outbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox"
	outboxmem "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox/memory"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

func TestSubscriberAdapter_DeliversMessages(t *testing.T) {
	store := outboxmem.New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Store a message directly.
	msg := pubsub.NewMessage(testNode("n1"))
	if err := store.Store(ctx, msg); err != nil {
		t.Fatal(err)
	}

	sub := outbox.NewSubscriber(store, 5*time.Millisecond)
	ch, err := sub.Subscribe(ctx, "my.topic")
	if err != nil {
		t.Fatal(err)
	}

	select {
	case delivered := <-ch:
		if delivered.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "n1" {
			t.Errorf("node ID = %q, want %q", delivered.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "n1")
		}
		delivered.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestSubscriberAdapter_NackRedelivers(t *testing.T) {
	store := outboxmem.New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := store.Store(ctx, pubsub.NewMessage(testNode("n1"))); err != nil {
		t.Fatal(err)
	}

	sub := outbox.NewSubscriber(store, 5*time.Millisecond)
	ch, err := sub.Subscribe(ctx, "t")
	if err != nil {
		t.Fatal(err)
	}

	// First delivery -- nack it.
	select {
	case msg := <-ch:
		msg.Nack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first delivery")
	}

	// Second delivery -- ack it.
	select {
	case msg := <-ch:
		if msg.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "n1" {
			t.Errorf("node ID = %q, want %q", msg.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "n1")
		}
		msg.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for redelivery")
	}
}

func TestSubscriberAdapter_TxCommitThenDeliver(t *testing.T) {
	store := outboxmem.New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub := outbox.NewSubscriber(store, 5*time.Millisecond)
	ch, err := sub.Subscribe(ctx, "t")
	if err != nil {
		t.Fatal(err)
	}

	// Start subscriber before any messages exist.
	// Store a message via transaction after a short delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		tx, _ := store.Begin(ctx)
		_ = outbox.NewPublisher(tx.Storage()).Publish("t", pubsub.NewMessage(testNode("delayed")))
		_ = tx.Commit()
	}()

	select {
	case msg := <-ch:
		if msg.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "delayed" {
			t.Errorf("node ID = %q, want %q", msg.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "delayed")
		}
		msg.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for tx-committed message")
	}
}

func TestSubscriberAdapter_ContextCancel(t *testing.T) {
	store := outboxmem.New()
	ctx, cancel := context.WithCancel(context.Background())

	sub := outbox.NewSubscriber(store, 5*time.Millisecond)
	ch, err := sub.Subscribe(ctx, "t")
	if err != nil {
		t.Fatal(err)
	}

	// Cancel context -- channel should close.
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed after cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}

func TestSubscriberAdapter_Close(t *testing.T) {
	store := outboxmem.New()
	ctx := context.Background()

	sub := outbox.NewSubscriber(store, 5*time.Millisecond)
	ch, err := sub.Subscribe(ctx, "t")
	if err != nil {
		t.Fatal(err)
	}

	// Close the subscriber -- channel should close.
	if err := sub.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed after Close()")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}

func TestSubscriberAdapter_ForwarderIntegration(t *testing.T) {
	// End-to-end: publisher writes enveloped messages to outbox,
	// subscriber delivers them, simulating the Forwarder relay path.
	store := outboxmem.New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Write 3 messages via tx (simulating handler → ForwarderPublisher → outbox).
	tx, _ := store.Begin(ctx)
	txPub := outbox.NewPublisher(tx.Storage())
	nodes := []*flowv1beta2.RunSnapshot_VarNode{testNode("a"), testNode("b"), testNode("c")}
	for _, n := range nodes {
		if err := txPub.Publish("_forwarder", pubsub.NewMessage(n)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// Subscribe and consume all 3.
	sub := outbox.NewSubscriber(store, 5*time.Millisecond)
	ch, err := sub.Subscribe(ctx, "_forwarder")
	if err != nil {
		t.Fatal(err)
	}

	var delivered []string
	for range 3 {
		select {
		case msg := <-ch:
			delivered = append(delivered, msg.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId())
			msg.Ack()
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out after %d deliveries", len(delivered))
		}
	}

	if len(delivered) != 3 {
		t.Fatalf("got %d deliveries, want 3", len(delivered))
	}
	for i, want := range []string{"a", "b", "c"} {
		if delivered[i] != want {
			t.Errorf("delivered[%d] = %q, want %q", i, delivered[i], want)
		}
	}
}

func TestSubscriberAdapter_DefaultPollInterval(t *testing.T) {
	store := outboxmem.New()

	// Pass 0 -- should use the 100ms default without panic.
	sub := outbox.NewSubscriber(store, 0)
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := sub.Subscribe(ctx, "t")
	if err != nil {
		t.Fatal(err)
	}

	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to close")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}

func TestSubscriberAdapter_CancelDuringSend(t *testing.T) {
	store := outboxmem.New()
	ctx, cancel := context.WithCancel(context.Background())

	// Store a message so poll has something to deliver.
	if err := store.Store(ctx, pubsub.NewMessage(testNode("n1"))); err != nil {
		t.Fatal(err)
	}

	sub := outbox.NewSubscriber(store, 5*time.Millisecond)
	ch, err := sub.Subscribe(ctx, "t")
	if err != nil {
		t.Fatal(err)
	}

	// Cancel before reading from channel -- poll blocks on `out <- msg`.
	// Give poll a moment to reach the send.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			// Drained a message -- that's fine, channel will close next.
			<-ch
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}

func TestSubscriberAdapter_CancelDuringAckWait(t *testing.T) {
	store := outboxmem.New()
	ctx, cancel := context.WithCancel(context.Background())

	if err := store.Store(ctx, pubsub.NewMessage(testNode("n1"))); err != nil {
		t.Fatal(err)
	}

	sub := outbox.NewSubscriber(store, 5*time.Millisecond)
	ch, err := sub.Subscribe(ctx, "t")
	if err != nil {
		t.Fatal(err)
	}

	// Read message but don't ack -- poll blocks on ack/nack select.
	select {
	case <-ch:
		// Got the message; now cancel while poll waits for ack.
		cancel()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	// Channel should close.
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel closed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}

func TestSubscriberAdapter_CancelDuringNackSleep(t *testing.T) {
	store := outboxmem.New()
	ctx, cancel := context.WithCancel(context.Background())

	if err := store.Store(ctx, pubsub.NewMessage(testNode("n1"))); err != nil {
		t.Fatal(err)
	}

	sub := outbox.NewSubscriber(store, 200*time.Millisecond)
	ch, err := sub.Subscribe(ctx, "t")
	if err != nil {
		t.Fatal(err)
	}

	// Nack the message -- poll enters the nack sleep select.
	select {
	case msg := <-ch:
		msg.Nack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	// Cancel during the nack sleep (200ms interval).
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel closed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}
