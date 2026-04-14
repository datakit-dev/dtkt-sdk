package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox"
	outboxmem "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox/memory"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

func TestGraph_Outbox_InputToVarToOutput(t *testing.T) {
	graph := loadFlow(t, "outbox_input_var_output.yaml")

	ps := newPubSub()
	defer func() { _ = ps.Close() }()

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
	require.Contains(t, snap.GetVars(), "vars.double", "snapshot should contain vars.double")
	varNode := snap.GetVars()["vars.double"]
	assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, varNode.GetPhase())
	assert.True(t, isEOFValue(varNode.GetValue()), "terminal var value should be EOF")
	assert.Equal(t, uint64(2), varNode.GetEvalCount(), "var should have been evaluated twice (inputs 5, 10)")
}

func TestGraph_Outbox_Chain(t *testing.T) {
	// input → var A → var B → output
	// All intermediate publishes go through the outbox.
	graph := loadFlow(t, "outbox_chain.yaml")

	ps := newPubSub()
	defer func() { _ = ps.Close() }()

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
	for _, varID := range []string{"vars.inc", "vars.sq"} {
		require.Contains(t, snap.GetVars(), varID)
		assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, snap.GetVars()[varID].GetPhase())
		assert.True(t, isEOFValue(snap.GetVars()[varID].GetValue()), "%s terminal value should be EOF", varID)
	}
	assert.Equal(t, uint64(2), snap.GetVars()["vars.inc"].GetEvalCount())
	assert.Equal(t, uint64(2), snap.GetVars()["vars.sq"].GetEvalCount())
}

func TestGraph_Outbox_RangeGenerator(t *testing.T) {
	// Range generator → output, with outbox wiring.
	graph := loadFlow(t, "outbox_range_generator.yaml")

	ps := newPubSub()
	defer func() { _ = ps.Close() }()

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
	// Verify that SnapshotAt reconstructs state from outbox events.
	graph := loadFlow(t, "outbox_snapshot.yaml")

	ps := newPubSub()
	defer func() { _ = ps.Close() }()

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
	require.Contains(t, snap.GetVars(), "vars.v")
	varNode := snap.GetVars()["vars.v"]
	assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, varNode.GetPhase())
	assert.True(t, isEOFValue(varNode.GetValue()), "terminal var value should be EOF")
	assert.Equal(t, uint64(1), varNode.GetEvalCount(), "var should have been evaluated once (input 3)")
}

// --- Unit tests for runtime/outbox.go types ---

type failingTxBeginner struct{}

func (f *failingTxBeginner) Begin(ctx context.Context) (outbox.Tx, error) {
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

func (f *failingCommitBeginner) Begin(ctx context.Context) (outbox.Tx, error) {
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

func (f *failingStoreBeginner) Begin(ctx context.Context) (outbox.Tx, error) {
	return &failingStoreTx{}, nil
}

func TestTxPublisher_BeginError(t *testing.T) {
	tp := &txPublisher{txBeginner: &failingTxBeginner{}, snap: &flowv1beta2.RunSnapshot{}}
	msg := pubsub.NewMessage(&flowv1beta2.RunSnapshot_FlowEvent{
		Data: &flowv1beta2.RunSnapshot_FlowEvent_Var{Var: &flowv1beta2.RunSnapshot_VarNode{Id: "n1"}},
	})
	err := tp.Publish("t", msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "begin outbox tx")
}

func TestTxPublisher_CommitError(t *testing.T) {
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
	tp := &txPublisher{txBeginner: &failingStoreBeginner{}, snap: &flowv1beta2.RunSnapshot{}}
	msg := pubsub.NewMessage(&flowv1beta2.RunSnapshot_FlowEvent{
		Data: &flowv1beta2.RunSnapshot_FlowEvent_Var{Var: &flowv1beta2.RunSnapshot_VarNode{Id: "n1"}},
	})
	err := tp.Publish("t", msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store failed")
}

func TestTxPublisher_NoMessages(t *testing.T) {
	store := outboxmem.New()
	tp := &txPublisher{txBeginner: store, snap: &flowv1beta2.RunSnapshot{}}
	// Publishing with no messages still opens and commits a tx.
	require.NoError(t, tp.Publish("t"))
}

func TestTxPublisher_Close(t *testing.T) {
	tp := &txPublisher{txBeginner: &failingTxBeginner{}, snap: &flowv1beta2.RunSnapshot{}}
	require.NoError(t, tp.Close())
}

func TestOutboxPubSub_Subscribe(t *testing.T) {
	ps := newPubSub()
	defer func() { _ = ps.Close() }()

	ops := &outboxPubSub{pub: ps, sub: ps}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := ops.Subscribe(ctx, "test.topic")
	require.NoError(t, err)
	require.NotNil(t, ch)
}

func TestOutboxPubSub_Close(t *testing.T) {
	ps := newPubSub()
	ops := &outboxPubSub{pub: ps, sub: ps}
	require.NoError(t, ops.Close())
}
