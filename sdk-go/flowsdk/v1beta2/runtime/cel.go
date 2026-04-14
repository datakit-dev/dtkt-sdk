package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// runtimeEnv wraps a *cel.Env and shared.Resolver to implement shared.Env.
// This bridges v1beta2's raw *cel.Env with the shared SDK infrastructure that
// expects a shared.Env (e.g. shared.ExprValueToNative).
type runtimeEnv struct {
	*cel.Env
	resolver shared.Resolver
}

func (e *runtimeEnv) TypeAdapter() types.Adapter   { return e.Env.CELTypeAdapter() }
func (e *runtimeEnv) TypeProvider() types.Provider { return e.Env.CELTypeProvider() }
func (e *runtimeEnv) Resolver() shared.Resolver    { return e.resolver }
func (e *runtimeEnv) Vars() cel.Activation         { return nil }

// nodeRef holds a reference to a subscription channel. recv() reads from the
// channel, blocking until a message arrives. The result is cached so that
// multiple accesses return the same value.
type nodeRef struct {
	ch         <-chan *pubsub.Message
	ctx        context.Context
	node       executor.StateNode
	chanClosed bool // channel itself was closed (not EOF value)
	consumed   bool
}

func (nr *nodeRef) recv() error {
	if nr.consumed {
		return nil
	}
	if nr.ch == nil {
		return nil
	}
	for {
		select {
		case <-nr.ctx.Done():
			return nr.ctx.Err()
		case msg, ok := <-nr.ch:
			if !ok {
				nr.chanClosed = true
				nr.consumed = true
				return nil
			}
			event := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			if event.GetEventType() == flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_UPDATE {
				msg.Ack()
				continue // skip NODE_UPDATE events; wait for the next NODE_OUTPUT
			}
			msg.Ack()
			nr.node = runtimeNodeFromEvent(event)
			nr.consumed = true
			return nil
		}
	}
}

// activation builds a CEL activation from subscription channel references and pre-set values.
type activation struct {
	refs    map[string]*nodeRef
	extras  map[string]*expr.Value
	adapter types.Adapter
}

func newActivation(adapter types.Adapter) *activation {
	return &activation{
		refs:    make(map[string]*nodeRef),
		extras:  make(map[string]*expr.Value),
		adapter: adapter,
	}
}

func newActivationFromChannels(ctx context.Context, inputs map[string]<-chan *pubsub.Message, adapter types.Adapter) *activation {
	a := newActivation(adapter)
	for nodeID, ch := range inputs {
		a.refs[nodeID] = &nodeRef{ch: ch, ctx: ctx}
	}
	return a
}

func (a *activation) SetExtra(key string, value *expr.Value) {
	a.extras[key] = value
}

// AnyEOF returns true if any resolved nodeRef received an EOF value or had its channel closed.
func (a *activation) AnyEOF() bool {
	for _, nr := range a.refs {
		if !nr.consumed {
			continue
		}
		if nr.chanClosed {
			return true
		}
		if nr.node != nil && isEOFValue(nr.node.GetValue()) {
			return true
		}
	}
	return false
}

// FirstInputValue returns the *expr.Value from the first resolved nodeRef
// that has a non-nil node. Used by action handlers to extract the input value
// for the method call.
func (a *activation) FirstInputValue() *expr.Value {
	for _, nr := range a.refs {
		if nr.consumed && nr.node != nil {
			return nr.node.GetValue()
		}
	}
	return nil
}

// Resolve eagerly receives from all subscription channels and builds the CEL
// variable map. Each node is placed directly into its namespace map as a typed
// proto message, enabling CEL to access fields natively (e.g. inputs.x.value,
// vars.sum.eval_count, streams.echo.response_closed).
func (a *activation) Resolve() (map[string]any, error) {
	vars := make(map[string]any)
	for nodeID, nr := range a.refs {
		if err := nr.recv(); err != nil {
			return nil, err
		}
		parts := strings.SplitN(nodeID, ".", 2)
		if len(parts) != 2 {
			continue
		}
		namespace, name := parts[0], parts[1]
		nsMap, ok := vars[namespace].(map[string]any)
		if !ok {
			nsMap = make(map[string]any)
			vars[namespace] = nsMap
		}
		if nr.chanClosed || nr.node == nil {
			nsMap[name] = types.NullValue
		} else {
			nsMap[name] = nodeToMap(a.adapter, nr.node)
		}
	}
	for k, v := range a.extras {
		vars[k] = exprToRefVal(a.adapter, v)
	}
	return vars, nil
}

