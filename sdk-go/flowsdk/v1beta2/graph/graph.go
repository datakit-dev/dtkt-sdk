package graph

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	graphlib "github.com/dominikbraun/graph"
	"google.golang.org/protobuf/types/known/structpb"
)

// nodeCategories are the valid category prefixes for qualified node references.
var nodeCategories = map[string]bool{
	"inputs": true, "generators": true, "vars": true,
	"actions": true, "streams": true, "interactions": true,
	"outputs": true,
}

// parseEnv is a minimal CEL environment used only for syntactic parsing
// during edge inference. It has no declared variables or functions.
var parseEnv, _ = cel.NewEnv()

// Build converts a Flow definition into a Graph with automatically inferred edges.
// It creates nodes for each flow element, infers data-dependency edges by scanning
// CEL expressions for node references, and validates the result is a DAG.
func Build(flow *flowv1beta2.Flow) (*flowv1beta2.Graph, error) {
	var nodes []*flowv1beta2.Node

	for _, c := range flow.GetConnections() {
		n := &flowv1beta2.Node{}
		n.SetId("connections." + c.GetId())
		n.SetConnection(c)
		nodes = append(nodes, n)
	}
	for _, inp := range flow.GetInputs() {
		n := &flowv1beta2.Node{}
		n.SetId("inputs." + inp.GetId())
		n.SetInput(inp)
		nodes = append(nodes, n)
	}
	for _, gen := range flow.GetGenerators() {
		n := &flowv1beta2.Node{}
		n.SetId("generators." + gen.GetId())
		n.SetGenerator(gen)
		nodes = append(nodes, n)
	}
	for _, v := range flow.GetVars() {
		n := &flowv1beta2.Node{}
		n.SetId("vars." + v.GetId())
		n.SetVar(v)
		nodes = append(nodes, n)
	}
	for _, a := range flow.GetActions() {
		n := &flowv1beta2.Node{}
		n.SetId("actions." + a.GetId())
		n.SetAction(a)
		nodes = append(nodes, n)
	}
	for _, s := range flow.GetStreams() {
		n := &flowv1beta2.Node{}
		n.SetId("streams." + s.GetId())
		n.SetStream(s)
		nodes = append(nodes, n)
	}
	for _, o := range flow.GetOutputs() {
		n := &flowv1beta2.Node{}
		n.SetId("outputs." + o.GetId())
		n.SetOutput(o)
		nodes = append(nodes, n)
	}
	for _, inter := range flow.GetInteractions() {
		n := &flowv1beta2.Node{}
		n.SetId("interactions." + inter.GetId())
		n.SetInteraction(inter)
		nodes = append(nodes, n)
	}

	// Validate unique IDs.
	nodeIDs := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if nodeIDs[n.GetId()] {
			return nil, fmt.Errorf("duplicate node ID: %s", n.GetId())
		}
		nodeIDs[n.GetId()] = true
	}

	// Infer edges from CEL expression references.
	edgeSet := make(map[string]bool)
	var edges []*flowv1beta2.Edge
	for _, n := range nodes {
		refs := extractRefs(collectExpressions(n))
		for _, ref := range refs {
			if ref == n.GetId() {
				continue
			}
			if !nodeIDs[ref] {
				return nil, fmt.Errorf("node %s references unknown node %s", n.GetId(), ref)
			}
			key := ref + "->" + n.GetId()
			if !edgeSet[key] {
				edgeSet[key] = true
				e := &flowv1beta2.Edge{}
				e.SetSource(ref)
				e.SetTarget(n.GetId())
				edges = append(edges, e)
			}
		}
	}

	// Validate DAG (no cycles).
	dag := graphlib.New(graphlib.StringHash, graphlib.Directed())
	for _, n := range nodes {
		_ = dag.AddVertex(n.GetId())
	}
	for _, e := range edges {
		_ = dag.AddEdge(e.GetSource(), e.GetTarget())
	}
	if _, err := graphlib.TopologicalSort(dag); err != nil {
		return nil, fmt.Errorf("graph contains cycles: %w", err)
	}

	g := &flowv1beta2.Graph{}
	g.SetNodes(nodes)
	g.SetEdges(edges)
	return g, nil
}

