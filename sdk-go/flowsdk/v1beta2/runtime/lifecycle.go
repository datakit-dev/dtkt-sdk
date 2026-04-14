package runtime

import (
	expr "cel.dev/expr"
	"google.golang.org/genproto/googleapis/rpc/status"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// nodeFactory builds StateNode protos for terminal and state-change publishing.
type nodeFactory struct {
	// buildTerminal creates a StateNode for terminal phases (with EOF value).
	buildTerminal func(id string, phase flowv1beta2.RunSnapshot_Phase, val *expr.Value, err *status.Status) executor.StateNode
	// buildState creates a StateNode for phase-change-only events.
	buildState func(id string, phase flowv1beta2.RunSnapshot_Phase, err *status.Status) executor.StateNode
}

// nodeFactoryMap builds the factory table lazily on first use, keyed by the
// return value of Node.WhichType() (an unexported int-like enum).
func nodeFactoryMap() map[any]nodeFactory {
	return map[any]nodeFactory{
		flowv1beta2.Node_Action_case: {
			buildTerminal: func(id string, phase flowv1beta2.RunSnapshot_Phase, val *expr.Value, err *status.Status) executor.StateNode {
				return flowv1beta2.RunSnapshot_ActionNode_builder{Id: id, Value: val, Phase: phase, Error: err}.Build()
			},
			buildState: func(id string, phase flowv1beta2.RunSnapshot_Phase, err *status.Status) executor.StateNode {
				return flowv1beta2.RunSnapshot_ActionNode_builder{Id: id, Phase: phase, Error: err}.Build()
			},
		},
		flowv1beta2.Node_Stream_case: {
			buildTerminal: func(id string, phase flowv1beta2.RunSnapshot_Phase, val *expr.Value, err *status.Status) executor.StateNode {
				return flowv1beta2.RunSnapshot_StreamNode_builder{Id: id, Value: val, ResponseClosed: true, Phase: phase, Error: err}.Build()
			},
			buildState: func(id string, phase flowv1beta2.RunSnapshot_Phase, err *status.Status) executor.StateNode {
				return flowv1beta2.RunSnapshot_StreamNode_builder{Id: id, Phase: phase, Error: err}.Build()
			},
		},
		flowv1beta2.Node_Var_case: {
			buildTerminal: func(id string, phase flowv1beta2.RunSnapshot_Phase, val *expr.Value, err *status.Status) executor.StateNode {
				return flowv1beta2.RunSnapshot_VarNode_builder{Id: id, Value: val, Phase: phase, Error: err}.Build()
			},
			buildState: func(id string, phase flowv1beta2.RunSnapshot_Phase, err *status.Status) executor.StateNode {
				return flowv1beta2.RunSnapshot_VarNode_builder{Id: id, Phase: phase, Error: err}.Build()
			},
		},
		flowv1beta2.Node_Generator_case: {
			buildTerminal: func(id string, phase flowv1beta2.RunSnapshot_Phase, val *expr.Value, err *status.Status) executor.StateNode {
				return flowv1beta2.RunSnapshot_GeneratorNode_builder{Id: id, Value: val, Done: true, Phase: phase, Error: err}.Build()
			},
			buildState: func(id string, phase flowv1beta2.RunSnapshot_Phase, err *status.Status) executor.StateNode {
				return flowv1beta2.RunSnapshot_GeneratorNode_builder{Id: id, Phase: phase, Error: err}.Build()
			},
		},
		flowv1beta2.Node_Output_case: {
			buildTerminal: func(id string, phase flowv1beta2.RunSnapshot_Phase, val *expr.Value, err *status.Status) executor.StateNode {
				return flowv1beta2.RunSnapshot_OutputNode_builder{Id: id, Value: val, Phase: phase, Error: err}.Build()
			},
			buildState: func(id string, phase flowv1beta2.RunSnapshot_Phase, err *status.Status) executor.StateNode {
				return flowv1beta2.RunSnapshot_OutputNode_builder{Id: id, Phase: phase, Error: err}.Build()
			},
		},
		flowv1beta2.Node_Interaction_case: {
			buildTerminal: func(id string, phase flowv1beta2.RunSnapshot_Phase, val *expr.Value, err *status.Status) executor.StateNode {
				return flowv1beta2.RunSnapshot_InteractionNode_builder{Id: id, Value: val, Phase: phase, Error: err}.Build()
			},
			buildState: func(id string, phase flowv1beta2.RunSnapshot_Phase, err *status.Status) executor.StateNode {
				return flowv1beta2.RunSnapshot_InteractionNode_builder{Id: id, Phase: phase, Error: err}.Build()
			},
		},
	}
}

// publishTerminalPhase publishes an ERRORED phase with an EOF value and error
// status for the given node. Downstream handlers will see the EOF and drain.
func publishTerminalPhase(pub pubsub.Publisher, topic string, node *flowv1beta2.Node, phase flowv1beta2.RunSnapshot_Phase, err error) error {
	f, ok := nodeFactoryMap()[node.WhichType()]
	if !ok {
		return nil
	}
	return publishNode(pub, topic, f.buildTerminal(node.GetId(), phase, newEOFValue(), grpcStatusProto(err)))
}

// publishPhaseChange publishes a phase transition as a STATE event (no EOF,
// no value). Downstream handlers skip STATE events, so this does not affect
// the data pipeline. Phase observers (monitoring, tests) see the transition.
func publishPhaseChange(pub pubsub.Publisher, topic string, node *flowv1beta2.Node, phase flowv1beta2.RunSnapshot_Phase, err error) error {
	f, ok := nodeFactoryMap()[node.WhichType()]
	if !ok {
		return nil
	}
	return publishStateEvent(pub, topic, f.buildState(node.GetId(), phase, grpcStatusProto(err)))
}

// isGenerator returns true for node types that produce values without upstream inputs.
func isGenerator(node *flowv1beta2.Node) bool {
	return node.WhichType() == flowv1beta2.Node_Generator_case
}
