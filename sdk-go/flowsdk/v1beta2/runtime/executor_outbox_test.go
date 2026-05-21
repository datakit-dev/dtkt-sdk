package runtime

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox"
	outboxmem "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox/memory"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

func TestGraph_Outbox_InputToVarToOutput(t *testing.T) {
	parallelByDefault(t)
	graph := loadFlow(t, "outbox_input_var_output.yaml")

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	store := outboxmem.New()
	feedInput(ps, "inputs.x", 5, 10)
	ctx := testContext(t)
	err := NewExecutor(ps, testTopics, WithOutbox(store)).Execute(ctx, graph)
	require.NoError(t, err)

	results := collectOutputs(ctx, ps, "outputs.result")
	require.Len(t, results, 2)
	assert.Equal(t, []int64{10, 20}, outputInt64s(results))

	// Verify events were written to the outbox.
	ctx = context.Background()
	msgs, err := store.ReadEvents(ctx, uuid.Nil, 100)
	require.NoError(t, err)
	require.NotEmpty(t, msgs, "expected outbox events to be written")

	// SnapshotAt should reconstruct final state from the outbox event log.
	snap, err := store.SnapshotAt(ctx, uuid.Max)
	require.NoError(t, err)
	// The var's terminal event should be in the snapshot with SUCCEEDED phase
	// and an EOF value (the computed values were emitted as separate events).
	// Snapshot map keys for vars are the bare spec id (Format A); category
	// is implicit in the field name `vars`.
	require.Contains(t, snap.GetVars(), "double", "snapshot should contain vars.double")
	varNode := snap.GetVars()["double"]
	assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, varNode.GetPhase())
	assert.True(t, isEOFValue(varNode.GetValue()), "terminal var value should be EOF")
	assert.Equal(t, uint64(2), varNode.GetEvalCount(), "var should have been evaluated twice (inputs 5, 10)")
}

func TestGraph_Outbox_Chain(t *testing.T) {
	parallelByDefault(t)
	// input → var A → var B → output
	// All intermediate publishes go through the outbox.
	graph := loadFlow(t, "outbox_chain.yaml")

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	store := outboxmem.New()
	feedInput(ps, "inputs.n", 2, 4)
	ctx := testContext(t)
	err := NewExecutor(ps, testTopics, WithOutbox(store)).Execute(ctx, graph)
	require.NoError(t, err)

	results := collectOutputs(ctx, ps, "outputs.result")
	require.Len(t, results, 2)
	// (2+1)^2 = 9, (4+1)^2 = 25
	assert.Equal(t, []int64{9, 25}, outputInt64s(results))

	// Verify snapshot captures final state for both vars in the chain.
	ctx = context.Background()
	snap, err := store.SnapshotAt(ctx, uuid.Max)
	require.NoError(t, err)
	// Snapshot map keys for vars are the bare spec id (Format A); category
	// is implicit in the field name `vars`.
	for _, varID := range []string{"inc", "sq"} {
		require.Contains(t, snap.GetVars(), varID)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, snap.GetVars()[varID].GetPhase())
		assert.True(t, isEOFValue(snap.GetVars()[varID].GetValue()), "%s terminal value should be EOF", varID)
	}
	assert.Equal(t, uint64(2), snap.GetVars()["inc"].GetEvalCount())
	assert.Equal(t, uint64(2), snap.GetVars()["sq"].GetEvalCount())
}

func TestGraph_Outbox_RangeGenerator(t *testing.T) {
	parallelByDefault(t)
	// Range generator → output, with outbox wiring.
	graph := loadFlow(t, "outbox_range_generator.yaml")

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	store := outboxmem.New()
	ctx := testContext(t)
	err := NewExecutor(ps, testTopics, WithOutbox(store)).Execute(ctx, graph)
	require.NoError(t, err)

	results := collectOutputs(ctx, ps, "outputs.result")
	require.Len(t, results, 3)
	assert.Equal(t, []int64{1, 2, 3}, outputInt64s(results))

	// Verify outbox captured events.
	ctx = context.Background()
	msgs, err := store.ReadEvents(ctx, uuid.Nil, 100)
	require.NoError(t, err)
	require.NotEmpty(t, msgs, "expected outbox events for range generator")
}

