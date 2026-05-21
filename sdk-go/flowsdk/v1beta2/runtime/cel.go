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

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
)

// runtimeEnv wraps a *cel.Env and shared.Resolver to implement shared.Env.
// This bridges v1beta2's raw *cel.Env with the shared SDK infrastructure that
// expects a shared.Env (e.g. shared.ExprValueToNative).
type runtimeEnv struct {
	*cel.Env
	resolver shared.Resolver
}

func (e *runtimeEnv) TypeAdapter() types.Adapter   { return e.CELTypeAdapter() }
func (e *runtimeEnv) TypeProvider() types.Provider { return e.CELTypeProvider() }
func (e *runtimeEnv) Resolver() shared.Resolver    { return e.resolver }
func (e *runtimeEnv) Vars() cel.Activation         { return nil }

// activation builds a CEL activation from subscription channel references and pre-set values.
type activation struct {
	refs   map[string]*nodeRef
	extras map[string]*expr.Value
	env    shared.Env
}

func newActivation(env shared.Env) *activation {
	return &activation{
		refs:   make(map[string]*nodeRef),
		extras: make(map[string]*expr.Value),
		env:    env,
	}
}

// newActivationFromChannelsInterruptible is the interrupt-aware variant. When
// suspendCh fires before any input arrives, recv returns errOperatorSuspended
// without consuming a message; the caller pauses, then re-creates the
// activation after resume to pick up where it left off. When stopCh fires,
// recv returns errOperatorStopped without consuming a message; the caller
// exits cleanly (no resume).
func newActivationFromChannelsInterruptible(ctx context.Context, inputs map[string]<-chan *pubsub.Message, env shared.Env, suspendCh, stopCh <-chan struct{}) *activation {
	a := newActivation(env)
	for nodeID, ch := range inputs {
		a.refs[nodeID] = &nodeRef{ch: ch, ctx: ctx, suspendCh: suspendCh, stopCh: stopCh}
	}
	return a
}

// newActivationFromMixedDeps creates an activation where cached deps
// are marked for last-seen / non-blocking-after-first recv semantics.
// All deps subscribe via pubsub the same way; the difference is the
// recv strategy on cached refs.
//
// `cachedSources` is the set of source IDs whose producer has
// cache:true. `cachedMem` provides per-source last-seen state that
// survives across activations on the same handler.
func newActivationFromMixedDeps(
	ctx context.Context,
	inputs map[string]<-chan *pubsub.Message,
	cachedSources map[string]bool,
	allCached bool,
	cachedMem map[string]*cachedRefState,
	env shared.Env,
	suspendCh, stopCh <-chan struct{},
) *activation {
	a := newActivation(env)
	for sourceID, ch := range inputs {
		ref := &nodeRef{ch: ch, ctx: ctx, suspendCh: suspendCh, stopCh: stopCh}
		if cachedSources[sourceID] {
			ref.isCached = true
			ref.allCached = allCached
			ref.lastSeen = cachedMem[sourceID]
		}
		a.refs[sourceID] = ref
	}
	return a
}

func (a *activation) SetExtra(key string, value *expr.Value) {
	a.extras[key] = value
}

// AnyEOF returns true if any resolved nodeRef has reached EOF: a
// closed pubsub channel or an EOF value. Cached refs only report EOF
// when allCached is true and the producer has terminated; mixed-deps
// cached refs ignore producer EOF and keep using lastSeen so the
// handler exits via its streaming deps.
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
			nsMap[name] = nodeToMap(a.env, nr.node)
		}
	}
	for k, v := range a.extras {
		vars[k] = exprToRefVal(a.env, v)
	}
	return vars, nil
}

// nodeToMap converts a StateNode to a map[string]any for CEL evaluation.
// The value field is unwrapped from *expr.Value to a native ref.Val.
func nodeToMap(env shared.Env, node executor.StateNode) map[string]any {
	eof := isEOFValue(node.GetValue())
	var value ref.Val
	if eof {
		value = types.NullValue
	} else {
		value = exprToRefVal(env, node.GetValue())
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
	}
}

