package runtime

import (
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// NodeControl: lifecycle triggers that act on the DECLARING NODE only.
// Other nodes are unaffected. Compiles to the same internal representation
// as FlowControl (compiledLifecycleControl in flow_control.go); the
// difference is which action functions are wired to each trigger.
//
// FlowControl wires to performStop / Executor.Terminate / Executor.Suspend
// (flow-wide). NodeControl wires to Executor.StopNode(id) /
// Executor.TerminateNode(id) / Executor.SuspendNode(id) (per-node).

// compileNodeControl compiles a NodeControl proto into a lifecycle control.
// Returns nil if nc is nil or all three CEL fields are empty.
func compileNodeControl(env shared.Env, nc *flowv1beta2.NodeControl) (*compiledLifecycleControl, error) {
	if nc == nil {
		return nil, nil
	}
	return compileLifecycleControl(env, nc.GetStopWhen(), nc.GetTerminateWhen(), nc.GetSuspendWhen())
}

// nodeNodeControl extracts the NodeControl proto from a Node and compiles it.
// Returns nil if the node type doesn't support node_control or has none set.
func nodeNodeControl(env shared.Env, node *flowv1beta2.Node) (*compiledLifecycleControl, error) {
	var nc *flowv1beta2.NodeControl
	switch node.WhichType() {
	case flowv1beta2.Node_Var_case:
		nc = node.GetVar().GetNodeControl()
	case flowv1beta2.Node_Action_case:
		nc = node.GetAction().GetNodeControl()
	case flowv1beta2.Node_Output_case:
		nc = node.GetOutput().GetNodeControl()
	case flowv1beta2.Node_Stream_case:
		nc = node.GetStream().GetNodeControl()
	case flowv1beta2.Node_Interaction_case:
		nc = node.GetInteraction().GetNodeControl()
	}
	return compileNodeControl(env, nc)
}

// makeNodeControlCallback wires a NodeControl's compiled programs to the
// per-node lifecycle action functions (StopNode/TerminateNode/SuspendNode
// for the declaring node).
func makeNodeControlCallback(nodeID string, c *compiledLifecycleControl, stopFn, terminateFn, suspendFn func()) lifecycleCallback {
	return makeLifecycleCallback(nodeID, "node_control", c, lifecycleActions{
		onStop:      stopFn,
		onTerminate: terminateFn,
		onSuspend:   suspendFn,
	})
}
