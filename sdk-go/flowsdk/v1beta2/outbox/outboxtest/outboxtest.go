package outboxtest

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// Factory creates a fresh Outbox instance for each test.
type Factory func(t *testing.T) outbox.Outbox

// Options configures which conformance tests to run.
type Options struct {
	// EventReader returns an EventReader for the same backend.
	// If non-nil, SnapshotAt and ReadEvents tests are included.
	EventReader func(t *testing.T, o outbox.Outbox) outbox.EventReader

	// SingleWriter skips tests that require concurrent write transactions
	// (e.g. SQLite only supports one writer at a time).
	SingleWriter bool
}

func testEvent(id string) *flowv1beta2.RunSnapshot_FlowEvent {
	varNode := &flowv1beta2.RunSnapshot_VarNode{}
	varNode.SetId(id)
	evt := &flowv1beta2.RunSnapshot_FlowEvent{}
	evt.SetEventType(flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT)
	evt.SetVar(varNode)
	return evt
}

func payloadVarID(payload interface{}) string {
	event, ok := payload.(*flowv1beta2.RunSnapshot_FlowEvent)
	if !ok {
		return ""
	}
	if event.WhichData() != flowv1beta2.RunSnapshot_FlowEvent_Var_case {
		return ""
	}
	return event.GetVar().GetId()
}