// buildCELEnv creates the flow-global shared.Env. ONE union resolver
// (spec-ordered connectors + platform) backs both the cel-go type
// universe (CustomTypeProvider/Adapter, populated by common.NewCELTypes
// via the resolver's RangeFiles) AND runtimeEnv.resolver (consulted by
// shared.ExprValueToNative at runtime for Any unmarshalling). Per-action
// handler envs share this same resolver, so every converter/decoder in
// the run uses one explicit type universe. See flowUnionResolver
// (resolver.go) for the rationale.
//
// orderedConnectors is the list of resolved connectors in flow-spec
// declared order (caller orders via graph traversal). platform is the
// SDK's platform-types resolver - api.GlobalResolver() by default,
// overridable via Executor's WithPlatformResolver Option. If platform
// is nil, defaults to api.GlobalResolver() so call sites that pre-date
// the option still work.
func buildCELEnv(orderedConnectors []*rpc.Connector, platform shared.Resolver) (shared.Env, error) {
	if platform == nil {
		platform = api.GlobalResolver()
	}
	opts := celEnvOptions()

	flowResolver := newFlowUnionResolver(orderedConnectors, platform)

	celTypes, err := common.NewCELTypes(flowResolver)
	if err != nil {
		return nil, fmt.Errorf("building CEL types: %w", err)
	}
	// Ensure flow's own RunSnapshot proto file is registered (covered by the
	// platform resolver's RangeFiles in production - belt-and-suspenders).
	_ = celTypes.RegisterDescriptor(
		(&flowv1beta2.RunSnapshot_InputNode{}).ProtoReflect().Descriptor().ParentFile(),
	)
	opts = append(opts,
		cel.CustomTypeProvider(celTypes),
		cel.CustomTypeAdapter(celTypes),
		cel.Container("dtkt"),
	)

	env, err := common.NewCELEnv(opts...)
	if err != nil {
		return nil, err
	}
	return &runtimeEnv{Env: env, resolver: flowResolver}, nil
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

// buildLintResponseEnv creates a *cel.Env where `this.response` is typed as
// the given method's response message (instead of `dyn`). The runtime binds
// `this` as `{"response": <msg>}` (a map) so the env mirrors that shape:
// `this: map<string, md.Output()>`. With that typing, CEL's own Check phase
// rejects unknown field accesses on `this.response.<chain>`. Only the
// resolver for this specific connection is consulted; the env is not shared
// across calls because each call has its own response type.
//
// The slight overpermission `this.bogus` (also typed as md.Output()) is
// accepted: the runtime only binds the "response" key, so any other key
// would be nil at eval time. A future tighter check could declare a
// synthetic proto wrapper, but the map shape is simpler and matches the
// runtime binding exactly.
func buildLintResponseEnv(resolver shared.Resolver, md protoreflect.MethodDescriptor) (*cel.Env, error) {
	outDesc := md.Output()
	thisType := cel.MapType(cel.StringType, cel.ObjectType(string(outDesc.FullName())))

	// Mirror celEnvOptions, but override `this` to the typed response message.
	// The other variables stay `map(string, dyn)` because lint expressions can
	// still reference `inputs.x.value` etc. inside a response expr.
	mapType := cel.MapType(cel.StringType, cel.DynType)
	opts := []cel.EnvOption{
		cel.Variable("this", thisType),
		cel.Variable("inputs", mapType),
		cel.Variable("generators", mapType),
		cel.Variable("vars", mapType),
		cel.Variable("actions", mapType),
		cel.Variable("streams", mapType),
		cel.Variable("interactions", mapType),
		cel.Variable("outputs", mapType),
		cel.Variable("connections", mapType),
		cel.Function("now",
			cel.SingletonFunctionBinding(func(...ref.Val) ref.Val {
				return types.Timestamp{Time: time.Now()}
			}),
			cel.Overload("now", nil, cel.TimestampType),
		),
	}

	// Register the connector's proto files so CEL knows about every type
	// reachable from `this.response.*` (nested messages, enums, etc.).
	var fileDescs []any
	resolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		fileDescs = append(fileDescs, fd)
		return true
	})
	if len(fileDescs) > 0 {
		opts = append(opts, cel.TypeDescs(fileDescs...), cel.Container("dtkt"))
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
