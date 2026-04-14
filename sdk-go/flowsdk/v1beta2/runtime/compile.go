package runtime

import (
	"fmt"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/robfig/cron/v3"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/cache"
	cachememory "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/cache/memory"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// Compiled node types hold pre-compiled CEL programs, request trees, retry
// strategies, and resolved connection references. Produced by compileNode,
// consumed by newHandler to construct runnable NodeHandlers.

type compiledVarValue struct {
	program    cel.Program
	transforms *transformPipeline
}

type compiledVarSwitch struct {
	valueProg   cel.Program
	cases       []switchCase
	defaultProg cel.Program
	transforms  *transformPipeline
}

type compiledTicker struct {
	interval     time.Duration
	delay        time.Duration
	valueProgram cel.Program
}

type compiledRange struct {
	start int64
	end   int64
	step  int64
	rate  *flowv1beta2.Rate
}

type compiledCron struct {
	schedule     cron.Schedule
	valueProgram cel.Program
}

type compiledCall struct {
	method               protoreflect.FullName
	kind                 rpc.MethodKind
	client               rpc.Client
	env                  shared.Env // connection-specific env
	whenProg             cel.Program
	closeRequestWhenProg cel.Program // nil for actions
	throttle             time.Duration
	request              *compiledRequest
	responseProg         cel.Program
	retry                *compiledRetryStrategy
	cache                cache.Cache // nil unless action with memoize
}

type compiledOutput struct {
	program    cel.Program
	transforms *transformPipeline
	throttle   time.Duration
}

// connRuntimeEnv creates a per-connection shared.Env by combining the compiled
// *cel.Env from the graph-level shared.Env with a connection-specific resolver.
func connRuntimeEnv(graphEnv shared.Env, resolver shared.Resolver) shared.Env {
	re := graphEnv.(*runtimeEnv)
	return &runtimeEnv{Env: re.Env, resolver: resolver}
}

// rateToDuration converts a proto Rate to a time.Duration per event.
// Returns 0 if the rate is nil or invalid.
func rateToDuration(rate *flowv1beta2.Rate) time.Duration {
	if rate == nil || !rate.GetInterval().IsValid() || rate.GetCount() == 0 {
		return 0
	}
	return rate.GetInterval().AsDuration() / time.Duration(rate.GetCount())
}

// lookupMethodKind resolves a method name via the Resolver and returns
// the corresponding MethodKind. Returns false if the method is not found.
func lookupMethodKind(resolver shared.Resolver, name protoreflect.FullName) (rpc.MethodKind, bool) {
	md, err := resolver.FindMethodByName(name)
	if err != nil {
		return 0, false
	}
	switch {
	case md.IsStreamingClient() && md.IsStreamingServer():
		return rpc.MethodBidiStream, true
	case md.IsStreamingClient():
		return rpc.MethodClientStream, true
	case md.IsStreamingServer():
		return rpc.MethodServerStream, true
	default:
		return rpc.MethodUnary, true
	}
}

// compileNode compiles a node definition into a compiled struct that holds all
// pre-compiled CEL programs, request trees, retry strategies, and resolved
// connection references. The returned value is one of the concrete compiled*
// types above. Returns an error for invalid CEL, unknown connections, or
// unsupported node types.
func compileNode(env shared.Env, node *flowv1beta2.Node, connectors map[string]*rpc.Connector, nodeCache cache.Cache) (any, error) {
	switch node.WhichType() {
	case flowv1beta2.Node_Var_case:
		return compileVar(env, node)
	case flowv1beta2.Node_Generator_case:
		return compileGenerator(env, node)
	case flowv1beta2.Node_Stream_case:
		return compileStream(env, node, connectors)
	case flowv1beta2.Node_Action_case:
		return compileAction(env, node, connectors, nodeCache)
	case flowv1beta2.Node_Output_case:
		return compileOutput(env, node)
	default:
		return nil, fmt.Errorf("unsupported node type for node %s", node.GetId())
	}
}

func compileVar(env shared.Env, node *flowv1beta2.Node) (any, error) {
	v := node.GetVar()
	transforms, err := compileTransforms(env, v.GetTransforms())
	if err != nil {
		return nil, fmt.Errorf("compiling transforms for %s: %w", node.GetId(), err)
	}

	switch v.WhichType() {
	case flowv1beta2.Var_Switch_case:
		sw := v.GetSwitch()
		valueProg, err := compileCEL(env, sw.GetValue())
		if err != nil {
			return nil, fmt.Errorf("compiling switch value CEL for %s: %w", node.GetId(), err)
		}
		var cases []switchCase
		for i, c := range sw.GetCase() {
			condProg, err := compileCEL(env, c.GetValue())
			if err != nil {
				return nil, fmt.Errorf("compiling switch case[%d] condition for %s: %w", i, node.GetId(), err)
			}
			retProg, err := compileCEL(env, c.GetReturn())
			if err != nil {
				return nil, fmt.Errorf("compiling switch case[%d] return for %s: %w", i, node.GetId(), err)
			}
			cases = append(cases, switchCase{condition: condProg, result: retProg})
		}
		defaultProg, err := compileCEL(env, sw.GetDefault())
		if err != nil {
			return nil, fmt.Errorf("compiling switch default CEL for %s: %w", node.GetId(), err)
		}
		return &compiledVarSwitch{
			valueProg:   valueProg,
			cases:       cases,
			defaultProg: defaultProg,
			transforms:  transforms,
		}, nil

	default:
		prog, err := compileCEL(env, v.GetValue())
		if err != nil {
			return nil, fmt.Errorf("compiling var CEL for %s: %w", node.GetId(), err)
		}
		return &compiledVarValue{
			program:    prog,
			transforms: transforms,
		}, nil
	}
}

func compileGenerator(env shared.Env, node *flowv1beta2.Node) (any, error) {
	gen := node.GetGenerator()

	switch gen.WhichType() {
	case flowv1beta2.Generator_Ticker_case:
		ticker := gen.GetTicker()
		var valueProg cel.Program
		var err error
		if ticker.GetValue() != "" {
			valueProg, err = compileCEL(env, ticker.GetValue())
			if err != nil {
				return nil, fmt.Errorf("compiling ticker CEL for %s: %w", node.GetId(), err)
			}
		}
		return &compiledTicker{
			interval:     ticker.GetInterval().AsDuration(),
			delay:        ticker.GetDelay().AsDuration(),
			valueProgram: valueProg,
		}, nil

	case flowv1beta2.Generator_Range_case:
		r := gen.GetRange()
		return &compiledRange{
			start: r.GetStart(),
			end:   r.GetEnd(),
			step:  r.GetStep(),
			rate:  r.GetRate(),
		}, nil

	case flowv1beta2.Generator_Cron_case:
		c := gen.GetCron()
		schedule, err := cron.ParseStandard(c.GetExpression())
		if err != nil {
			return nil, fmt.Errorf("parsing cron expression for %s: %w", node.GetId(), err)
		}
		var valueProg cel.Program
		if c.GetValue() != "" {
			valueProg, err = compileCEL(env, c.GetValue())
			if err != nil {
				return nil, fmt.Errorf("compiling cron CEL for %s: %w", node.GetId(), err)
			}
		}
		return &compiledCron{
			schedule:     schedule,
			valueProgram: valueProg,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported generator type for node %s", node.GetId())
	}
}

func compileStream(env shared.Env, node *flowv1beta2.Node, connectors map[string]*rpc.Connector) (any, error) {
	stream := node.GetStream()
	var whenProg, closeReqWhenProg cel.Program
	if w := stream.GetWhen(); w != "" {
		var err error
		whenProg, err = compileCEL(env, w)
		if err != nil {
			return nil, fmt.Errorf("compiling stream when CEL for %s: %w", node.GetId(), err)
		}
	}
	if crw := stream.GetCloseRequestWhen(); crw != "" {
		var err error
		closeReqWhenProg, err = compileCEL(env, crw)
		if err != nil {
			return nil, fmt.Errorf("compiling stream close_request_when CEL for %s: %w", node.GetId(), err)
		}
	}
	retry, err := compileRetryStrategy(env, stream.GetRetryStrategy())
	if err != nil {
		return nil, fmt.Errorf("compiling retry strategy for %s: %w", node.GetId(), err)
	}

	switch stream.WhichType() {
	case flowv1beta2.Stream_Call_case:
		call := stream.GetCall()
		connID := call.GetConnection()
		conn := connectors[connID]
		if conn == nil {
			return nil, fmt.Errorf("no connector for connection %q on stream call node %s", connID, node.GetId())
		}
		methodName := protoreflect.FullName(call.GetMethod())
		kind, ok := lookupMethodKind(conn.Resolver, methodName)
		if !ok {
			return nil, fmt.Errorf("method %q not found in connection %q for node %s", call.GetMethod(), connID, node.GetId())
		}
		var reqTree *compiledRequest
		if call.GetRequest() != nil {
			reqTree, err = compileRequestTree(env, call.GetRequest())
			if err != nil {
				return nil, fmt.Errorf("compiling request tree for %s: %w", node.GetId(), err)
			}
		}
		var respProg cel.Program
		if resp := call.GetResponse(); resp != "" {
			respProg, err = compileCEL(env, resp)
			if err != nil {
				return nil, fmt.Errorf("compiling response CEL for %s: %w", node.GetId(), err)
			}
		}
		return &compiledCall{
			method:               methodName,
			kind:                 kind,
			client:               conn.Client,
			env:                  connRuntimeEnv(env, conn.Resolver),
			whenProg:             whenProg,
			closeRequestWhenProg: closeReqWhenProg,
			throttle:             rateToDuration(stream.GetThrottle()),
			request:              reqTree,
			responseProg:         respProg,
			retry:                retry,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported stream type for node %s", node.GetId())
	}
}

func compileAction(env shared.Env, node *flowv1beta2.Node, connectors map[string]*rpc.Connector, nodeCache cache.Cache) (any, error) {
	action := node.GetAction()
	call := action.GetCall()
	connID := call.GetConnection()
	conn := connectors[connID]
	if conn == nil {
		return nil, fmt.Errorf("no connector for connection %q on action node %s", connID, node.GetId())
	}
	methodName := protoreflect.FullName(call.GetMethod())
	kind, ok := lookupMethodKind(conn.Resolver, methodName)
	if !ok {
		return nil, fmt.Errorf("action method %q not found in connection %q for node %s", call.GetMethod(), connID, node.GetId())
	}
	if kind != rpc.MethodUnary {
		return nil, fmt.Errorf("action method %q must be unary for node %s", call.GetMethod(), node.GetId())
	}

	var whenProg cel.Program
	if w := action.GetWhen(); w != "" {
		var err error
		whenProg, err = compileCEL(env, w)
		if err != nil {
			return nil, fmt.Errorf("compiling action when CEL for %s: %w", node.GetId(), err)
		}
	}

	var actionCache cache.Cache
	if action.GetMemoize() {
		actionCache = nodeCache
		if actionCache == nil {
			actionCache = cachememory.New()
		}
	}

	var reqTree *compiledRequest
	if call.GetRequest() != nil {
		var err error
		reqTree, err = compileRequestTree(env, call.GetRequest())
		if err != nil {
			return nil, fmt.Errorf("compiling request tree for %s: %w", node.GetId(), err)
		}
	}
	var respProg cel.Program
	if resp := call.GetResponse(); resp != "" {
		var err error
		respProg, err = compileCEL(env, resp)
		if err != nil {
			return nil, fmt.Errorf("compiling response CEL for %s: %w", node.GetId(), err)
		}
	}

	retry, err := compileRetryStrategy(env, action.GetRetryStrategy())
	if err != nil {
		return nil, fmt.Errorf("compiling retry strategy for %s: %w", node.GetId(), err)
	}

	return &compiledCall{
		method:       methodName,
		kind:         kind,
		client:       conn.Client,
		env:          connRuntimeEnv(env, conn.Resolver),
		whenProg:     whenProg,
		throttle:     rateToDuration(action.GetThrottle()),
		request:      reqTree,
		responseProg: respProg,
		retry:        retry,
		cache:        actionCache,
	}, nil
}

func compileOutput(env shared.Env, node *flowv1beta2.Node) (any, error) {
	output := node.GetOutput()
	prog, err := compileCEL(env, output.GetValue())
	if err != nil {
		return nil, fmt.Errorf("compiling output CEL for %s: %w", node.GetId(), err)
	}
	transforms, err := compileTransforms(env, output.GetTransforms())
	if err != nil {
		return nil, fmt.Errorf("compiling transforms for %s: %w", node.GetId(), err)
	}
	return &compiledOutput{
		program:    prog,
		transforms: transforms,
		throttle:   rateToDuration(output.GetThrottle()),
	}, nil
}
