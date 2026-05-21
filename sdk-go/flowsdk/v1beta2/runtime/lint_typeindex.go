package runtime

import (
	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/reflect/protoreflect"

	graphlib "github.com/dominikbraun/graph"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// nodeTypeInfo describes the logical type of a node's `.value` for lint
// purposes. The walker uses `descriptor` to validate field chains; when
// `descriptor` is nil the type is scalar / list / map / dyn / unresolved
// and the walker stops chain validation (no false positives, no chain
// coverage either).
//
// This is the LOGICAL type a user-facing CEL expression sees -- not the
// proto schema's `cel.expr.Value` wrapper. nodeToMap (cel.go) decodes the
// wrapper to a typed ref.Val at runtime; the type index pulls that runtime
// knowledge forward to lint time.
type nodeTypeInfo struct {
	descriptor protoreflect.MessageDescriptor
}

// nodeTypeIndex maps node ID -> nodeTypeInfo. Built once per lint pass.
// Missing entries mean "type not computed" and behave identically to
// {descriptor: nil} at the walker -- chain validation skipped.
type nodeTypeIndex map[string]nodeTypeInfo

// buildNodeTypeIndex resolves the .value type for every node in the
// graph. The graph is guaranteed acyclic by ParseGraph (graph.go:121-123),
// so a single topological pass suffices -- no cycle handling needed.
//
// When resolvers is nil we return an empty index: chain validation
// requires connector descriptors to look up md.Input/Output, and without
// them the walker emits nothing (matches P6's conns-absent contract).
func buildNodeTypeIndex(graph *flowv1beta2.Graph, resolvers map[string]shared.Resolver, env shared.Env) nodeTypeIndex {
	idx := nodeTypeIndex{}
	if resolvers == nil {
		return idx
	}

	nodesByID := make(map[string]*flowv1beta2.Node, len(graph.GetNodes()))
	for _, n := range graph.GetNodes() {
		nodesByID[n.GetId()] = n
	}

	// Topologically sort so consumer types (vars/outputs/action-with-response)
	// can rely on their deps' types already being in the index. ParseGraph
	// validates DAG-ness so this never fails for graphs that reached Lint.
	dag := graphlib.New(graphlib.StringHash, graphlib.Directed())
	for _, n := range graph.GetNodes() {
		_ = dag.AddVertex(n.GetId())
	}
	for _, e := range graph.GetEdges() {
		_ = dag.AddEdge(e.GetSource(), e.GetTarget())
	}
	order, err := graphlib.TopologicalSort(dag)
	if err != nil {
		// Should not happen for a parsed graph; bail to empty index.
		return idx
	}

	for _, id := range order {
		node, ok := nodesByID[id]
		if !ok {
			continue
		}
		info := resolveNodeType(node, resolvers, env, idx)
		if info.descriptor != nil {
			idx[id] = info
		}
		// nil descriptors are intentionally not stored -- missing == unknown,
		// keeps the map smaller and the lookup semantics identical.
	}
	return idx
}

// resolveNodeType computes the .value type for a single node. May read
// from idx for already-computed upstream types (vars/outputs that
// reference other nodes); upstream entries are guaranteed present in
// topological order.
func resolveNodeType(node *flowv1beta2.Node, resolvers map[string]shared.Resolver, env shared.Env, idx nodeTypeIndex) nodeTypeInfo {
	switch node.WhichType() {
	case flowv1beta2.Node_Input_case:
		return inputTypeInfo(node.GetInput(), resolvers)
	case flowv1beta2.Node_Action_case:
		return callTypeInfo(node.GetAction().GetCall(), resolvers, env)
	case flowv1beta2.Node_Stream_case:
		return callTypeInfo(node.GetStream().GetCall(), resolvers, env)
	case flowv1beta2.Node_Var_case:
		return derivedTypeInfo(node.GetVar().GetValue(), resolvers, env, idx)
	case flowv1beta2.Node_Output_case:
		return derivedTypeInfo(node.GetOutput().GetValue(), resolvers, env, idx)
	}
	// Generator (int64 counter), Connection (opaque), Interaction (form):
	// no descriptor-walkable .value type.
	return nodeTypeInfo{}
}

// inputTypeInfo extracts a proto descriptor from an Input spec when the
// input is declared as a message. List/map/scalar inputs have no
// descriptor-walkable .value at lint time.
func inputTypeInfo(input *flowv1beta2.Input, resolvers map[string]shared.Resolver) nodeTypeInfo {
	msg := input.GetMessage()
	if msg == nil {
		return nodeTypeInfo{}
	}
	fullName := protoreflect.FullName(msg.GetType())
	if fullName == "" {
		return nodeTypeInfo{}
	}
	return lookupDescriptor(fullName, resolvers)
}

// callTypeInfo resolves an action/stream call's .value type. Without
// `call.response`, .value is the raw RPC response (md.Output()). With
// `call.response` set, the value is whatever that CEL expression
// computes -- and we compile it against a response-typed env (P6 helper)
// to read its OutputType.
func callTypeInfo(call *flowv1beta2.MethodCall, resolvers map[string]shared.Resolver, env shared.Env) nodeTypeInfo {
	if call == nil {
		return nodeTypeInfo{}
	}
	resolver, ok := resolvers[call.GetConnection()]
	if !ok {
		return nodeTypeInfo{}
	}
	md, err := resolver.FindMethodByName(protoreflect.FullName(call.GetMethod()))
	if err != nil {
		return nodeTypeInfo{}
	}

	if resp := call.GetResponse(); resp != "" {
		return responseExprTypeInfo(resp, resolver, md, resolvers)
	}
	return nodeTypeInfo{descriptor: md.Output()}
}

// responseExprTypeInfo compiles a `call.response` CEL expression with
// `this.response` typed as md.Output() (P6 helper) and reads the
// expression's OutputType. If the output is a concrete proto struct we
// look up its descriptor in the resolver pool; otherwise (scalar, list,
// dyn) the type isn't descriptor-walkable.
func responseExprTypeInfo(expression string, resolver shared.Resolver, md protoreflect.MethodDescriptor, resolvers map[string]shared.Resolver) nodeTypeInfo {
	src, ok := shared.IsValidExpr(expression)
	if !ok {
		return nodeTypeInfo{}
	}
	respEnv, err := buildLintResponseEnv(resolver, md)
	if err != nil {
		return nodeTypeInfo{}
	}
	ast, issues := respEnv.Compile(src)
	if issues != nil && issues.Err() != nil {
		return nodeTypeInfo{}
	}
	out := ast.OutputType()
	if out == nil || out.Kind() != cel.StructKind {
		return nodeTypeInfo{}
	}
	return lookupDescriptor(protoreflect.FullName(out.TypeName()), resolvers)
}

// derivedTypeInfo computes the .value type of a Var or Output. It tries
// two paths in priority order:
//  1. Node-ref-aware inference: if the expression is a node-ref chain
//     rooted in a category (e.g. `actions.foo.value` or
//     `vars.passthrough.value.nested`), walk the chain against the
//     index. This handles passthrough vars and nested-field selections
//     that CEL's typer can't see (because `actions` is declared
//     `map<string, dyn>` in the lint env).
//  2. CEL OutputType: fall back to compiling the expression and reading
//     its inferred type. Works for proto constructors like
//     `test.TestResponse{...}` whose type is statically known.
//
// Returns {} when neither path yields a concrete descriptor (e.g.
// scalar, list, dyn from a conditional, transform pipeline). Walker
// silently skips chains rooted in nodes with no descriptor -- that's
// the zero-false-positives invariant.
func derivedTypeInfo(expression string, resolvers map[string]shared.Resolver, env shared.Env, idx nodeTypeIndex) nodeTypeInfo {
	if d, ok := inferTypeFromNodeRefChain(expression, env, idx); ok {
		return nodeTypeInfo{descriptor: d}
	}
	return celExprTypeInfo(expression, resolvers, env)
}

// celExprTypeInfo computes the .value type of a Var or Output by
// compiling its CEL expression against the lint env and reading the
// OutputType. Only ObjectType outputs are descriptor-walkable; scalar /
// list / map / dyn yield nil descriptor (chain skipped). Because the
// lint env already has all connector and node-ref variables declared,
// this naturally chains through `vars.x.value.foo` references to
// upstream-computed types.
func celExprTypeInfo(expression string, resolvers map[string]shared.Resolver, env shared.Env) nodeTypeInfo {
	src, ok := shared.IsValidExpr(expression)
	if !ok {
		return nodeTypeInfo{}
	}
	if env == nil {
		return nodeTypeInfo{}
	}
	ast, issues := env.Compile(src)
	if issues != nil && issues.Err() != nil {
		return nodeTypeInfo{}
	}
	out := ast.OutputType()
	if out == nil || out.Kind() != cel.StructKind {
		return nodeTypeInfo{}
	}
	return lookupDescriptor(protoreflect.FullName(out.TypeName()), resolvers)
}

// inferTypeFromNodeRefChain parses the expression and, if its root is
// exactly a node-ref chain like `<category>.<id>.value[.<field>...]`,
// resolves the chain against the type index and returns the descriptor
// at the end of the chain. Returns (nil, false) for any non-chain root
// (constructors, lists, function calls, etc.) -- caller falls back to
// CEL OutputType.
//
// The chain must reach `.value` to be descriptor-walkable; chains that
// stop at a wrapper field other than .value resolve to a scalar and
// have no descriptor.
func inferTypeFromNodeRefChain(expression string, env shared.Env, idx nodeTypeIndex) (protoreflect.MessageDescriptor, bool) {
	if env == nil {
		return nil, false
	}
	src, ok := shared.IsValidExpr(expression)
	if !ok {
		return nil, false
	}
	ast, issues := env.Parse(src)
	if issues != nil && issues.Err() != nil {
		return nil, false
	}
	// Reuse the walker's chain collector (lint_noderef.go) to keep the
	// chain-shape definition in ONE place.
	category, chain, ok := collectChain(ast.NativeRep().Expr())
	if !ok {
		return nil, false
	}
	whitelist, isNodeRef := nodeRefWrapperFields[category]
	if !isNodeRef || whitelist == nil {
		return nil, false
	}
	// chain = [bareID, wrapperField, ...rest]; the chain must reach
	// `.value` to be descriptor-walkable -- other wrapper fields are
	// scalars and have no further chain.
	if len(chain) < 2 || chain[1] != "value" {
		return nil, false
	}
	info, ok := idx[category+"."+chain[0]]
	if !ok || info.descriptor == nil {
		return nil, false
	}
	// Walk any remaining selectors against the descriptor. Stops on the
	// first non-singular-message field (list/map/scalar/enum); the chain
	// could still be valid if the user indexes/iterates next, but we
	// can't follow that without losing type certainty -- the
	// zero-false-positives invariant says skip.
	md := info.descriptor
	for _, field := range chain[2:] {
		fd := md.Fields().ByName(protoreflect.Name(field))
		if fd == nil {
			fd = md.Fields().ByJSONName(field)
		}
		if fd == nil || fd.Kind() != protoreflect.MessageKind || fd.IsList() || fd.IsMap() {
			return nil, false
		}
		md = fd.Message()
	}
	return md, true
}

// lookupDescriptor finds a MessageDescriptor by full name across the
// available resolver pool. Returns {} if not found.
func lookupDescriptor(fullName protoreflect.FullName, resolvers map[string]shared.Resolver) nodeTypeInfo {
	if fullName == "" {
		return nodeTypeInfo{}
	}
	for _, r := range resolvers {
		if mt, err := r.FindMessageByName(fullName); err == nil && mt != nil {
			return nodeTypeInfo{descriptor: mt.Descriptor()}
		}
	}
	return nodeTypeInfo{}
}
