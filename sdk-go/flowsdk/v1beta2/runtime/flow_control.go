package runtime

import (
	"log/slog"

	"github.com/google/cel-go/cel"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// -----------------------------------------------------------------------
// Lifecycle control core (shared by FlowControl and NodeControl)
//
// FlowControl and NodeControl are structurally identical: each declares
// stop_when / terminate_when / suspend_when CEL expressions. They differ
// only in scope (whole flow vs. one node). Both compile to the same
// internal representation; only the action functions wired to each
// expression differ.
// -----------------------------------------------------------------------

// LifecycleAction is the action selected by a lifecycle-control evaluation.
type LifecycleAction int

const (
	LifecycleNone LifecycleAction = iota
	LifecycleStop
	LifecycleTerminate
	LifecycleSuspend
)

// compiledLifecycleControl holds the compiled CEL programs for a
// stop_when / terminate_when / suspend_when triple. Either FlowControl or
// NodeControl proto messages compile to this shape.
type compiledLifecycleControl struct {
	stopWhen      cel.Program
	terminateWhen cel.Program
	suspendWhen   cel.Program
}

// compileLifecycleControl compiles three CEL strings into programs.
// Returns nil if all three are empty.
func compileLifecycleControl(env shared.Env, stopWhen, terminateWhen, suspendWhen string) (*compiledLifecycleControl, error) {
	var c compiledLifecycleControl
	var hasAny bool
	if s := stopWhen; s != "" {
		prog, err := compileCEL(env, s)
		if err != nil {
			return nil, err
		}
		c.stopWhen = prog
		hasAny = true
	}
	if s := terminateWhen; s != "" {
		prog, err := compileCEL(env, s)
		if err != nil {
			return nil, err
		}
		c.terminateWhen = prog
		hasAny = true
	}
	if s := suspendWhen; s != "" {
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

// checkLifecycleControl evaluates the compiled programs and returns the
// highest-priority action that fired. Priority: terminate > suspend > stop.
func checkLifecycleControl(c *compiledLifecycleControl, vars map[string]any) LifecycleAction {
	if c == nil {
		return LifecycleNone
	}
	if c.terminateWhen != nil {
		result, err := evalCEL(c.terminateWhen, vars)
		if err == nil && result.Value() == true {
			return LifecycleTerminate
		}
	}
	if c.suspendWhen != nil {
		result, err := evalCEL(c.suspendWhen, vars)
		if err == nil && result.Value() == true {
			return LifecycleSuspend
		}
	}
	if c.stopWhen != nil {
		result, err := evalCEL(c.stopWhen, vars)
		if err == nil && result.Value() == true {
			return LifecycleStop
		}
	}
	return LifecycleNone
}

// lifecycleActions binds a compiledLifecycleControl to runtime action
// functions. FlowControl wires these to flow-wide stop/terminate/suspend.
// NodeControl wires them to per-node StopNode/TerminateNode/SuspendNode.
type lifecycleActions struct {
	onStop      func()
	onTerminate func()
	onSuspend   func()
}

// lifecycleCallback is invoked by handlers after each evaluation with the
// current CEL vars. Returns the LifecycleAction that fired (LifecycleNone
// if nothing matched). The callback also dispatches the action's side
// effects (e.g. calling Executor.StopNode) before returning.
type lifecycleCallback func(vars map[string]any) LifecycleAction

// makeLifecycleCallback wraps a compiled lifecycle control and its action
// functions into a callback the handler invokes per iteration. `kind` is
// "flow_control" or "node_control" -- used in the log line so observers
// can tell which trigger fired.
//
// The callback both fires side effects AND returns the action so the
// handler can branch on it (e.g. retry-suspend escalation breaks the loop
// when stop fired, instead of calling selfSuspend).
func makeLifecycleCallback(nodeID, kind string, c *compiledLifecycleControl, a lifecycleActions) lifecycleCallback {
	if c == nil {
		return nil
	}
	return func(vars map[string]any) LifecycleAction {
		action := checkLifecycleControl(c, vars)
		switch action {
		case LifecycleStop:
			slog.Info(kind+": stop triggered", slog.String("node", nodeID))
			a.onStop()
		case LifecycleTerminate:
			slog.Info(kind+": terminate triggered", slog.String("node", nodeID))
			a.onTerminate()
		case LifecycleSuspend:
			slog.Info(kind+": suspend triggered", slog.String("node", nodeID))
			a.onSuspend()
		}
		return action
	}
}

// -----------------------------------------------------------------------
// FlowControl: lifecycle triggers that act on the WHOLE flow.
// -----------------------------------------------------------------------

// compileFlowControl compiles a FlowControl proto into a lifecycle control.
// Returns nil if fc is nil or all three CEL fields are empty.
func compileFlowControl(env shared.Env, fc *flowv1beta2.FlowControl) (*compiledLifecycleControl, error) {
	if fc == nil {
		return nil, nil
	}
	return compileLifecycleControl(env, fc.GetStopWhen(), fc.GetTerminateWhen(), fc.GetSuspendWhen())
}

// nodeFlowControl extracts the FlowControl proto from a Node and compiles it.
// Returns nil if the node type doesn't support flow_control or has none set.
func nodeFlowControl(env shared.Env, node *flowv1beta2.Node) (*compiledLifecycleControl, error) {
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
	return compileFlowControl(env, fc)
}

// makeFlowControlCallback wires a FlowControl's compiled programs to the
// flow-wide lifecycle action functions.
func makeFlowControlCallback(nodeID string, c *compiledLifecycleControl, stopFn, terminateFn, suspendFn func()) lifecycleCallback {
	return makeLifecycleCallback(nodeID, "flow_control", c, lifecycleActions{
		onStop:      stopFn,
		onTerminate: terminateFn,
		onSuspend:   suspendFn,
	})
}

// -----------------------------------------------------------------------
// Handler-side mixin: stores both FlowControl and NodeControl callbacks.
// Each iteration, handlers invoke checkLifecycle which runs both.
// -----------------------------------------------------------------------

// lifecycleMixin gives a handler one callback slot for FlowControl and one
// for NodeControl. Each iteration the handler calls checkLifecycle, which
// runs whichever callbacks are installed -- order is NodeControl then
// FlowControl, both fire independently. A node can have both controls
// active and trigger different actions (e.g. flow_control.stop_when fires
// flow drain while node_control.terminate_when cancels just this node).
type lifecycleMixin struct {
	onFlowControl lifecycleCallback
	onNodeControl lifecycleCallback
}

// lifecycleHolder is the interface that executor_setup uses to install
// FlowControl and NodeControl callbacks on a handler. Every handler that
// supports either control implements it via lifecycleMixin.
type lifecycleHolder interface {
	setFlowControlCallback(fn lifecycleCallback)
	setNodeControlCallback(fn lifecycleCallback)
}

func (m *lifecycleMixin) setFlowControlCallback(fn lifecycleCallback) { m.onFlowControl = fn }
func (m *lifecycleMixin) setNodeControlCallback(fn lifecycleCallback) { m.onNodeControl = fn }

// checkLifecycle evaluates node_control first, then flow_control. Returns
// the action each control selected (LifecycleNone if not configured or
// no _when matched). Each callback also fires its action's side effects
// (e.g. SuspendNode, performStop) before returning.
//
// Order matters: NC runs first because it is narrower in scope (a single
// node) and its state events (e.g. PHASE_STOPPING for the controlled node)
// must land cleanly before any FC-driven flow-level cancel races with
// them. Concrete failure mode if FC ran first: FC.terminate cancels
// runCtx, the handler's publish channel starts tearing down, and a
// subsequent NC.publishPhaseChange(PHASE_STOPPING) on the same handler
// races against the dying state. Reversing the order keeps the per-node
// state event ahead of any flow-wide cancel.
//
// Both callbacks always run (they're independent triggers); ordering
// only affects publish ordering on the wire when both fire the same
// iteration.
func (m *lifecycleMixin) checkLifecycle(vars map[string]any) (nc, fc LifecycleAction) {
	if m.onNodeControl != nil {
		nc = m.onNodeControl(vars)
	}
	if m.onFlowControl != nil {
		fc = m.onFlowControl(vars)
	}
	return
}