// nodeToMap converts a StateNode to a map[string]any for CEL evaluation.
// The value field is unwrapped from *expr.Value to a native ref.Val.
func nodeToMap(adapter types.Adapter, node executor.StateNode) map[string]any {
	eof := isEOFValue(node.GetValue())
	var value ref.Val
	if eof {
		value = types.NullValue
	} else {
		value = exprToRefVal(adapter, node.GetValue())
	}

	m := map[string]any{"value": value}
	switch n := node.(type) {
	case *flowv1beta2.RunSnapshot_InputNode:
		m["closed"] = n.GetClosed()
	case *flowv1beta2.RunSnapshot_GeneratorNode:
		m["done"] = n.GetDone()
		m["eval_count"] = int64(n.GetEvalCount())
	case *flowv1beta2.RunSnapshot_VarNode:
		m["eval_count"] = int64(n.GetEvalCount())
	case *flowv1beta2.RunSnapshot_ActionNode:
		m["eval_count"] = int64(n.GetEvalCount())
	case *flowv1beta2.RunSnapshot_StreamNode:
		m["request_closed"] = n.GetRequestClosed()
		m["response_closed"] = n.GetResponseClosed()
		m["request_count"] = int64(n.GetRequestCount())
		m["response_count"] = int64(n.GetResponseCount())
	case *flowv1beta2.RunSnapshot_OutputNode:
		m["eval_count"] = int64(n.GetEvalCount())
	case *flowv1beta2.RunSnapshot_InteractionNode:
		m["submitted"] = n.GetSubmitted()
	default:
		m["closed"] = eof
	}
	return m
}

// celEnvOptions returns the shared CEL environment options used by both
// parseCEL and compileCEL. These are the base options without connection types.
func celEnvOptions() []cel.EnvOption {
	mapType := cel.MapType(cel.StringType, cel.DynType)
	return []cel.EnvOption{
		cel.Variable("this", cel.DynType),
		cel.Variable("inputs", mapType),
		cel.Variable("generators", mapType),
		cel.Variable("vars", mapType),
		cel.Variable("actions", mapType),
		cel.Variable("streams", mapType),
		cel.Variable("interactions", mapType),
		cel.Variable("outputs", mapType),
		cel.Variable("connections", mapType),
		cel.Function("now",
			cel.SingletonFunctionBinding(
				func(...ref.Val) ref.Val {
					return types.Timestamp{
						Time: time.Now(),
					}
				},
			),
			cel.Overload("now", nil, cel.TimestampType),
		),
		cel.Function("EOF",
			cel.Overload("eof_zero",
				[]*cel.Type{},
				cel.DynType,
				cel.FunctionBinding(func(values ...ref.Val) ref.Val {
					return eofRefValInstance
				}),
			),
		),
	}
}

// buildCELEnv creates a shared.Env with connection resolver types registered.
// When connectors have CELResolver implementations, their proto file descriptors
// are registered via common.CELTypes for full struct/field type resolution.
// Falls back to basic cel.Types registration when no resolvers are available.
// Uses common.NewCELEnv to include the standard extension set (URL validation,
// encoders, string v4, list, proto, enum).
func buildCELEnv(connectors map[string]*rpc.Connector) (shared.Env, error) {
	opts := celEnvOptions()

	// Collect the first available resolver for the runtimeEnv wrapper.
	var firstResolver shared.Resolver
	var celTypes *common.CELTypes
	for _, conn := range connectors {
		if firstResolver == nil {
			firstResolver = conn.Resolver
		}
		cr, ok := conn.Resolver.(common.CELResolver)
		if !ok {
			continue
		}
		if celTypes == nil {
			var err error
			celTypes, err = common.NewCELTypes(cr)
			if err != nil {
				return nil, fmt.Errorf("building CEL types: %w", err)
			}
		} else {
			cr.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
				_ = celTypes.RegisterDescriptor(fd)
				return true
			})
		}
	}

	if celTypes != nil {
		// Register our own flow proto types so RunSnapshot nodes are known.
		_ = celTypes.RegisterDescriptor(
			(&flowv1beta2.RunSnapshot_InputNode{}).ProtoReflect().Descriptor().ParentFile(),
		)
		opts = append(opts,
			cel.CustomTypeProvider(celTypes),
			cel.CustomTypeAdapter(celTypes),
			cel.Container("dtkt"),
		)
	} else {
		// No connection resolvers; register only built-in types.
		opts = append(opts,
			cel.Types(
				&flowv1beta2.RunSnapshot_InputNode{},
				&flowv1beta2.RunSnapshot_GeneratorNode{},
				&flowv1beta2.RunSnapshot_VarNode{},
				&flowv1beta2.RunSnapshot_ActionNode{},
				&flowv1beta2.RunSnapshot_StreamNode{},
				&flowv1beta2.RunSnapshot_OutputNode{},
				&flowv1beta2.RunSnapshot_InteractionNode{},
			),
		)
	}

	env, err := common.NewCELEnv(opts...)
	if err != nil {
		return nil, err
	}
	return &runtimeEnv{Env: env, resolver: firstResolver}, nil
}

