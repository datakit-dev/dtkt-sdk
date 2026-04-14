package middleware_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub/memory"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub/middleware"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

func testNode(id string) *flowv1beta2.RunSnapshot_VarNode {
	return &flowv1beta2.RunSnapshot_VarNode{Id: id}
}

func okHandler(msg *pubsub.Message) ([]*pubsub.Message, error) {
	return []*pubsub.Message{msg}, nil
}

var errFail = errors.New("fail")

func failHandler(_ *pubsub.Message) ([]*pubsub.Message, error) {
	return nil, errFail
}

func TestRetry_Success(t *testing.T) {
	r := middleware.Retry{MaxRetries: 3, InitialInterval: time.Millisecond}
	h := r.Middleware(okHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	msgs, err := h(msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
}

func TestRetry_SuccessAfterFailures(t *testing.T) {
	calls := 0
	h := func(msg *pubsub.Message) ([]*pubsub.Message, error) {
		calls++
		if calls < 3 {
			return nil, errFail
		}
		return []*pubsub.Message{msg}, nil
	}
	r := middleware.Retry{MaxRetries: 5, InitialInterval: time.Millisecond, BackoffFactor: 1.0}
	wrapped := r.Middleware(h)
	msg := pubsub.NewMessage(testNode("n1"))
	msgs, err := wrapped(msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_ExhaustedRetries(t *testing.T) {
	r := middleware.Retry{MaxRetries: 2, InitialInterval: time.Millisecond, BackoffFactor: 1.0}
	h := r.Middleware(failHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	_, err := h(msg)
	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
	if !strings.Contains(err.Error(), "2 retries") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRetry_ZeroRetries(t *testing.T) {
	r := middleware.Retry{MaxRetries: 0}
	h := r.Middleware(failHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	_, err := h(msg)
	if !errors.Is(err, errFail) {
		t.Errorf("expected errFail, got %v", err)
	}
}

func TestPoisonQueue_RoutesAfterFailures(t *testing.T) {
	ps := memory.New()
	defer func() { _ = ps.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dlq, err := ps.Subscribe(ctx, "dead-letters")
	if err != nil {
		t.Fatal(err)
	}
	pq := middleware.PoisonQueue{
		MaxRetries: 1,
		Publisher:  ps,
		Topic:      "dead-letters",
	}
	h := pq.Middleware(failHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	msgs, err := h(msg)
	if err != nil {
		t.Fatalf("poison queue should not return error: %v", err)
	}
	if msgs != nil {
		t.Errorf("expected nil msgs, got %d", len(msgs))
	}
	select {
	case received := <-dlq:
		if received.Metadata["poison_reason"] == "" {
			t.Error("expected poison_reason metadata")
		}
		received.Ack()
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for dead-letter message")
	}
}

func TestPoisonQueue_PassesOnSuccess(t *testing.T) {
	pq := middleware.PoisonQueue{MaxRetries: 2, Publisher: memory.New(), Topic: "dlq"}
	h := pq.Middleware(okHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	msgs, err := h(msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
}

func TestRecoverer_NoPanic(t *testing.T) {
	h := middleware.Recoverer(okHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	msgs, err := h(msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
}

func TestRecoverer_CatchesPanic(t *testing.T) {
	panicHandler := func(_ *pubsub.Message) ([]*pubsub.Message, error) {
		panic("boom")
	}
	h := middleware.Recoverer(panicHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	_, err := h(msg)
	if err == nil {
		t.Fatal("expected error from recovered panic")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected panic value in error, got: %v", err)
	}
}

func TestThrottle_RateLimits(t *testing.T) {
	th := middleware.Throttle{Interval: 50 * time.Millisecond}
	h := th.Middleware(okHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	start := time.Now()
	for i := 0; i < 3; i++ {
		if _, err := h(msg); err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)
	if elapsed < 100*time.Millisecond {
		t.Errorf("expected at least 100ms for 3 calls at 50ms interval, got %v", elapsed)
	}
}

func TestThrottle_ZeroInterval(t *testing.T) {
	th := middleware.Throttle{Interval: 0}
	h := th.Middleware(okHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	if _, err := h(msg); err != nil {
		t.Fatal(err)
	}
}

func TestTimeout_Success(t *testing.T) {
	to := middleware.Timeout{Duration: time.Second}
	h := to.Middleware(okHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	msgs, err := h(msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
}

func TestTimeout_Exceeded(t *testing.T) {
	slowHandler := func(msg *pubsub.Message) ([]*pubsub.Message, error) {
		select {
		case <-msg.Context().Done():
			return nil, msg.Context().Err()
		case <-time.After(5 * time.Second):
			return []*pubsub.Message{msg}, nil
		}
	}
	to := middleware.Timeout{Duration: 50 * time.Millisecond}
	h := to.Middleware(slowHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	_, err := h(msg)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout message, got: %v", err)
	}
}

func TestTimeout_ZeroDuration(t *testing.T) {
	to := middleware.Timeout{Duration: 0}
	h := to.Middleware(failHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	_, err := h(msg)
	if !errors.Is(err, errFail) {
		t.Errorf("expected passthrough, got %v", err)
	}
}

func TestCorrelationID_Generates(t *testing.T) {
	outHandler := func(msg *pubsub.Message) ([]*pubsub.Message, error) {
		out := pubsub.NewMessage(testNode("n2"))
		return []*pubsub.Message{out}, nil
	}
	h := middleware.CorrelationID(outHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	msgs, err := h(msg)
	if err != nil {
		t.Fatal(err)
	}
	cid := msg.Metadata[middleware.CorrelationIDKey]
	if cid == "" {
		t.Fatal("expected correlation ID on incoming message")
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 output, got %d", len(msgs))
	}
	if msgs[0].Metadata[middleware.CorrelationIDKey] != cid {
		t.Errorf("output correlation ID %q != input %q",
			msgs[0].Metadata[middleware.CorrelationIDKey], cid)
	}
}

func TestCorrelationID_Preserves(t *testing.T) {
	outHandler := func(msg *pubsub.Message) ([]*pubsub.Message, error) {
		out := pubsub.NewMessage(testNode("n2"))
		return []*pubsub.Message{out}, nil
	}
	h := middleware.CorrelationID(outHandler)
	msg := pubsub.NewMessage(testNode("n1"))
	msg.Metadata[middleware.CorrelationIDKey] = "existing-id"
	msgs, err := h(msg)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Metadata[middleware.CorrelationIDKey] != "existing-id" {
		t.Error("existing correlation ID was overwritten")
	}
	if msgs[0].Metadata[middleware.CorrelationIDKey] != "existing-id" {
		t.Error("output did not inherit existing correlation ID")
	}
}
