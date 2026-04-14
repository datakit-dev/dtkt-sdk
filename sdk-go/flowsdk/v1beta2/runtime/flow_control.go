package runtime

import (
	"log/slog"

	"github.com/google/cel-go/cel"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// FlowControlAction indicates which flow-level lifecycle action to trigger.
type FlowControlAction int

const (
	FlowControlNone      FlowControlAction = iota
	FlowControlStop                        // graceful drain
	FlowControlTerminate                   // immediate cancel
	FlowControlSuspend                     // pause
)

// compiledFlowControl holds compiled CEL programs for flow_control expressions.
type compiledFlowControl struct {
	stopWhen      cel.Program
	terminateWhen cel.Program
	suspendWhen   cel.Program
}

// flowControlCallback is called by handlers after each evaluation with the
// current CEL vars. It evaluates "flow_control" programs and dispatches the
// appropriate flow-level action (Stop/Terminate/Suspend). Returns true if any
// action was triggered (the handler should continue its loop -- the flow will
// drain or pause via normal mechanisms).
type flowControlCallback func(vars map[string]any) bool

// flowControlHolder is an interface for handlers that support flow_control.
type flowControlHolder interface {
	setFlowControlCallback(fn flowControlCallback)
}

// flowControlMixin is embedded in handlers to provide flow_control support.
type flowControlMixin struct {
	onFlowControl flowControlCallback
}

func (m *flowControlMixin) setFlowControlCallback(fn flowControlCallback) {
	m.onFlowControl = fn
}

// checkFC evaluates flow control. Returns true if a flow action was triggered.
func (m *flowControlMixin) checkFC(vars map[string]any) bool {
	if m.onFlowControl == nil {
		return false
	}
	return m.onFlowControl(vars)
}

// nodeFlowControl extracts the FlowControl proto from a Node and compiles it.
// Returns nil if the node type doesn't support flow_control or has none set.
func nodeFlowControl(env shared.Env, node *flowv1beta2.Node) (*compiledFlowControl, error) {
	var fc *flowv1beta2.FlowControl
	switch node.WhichType() {
	case flowv1beta2.Node_Var_case:
		fc = node.GetVar().GetFlowControl()
	case flowv1beta2.Node_Action_case:
		fc = node.GetAction().GetFlowControl()
	case flowv1beta2.Node_Output_case:
		fc = node.GetOutput().GetFlowControl()
	case flowv1beta2.Node_Stream_case:
		fc = node.GetStream().GetFlowControl()
	case flowv1beta2.Node_Interaction_case:
		fc = node.GetInteraction().GetFlowControl()
	}
	if fc == nil {
		return nil, nil
	}
	return compileFlowControl(env, fc)
}

// compileFlowControl compiles the CEL expressions in a FlowControl proto.
// Returns nil if fc is nil or all fields are empty.
func compileFlowControl(env shared.Env, fc *flowv1beta2.FlowControl) (*compiledFlowControl, error) {
	if fc == nil {
		return nil, nil
	}
	var c compiledFlowControl
	var hasAny bool
	if s := fc.GetStopWhen(); s != "" {
		prog, err := compileCEL(env, s)
		if err != nil {
			return nil, err
		}
		c.stopWhen = prog
		hasAny = true
	}
	if s := fc.GetTerminateWhen(); s != "" {
		prog, err := compileCEL(env, s)
		if err != nil {
			return nil, err
		}
		c.terminateWhen = prog
		hasAny = true
	}
	if s := fc.GetSuspendWhen(); s != "" {
		prog, err := compileCEL(env, s)
		if err != nil {
			return nil, err
		}
		c.suspendWhen = prog
		hasAny = true
	}
	if !hasAny {
		return nil, nil
	}
	return &c, nil
}

// checkFlowControl evaluates flow control expressions against the given vars.
// Returns the highest-priority action that evaluated to true.
// Priority: terminate > suspend > stop.
func checkFlowControl(fc *compiledFlowControl, vars map[string]any) FlowControlAction {
	if fc == nil {
		return FlowControlNone
	}
	// Terminate takes highest priority.
	if fc.terminateWhen != nil {
		result, err := evalCEL(fc.terminateWhen, vars)
		if err == nil && result.Value() == true {
			return FlowControlTerminate
		}
	}
	// Suspend next.
	if fc.suspendWhen != nil {
		result, err := evalCEL(fc.suspendWhen, vars)
		if err == nil && result.Value() == true {
			return FlowControlSuspend
		}
	}
	// Stop is lowest priority.
	if fc.stopWhen != nil {
		result, err := evalCEL(fc.stopWhen, vars)
		if err == nil && result.Value() == true {
			return FlowControlStop
		}
	}
	return FlowControlNone
}

// makeFlowControlCallback creates a flowControlCallback that evaluates
// the compiled flow control and dispatches the action via the provided
// stop/terminate/suspend functions. Returns nil if fc is nil.
func makeFlowControlCallback(
	nodeID string,
	fc *compiledFlowControl,
	stopFn func(),
	terminateFn func(),
	suspendFn func(),
) flowControlCallback {
	if fc == nil {
		return nil
	}
	return func(vars map[string]any) bool {
		action := checkFlowControl(fc, vars)
		switch action {
		case FlowControlStop:
			slog.Info("flow_control: stop triggered", "node", nodeID)
			stopFn()
			return true
		case FlowControlTerminate:
			slog.Info("flow_control: terminate triggered", "node", nodeID)
			terminateFn()
			return true
		case FlowControlSuspend:
			slog.Info("flow_control: suspend triggered", "node", nodeID)
			suspendFn()
			return true
		}
		return false
	}
}