// parseCEL validates a CEL expression string (syntax + type check) without
// producing an executable program. Used by Lint for fast validation.
func parseCEL(expression string) (*cel.Ast, error) {
	src, ok := shared.IsValidExpr(expression)
	if !ok {
		return nil, fmt.Errorf("invalid CEL expression: %q", expression)
	}

	env, err := common.NewCELEnv(append(celEnvOptions(),
		cel.Types(
			&flowv1beta2.RunSnapshot_InputNode{},
			&flowv1beta2.RunSnapshot_GeneratorNode{},
			&flowv1beta2.RunSnapshot_VarNode{},
			&flowv1beta2.RunSnapshot_ActionNode{},
			&flowv1beta2.RunSnapshot_StreamNode{},
			&flowv1beta2.RunSnapshot_OutputNode{},
			&flowv1beta2.RunSnapshot_InteractionNode{},
		),
	)...)
	if err != nil {
		return nil, fmt.Errorf("creating CEL env: %w", err)
	}

	ast, issues := env.Compile(src)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("compiling CEL expression %q: %w", src, issues.Err())
	}

	return ast, nil
}

// compileCEL compiles a CEL expression string into an executable program.
// The provided env must be built with buildCELEnv to include connection types.
func compileCEL(env shared.Env, expression string) (cel.Program, error) {
	src, ok := shared.IsValidExpr(expression)
	if !ok {
		return nil, fmt.Errorf("invalid CEL expression: %q", expression)
	}

	ast, issues := env.Compile(src)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("compiling CEL expression %q: %w", src, issues.Err())
	}

	prog, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("creating CEL program for %q: %w", src, err)
	}

	return prog, nil
}

// buildLintCELEnv creates a *cel.Env with resolver types registered for use
// during linting. Connection proto types are registered via cel.TypeDescs so
// that CEL expressions referencing those types can be fully type-checked.
func buildLintCELEnv(resolvers map[string]shared.Resolver) (*cel.Env, error) {
	opts := celEnvOptions()

	var fileDescs []any
	for _, resolver := range resolvers {
		resolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
			fileDescs = append(fileDescs, fd)
			return true
		})
	}

	if len(fileDescs) > 0 {
		// Also register flow proto types so RunSnapshot nodes are known.
		fileDescs = append(fileDescs,
			(&flowv1beta2.RunSnapshot_InputNode{}).ProtoReflect().Descriptor().ParentFile(),
		)
		opts = append(opts,
			cel.TypeDescs(fileDescs...),
			cel.Container("dtkt"),
		)
	} else {
		opts = append(opts,
			cel.Types(
				&flowv1beta2.RunSnapshot_InputNode{},
				&flowv1beta2.RunSnapshot_GeneratorNode{},
				&flowv1beta2.RunSnapshot_VarNode{},
				&flowv1beta2.RunSnapshot_ActionNode{},
				&flowv1beta2.RunSnapshot_StreamNode{},
				&flowv1beta2.RunSnapshot_OutputNode{},
				&flowv1beta2.RunSnapshot_InteractionNode{},
			),
		)
	}

	return common.NewCELEnv(opts...)
}

// checkCELOutputType compiles a CEL expression using the provided environment
// and returns its checked output type. Returns nil if compilation fails or
// the type is dynamic/unresolvable (i.e. type checking cannot be performed).
func checkCELOutputType(env shared.Env, expression string) *cel.Type {
	src, ok := shared.IsValidExpr(expression)
	if !ok {
		return nil
	}
	ast, issues := env.Compile(src)
	if issues != nil && issues.Err() != nil {
		return nil
	}
	outType := ast.OutputType()
	if outType == nil {
		return nil
	}
	switch outType.Kind() {
	case cel.DynKind, cel.TypeParamKind:
		return nil
	}
	return outType
}

// evalCEL runs a compiled CEL program with the given activation.
func evalCEL(prog cel.Program, vars map[string]any) (ref.Val, error) {
	out, _, err := prog.Eval(vars)
	if err != nil {
		return nil, err
	}
	return out, nil
}
