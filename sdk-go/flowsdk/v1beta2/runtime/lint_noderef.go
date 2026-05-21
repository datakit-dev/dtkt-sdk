package runtime

import (
	"fmt"
	"sort"
	"strings"

	celast "github.com/google/cel-go/common/ast"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// Per-category wrapper-field whitelists, derived from `nodeToMap`
// (cel.go:175-197). A wrapper field NOT in the whitelist is flagged as
// unknown -- it would resolve to nil at runtime (the activation does not
// bind it), which is almost always a typo.
//
// `connections` is intentionally absent: connection refs are opaque
// metadata and may grow fields in future; the walker skips them.
//
// Proto wrapper fields like `.error`, `.phase`, `.event_time` exist on
// the RunSnapshot_*Node protos but are NOT in the activation. The
// whitelist matches the activation, so referencing them lint-errors --
// consistent with the runtime behavior of returning nil.
var nodeRefWrapperFields = map[string]map[string]bool{
	"inputs":       {"value": true, "closed": true},
	"generators":   {"value": true, "done": true, "eval_count": true},
	"vars":         {"value": true, "eval_count": true},
	"actions":      {"value": true, "eval_count": true},
	"streams":      {"value": true, "request_closed": true, "response_closed": true, "request_count": true, "response_count": true},
	"outputs":      {"value": true, "eval_count": true},
	"interactions": {"value": true, "submitted": true},
}

// lintNodeRefChains validates node-ref selector chains across every CEL
// expression in the graph. For each expression: walk the AST, find each
// outermost `<category>.<id>.<chain>` pattern, then:
//  1. Verify the wrapper field (chain[1]) is in the per-category whitelist.
//  2. For `.value` specifically, walk the remaining selectors against the
//     resolved proto descriptor from the type index.
//
// Emits nothing when resolvers is nil (matches P6 conns-absent contract).
// Stops walking at any dyn / runtime-index / non-message-field boundary --
// the contract is ZERO FALSE POSITIVES: only emit when ground truth is
// certain.
//
// Validation contract (documented at the package level, also referenced
// by the plan doc P14 entry):
//   - CATCHES: field-name typos, schema drift, wrong wrapper fields,
//     transitive chains through vars/outputs of known types.
//   - DOES NOT CATCH: runtime index/key values, conditional widening to
//     dyn, side-effect correctness, expressions where the upstream type
//     can't be resolved (e.g., var whose .Value returns dyn).
func lintNodeRefChains(graph *flowv1beta2.Graph, resolvers map[string]shared.Resolver, env shared.Env) []LintDiagnostic {
	if resolvers == nil || env == nil {
		return nil
	}
	idx := buildNodeTypeIndex(graph, resolvers, env)
	if len(idx) == 0 {
		return nil
	}
	var diags []LintDiagnostic
	for _, site := range enumerateExpressionSites(graph) {
		diags = append(diags, lintNodeRefChainsForExpr(site.expression, site.path, idx, env)...)
	}
	return diags
}

// exprSite is a single CEL expression within the flow graph, paired
// with the node-relative path that locates it for diagnostics.
type exprSite struct {
	path       string // e.g. "outputs.foo.value", "actions.x.call.request.name"
	expression string
}

// enumerateExpressionSites yields every CEL expression in the graph in a
// deterministic order. Mirrors the dep-extraction enumeration at
// graph.go:138-217 but emits sites for lint rather than dep edges.
func enumerateExpressionSites(graph *flowv1beta2.Graph) []exprSite {
	var sites []exprSite
	for _, node := range graph.GetNodes() {
		nodePath := node.GetId()
		switch node.WhichType() {
		case flowv1beta2.Node_Input_case:
			for i, tr := range node.GetInput().GetTransforms() {
				sites = append(sites, transformSites(tr, fmt.Sprintf("%s.transforms[%d]", nodePath, i))...)
			}
		case flowv1beta2.Node_Var_case:
			v := node.GetVar()
			if e := v.GetValue(); e != "" {
				sites = append(sites, exprSite{nodePath + ".value", e})
			}
			if sw := v.GetSwitch(); sw != nil {
				if e := sw.GetValue(); e != "" {
					sites = append(sites, exprSite{nodePath + ".switch.value", e})
				}
				for i, c := range sw.GetCase() {
					if e := c.GetValue(); e != "" {
						sites = append(sites, exprSite{fmt.Sprintf("%s.switch.cases[%d].value", nodePath, i), e})
					}
					if e := c.GetReturn(); e != "" {
						sites = append(sites, exprSite{fmt.Sprintf("%s.switch.cases[%d].return", nodePath, i), e})
					}
				}
				if e := sw.GetDefault(); e != "" {
					sites = append(sites, exprSite{nodePath + ".switch.default", e})
				}
			}
			for i, tr := range v.GetTransforms() {
				sites = append(sites, transformSites(tr, fmt.Sprintf("%s.transforms[%d]", nodePath, i))...)
			}
			sites = append(sites, flowControlSites(v.GetFlowControl(), nodePath+".flow_control")...)
			sites = append(sites, nodeControlSites(v.GetNodeControl(), nodePath+".node_control")...)
		case flowv1beta2.Node_Action_case:
			a := node.GetAction()
			if e := a.GetWhen(); e != "" {
				sites = append(sites, exprSite{nodePath + ".when", e})
			}
			sites = append(sites, callSites(a.GetCall(), nodePath+".call")...)
			sites = append(sites, retrySites(a.GetRetryStrategy(), nodePath+".retry_strategy")...)
			sites = append(sites, flowControlSites(a.GetFlowControl(), nodePath+".flow_control")...)
			sites = append(sites, nodeControlSites(a.GetNodeControl(), nodePath+".node_control")...)
		case flowv1beta2.Node_Stream_case:
			s := node.GetStream()
			if e := s.GetWhen(); e != "" {
				sites = append(sites, exprSite{nodePath + ".when", e})
			}
			sites = append(sites, callSites(s.GetCall(), nodePath+".call")...)
			sites = append(sites, retrySites(s.GetRetryStrategy(), nodePath+".retry_strategy")...)
			sites = append(sites, flowControlSites(s.GetFlowControl(), nodePath+".flow_control")...)
			sites = append(sites, nodeControlSites(s.GetNodeControl(), nodePath+".node_control")...)
		case flowv1beta2.Node_Generator_case:
			g := node.GetGenerator()
			if t := g.GetTicker(); t != nil {
				if e := t.GetValue(); e != "" {
					sites = append(sites, exprSite{nodePath + ".ticker.value", e})
				}
			}
			if c := g.GetCron(); c != nil {
				if e := c.GetValue(); e != "" {
					sites = append(sites, exprSite{nodePath + ".cron.value", e})
				}
			}
		case flowv1beta2.Node_Output_case:
			o := node.GetOutput()
			if e := o.GetValue(); e != "" {
				sites = append(sites, exprSite{nodePath + ".value", e})
			}
			for i, tr := range o.GetTransforms() {
				sites = append(sites, transformSites(tr, fmt.Sprintf("%s.transforms[%d]", nodePath, i))...)
			}
			sites = append(sites, flowControlSites(o.GetFlowControl(), nodePath+".flow_control")...)
			sites = append(sites, nodeControlSites(o.GetNodeControl(), nodePath+".node_control")...)
		case flowv1beta2.Node_Interaction_case:
			it := node.GetInteraction()
			if e := it.GetWhen(); e != "" {
				sites = append(sites, exprSite{nodePath + ".when", e})
			}
			for i, tr := range it.GetTransforms() {
				sites = append(sites, transformSites(tr, fmt.Sprintf("%s.transforms[%d]", nodePath, i))...)
			}
			sites = append(sites, flowControlSites(it.GetFlowControl(), nodePath+".flow_control")...)
			sites = append(sites, nodeControlSites(it.GetNodeControl(), nodePath+".node_control")...)
		}
	}
	return sites
}

func callSites(call *flowv1beta2.MethodCall, prefix string) []exprSite {
	if call == nil {
		return nil
	}
	var sites []exprSite
	if req := call.GetRequest(); req != nil {
		sites = append(sites, requestTreeSites(req, prefix+".request")...)
	}
	if e := call.GetResponse(); e != "" {
		sites = append(sites, exprSite{prefix + ".response", e})
	}
	return sites
}

// requestTreeSites recursively yields CEL leaves from a structpb-shaped
// request tree. Mirrors lintRequestTree's traversal.
func requestTreeSites(v *structpb.Value, path string) []exprSite {
	if v == nil {
		return nil
	}
	switch v.GetKind().(type) {
	case *structpb.Value_StringValue:
		s := v.GetStringValue()
		if _, ok := shared.IsValidExpr(s); ok {
			return []exprSite{{path, s}}
		}
	case *structpb.Value_StructValue:
		var sites []exprSite
		for key, field := range v.GetStructValue().GetFields() {
			sites = append(sites, requestTreeSites(field, path+"."+key)...)
		}
		return sites
	case *structpb.Value_ListValue:
		var sites []exprSite
		for i, elem := range v.GetListValue().GetValues() {
			sites = append(sites, requestTreeSites(elem, fmt.Sprintf("%s[%d]", path, i))...)
		}
		return sites
	}
	return nil
}

func transformSites(t *flowv1beta2.Transform, path string) []exprSite {
	if t == nil {
		return nil
	}
	var sites []exprSite
	if e := t.GetMap(); e != "" {
		sites = append(sites, exprSite{path + ".map", e})
	}
	if e := t.GetFilter(); e != "" {
		sites = append(sites, exprSite{path + ".filter", e})
	}
	if r := t.GetReduce(); r != nil {
		if e := r.GetInitial(); e != "" {
			sites = append(sites, exprSite{path + ".reduce.initial", e})
		}
		if e := r.GetAccumulator(); e != "" {
			sites = append(sites, exprSite{path + ".reduce.accumulator", e})
		}
	}
	if s := t.GetScan(); s != nil {
		if e := s.GetInitial(); e != "" {
			sites = append(sites, exprSite{path + ".scan.initial", e})
		}
		if e := s.GetAccumulator(); e != "" {
			sites = append(sites, exprSite{path + ".scan.accumulator", e})
		}
	}
	return sites
}

func retrySites(r *flowv1beta2.RetryStrategy, prefix string) []exprSite {
	if r == nil {
		return nil
	}
	var sites []exprSite
	if e := r.GetWhen(); e != "" {
		sites = append(sites, exprSite{prefix + ".when", e})
	}
	if e := r.GetSkipWhen(); e != "" {
		sites = append(sites, exprSite{prefix + ".skip_when", e})
	}
	if e := r.GetSuspendWhen(); e != "" {
		sites = append(sites, exprSite{prefix + ".suspend_when", e})
	}
	if e := r.GetTerminateWhen(); e != "" {
		sites = append(sites, exprSite{prefix + ".terminate_when", e})
	}
	if e := r.GetContinueWhen(); e != "" {
		sites = append(sites, exprSite{prefix + ".continue_when", e})
	}
	return sites
}

func flowControlSites(c *flowv1beta2.FlowControl, prefix string) []exprSite {
	if c == nil {
		return nil
	}
	var sites []exprSite
	if e := c.GetStopWhen(); e != "" {
		sites = append(sites, exprSite{prefix + ".stop_when", e})
	}
	if e := c.GetTerminateWhen(); e != "" {
		sites = append(sites, exprSite{prefix + ".terminate_when", e})
	}
	if e := c.GetSuspendWhen(); e != "" {
		sites = append(sites, exprSite{prefix + ".suspend_when", e})
	}
	return sites
}

func nodeControlSites(c *flowv1beta2.NodeControl, prefix string) []exprSite {
	if c == nil {
		return nil
	}
	var sites []exprSite
	if e := c.GetStopWhen(); e != "" {
		sites = append(sites, exprSite{prefix + ".stop_when", e})
	}
	if e := c.GetTerminateWhen(); e != "" {
		sites = append(sites, exprSite{prefix + ".terminate_when", e})
	}
	if e := c.GetSuspendWhen(); e != "" {
		sites = append(sites, exprSite{prefix + ".suspend_when", e})
	}
	return sites
}

// lintNodeRefChainsForExpr validates a single CEL expression against
// the type index. Parses with the lint env (so connector types resolve),
// then walks the AST for outermost SelectKind chains rooted in
// node-category idents.
func lintNodeRefChainsForExpr(expression, path string, idx nodeTypeIndex, env shared.Env) []LintDiagnostic {
	src, ok := shared.IsValidExpr(expression)
	if !ok {
		return nil
	}
	if env == nil {
		return nil
	}
	ast, issues := env.Parse(src)
	if issues != nil && issues.Err() != nil {
		return nil // parse errors already reported elsewhere
	}
	var diags []LintDiagnostic
	navAST := celast.NavigateAST(ast.NativeRep())
	celast.PreOrderVisit(navAST, celast.NewExprVisitor(func(e celast.Expr) {
		if e.Kind() != celast.SelectKind {
			return
		}
		// Only process the OUTERMOST link of a chain; nested Selects
		// are walked from here by collectChain.
		if nav, ok := e.(celast.NavigableExpr); ok {
			if parent, has := nav.Parent(); has && parent.Kind() == celast.SelectKind {
				return
			}
		}
		category, chain, ok := collectChain(e)
		if !ok {
			return
		}
		whitelist, isNodeRef := nodeRefWrapperFields[category]
		if !isNodeRef {
			return // category isn't a node-ref root (e.g., "connections" - opaque)
		}
		// chain = [<nodeId>, <wrapperField>?, <subfield>?, ...]
		if len(chain) < 2 {
			return // just `actions.foo` -- wrapper accessed; no chain yet
		}
		bareID := chain[0]
		// ParseGraph prefixes node IDs with the category (graph.go:34-76),
		// so the canonical ID in the type index is `<category>.<bareID>`.
		nodeID := category + "." + bareID
		wrapperField := chain[1]
		rest := chain[2:]

		// 1. Validate wrapper field against whitelist.
		if !whitelist[wrapperField] {
			diags = append(diags, LintDiagnostic{
				Severity: SeverityError,
				Path:     path,
				Message:  fmt.Sprintf("unknown field %q on %s.%s (valid: %s)", wrapperField, category, bareID, sortedKeyList(whitelist)),
				Code:     CodeUnknownField,
			})
			return
		}

		// 2. Only `.value` has a descriptor-walkable type. Other wrapper
		//    fields (eval_count, closed, done, submitted, *_count) are
		//    scalars; further chain is a runtime nil, but lint stays
		//    silent (no false positives from validating CEL on scalars).
		if wrapperField != "value" {
			return
		}

		// 3. Walk `rest` against the resolved .value descriptor.
		info, ok := idx[nodeID]
		if !ok || info.descriptor == nil {
			return // type unresolved; cannot validate further
		}
		diags = append(diags, walkFieldChain(info.descriptor, rest, category, bareID, path)...)
	}))
	return diags
}

// collectChain walks down an outermost SelectKind to find the rooted
// identifier and accumulates the chain of field names. Returns
// (category, [nodeId, field1, field2, ...], true) on success; false if
// the chain isn't rooted in an Ident.
func collectChain(outer celast.Expr) (string, []string, bool) {
	chain := []string{}
	cur := outer
	for cur.Kind() == celast.SelectKind {
		sel := cur.AsSelect()
		// Prepend; we're walking outer-to-inner but the conceptual chain
		// reads root-to-leaf.
		chain = append([]string{sel.FieldName()}, chain...)
		cur = sel.Operand()
	}
	if cur.Kind() != celast.IdentKind {
		return "", nil, false
	}
	return cur.AsIdent(), chain, true
}

// walkFieldChain walks proto fields starting from `md` along `chain` and
// emits a diagnostic for the first missing field. Stops walking on
// non-singular-message fields (lists, maps, scalars) without emitting --
// chain could still be valid (e.g., list index next).
//
// `bareID` is the node ID without the category prefix (e.g. "run" not
// "actions.run"); it is combined with `category` for display only.
func walkFieldChain(md protoreflect.MessageDescriptor, chain []string, category, bareID, exprPath string) []LintDiagnostic {
	current := md
	displayPath := fmt.Sprintf("%s.%s.value", category, bareID)
	for _, field := range chain {
		displayPath += "." + field
		fd := current.Fields().ByName(protoreflect.Name(field))
		if fd == nil {
			// Try JSON name as fallback (proto3 allows snake_case or camelCase access).
			fd = current.Fields().ByJSONName(field)
		}
		if fd == nil {
			return []LintDiagnostic{{
				Severity: SeverityError,
				Path:     exprPath,
				Message:  fmt.Sprintf("unknown field %q in message %s (referenced as %s)", field, current.FullName(), displayPath),
				Code:     CodeUnknownField,
			}}
		}
		// Continue walking only when the next position is a singular
		// message field. Anything else (scalar, enum, list, map) ends
		// proto-descriptor walking; users would index/iterate next, which
		// we don't follow (Stage 1 boundary -- documented in the plan).
		if fd.Kind() != protoreflect.MessageKind || fd.IsList() || fd.IsMap() {
			return nil
		}
		current = fd.Message()
	}
	return nil
}

// sortedKeyList returns a deterministic comma-separated list of map keys
// for use in diagnostic messages. Without this, the diagnostic text
// would vary across runs and break test assertions on message content.
func sortedKeyList(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

