package memory

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

func testNode(id string) *flowv1beta2.RunSnapshot_VarNode {
	return &flowv1beta2.RunSnapshot_VarNode{Id: id}
}

func testEvent(id string) *flowv1beta2.RunSnapshot_FlowEvent {
	return &flowv1beta2.RunSnapshot_FlowEvent{
		EventType: flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT,
		Data:      &flowv1beta2.RunSnapshot_FlowEvent_Var{Var: testNode(id)},
	}
}

func TestStore_DirectStore_ReadEvents(t *testing.T) {
	s := New()
	ctx := context.Background()

	msg := pubsub.NewMessage(testNode("n1"))
	if err := s.Store(ctx, msg); err != nil {
		t.Fatal(err)
	}

	got, err := s.ReadEvents(ctx, uuid.Nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d messages, want 1", len(got))
	}
	if got[0].UUID == uuid.Nil {
		t.Error("UUID should not be nil")
	}
	if got[0].Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "n1" {
		t.Errorf("node ID = %q, want %q", got[0].Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "n1")
	}
}

func TestStore_ReadEvents_AfterUID(t *testing.T) {
	s := New()
	ctx := context.Background()

	for i := range 5 {
		msg := pubsub.NewMessage(testNode("n"))
		msg.Metadata["i"] = string(rune('0' + i))
		if err := s.Store(ctx, msg); err != nil {
			t.Fatal(err)
		}
	}

	all, err := s.ReadEvents(ctx, uuid.Nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 5 {
		t.Fatalf("got %d messages, want 5", len(all))
	}

	got, err := s.ReadEvents(ctx, all[2].UUID, 10)
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
}

func TestStore_ReadEvents_Limit(t *testing.T) {
	s := New()
	ctx := context.Background()

	for range 5 {
		if err := s.Store(ctx, pubsub.NewMessage(testNode("n"))); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.ReadEvents(ctx, uuid.Nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d messages, want 2", len(got))
	}
}

func TestStore_ReadEvents_Empty(t *testing.T) {
	s := New()
	ctx := context.Background()

	got, err := s.ReadEvents(ctx, uuid.Nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d messages, want 0", len(got))
	}
}

func TestTx_Commit_MakesWritesVisible(t *testing.T) {
	s := New()
	ctx := context.Background()

	tx, err := s.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	storage := tx.Storage()
	if err := storage.Store(ctx, pubsub.NewMessage(testNode("n1"))); err != nil {
		t.Fatal(err)
	}
	if err := storage.Store(ctx, pubsub.NewMessage(testNode("n2"))); err != nil {
		t.Fatal(err)
	}

	got, err := s.ReadEvents(ctx, uuid.Nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("before commit: got %d messages, want 0", len(got))
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	got, err = s.ReadEvents(ctx, uuid.Nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("after commit: got %d messages, want 2", len(got))
	}
}

func TestTx_Rollback_DiscardsWrites(t *testing.T) {
	s := New()
	ctx := context.Background()

	tx, err := s.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	storage := tx.Storage()
	if err := storage.Store(ctx, pubsub.NewMessage(testNode("n1"))); err != nil {
		t.Fatal(err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	got, err := s.ReadEvents(ctx, uuid.Nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("after rollback: got %d messages, want 0", len(got))
	}
}

func TestTx_DoubleCommit_Error(t *testing.T) {
	s := New()
	ctx := context.Background()

	tx, err := s.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	if err := tx.Commit(); err == nil {
		t.Fatal("expected error on double commit, got nil")
	}
}

func TestTx_DoubleRollback_Error(t *testing.T) {
	s := New()
	ctx := context.Background()

	tx, err := s.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	if err := tx.Rollback(); err == nil {
		t.Fatal("expected error on double rollback, got nil")
	}
}

func TestTx_CommitThenRollback_Error(t *testing.T) {
	s := New()
	ctx := context.Background()

	tx, err := s.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	if err := tx.Rollback(); err == nil {
		t.Fatal("expected error on rollback after commit, got nil")
	}
}

func TestTx_StoreAfterCommit_Error(t *testing.T) {
	s := New()
	ctx := context.Background()

	tx, err := s.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	err = tx.Storage().Store(ctx, pubsub.NewMessage(testNode("n")))
	if err == nil {
		t.Fatal("expected error storing after commit, got nil")
	}
}

func TestTx_StoreAfterRollback_Error(t *testing.T) {
	s := New()
	ctx := context.Background()

	tx, err := s.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	err = tx.Storage().Store(ctx, pubsub.NewMessage(testNode("n")))
	if err == nil {
		t.Fatal("expected error storing after rollback, got nil")
	}
}

func TestTx_SequenceNumbers_Increment(t *testing.T) {
	s := New()
	ctx := context.Background()

	if err := s.Store(ctx, pubsub.NewMessage(testNode("n"))); err != nil {
		t.Fatal(err)
	}

	tx, err := s.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	storage := tx.Storage()
	if err := storage.Store(ctx, pubsub.NewMessage(testNode("n"))); err != nil {
		t.Fatal(err)
	}
	if err := storage.Store(ctx, pubsub.NewMessage(testNode("n"))); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	got, err := s.ReadEvents(ctx, uuid.Nil, 10)
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
}

func TestTx_ReadDelegates_ToStore(t *testing.T) {
	s := New()
	ctx := context.Background()

	if err := s.Store(ctx, pubsub.NewMessage(testNode("n1"))); err != nil {
		t.Fatal(err)
	}

	got, err := s.ReadEvents(ctx, uuid.Nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d messages, want 1", len(got))
	}
}

func TestSnapshotAt(t *testing.T) {
	s := New()
	ctx := context.Background()

	if err := s.Store(ctx, pubsub.NewMessage(testEvent("n1"))); err != nil {
		t.Fatal(err)
	}
	if err := s.Store(ctx, pubsub.NewMessage(testEvent("n2"))); err != nil {
		t.Fatal(err)
	}
	if err := s.Store(ctx, pubsub.NewMessage(testEvent("n1"))); err != nil {
		t.Fatal(err)
	}

	all, _ := s.ReadEvents(ctx, uuid.Nil, 10)

	snap, err := s.SnapshotAt(ctx, all[1].UUID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.GetVars()) != 2 {
		t.Fatalf("snapshot(2): got %d nodes, want 2", len(snap.GetVars()))
	}
	if _, ok := snap.GetVars()["n1"]; !ok {
		t.Error("snapshot(2): missing n1")
	}
	if _, ok := snap.GetVars()["n2"]; !ok {
		t.Error("snapshot(2): missing n2")
	}

	snap, err = s.SnapshotAt(ctx, all[2].UUID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.GetVars()) != 2 {
		t.Fatalf("snapshot(3): got %d nodes, want 2", len(snap.GetVars()))
	}
}

func TestSnapshotAt_Empty(t *testing.T) {
	s := New()
	ctx := context.Background()

	snap, err := s.SnapshotAt(ctx, uuid.Max)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.GetVars()) != 0 {
		t.Fatalf("got %d nodes, want 0", len(snap.GetVars()))
	}
}

func TestSnapshotAt_NilNode(t *testing.T) {
	s := New()
	ctx := context.Background()

	if err := s.Store(ctx, pubsub.NewMessage(nil)); err != nil {
		t.Fatal(err)
	}

	all, _ := s.ReadEvents(ctx, uuid.Nil, 10)

	snap, err := s.SnapshotAt(ctx, all[0].UUID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.GetVars()) != 0 {
		t.Fatalf("got %d nodes, want 0 (nil node should be skipped)", len(snap.GetVars()))
	}
}

func TestMultipleTx_IndependentCommits(t *testing.T) {
	s := New()
	ctx := context.Background()

	tx1, err := s.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tx2, err := s.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := tx1.Storage().Store(ctx, pubsub.NewMessage(testNode("from-tx1"))); err != nil {
		t.Fatal(err)
	}
	if err := tx2.Storage().Store(ctx, pubsub.NewMessage(testNode("from-tx2"))); err != nil {
		t.Fatal(err)
	}

	if err := tx1.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := tx2.Rollback(); err != nil {
		t.Fatal(err)
	}

	got, err := s.ReadEvents(ctx, uuid.Nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d messages, want 1", len(got))
	}
	if got[0].Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId() != "from-tx1" {
		t.Errorf("node ID = %q, want %q", got[0].Payload.(*flowv1beta2.RunSnapshot_VarNode).GetId(), "from-tx1")
	}
}

func TestStateWriter_CommitPersistsState(t *testing.T) {
	s := New()
	ctx := context.Background()

	tx, err := s.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	eventUID := uuid.New()
	snap := &flowv1beta2.RunSnapshot{
		Vars: map[string]*flowv1beta2.RunSnapshot_VarNode{
			"n1": {Id: "n1"},
		},
	}
	if err := tx.StateWriter().WriteState(ctx, snap, eventUID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	got, err := s.SnapshotAt(ctx, eventUID)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.GetVars()["n1"]; !ok {
		t.Fatal("snapshot missing n1")
	}

	latest, err := s.SnapshotAt(ctx, uuid.Max)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := latest.GetVars()["n1"]; !ok {
		t.Fatal("latest snapshot missing n1")
	}
}

func TestStateWriter_RollbackDoesNotPersistState(t *testing.T) {
	s := New()
	ctx := context.Background()

	tx, err := s.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	eventUID := uuid.New()
	snap := &flowv1beta2.RunSnapshot{
		Vars: map[string]*flowv1beta2.RunSnapshot_VarNode{
			"n1": {Id: "n1"},
		},
	}
	if err := tx.StateWriter().WriteState(ctx, snap, eventUID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	got, err := s.SnapshotAt(ctx, eventUID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.GetVars()) != 0 {
		t.Fatalf("got %d vars, want 0", len(got.GetVars()))
	}
}

func TestStateWriter_ClonesStateOnWrite(t *testing.T) {
	s := New()
	ctx := context.Background()

	tx, err := s.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	eventUID := uuid.New()
	snap := &flowv1beta2.RunSnapshot{
		Vars: map[string]*flowv1beta2.RunSnapshot_VarNode{
			"n1": {Id: "n1"},
		},
	}
	if err := tx.StateWriter().WriteState(ctx, snap, eventUID); err != nil {
		t.Fatal(err)
	}

	// Mutate source snapshot after WriteState; committed state should be unchanged.
	snap.Vars["n2"] = &flowv1beta2.RunSnapshot_VarNode{Id: "n2"}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	got, err := s.SnapshotAt(ctx, eventUID)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.GetVars()["n1"]; !ok {
		t.Fatal("snapshot missing n1")
	}
	if _, ok := got.GetVars()["n2"]; ok {
		t.Fatal("snapshot unexpectedly contains n2")
	}
}