// collectExpressions gathers all CEL expression strings from a node for edge inference.
func collectExpressions(node *flowv1beta2.Node) []string {
	var exprs []string

	switch node.WhichType() {
	case flowv1beta2.Node_Input_case:
		exprs = append(exprs, collectTransformExpressions(node.GetInput().GetTransforms())...)

	case flowv1beta2.Node_Var_case:
		v := node.GetVar()
		switch v.WhichType() {
		case flowv1beta2.Var_Value_case:
			exprs = append(exprs, v.GetValue())
		case flowv1beta2.Var_Switch_case:
			sw := v.GetSwitch()
			exprs = append(exprs, sw.GetValue())
			for _, c := range sw.GetCase() {
				exprs = append(exprs, c.GetValue(), c.GetReturn())
			}
			if sw.GetDefault() != "" {
				exprs = append(exprs, sw.GetDefault())
			}
		}
		exprs = append(exprs, collectTransformExpressions(v.GetTransforms())...)
		exprs = append(exprs, collectFlowControlExpressions(v.GetFlowControl())...)

	case flowv1beta2.Node_Action_case:
		a := node.GetAction()
		if a.GetWhen() != "" {
			exprs = append(exprs, a.GetWhen())
		}
		if call := a.GetCall(); call != nil {
			exprs = append(exprs, collectMethodCallExpressions(call)...)
		}
		exprs = append(exprs, collectFlowControlExpressions(a.GetFlowControl())...)

	case flowv1beta2.Node_Stream_case:
		s := node.GetStream()
		if s.GetWhen() != "" {
			exprs = append(exprs, s.GetWhen())
		}
		if s.GetCloseRequestWhen() != "" {
			exprs = append(exprs, s.GetCloseRequestWhen())
		}
		if call := s.GetCall(); call != nil {
			exprs = append(exprs, collectMethodCallExpressions(call)...)
		}
		exprs = append(exprs, collectFlowControlExpressions(s.GetFlowControl())...)

	case flowv1beta2.Node_Generator_case:
		gen := node.GetGenerator()
		switch gen.WhichType() {
		case flowv1beta2.Generator_Ticker_case:
			if v := gen.GetTicker().GetValue(); v != "" {
				exprs = append(exprs, v)
			}
		case flowv1beta2.Generator_Cron_case:
			if v := gen.GetCron().GetValue(); v != "" {
				exprs = append(exprs, v)
			}
		}

	case flowv1beta2.Node_Output_case:
		exprs = append(exprs, node.GetOutput().GetValue())
		exprs = append(exprs, collectTransformExpressions(node.GetOutput().GetTransforms())...)
		exprs = append(exprs, collectFlowControlExpressions(node.GetOutput().GetFlowControl())...)

	case flowv1beta2.Node_Interaction_case:
		if inter := node.GetInteraction(); inter != nil {
			if inter.GetWhen() != "" {
				exprs = append(exprs, inter.GetWhen())
			}
			exprs = append(exprs, collectTransformExpressions(inter.GetTransforms())...)
			exprs = append(exprs, collectFlowControlExpressions(inter.GetFlowControl())...)
		}
	}

	return exprs
}

// collectFlowControlExpressions gathers CEL expression strings from a FlowControl proto.
func collectFlowControlExpressions(fc *flowv1beta2.FlowControl) []string {
	if fc == nil {
		return nil
	}
	var exprs []string
	if s := fc.GetStopWhen(); s != "" {
		exprs = append(exprs, s)
	}
	if s := fc.GetTerminateWhen(); s != "" {
		exprs = append(exprs, s)
	}
	if s := fc.GetSuspendWhen(); s != "" {
		exprs = append(exprs, s)
	}
	return exprs
}