func TestGraph_Outbox_SnapshotCaptures(t *testing.T) {
	parallelByDefault(t)
	// Verify that SnapshotAt reconstructs state from outbox events.
	graph := loadFlow(t, "outbox_snapshot.yaml")

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	store := outboxmem.New()
	feedInput(ps, "inputs.x", 3)
	ctx := testContext(t)
	err := NewExecutor(ps, testTopics, WithOutbox(store)).Execute(ctx, graph)
	require.NoError(t, err)
	results := collectOutputs(ctx, ps, "outputs.result")
	require.Len(t, results, 1)
	assert.Equal(t, int64(30), results[0].GetValue().GetInt64Value())

	// Give forwarder time to mark messages as forwarded.
	time.Sleep(100 * time.Millisecond)

	// SnapshotAt with a high seq should include the var node's state.
	ctx = context.Background()
	snap, err := store.SnapshotAt(ctx, uuid.Max)
	require.NoError(t, err)
	// Snapshot map keys for vars are the bare spec id (Format A).
	require.Contains(t, snap.GetVars(), "v")
	varNode := snap.GetVars()["v"]
	assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, varNode.GetPhase())
	assert.True(t, isEOFValue(varNode.GetValue()), "terminal var value should be EOF")
	assert.Equal(t, uint64(1), varNode.GetEvalCount(), "var should have been evaluated once (input 3)")
}

// Outbox replay: paginated cursor-based event readback after a flow run.
//
// Spec contract (outbox.EventReader docstring): "ReadEvents returns events
// after the given UID, ordered chronologically (by UUIDv7). Returns at
// most limit events. Used for cursor-based replay when clients provide
// after_event_id."
//
// Existing TestGraph_Outbox_* tests call ReadEvents(uuid.Nil, 100) once
// and assert non-empty -- they prove events are written but not that
// pagination + cursor advancement work at the executor-integration layer.
// Storage-layer pagination IS covered by outboxtest.ReadEventsAfterUID,
// but that uses a hand-rolled event stream, not a real flow run. This
// test feeds the gap: run a real flow, paginate the resulting event log,
// and assert the cursor-walking client sees every event exactly once in
// chronological order.

func TestGraph_Outbox_PaginatedReplay(t *testing.T) {
	parallelByDefault(t)
	graph := loadFlow(t, "outbox_input_var_output.yaml")

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	store := outboxmem.New()
	// Feed several values to produce a multi-event log.
	feedInput(ps, "inputs.x", 5, 10, 15)
	ctx := testContext(t)
	require.NoError(t, NewExecutor(ps, testTopics, WithOutbox(store)).Execute(ctx, graph))

	// Wait a beat so the relay finishes draining; without this the
	// final var/output EOF events may not be committed yet when we
	// start paginating.
	time.Sleep(100 * time.Millisecond)

	// Single-shot read to establish the ground-truth event count.
	bgCtx := context.Background()
	all, err := store.ReadEvents(bgCtx, uuid.Nil, 1000)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(all), 6,
		"3 inputs through input -> var -> output should produce well over 6 events; got %d", len(all))

	// Walk the cursor in pages of 2. Each page's last UUID becomes the
	// next cursor. Stop when a page comes back empty -- that's the
	// documented end-of-log signal.
	const pageSize = 2
	var paginated []*pubsub.Message
	cursor := uuid.Nil
	for {
		page, perr := store.ReadEvents(bgCtx, cursor, pageSize)
		require.NoError(t, perr)
		if len(page) == 0 {
			break
		}
		paginated = append(paginated, page...)
		cursor = page[len(page)-1].UUID
	}

	require.Equal(t, len(all), len(paginated),
		"paginated walk must surface the same number of events as a single-shot read; "+
			"a mismatch means the cursor either skipped (paginated<all) or repeated (paginated>all) events")

	// Same UUIDs in the same chronological order. UUIDv7 monotonicity
	// means bytes.Compare on adjacent UUIDs must be strictly ascending.
	for i, msg := range paginated {
		require.Equal(t, all[i].UUID, msg.UUID,
			"event %d differs between single-shot and paginated reads: cursor advancement is wrong", i)
		if i > 0 {
			require.Less(t, bytes.Compare(paginated[i-1].UUID[:], msg.UUID[:]), 0,
				"event %d UUID (%v) must be strictly greater than predecessor (%v): "+
					"UUIDv7 chronological-order contract violated", i, msg.UUID, paginated[i-1].UUID)
		}
	}

	// Idempotent re-read: from any cursor mid-log, a client must see
	// the same suffix every time. Pick a mid-log cursor and re-paginate.
	if len(all) >= 4 {
		mid := all[len(all)/2].UUID
		suffix1, err := store.ReadEvents(bgCtx, mid, 1000)
		require.NoError(t, err)
		suffix2, err := store.ReadEvents(bgCtx, mid, 1000)
		require.NoError(t, err)
		require.Equal(t, len(suffix1), len(suffix2),
			"two reads from the same cursor must return the same suffix length")
		for i := range suffix1 {
			require.Equal(t, suffix1[i].UUID, suffix2[i].UUID,
				"two reads from the same cursor must return the same events in the same order at index %d", i)
		}
	}
}

