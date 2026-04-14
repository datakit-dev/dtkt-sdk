package outbox

import (
	"context"

	"github.com/google/uuid"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// Storage persists events to the outbox.
type Storage interface {
	// Store writes a message to the outbox.
	Store(ctx context.Context, msg *pubsub.Message) error
}

// EventReader reads from the event log: reconstruct state at a point in time,
// or paginate through raw events for client replay.
type EventReader interface {
	// SnapshotAt reconstructs state at a point in time by applying events up to uid.
	// Returns a RunSnapshot message with typed node maps populated from FlowEvents.
	// Used for history viewing (local) and checkpoint loading (cloud).
	SnapshotAt(ctx context.Context, uid uuid.UUID) (*flowv1beta2.RunSnapshot, error)

	// ReadEvents returns events after the given UID, ordered chronologically
	// (by UUIDv7). Returns at most limit events. Used for cursor-based replay
	// when clients provide after_event_id.
	ReadEvents(ctx context.Context, afterUID uuid.UUID, limit int) ([]*pubsub.Message, error)
}

// StateWriter writes the materialized RunSnapshot and last-event cursor
// within the same transaction that stores the FlowEvent. This is the
// core of the outbox pattern: event + state written atomically.
//
// Implementations that don't track persistent state (e.g. in-memory)
// may return a no-op writer from Tx.StateWriter().
type StateWriter interface {
	// WriteState persists the materialized snapshot and the UID of the
	// event that produced it. Called once per event in the same tx as
	// Storage().Store().
	WriteState(ctx context.Context, snap *flowv1beta2.RunSnapshot, eventUID uuid.UUID) error
}

// Tx represents a transaction that can atomically commit state + outbox writes.
type Tx interface {
	// Storage returns an outbox Storage bound to this transaction.
	Storage() Storage
	// StateWriter returns a writer for the materialized RunSnapshot.
	// The returned writer must operate within this transaction so that
	// event storage and state updates commit atomically.
	StateWriter() StateWriter
	Commit() error
	Rollback() error
}

// TxBeginner opens new transactions.
type TxBeginner interface {
	Begin(ctx context.Context) (Tx, error)
}

// Outbox combines TxBeginner, Storage, and EventReader for outbox backends
// that support transactional writes and event reads. All backends (memory, ent)
// implement this interface.
type Outbox interface {
	TxBeginner
	Storage
	EventReader
}

// ApplyFlowEvent dispatches a FlowEvent into the appropriate field on snap.
// Node events update the typed node maps; flow events update the flow state.
func ApplyFlowEvent(snap *flowv1beta2.RunSnapshot, event *flowv1beta2.RunSnapshot_FlowEvent) {
	switch event.WhichData() {
	case flowv1beta2.RunSnapshot_FlowEvent_Input_case:
		m := snap.GetInputs()
		if m == nil {
			m = make(map[string]*flowv1beta2.RunSnapshot_InputNode)
		}
		m[event.GetInput().GetId()] = event.GetInput()
		snap.SetInputs(m)
	case flowv1beta2.RunSnapshot_FlowEvent_Generator_case:
		m := snap.GetGenerators()
		if m == nil {
			m = make(map[string]*flowv1beta2.RunSnapshot_GeneratorNode)
		}
		m[event.GetGenerator().GetId()] = event.GetGenerator()
		snap.SetGenerators(m)
	case flowv1beta2.RunSnapshot_FlowEvent_Var_case:
		m := snap.GetVars()
		if m == nil {
			m = make(map[string]*flowv1beta2.RunSnapshot_VarNode)
		}
		m[event.GetVar().GetId()] = event.GetVar()
		snap.SetVars(m)
	case flowv1beta2.RunSnapshot_FlowEvent_Action_case:
		m := snap.GetActions()
		if m == nil {
			m = make(map[string]*flowv1beta2.RunSnapshot_ActionNode)
		}
		m[event.GetAction().GetId()] = event.GetAction()
		snap.SetActions(m)
	case flowv1beta2.RunSnapshot_FlowEvent_Stream_case:
		m := snap.GetStreams()
		if m == nil {
			m = make(map[string]*flowv1beta2.RunSnapshot_StreamNode)
		}
		m[event.GetStream().GetId()] = event.GetStream()
		snap.SetStreams(m)
	case flowv1beta2.RunSnapshot_FlowEvent_Output_case:
		m := snap.GetOutputs()
		if m == nil {
			m = make(map[string]*flowv1beta2.RunSnapshot_OutputNode)
		}
		m[event.GetOutput().GetId()] = event.GetOutput()
		snap.SetOutputs(m)
	case flowv1beta2.RunSnapshot_FlowEvent_Interaction_case:
		m := snap.GetInteractions()
		if m == nil {
			m = make(map[string]*flowv1beta2.RunSnapshot_InteractionNode)
		}
		m[event.GetInteraction().GetId()] = event.GetInteraction()
		snap.SetInteractions(m)
	case flowv1beta2.RunSnapshot_FlowEvent_Flow_case:
		snap.SetFlow(event.GetFlow())
	}
}
