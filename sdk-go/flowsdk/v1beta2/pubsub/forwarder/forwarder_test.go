package forwarder_test

import (
	"context"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub/forwarder"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub/memory"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

func testNode(id string) *flowv1beta2.RunSnapshot_VarNode {
	return &flowv1beta2.RunSnapshot_VarNode{Id: id}
}

func TestForwarder_EndToEnd(t *testing.T) {
	ps := memory.New()
	defer func() { _ = ps.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Subscribe to the real destination topic.
	destCh, err := ps.Subscribe(ctx, "real-topic")
	if err != nil {
		t.Fatal(err)
	}

	// Start the forwarder daemon.
	fwd := forwarder.NewForwarder(ps, ps)
	errCh := make(chan error, 1)
	go func() {
		errCh <- fwd.Run(ctx)
	}()

	// Publish through the ForwarderPublisher.
	fp := forwarder.NewPublisher(ps)
	msg := pubsub.NewMessage(testNode("n1"))
	msg.Metadata["user_key"] = "user_val"
	if err := fp.Publish("real-topic", msg); err != nil {
		t.Fatal(err)
	}

	// The forwarder should relay the message to the real destination.
	select {
	case received := <-destCh:
		if received.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "n1" {
			t.Errorf("expected node n1, got %s", received.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId())
		}
		if received.Metadata["user_key"] != "user_val" {
			t.Error("user metadata not preserved")
		}
		if _, exists := received.Metadata["_forwarder_dest"]; exists {
			t.Error("forwarder envelope metadata should be removed")
		}
		received.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for forwarded message")
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Errorf("unexpected forwarder error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("forwarder did not exit after cancel")
	}
}

func TestForwarder_MultipleMessages(t *testing.T) {
	ps := memory.New()
	defer func() { _ = ps.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	destCh, err := ps.Subscribe(ctx, "dest")
	if err != nil {
		t.Fatal(err)
	}

	fwd := forwarder.NewForwarder(ps, ps)
	go func() { _ = fwd.Run(ctx) }()
	time.Sleep(50 * time.Millisecond) // let forwarder subscribe

	fp := forwarder.NewPublisher(ps)
	for i := 0; i < 3; i++ {
		msg := pubsub.NewMessage(testNode("n1"))
		if err := fp.Publish("dest", msg); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 3; i++ {
		select {
		case received := <-destCh:
			received.Ack()
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for message %d", i)
		}
	}
}

func TestForwarder_DifferentTopics(t *testing.T) {
	ps := memory.New()
	defer func() { _ = ps.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch1, err := ps.Subscribe(ctx, "topic-a")
	if err != nil {
		t.Fatal(err)
	}
	ch2, err := ps.Subscribe(ctx, "topic-b")
	if err != nil {
		t.Fatal(err)
	}

	fwd := forwarder.NewForwarder(ps, ps)
	go func() { _ = fwd.Run(ctx) }()
	time.Sleep(50 * time.Millisecond) // let forwarder subscribe

	fp := forwarder.NewPublisher(ps)

	msgA := pubsub.NewMessage(testNode("a"))
	if err := fp.Publish("topic-a", msgA); err != nil {
		t.Fatal(err)
	}
	msgB := pubsub.NewMessage(testNode("b"))
	if err := fp.Publish("topic-b", msgB); err != nil {
		t.Fatal(err)
	}

	select {
	case r := <-ch1:
		if r.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "a" {
			t.Errorf("topic-a: expected node a, got %s", r.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId())
		}
		r.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for topic-a message")
	}

	select {
	case r := <-ch2:
		if r.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "b" {
			t.Errorf("topic-b: expected node b, got %s", r.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId())
		}
		r.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for topic-b message")
	}
}

func TestForwarderPublisher_UUIDPreserved(t *testing.T) {
	ps := memory.New()
	defer func() { _ = ps.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	destCh, err := ps.Subscribe(ctx, "dest")
	if err != nil {
		t.Fatal(err)
	}

	fwd := forwarder.NewForwarder(ps, ps)
	go func() { _ = fwd.Run(ctx) }()
	time.Sleep(50 * time.Millisecond) // let forwarder subscribe

	fp := forwarder.NewPublisher(ps)
	msg := pubsub.NewMessage(testNode("n1"))
	origUUID := msg.UUID
	if err := fp.Publish("dest", msg); err != nil {
		t.Fatal(err)
	}

	select {
	case received := <-destCh:
		if received.UUID != origUUID {
			t.Errorf("UUID mismatch: got %v, want %v", received.UUID, origUUID)
		}
		received.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}
}