// Run executes the full outbox conformance test suite against the given factory.
func Run(t *testing.T, factory Factory, opts Options) {
	t.Helper()

	t.Run("StoreAndReadEvents", func(t *testing.T) {
		if opts.EventReader == nil {
			t.Skip("skipping: no EventReader provided")
		}
		o := factory(t)
		ctx := context.Background()
		er := opts.EventReader(t, o)

		msg := pubsub.NewMessage(testEvent("n1"))
		if err := o.Store(ctx, msg); err != nil {
			t.Fatal(err)
		}

		got, err := er.ReadEvents(ctx, uuid.Nil, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d messages, want 1", len(got))
		}
		if got[0].UUID == uuid.Nil {
			t.Error("UUID should not be nil")
		}
		if id := payloadVarID(got[0].Payload); id != "n1" {
			t.Errorf("node ID = %q, want %q", id, "n1")
		}
	})

	t.Run("ReadEventsAfterUID", func(t *testing.T) {
		if opts.EventReader == nil {
			t.Skip("skipping: no EventReader provided")
		}
		o := factory(t)
		ctx := context.Background()
		er := opts.EventReader(t, o)

		for range 5 {
			if err := o.Store(ctx, pubsub.NewMessage(testEvent("n"))); err != nil {
				t.Fatal(err)
			}
		}

		all, err := er.ReadEvents(ctx, uuid.Nil, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(all) != 5 {
			t.Fatalf("got %d messages, want 5", len(all))
		}

		got, err := er.ReadEvents(ctx, all[2].UUID, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d messages, want 2", len(got))
		}
		if got[0].UUID != all[3].UUID {
			t.Errorf("first UUID = %v, want %v", got[0].UUID, all[3].UUID)
		}
		if got[1].UUID != all[4].UUID {
			t.Errorf("second UUID = %v, want %v", got[1].UUID, all[4].UUID)
		}
	})

	t.Run("ReadEventsLimit", func(t *testing.T) {
		if opts.EventReader == nil {
			t.Skip("skipping: no EventReader provided")
		}
		o := factory(t)
		ctx := context.Background()
		er := opts.EventReader(t, o)

		for range 5 {
			if err := o.Store(ctx, pubsub.NewMessage(testEvent("n"))); err != nil {
				t.Fatal(err)
			}
		}

		got, err := er.ReadEvents(ctx, uuid.Nil, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d messages, want 2", len(got))
		}
	})

	t.Run("ReadEventsEmpty", func(t *testing.T) {
		if opts.EventReader == nil {
			t.Skip("skipping: no EventReader provided")
		}
		o := factory(t)
		ctx := context.Background()
		er := opts.EventReader(t, o)

		got, err := er.ReadEvents(ctx, uuid.Nil, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("got %d messages, want 0", len(got))
		}
	})

	t.Run("TxCommitVisible", func(t *testing.T) {
		if opts.EventReader == nil {
			t.Skip("skipping: no EventReader provided")
		}
		o := factory(t)
		ctx := context.Background()
		er := opts.EventReader(t, o)

		tx, err := o.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}

		storage := tx.Storage()
		if err := storage.Store(ctx, pubsub.NewMessage(testEvent("n1"))); err != nil {
			t.Fatal(err)
		}
		if err := storage.Store(ctx, pubsub.NewMessage(testEvent("n2"))); err != nil {
			t.Fatal(err)
		}

		// Not visible before commit.
		got, err := er.ReadEvents(ctx, uuid.Nil, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("before commit: got %d messages, want 0", len(got))
		}

		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}

		// Visible after commit.
		got, err = er.ReadEvents(ctx, uuid.Nil, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("after commit: got %d messages, want 2", len(got))
		}
	})

	t.Run("TxRollbackDiscards", func(t *testing.T) {
		if opts.EventReader == nil {
			t.Skip("skipping: no EventReader provided")
		}
		o := factory(t)
		ctx := context.Background()
		er := opts.EventReader(t, o)

		tx, err := o.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if err := tx.Storage().Store(ctx, pubsub.NewMessage(testEvent("n1"))); err != nil {
			t.Fatal(err)
		}

		if err := tx.Rollback(); err != nil {
			t.Fatal(err)
		}

		got, err := er.ReadEvents(ctx, uuid.Nil, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("after rollback: got %d messages, want 0", len(got))
		}
	})

	t.Run("TxDoubleCommitError", func(t *testing.T) {
		o := factory(t)
		ctx := context.Background()

		tx, err := o.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err == nil {
			t.Fatal("expected error on double commit")
		}
	})

	t.Run("TxDoubleRollbackError", func(t *testing.T) {
		o := factory(t)
		ctx := context.Background()

		tx, err := o.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Rollback(); err != nil {
			t.Fatal(err)
		}
		if err := tx.Rollback(); err == nil {
			t.Fatal("expected error on double rollback")
		}
	})

	t.Run("TxStoreAfterCommitError", func(t *testing.T) {
		o := factory(t)
		ctx := context.Background()

		tx, err := o.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		if err := tx.Storage().Store(ctx, pubsub.NewMessage(testEvent("n"))); err == nil {
			t.Fatal("expected error storing after commit")
		}
	})

	t.Run("TxStoreAfterRollbackError", func(t *testing.T) {
		o := factory(t)
		ctx := context.Background()

		tx, err := o.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Rollback(); err != nil {
			t.Fatal(err)
		}
		if err := tx.Storage().Store(ctx, pubsub.NewMessage(testEvent("n"))); err == nil {
			t.Fatal("expected error storing after rollback")
		}
	})

	t.Run("SequenceIncrement", func(t *testing.T) {
		if opts.EventReader == nil {
			t.Skip("skipping: no EventReader provided")
		}
		o := factory(t)
		ctx := context.Background()
		er := opts.EventReader(t, o)

		if err := o.Store(ctx, pubsub.NewMessage(testEvent("n"))); err != nil {
			t.Fatal(err)
		}

		tx, err := o.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		storage := tx.Storage()
		if err := storage.Store(ctx, pubsub.NewMessage(testEvent("n"))); err != nil {
			t.Fatal(err)
		}
		if err := storage.Store(ctx, pubsub.NewMessage(testEvent("n"))); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}

		got, err := er.ReadEvents(ctx, uuid.Nil, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d messages, want 3", len(got))
		}
		for i := 1; i < len(got); i++ {
			if bytes.Compare(got[i].UUID[:], got[i-1].UUID[:]) <= 0 {
				t.Errorf("got[%d].UUID (%v) not after got[%d].UUID (%v)", i, got[i].UUID, i-1, got[i-1].UUID)
			}
		}
	})

	t.Run("MultipleTxIndependent", func(t *testing.T) {
		if opts.SingleWriter {
			t.Skip("skipping: backend does not support concurrent write transactions")
		}
		if opts.EventReader == nil {
			t.Skip("skipping: no EventReader provided")
		}
		o := factory(t)
		ctx := context.Background()
		er := opts.EventReader(t, o)

		tx1, err := o.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		tx2, err := o.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if err := tx1.Storage().Store(ctx, pubsub.NewMessage(testEvent("from-tx1"))); err != nil {
			t.Fatal(err)
		}
		if err := tx2.Storage().Store(ctx, pubsub.NewMessage(testEvent("from-tx2"))); err != nil {
			t.Fatal(err)
		}

		if err := tx1.Commit(); err != nil {
			t.Fatal(err)
		}
		if err := tx2.Rollback(); err != nil {
			t.Fatal(err)
		}

		got, err := er.ReadEvents(ctx, uuid.Nil, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d messages, want 1", len(got))
		}
		if id := payloadVarID(got[0].Payload); id != "from-tx1" {
			t.Errorf("node ID = %q, want %q", id, "from-tx1")
		}
	})

	// EventReader tests (optional).
	if opts.EventReader != nil {
		t.Run("SnapshotAt", func(t *testing.T) {
			o := factory(t)
			ctx := context.Background()
			er := opts.EventReader(t, o)

			if err := o.Store(ctx, pubsub.NewMessage(testEvent("n1"))); err != nil {
				t.Fatal(err)
			}
			if err := o.Store(ctx, pubsub.NewMessage(testEvent("n2"))); err != nil {
				t.Fatal(err)
			}
			if err := o.Store(ctx, pubsub.NewMessage(testEvent("n1"))); err != nil {
				t.Fatal(err)
			}

			all, err := er.ReadEvents(ctx, uuid.Nil, 10)
			if err != nil {
				t.Fatal(err)
			}

			snap, err := er.SnapshotAt(ctx, all[1].UUID)
			if err != nil {
				t.Fatal(err)
			}
			if len(snap.GetVars()) != 2 {
				t.Fatalf("snapshot(2): got %d vars, want 2", len(snap.GetVars()))
			}
			if _, ok := snap.GetVars()["n1"]; !ok {
				t.Error("snapshot(2): missing n1")
			}
			if _, ok := snap.GetVars()["n2"]; !ok {
				t.Error("snapshot(2): missing n2")
			}
		})

		t.Run("SnapshotAtEmpty", func(t *testing.T) {
			o := factory(t)
			ctx := context.Background()
			er := opts.EventReader(t, o)

			snap, err := er.SnapshotAt(ctx, uuid.Max)
			if err != nil {
				t.Fatal(err)
			}
			if len(snap.GetVars()) != 0 {
				t.Fatalf("got %d vars, want 0", len(snap.GetVars()))
			}
		})
	}
}