// collectTransformExpressions gathers all CEL expression strings from a transform pipeline.
func collectTransformExpressions(transforms []*flowv1beta2.Transform) []string {
	var exprs []string
	for _, t := range transforms {
		switch t.WhichType() {
		case flowv1beta2.Transform_Map_case:
			exprs = append(exprs, t.GetMap())
		case flowv1beta2.Transform_Filter_case:
			exprs = append(exprs, t.GetFilter())
		case flowv1beta2.Transform_Reduce_case:
			r := t.GetReduce()
			exprs = append(exprs, r.GetInitial(), r.GetAccumulator())
			if gb := r.GetGroupBy(); gb != nil {
				if gb.GetKey() != "" {
					exprs = append(exprs, gb.GetKey())
				}
				if w := gb.GetWindow(); w != nil {
					if e := w.GetEvent(); e != nil {
						exprs = append(exprs, e.GetWhen())
					}
				}
			}
		case flowv1beta2.Transform_Scan_case:
			s := t.GetScan()
			exprs = append(exprs, s.GetInitial(), s.GetAccumulator())
			if gb := s.GetGroupBy(); gb != nil {
				if gb.GetKey() != "" {
					exprs = append(exprs, gb.GetKey())
				}
				if w := gb.GetWindow(); w != nil {
					if e := w.GetEvent(); e != nil {
						exprs = append(exprs, e.GetWhen())
					}
				}
			}
		}
	}
	return exprs
}

// collectMethodCallExpressions gathers CEL expressions from a MethodCall.
func collectMethodCallExpressions(call *flowv1beta2.MethodCall) []string {
	var exprs []string
	if call.GetResponse() != "" {
		exprs = append(exprs, call.GetResponse())
	}
	exprs = append(exprs, collectStructpbExpressions(call.GetRequest())...)
	return exprs
}

// collectStructpbExpressions recursively finds CEL expression strings inside a structpb.Value.
// CEL expressions are string values prefixed with "=".
func collectStructpbExpressions(v *structpb.Value) []string {
	if v == nil {
		return nil
	}
	switch v.GetKind().(type) {
	case *structpb.Value_StringValue:
		s := v.GetStringValue()
		if strings.HasPrefix(strings.TrimSpace(s), "=") {
			return []string{s}
		}
	case *structpb.Value_StructValue:
		var exprs []string
		for _, f := range v.GetStructValue().GetFields() {
			exprs = append(exprs, collectStructpbExpressions(f)...)
		}
		return exprs
	case *structpb.Value_ListValue:
		var exprs []string
		for _, elem := range v.GetListValue().GetValues() {
			exprs = append(exprs, collectStructpbExpressions(elem)...)
		}
		return exprs
	}
	return nil
}

// extractRefs extracts unique qualified node references (e.g. "inputs.number")
// from a set of CEL expression strings by parsing each expression into an AST
// and walking it to find Select nodes with a category-level Ident operand.
// Expressions that fail to parse are silently skipped (syntax errors are caught
// later in the compile phase).
func extractRefs(expressions []string) []string {
	seen := make(map[string]bool)
	var refs []string
	for _, e := range expressions {
		// Strip leading "=" from structpb CEL expressions.
		e = strings.TrimSpace(e)
		if strings.HasPrefix(e, "=") {
			e = strings.TrimSpace(e[1:])
		}
		if e == "" {
			continue
		}
		parsed, iss := parseEnv.Parse(e)
		if iss != nil && iss.Err() != nil {
			continue
		}
		ast.PreOrderVisit(ast.NavigateAST(parsed.NativeRep()), ast.NewExprVisitor(func(expr ast.Expr) {
			if expr.Kind() != ast.SelectKind {
				return
			}
			sel := expr.AsSelect()
			if sel.Operand().Kind() != ast.IdentKind {
				return
			}
			category := sel.Operand().AsIdent()
			if !nodeCategories[category] {
				return
			}
			ref := category + "." + sel.FieldName()
			if !seen[ref] {
				seen[ref] = true
				refs = append(refs, ref)
			}
		}))
	}
	return refs
}
