package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub/memory"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub/pubsubtest"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

func testNode(id string) *flowv1beta2.RunSnapshot_VarNode {
	return &flowv1beta2.RunSnapshot_VarNode{Id: id}
}

func TestConformance(t *testing.T) {
	pubsubtest.Run(t, func(t *testing.T) pubsub.PubSub {
		t.Helper()
		return memory.New()
	}, pubsubtest.Options{})
}

func TestPublishSubscribe_Basic(t *testing.T) {
	ps := memory.New()
	defer func() { _ = ps.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := ps.Subscribe(ctx, "topic1")
	if err != nil {
		t.Fatal(err)
	}
	msg := pubsub.NewMessage(testNode("n1"))
	if err := ps.Publish("topic1", msg); err != nil {
		t.Fatal(err)
	}
	select {
	case received := <-ch:
		if received.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "n1" {
			t.Errorf("got node ID %q, want %q", received.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "n1")
		}
		received.Ack()
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestPublishSubscribe_UUID(t *testing.T) {
	ps := memory.New()
	defer func() { _ = ps.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := ps.Subscribe(ctx, "topic1")
	if err != nil {
		t.Fatal(err)
	}
	msg := pubsub.NewMessage(testNode("n1"))
	if msg.UUID == (uuid.UUID{}) {
		t.Fatal("NewMessage should assign a UUID")
	}
	if err := ps.Publish("topic1", msg); err != nil {
		t.Fatal(err)
	}
	select {
	case received := <-ch:
		if received.UUID != msg.UUID {
			t.Errorf("UUID mismatch: got %v, want %v", received.UUID, msg.UUID)
		}
		received.Ack()
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestAckBeforeNext(t *testing.T) {
	ps := memory.New()
	defer func() { _ = ps.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := ps.Subscribe(ctx, "topic1")
	if err != nil {
		t.Fatal(err)
	}
	msg1 := pubsub.NewMessage(testNode("n1"))
	msg2 := pubsub.NewMessage(testNode("n2"))
	if err := ps.Publish("topic1", msg1, msg2); err != nil {
		t.Fatal(err)
	}
	var first *pubsub.Message
	select {
	case first = <-ch:
		if first.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "n1" {
			t.Fatalf("expected n1, got %s", first.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId())
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first message")
	}
	select {
	case <-ch:
		t.Fatal("received second message before acking first")
	case <-time.After(50 * time.Millisecond):
	}
	first.Ack()
	select {
	case second := <-ch:
		if second.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "n2" {
			t.Fatalf("expected n2, got %s", second.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId())
		}
		second.Ack()
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second message after ack")
	}
}

func TestNackRedelivery(t *testing.T) {
	ps := memory.New()
	defer func() { _ = ps.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := ps.Subscribe(ctx, "topic1")
	if err != nil {
		t.Fatal(err)
	}
	msg := pubsub.NewMessage(testNode("n1"))
	originalUUID := msg.UUID
	if err := ps.Publish("topic1", msg); err != nil {
		t.Fatal(err)
	}
	select {
	case received := <-ch:
		received.Nack()
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
	select {
	case redelivered := <-ch:
		if redelivered.UUID != originalUUID {
			t.Errorf("redelivered UUID %v differs from original %v", redelivered.UUID, originalUUID)
		}
		if redelivered.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "n1" {
			t.Errorf("expected node n1, got %s", redelivered.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId())
		}
		redelivered.Ack()
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for redelivery")
	}
}

func TestFanOut_UUIDPreserved(t *testing.T) {
	ps := memory.New()
	defer func() { _ = ps.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch1, err := ps.Subscribe(ctx, "topic1")
	if err != nil {
		t.Fatal(err)
	}
	ch2, err := ps.Subscribe(ctx, "topic1")
	if err != nil {
		t.Fatal(err)
	}
	msg := pubsub.NewMessage(testNode("n1"))
	if err := ps.Publish("topic1", msg); err != nil {
		t.Fatal(err)
	}
	var received1, received2 *pubsub.Message
	select {
	case received1 = <-ch1:
	case <-time.After(time.Second):
		t.Fatal("timed out on subscriber 1")
	}
	select {
	case received2 = <-ch2:
	case <-time.After(time.Second):
		t.Fatal("timed out on subscriber 2")
	}
	if received1.UUID != received2.UUID {
		t.Errorf("fan-out UUID mismatch: %v vs %v", received1.UUID, received2.UUID)
	}
	if received1.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "n1" || received2.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "n1" {
		t.Error("fan-out node mismatch")
	}
	received1.Ack()
	received2.Ack()
}

func TestFanOut_IndependentAck(t *testing.T) {
	ps := memory.New()
	defer func() { _ = ps.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch1, err := ps.Subscribe(ctx, "topic1")
	if err != nil {
		t.Fatal(err)
	}
	ch2, err := ps.Subscribe(ctx, "topic1")
	if err != nil {
		t.Fatal(err)
	}
	msg1 := pubsub.NewMessage(testNode("n1"))
	msg2 := pubsub.NewMessage(testNode("n2"))
	if err := ps.Publish("topic1", msg1, msg2); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-ch1:
		r.Ack()
	case <-time.After(time.Second):
		t.Fatal("sub1 timed out")
	}
	var sub2Msg1 *pubsub.Message
	select {
	case sub2Msg1 = <-ch2:
	case <-time.After(time.Second):
		t.Fatal("sub2 timed out on msg1")
	}
	select {
	case r := <-ch1:
		if r.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "n2" {
			t.Errorf("sub1 expected n2, got %s", r.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId())
		}
		r.Ack()
	case <-time.After(time.Second):
		t.Fatal("sub1 timed out waiting for msg2")
	}
	select {
	case <-ch2:
		t.Fatal("sub2 received msg2 before acking msg1")
	case <-time.After(50 * time.Millisecond):
	}
	sub2Msg1.Ack()
	select {
	case r := <-ch2:
		if r.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "n2" {
			t.Errorf("sub2 expected n2, got %s", r.Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId())
		}
		r.Ack()
	case <-time.After(time.Second):
		t.Fatal("sub2 timed out waiting for msg2 after ack")
	}
}

func TestContextCancellation(t *testing.T) {
	ps := memory.New()
	defer func() { _ = ps.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := ps.Subscribe(ctx, "topic1")
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed after context cancel")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}

func TestAckIdempotent(t *testing.T) {
	msg := pubsub.NewMessage(testNode("n1"))
	msg.Ack()
	msg.Ack()
}

func TestNackIdempotent(t *testing.T) {
	msg := pubsub.NewMessage(testNode("n1"))
	msg.Nack()
	msg.Nack()
}

func TestAckThenNack_NoOp(t *testing.T) {
	msg := pubsub.NewMessage(testNode("n1"))
	msg.Ack()
	msg.Nack()
	select {
	case <-msg.Nacked():
		t.Error("Nacked() signaled after Ack()")
	default:
	}
}

func TestCopyMessage_FreshAckNack(t *testing.T) {
	msg := pubsub.NewMessage(testNode("n1"))
	msg.Metadata["key"] = "value"
	cp := pubsub.CopyMessage(msg)
	if cp.UUID != msg.UUID {
		t.Errorf("UUID mismatch: %v vs %v", cp.UUID, msg.UUID)
	}
	if cp.Metadata["key"] != "value" {
		t.Error("metadata not copied")
	}
	if cp.Payload != msg.Payload {
		t.Error("payload should be the same pointer")
	}
	msg.Ack()
	select {
	case <-cp.Acked():
		t.Error("copy Acked() should not be signaled by original Ack()")
	default:
	}
}

func TestPublishToNoSubscribers(t *testing.T) {
	ps := memory.New()
	defer func() { _ = ps.Close() }()
	msg := pubsub.NewMessage(testNode("n1"))
	if err := ps.Publish("nobody", msg); err != nil {
		t.Fatal(err)
	}
}

func TestMessageContext(t *testing.T) {
	msg := pubsub.NewMessage(testNode("n1"))
	if msg.Context() == nil {
		t.Fatal("default context should not be nil")
	}
	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("k"), "v")
	msg.SetContext(ctx)
	if msg.Context().Value(ctxKey("k")) != "v" {
		t.Error("SetContext did not take effect")
	}
}
