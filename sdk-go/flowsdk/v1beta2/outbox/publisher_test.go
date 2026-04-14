package outbox_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox"
	outboxmem "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox/memory"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

func testNode(id string) *flowv1beta2.RunSnapshot_VarNode {
	return &flowv1beta2.RunSnapshot_VarNode{Id: id}
}

// failingStorage is a Storage that returns an error on Store.
type failingStorage struct {
	outbox.Storage
}

func (f *failingStorage) Store(_ context.Context, _ *pubsub.Message) error {
	return errors.New("store failed")
}

func TestPublisherAdapter_Publish(t *testing.T) {
	store := outboxmem.New()
	pub := outbox.NewPublisher(store)

	msg := pubsub.NewMessage(testNode("n1"))
	if err := pub.Publish("topic.a", msg); err != nil {
		t.Fatal(err)
	}

	got, err := store.ReadEvents(context.Background(), uuid.UUID{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d messages, want 1", len(got))
	}
	if got[0].Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "n1" {
		t.Errorf("node ID = %q, want %q", got[0].Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "n1")
	}
}

func TestPublisherAdapter_Publish_MultipleMessages(t *testing.T) {
	store := outboxmem.New()
	pub := outbox.NewPublisher(store)

	msgs := []*pubsub.Message{
		pubsub.NewMessage(testNode("a")),
		pubsub.NewMessage(testNode("b")),
		pubsub.NewMessage(testNode("c")),
	}
	if err := pub.Publish("t", msgs...); err != nil {
		t.Fatal(err)
	}

	got, err := store.ReadEvents(context.Background(), uuid.UUID{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d messages, want 3", len(got))
	}
	for i, want := range []string{"a", "b", "c"} {
		if got[i].Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != want {
			t.Errorf("msg[%d] node ID = %q, want %q", i, got[i].Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId(), want)
		}
	}
}

func TestPublisherAdapter_TxBound(t *testing.T) {
	store := outboxmem.New()
	ctx := context.Background()

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	pub := outbox.NewPublisher(tx.Storage())
	if err := pub.Publish("t", pubsub.NewMessage(testNode("n1"))); err != nil {
		t.Fatal(err)
	}

	got, _ := store.ReadEvents(ctx, uuid.UUID{}, 10)
	if len(got) != 0 {
		t.Fatalf("got %d messages before commit, want 0", len(got))
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	got, _ = store.ReadEvents(ctx, uuid.UUID{}, 10)
	if len(got) != 1 {
		t.Fatalf("got %d messages after commit, want 1", len(got))
	}
}

func TestPublisherAdapter_Close(t *testing.T) {
	store := outboxmem.New()
	pub := outbox.NewPublisher(store)
	if err := pub.Close(); err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
}

func TestPublisherAdapter_Publish_StoreError(t *testing.T) {
	pub := outbox.NewPublisher(&failingStorage{})
	err := pub.Publish("t", pubsub.NewMessage(testNode("n1")))
	if err == nil {
		t.Fatal("expected error from Publish, got nil")
	}
	if err.Error() != "store failed" {
		t.Errorf("error = %q, want %q", err.Error(), "store failed")
	}
}