// Outbox replay: intermediate-point snapshot reconstruction.
//
// Spec contract (outbox.SnapshotReader docstring): "Reconstruct state at
// a point in time by applying events up to uid. Used for history viewing
// (local) and checkpoint loading (cloud)."
//
// Existing TestGraph_Outbox_SnapshotCaptures only calls
// SnapshotAt(uuid.Max) -- the final-state read. It cannot distinguish a
// correct intermediate-point reconstruction from a snapshot that always
// returns the latest cached state regardless of cursor. This test feeds
// the gap: take snapshots at THREE points (early, middle, final) and
// assert each shows the correct evalCount for that cursor position.

func TestGraph_Outbox_IntermediateSnapshot(t *testing.T) {
	parallelByDefault(t)
	graph := loadFlow(t, "outbox_input_var_output.yaml")

	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	store := outboxmem.New()
	// Three inputs -> three var evaluations. Each var event has an
	// incrementing evalCount: 1, 2, 3. SnapshotAt the i'th var event
	// must report evalCount=i (events earlier than that var event do
	// not contribute to the var's evalCount).
	feedInput(ps, "inputs.x", 5, 10, 15)
	ctx := testContext(t)
	require.NoError(t, NewExecutor(ps, testTopics, WithOutbox(store)).Execute(ctx, graph))

	time.Sleep(100 * time.Millisecond)

	bgCtx := context.Background()
	all, err := store.ReadEvents(bgCtx, uuid.Nil, 1000)
	require.NoError(t, err)

	// Pull out the var events (where the var.double node updates land).
	// Each var event has a monotonically increasing evalCount.
	type varEvent struct {
		uid       uuid.UUID
		evalCount uint64
		valueIsEOF bool
	}
	var varEvents []varEvent
	for _, msg := range all {
		evt, ok := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
		if !ok {
			continue
		}
		if evt.WhichData() != flowv1beta2.RunSnapshot_FlowEvent_Var_case {
			continue
		}
		v := evt.GetVar()
		if v.GetId() != "double" {
			continue
		}
		varEvents = append(varEvents, varEvent{
			uid:        msg.UUID,
			evalCount:  v.GetEvalCount(),
			valueIsEOF: isEOFValue(v.GetValue()),
		})
	}

	require.GreaterOrEqual(t, len(varEvents), 3,
		"3 inputs -> 3 var data events (excluding any EOF terminal); got %d", len(varEvents))

	// Find the first var event whose evalCount jumped from 0 to 1.
	// That's "after input 1 processed". Snapshot AT that UID must show
	// the var with evalCount=1, value=10 (5*2).
	var firstEvalUID uuid.UUID
	for _, ve := range varEvents {
		if ve.evalCount == 1 && !ve.valueIsEOF {
			firstEvalUID = ve.uid
			break
		}
	}
	require.NotEqual(t, uuid.Nil, firstEvalUID,
		"could not find a var event with evalCount=1; runtime is not emitting per-eval var events")

	snap1, err := store.SnapshotAt(bgCtx, firstEvalUID)
	require.NoError(t, err)
	require.Contains(t, snap1.GetVars(), "double")
	assert.Equal(t, uint64(1), snap1.GetVars()["double"].GetEvalCount(),
		"snapshot at the first var event must show evalCount=1, not the final state")
	assert.Equal(t, int64(10), snap1.GetVars()["double"].GetValue().GetInt64Value(),
		"snapshot at the first var event must show value=10 (first input 5 doubled), not the latest")
	assert.NotEqual(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, snap1.GetVars()["double"].GetPhase(),
		"snapshot at the first var event must NOT yet show terminal phase; "+
			"if SUCCEEDED appears here the SnapshotAt is collapsing to final state regardless of cursor")

	// Snapshot at the SECOND var-eval UID: evalCount=2, value=20.
	var secondEvalUID uuid.UUID
	for _, ve := range varEvents {
		if ve.evalCount == 2 && !ve.valueIsEOF {
			secondEvalUID = ve.uid
			break
		}
	}
	require.NotEqual(t, uuid.Nil, secondEvalUID, "missing var event with evalCount=2")

	snap2, err := store.SnapshotAt(bgCtx, secondEvalUID)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), snap2.GetVars()["double"].GetEvalCount())
	assert.Equal(t, int64(20), snap2.GetVars()["double"].GetValue().GetInt64Value())

	// Final-state snapshot via uuid.Max: var must be EOF/SUCCEEDED with
	// evalCount=3 (all three inputs processed).
	snapFinal, err := store.SnapshotAt(bgCtx, uuid.Max)
	require.NoError(t, err)
	finalVar := snapFinal.GetVars()["double"]
	require.NotNil(t, finalVar)
	assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, finalVar.GetPhase())
	assert.True(t, isEOFValue(finalVar.GetValue()))
	assert.Equal(t, uint64(3), finalVar.GetEvalCount())

	// Sanity: snap1 != snap2 != snapFinal at the eval-count discriminator
	// (proves SnapshotAt actually reflects the cursor rather than
	// always returning the same state).
	require.NotEqual(t, snap1.GetVars()["double"].GetEvalCount(), snap2.GetVars()["double"].GetEvalCount(),
		"snapshots at distinct cursors must yield distinct eval counts")
	require.NotEqual(t, snap2.GetVars()["double"].GetEvalCount(), finalVar.GetEvalCount(),
		"intermediate vs final snapshot must yield distinct eval counts")
}

