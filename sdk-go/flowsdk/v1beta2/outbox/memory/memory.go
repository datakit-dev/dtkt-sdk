package memory

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// Store is an in-memory outbox backend that implements Storage, EventReader,
// and TxBeginner. It uses a mutex-guarded append buffer with transaction
// semantics: writes are staged during a transaction and become visible only
// on Commit.
type Store struct {
	mu           sync.Mutex
	records      []record
	state        *flowv1beta2.RunSnapshot
	lastEventUID *uuid.UUID
}

type record struct {
	msg *pubsub.Message
}

// New creates a new in-memory outbox store.
func New() *Store {
	return &Store{}
}

// Begin opens a new transaction.
func (s *Store) Begin(_ context.Context) (outbox.Tx, error) {
	return &memTx{store: s}, nil
}

// SnapshotAt reconstructs state at a point in time by applying events up to uid.
func (s *Store) SnapshotAt(_ context.Context, uid uuid.UUID) (*flowv1beta2.RunSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if uid == uuid.Max && s.state != nil {
		return cloneRunSnapshot(s.state), nil
	}
	if s.lastEventUID != nil && *s.lastEventUID == uid && s.state != nil {
		return cloneRunSnapshot(s.state), nil
	}

	snap := &flowv1beta2.RunSnapshot{}
	for _, r := range s.records {
		if bytes.Compare(r.msg.UUID[:], uid[:]) > 0 {
			break
		}
		if event, ok := r.msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent); ok {
			outbox.ApplyFlowEvent(snap, event)
		}
	}
	return snap, nil
}

// ReadEvents returns all events after the given UID, ordered chronologically.
// Used for cursor-based client replay.
func (s *Store) ReadEvents(_ context.Context, afterUID uuid.UUID, limit int) ([]*pubsub.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	startIdx := 0
	if afterUID != (uuid.UUID{}) {
		for i := range s.records {
			if s.records[i].msg.UUID == afterUID {
				startIdx = i + 1
				break
			}
		}
	}

	var result []*pubsub.Message
	for i := startIdx; i < len(s.records); i++ {
		result = append(result, s.records[i].msg)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

// commit appends staged records to the store.
func (s *Store) commit(staged []record, stagedState *flowv1beta2.RunSnapshot, stagedEventUID *uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, staged...)
	if stagedEventUID != nil {
		s.state = cloneRunSnapshot(stagedState)
		eventUID := *stagedEventUID
		s.lastEventUID = &eventUID
	}
}

// memTx is an in-memory transaction.
type memTx struct {
	store          *Store
	staged         []record
	stagedState    *flowv1beta2.RunSnapshot
	stagedEventUID *uuid.UUID
	stateWritten   bool
	done           bool
	mu             sync.Mutex
}

// Storage returns a tx-bound Storage that stages writes.
func (tx *memTx) Storage() outbox.Storage {
	return &txStorage{tx: tx}
}

// StateWriter returns a tx-bound writer that stages state updates.
func (tx *memTx) StateWriter() outbox.StateWriter {
	return &memStateWriter{tx: tx}
}

// Commit makes all staged writes visible.
func (tx *memTx) Commit() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.done {
		return fmt.Errorf("transaction already completed")
	}
	tx.done = true
	var eventUID *uuid.UUID
	var state *flowv1beta2.RunSnapshot
	if tx.stateWritten {
		eventUID = tx.stagedEventUID
		state = tx.stagedState
	}
	tx.store.commit(tx.staged, state, eventUID)
	return nil
}

// Rollback discards all staged writes.
func (tx *memTx) Rollback() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.done {
		return fmt.Errorf("transaction already completed")
	}
	tx.done = true
	tx.staged = nil
	tx.stagedState = nil
	tx.stagedEventUID = nil
	tx.stateWritten = false
	return nil
}

// txStorage stages writes for the transaction.
type txStorage struct {
	tx *memTx
}

func (ts *txStorage) Store(_ context.Context, msg *pubsub.Message) error {
	ts.tx.mu.Lock()
	defer ts.tx.mu.Unlock()
	if ts.tx.done {
		return fmt.Errorf("transaction already completed")
	}
	ts.tx.staged = append(ts.tx.staged, record{
		msg: msg,
	})
	return nil
}

// Store implements Storage for non-transactional writes (directly appends to the store).
func (s *Store) Store(_ context.Context, msg *pubsub.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, record{
		msg: msg,
	})
	return nil
}

type memStateWriter struct {
	tx *memTx
}

func (sw *memStateWriter) WriteState(_ context.Context, snap *flowv1beta2.RunSnapshot, eventUID uuid.UUID) error {
	sw.tx.mu.Lock()
	defer sw.tx.mu.Unlock()
	if sw.tx.done {
		return fmt.Errorf("transaction already completed")
	}
	sw.tx.stagedState = cloneRunSnapshot(snap)
	uidCopy := eventUID
	sw.tx.stagedEventUID = &uidCopy
	sw.tx.stateWritten = true
	return nil
}

func cloneRunSnapshot(snap *flowv1beta2.RunSnapshot) *flowv1beta2.RunSnapshot {
	if snap == nil {
		return &flowv1beta2.RunSnapshot{}
	}
	cloned, ok := proto.Clone(snap).(*flowv1beta2.RunSnapshot)
	if !ok {
		return &flowv1beta2.RunSnapshot{}
	}
	return cloned
}

// Compile-time interface assertions.
var (
	_ outbox.Storage     = (*Store)(nil)
	_ outbox.EventReader = (*Store)(nil)
	_ outbox.TxBeginner  = (*Store)(nil)
	_ outbox.Outbox      = (*Store)(nil)
	_ outbox.Storage     = (*txStorage)(nil)
	_ outbox.StateWriter = (*memStateWriter)(nil)
)