// --- Unit tests for runtime/outbox.go types ---

type failingTxBeginner struct{}

func (f *failingTxBeginner) BeginStateful(ctx context.Context) (outbox.StatefulTx, error) {
	return nil, errors.New("begin failed")
}

// nopStateWriter implements outbox.StateWriter as a no-op for tests.
type nopStateWriter struct{}

func (nopStateWriter) WriteState(_ context.Context, _ *flowv1beta2.RunSnapshot, _ uuid.UUID) error {
	return nil
}

type failingCommitTx struct {
	storage outbox.Storage
}

func (f *failingCommitTx) Storage() outbox.Storage         { return f.storage }
func (f *failingCommitTx) StateWriter() outbox.StateWriter { return nopStateWriter{} }
func (f *failingCommitTx) Commit() error                   { return errors.New("commit failed") }
func (f *failingCommitTx) Rollback() error                 { return nil }

type failingCommitBeginner struct {
	storage outbox.Storage
}

func (f *failingCommitBeginner) BeginStateful(ctx context.Context) (outbox.StatefulTx, error) {
	return &failingCommitTx{storage: f.storage}, nil
}

// failingStoreTx returns a tx whose Storage.Store always fails.
type failingStoreTx struct{}

func (f *failingStoreTx) Storage() outbox.Storage         { return &failingStore{} }
func (f *failingStoreTx) StateWriter() outbox.StateWriter { return nopStateWriter{} }
func (f *failingStoreTx) Commit() error                   { return nil }
func (f *failingStoreTx) Rollback() error                 { return nil }

type failingStore struct{}

func (f *failingStore) Store(ctx context.Context, msg *pubsub.Message) error {
	return errors.New("store failed")
}

type failingStoreBeginner struct{}

func (f *failingStoreBeginner) BeginStateful(ctx context.Context) (outbox.StatefulTx, error) {
	return &failingStoreTx{}, nil
}

func TestTxPublisher_BeginError(t *testing.T) {
	parallelByDefault(t)
	tp := &txPublisher{txBeginner: &failingTxBeginner{}, snap: &flowv1beta2.RunSnapshot{}}
	msg := pubsub.NewMessage(&flowv1beta2.RunSnapshot_FlowEvent{
		Data: &flowv1beta2.RunSnapshot_FlowEvent_Var{Var: &flowv1beta2.RunSnapshot_VarNode{Id: "n1"}},
	})
	err := tp.Publish("t", msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "begin outbox tx")
}

func TestTxPublisher_CommitError(t *testing.T) {
	parallelByDefault(t)
	store := outboxmem.New()
	tp := &txPublisher{txBeginner: &failingCommitBeginner{storage: store}, snap: &flowv1beta2.RunSnapshot{}}
	msg := pubsub.NewMessage(&flowv1beta2.RunSnapshot_FlowEvent{
		Data: &flowv1beta2.RunSnapshot_FlowEvent_Var{Var: &flowv1beta2.RunSnapshot_VarNode{Id: "n1"}},
	})
	err := tp.Publish("t", msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "commit outbox tx")
}

func TestTxPublisher_PublishError_Rollback(t *testing.T) {
	parallelByDefault(t)
	tp := &txPublisher{txBeginner: &failingStoreBeginner{}, snap: &flowv1beta2.RunSnapshot{}}
	msg := pubsub.NewMessage(&flowv1beta2.RunSnapshot_FlowEvent{
		Data: &flowv1beta2.RunSnapshot_FlowEvent_Var{Var: &flowv1beta2.RunSnapshot_VarNode{Id: "n1"}},
	})
	err := tp.Publish("t", msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store failed")
}

func TestTxPublisher_NoMessages(t *testing.T) {
	parallelByDefault(t)
	store := outboxmem.New()
	tp := &txPublisher{txBeginner: store, snap: &flowv1beta2.RunSnapshot{}}
	// Publishing with no messages still opens and commits a tx.
	require.NoError(t, tp.Publish("t"))
}

func TestTxPublisher_Close(t *testing.T) {
	parallelByDefault(t)
	tp := &txPublisher{txBeginner: &failingTxBeginner{}, snap: &flowv1beta2.RunSnapshot{}}
	require.NoError(t, tp.Close())
}

func TestOutboxPubSub_Subscribe(t *testing.T) {
	parallelByDefault(t)
	ps := newPubSub()
	defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

	ops := &outboxPubSub{pub: ps, sub: ps}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := ops.Subscribe(ctx, "test.topic")
	require.NoError(t, err)
	require.NotNil(t, ch)

	// Publish via the pubsub directly and verify the message arrives on
	// the subscription channel. Proves the subscribe wiring delivers
	// messages, not just returns a dead channel.
	val, _ := nativeToExpr(int64(7))
	require.NoError(t, ps.Publish("test.topic", pubsub.NewMessage(val)))
	select {
	case msg := <-ch:
		require.NotNil(t, msg, "expected message on outbox subscribe channel")
		msg.Ack()
	case <-time.After(500 * time.Millisecond):
		t.Fatal("subscribed channel did not receive published message; outbox Subscribe is not wired through")
	}
}

func TestOutboxPubSub_Close(t *testing.T) {
	parallelByDefault(t)
	ps := newPubSub()
	ops := &outboxPubSub{pub: ps, sub: ps}
	require.NoError(t, ops.Close())
}
