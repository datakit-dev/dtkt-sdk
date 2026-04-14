package runtime

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// Lint validates all CEL expressions and transform pipelines in a graph.
// It parses every expression to check for syntax/type errors without compiling
// to executable programs.
//
// When resolvers are provided (keyed by connection ID), Lint additionally
// validates request trees against proto schemas -- checking that field names
// exist and that literal value types are compatible with the proto field types.
func Lint(graph *flowv1beta2.Graph, resolvers ...map[string]shared.Resolver) *LintResult {
	var (
		diags       []LintDiagnostic
		resolverMap map[string]shared.Resolver
	)
	if len(resolvers) > 0 {
		resolverMap = resolvers[0]
	}

	// Build a CEL env with connection types when resolvers are available.
	// This enables type-checking CEL expressions against proto field types.
	var celEnv shared.Env
	if resolverMap != nil {
		diags = append(diags, lintProtoConflicts(resolverMap)...)

		rawEnv, err := buildLintCELEnv(resolverMap)
		if err != nil {
			celEnv = nil // fall back to no CEL type checking
		} else {
			celEnv = &runtimeEnv{Env: rawEnv}
		}
	}

	// Collect declared connection IDs for reference validation.
	connections := make(map[string]bool)
	for _, node := range graph.GetNodes() {
		if node.WhichType() == flowv1beta2.Node_Connection_case {
			connections[node.GetConnection().GetId()] = true
		}
	}

	for _, node := range graph.GetNodes() {
		diags = append(diags, lintNode(node, resolverMap, celEnv)...)
		if d := lintNodeConnection(node, connections); d != nil {
			diags = append(diags, *d)
		}
	}

	// Warn for orphaned nodes (no consumers) that don't have side effects.
	consumed := make(map[string]bool, len(graph.GetEdges()))
	for _, e := range graph.GetEdges() {
		consumed[e.GetSource()] = true
	}
	for _, node := range graph.GetNodes() {
		if consumed[node.GetId()] || hasSideEffect(node) {
			continue
		}
		diags = append(diags, LintDiagnostic{
			Severity: SeverityWarning,
			Node:     node.GetId(),
			Message:  "orphaned node has no consumers",
			Code:     CodeOrphanedNode,
		})
	}

	// Error for non-source nodes with no upstream edges (no producers).
	// Inputs, generators, and connections are source nodes -- they produce data
	// without needing upstream dependencies. All other node types require at
	// least one inbound edge; without one the handler either loops forever
	// (streams/actions/interactions) or never receives data (vars/outputs).
	fed := make(map[string]bool, len(graph.GetEdges()))
	for _, e := range graph.GetEdges() {
		fed[e.GetTarget()] = true
	}
	for _, node := range graph.GetNodes() {
		if isSource(node) || fed[node.GetId()] {
			continue
		}
		diags = append(diags, LintDiagnostic{
			Severity: SeverityError,
			Node:     node.GetId(),
			Message:  "has no upstream dependencies",
			Code:     CodeNoUpstream,
		})
	}

	return &LintResult{Diagnostics: diags}
}

// hasSideEffect returns true for nodes that perform external effects and thus
// are not considered orphaned even without consumers (actions, streams, outputs,
// interactions, connections).
func hasSideEffect(node *flowv1beta2.Node) bool {
	switch node.WhichType() {
	case flowv1beta2.Node_Action_case,
		flowv1beta2.Node_Stream_case,
		flowv1beta2.Node_Output_case,
		flowv1beta2.Node_Interaction_case,
		flowv1beta2.Node_Connection_case:
		return true
	}
	return false
}

// isSource returns true for nodes that produce data without needing upstream
// dependencies (inputs, generators, connections).
func isSource(node *flowv1beta2.Node) bool {
	switch node.WhichType() {
	case flowv1beta2.Node_Input_case,
		flowv1beta2.Node_Generator_case,
		flowv1beta2.Node_Connection_case:
		return true
	}
	return false
}

func lintNode(node *flowv1beta2.Node, resolvers map[string]shared.Resolver, env shared.Env) []LintDiagnostic {
	var diags []LintDiagnostic
	nodeID := node.GetId()

	diag := func(path, message, code string) {
		diags = append(diags, LintDiagnostic{
			Severity: SeverityError,
			Node:     nodeID,
			Path:     path,
			Message:  message,
			Code:     code,
		})
	}
	celDiag := func(path string, err error) {
		diag(path, err.Error(), CodeInvalidCEL)
	}
	appendWith := func(ds []LintDiagnostic) {
		for _, d := range ds {
			d.Node = nodeID
			diags = append(diags, d)
		}
	}

	switch node.WhichType() {
	case flowv1beta2.Node_Input_case:
		inp := node.GetInput()
		if inp.GetConstant() && inputTypeHasDefault(inp) {
			diag("", "constant and default are mutually exclusive", CodeConstantDefaultExclusive)
		}
		appendWith(lintTransforms(inp.GetTransforms()))

	case flowv1beta2.Node_Var_case:
		v := node.GetVar()
		switch v.WhichType() {
		case flowv1beta2.Var_Value_case:
			if _, err := parseCEL(v.GetValue()); err != nil {
				celDiag("value", err)
			}
		case flowv1beta2.Var_Switch_case:
			sw := v.GetSwitch()
			if _, err := parseCEL(sw.GetValue()); err != nil {
				celDiag("switch.value", err)
			}
			for i, c := range sw.GetCase() {
				if _, err := parseCEL(c.GetValue()); err != nil {
					celDiag(fmt.Sprintf("switch.cases[%d].value", i), err)
				}
				if _, err := parseCEL(c.GetReturn()); err != nil {
					celDiag(fmt.Sprintf("switch.cases[%d].return", i), err)
				}
			}
			if sw.GetDefault() != "" {
				if _, err := parseCEL(sw.GetDefault()); err != nil {
					celDiag("switch.default", err)
				}
			}
		}
		appendWith(lintTransforms(v.GetTransforms()))

	case flowv1beta2.Node_Generator_case:
		gen := node.GetGenerator()
		switch gen.WhichType() {
		case flowv1beta2.Generator_Ticker_case:
			if v := gen.GetTicker().GetValue(); v != "" {
				if _, err := parseCEL(v); err != nil {
					celDiag("ticker.value", err)
				}
			}
		case flowv1beta2.Generator_Cron_case:
			if v := gen.GetCron().GetValue(); v != "" {
				if _, err := parseCEL(v); err != nil {
					celDiag("cron.value", err)
				}
			}
		}

	case flowv1beta2.Node_Stream_case:
		s := node.GetStream()
		if w := s.GetWhen(); w != "" {
			if _, err := parseCEL(w); err != nil {
				celDiag("when", err)
			}
		}
		if crw := s.GetCloseRequestWhen(); crw != "" {
			if _, err := parseCEL(crw); err != nil {
				celDiag("close_request_when", err)
			}
		}
		if call := s.GetCall(); call != nil {
			appendWith(lintMethodCall(call, resolvers, env))
		}
		appendWith(lintRetryStrategy(s.GetRetryStrategy()))

	case flowv1beta2.Node_Action_case:
		a := node.GetAction()
		if w := a.GetWhen(); w != "" {
			if _, err := parseCEL(w); err != nil {
				celDiag("when", err)
			}
		}
		if call := a.GetCall(); call != nil {
			appendWith(lintMethodCall(call, resolvers, env))
		}
		appendWith(lintRetryStrategy(a.GetRetryStrategy()))

	case flowv1beta2.Node_Output_case:
		if _, err := parseCEL(node.GetOutput().GetValue()); err != nil {
			celDiag("value", err)
		}
		appendWith(lintTransforms(node.GetOutput().GetTransforms()))
	}

	return diags
}

// lintRequestSchemas validates request trees for all action/stream calls in
// a graph against their proto schemas. Used by Execute() to catch schema
// issues early without re-parsing CEL expressions.
func lintRequestSchemas(graph *flowv1beta2.Graph, resolvers map[string]shared.Resolver, env shared.Env) *LintResult {
	var diags []LintDiagnostic
	for _, node := range graph.GetNodes() {
		var call *flowv1beta2.MethodCall
		switch node.WhichType() {
		case flowv1beta2.Node_Action_case:
			call = node.GetAction().GetCall()
		case flowv1beta2.Node_Stream_case:
			call = node.GetStream().GetCall()
		}
		if call == nil || call.GetConnection() == "" || call.GetMethod() == "" || call.GetRequest() == nil {
			continue
		}
		for _, d := range lintCallSchema(call, resolvers, env) {
			d.Node = node.GetId()
			diags = append(diags, d)
		}
	}
	return &LintResult{Diagnostics: diags}
}

// lintCallSchema validates a single MethodCall's request tree against the
// proto schema of its method's input message.
func lintCallSchema(call *flowv1beta2.MethodCall, resolvers map[string]shared.Resolver, env shared.Env) []LintDiagnostic {
	resolver, ok := resolvers[call.GetConnection()]
	if !ok {
		return nil
	}
	md, err := resolver.FindMethodByName(protoreflect.FullName(call.GetMethod()))
	if err != nil {
		return nil
	}
	return lintRequestSchema(call.GetRequest(), md.Input(), "request", env)
}

// lintMethodCall validates CEL expressions in a MethodCall's request tree and
// response expression without producing executable programs. When resolvers are
// available, it also validates request field names and literal value types
// against the proto schema of the method's input message.
func lintMethodCall(call *flowv1beta2.MethodCall, resolvers map[string]shared.Resolver, env shared.Env) []LintDiagnostic {
	var diags []LintDiagnostic
	if call.GetConnection() == "" {
		diags = append(diags, LintDiagnostic{
			Severity: SeverityError,
			Path:     "call.connection",
			Message:  "missing connection",
			Code:     CodeMissingField,
		})
	}
	if call.GetMethod() == "" {
		diags = append(diags, LintDiagnostic{
			Severity: SeverityError,
			Path:     "call.method",
			Message:  "missing method",
			Code:     CodeMissingField,
		})
	}
	if call.GetRequest() != nil {
		diags = append(diags, lintRequestTree(call.GetRequest(), "request")...)
	}
	if resp := call.GetResponse(); resp != "" {
		if _, err := parseCEL(resp); err != nil {
			diags = append(diags, LintDiagnostic{
				Severity: SeverityError,
				Path:     "response",
				Message:  err.Error(),
				Code:     CodeInvalidCEL,
			})
		}
	}

	// Schema validation: when a resolver is available for this connection,
	// validate request tree fields against the proto input descriptor.
	if resolvers != nil && call.GetConnection() != "" && call.GetMethod() != "" && call.GetRequest() != nil {
		diags = append(diags, lintCallSchema(call, resolvers, env)...)
	}

	return diags
}

// lintNodeConnection validates that action/stream call.connection references a
// declared connection. Missing connections produce a warning rather than an error
// so that flows using externally provided (mocked) connections still pass lint.
func lintNodeConnection(node *flowv1beta2.Node, connections map[string]bool) *LintDiagnostic {
	var call *flowv1beta2.MethodCall
	switch node.WhichType() {
	case flowv1beta2.Node_Action_case:
		call = node.GetAction().GetCall()
	case flowv1beta2.Node_Stream_case:
		call = node.GetStream().GetCall()
	}
	if call == nil || call.GetConnection() == "" {
		return nil
	}
	if !connections[call.GetConnection()] {
		return &LintDiagnostic{
			Severity: SeverityWarning,
			Node:     node.GetId(),
			Path:     "call.connection",
			Message:  fmt.Sprintf("connection %q not declared in connections", call.GetConnection()),
			Code:     CodeUndeclaredConnection,
		}
	}
	return nil
}

// inputTypeHasDefault checks whether the Input's type variant has a default value set.
func inputTypeHasDefault(inp *flowv1beta2.Input) bool {
	switch inp.WhichType() {
	case flowv1beta2.Input_Bool_case:
		return inp.GetBool().HasDefault()
	case flowv1beta2.Input_Bytes_case:
		return inp.GetBytes().HasDefault()
	case flowv1beta2.Input_Double_case:
		return inp.GetDouble().HasDefault()
	case flowv1beta2.Input_Float_case:
		return inp.GetFloat().HasDefault()
	case flowv1beta2.Input_Int64_case:
		return inp.GetInt64().HasDefault()
	case flowv1beta2.Input_Uint64_case:
		return inp.GetUint64().HasDefault()
	case flowv1beta2.Input_Int32_case:
		return inp.GetInt32().HasDefault()
	case flowv1beta2.Input_Uint32_case:
		return inp.GetUint32().HasDefault()
	case flowv1beta2.Input_String__case:
		return inp.GetString().HasDefault()
	case flowv1beta2.Input_List_case:
		return inp.GetList().GetDefault() != nil
	case flowv1beta2.Input_Map_case:
		return inp.GetMap().GetDefault() != nil
	case flowv1beta2.Input_Message_case:
		return inp.GetMessage().GetDefault() != nil
	}
	return false
}

// lintRequestSchema validates a structpb.Value request tree against a proto
// MessageDescriptor. It checks that field names exist in the schema and that
// literal (non-CEL) value types are compatible with the proto field types.
func lintRequestSchema(v *structpb.Value, md protoreflect.MessageDescriptor, path string, env shared.Env) []LintDiagnostic {
	if v == nil {
		return nil
	}

	switch v.GetKind().(type) {
	case *structpb.Value_StructValue:
		var diags []LintDiagnostic
		for key, field := range v.GetStructValue().GetFields() {
			fd := md.Fields().ByName(protoreflect.Name(key))
			if fd == nil {
				fd = md.Fields().ByJSONName(key)
			}
			if fd == nil {
				diags = append(diags, LintDiagnostic{
					Severity: SeverityError,
					Path:     path + "." + key,
					Message:  fmt.Sprintf("unknown field %q in message %s", key, md.FullName()),
					Code:     CodeUnknownField,
				})
				continue
			}
			diags = append(diags, lintFieldValue(field, fd, path+"."+key, env)...)
		}
		return diags

	case *structpb.Value_StringValue:
		if _, ok := shared.IsValidExpr(v.GetStringValue()); ok {
			if env != nil {
				outType := checkCELOutputType(env, v.GetStringValue())
				if outType != nil {
					kind := outType.Kind()
					if kind != cel.NullTypeKind && kind != cel.StructKind && kind != cel.MapKind {
						return []LintDiagnostic{{
							Severity: SeverityError,
							Path:     path,
							Message:  fmt.Sprintf("CEL expression returns %s, expected message %s", cel.FormatCELType(outType), md.FullName()),
							Code:     CodeTypeMismatch,
						}}
					}
				}
			}
			return nil
		}
		return []LintDiagnostic{{
			Severity: SeverityError,
			Path:     path,
			Message:  fmt.Sprintf("expected message %s, got string literal", md.FullName()),
			Code:     CodeTypeMismatch,
		}}

	case *structpb.Value_NumberValue:
		return []LintDiagnostic{{
			Severity: SeverityError,
			Path:     path,
			Message:  fmt.Sprintf("expected message %s, got number literal", md.FullName()),
			Code:     CodeTypeMismatch,
		}}

	case *structpb.Value_BoolValue:
		return []LintDiagnostic{{
			Severity: SeverityError,
			Path:     path,
			Message:  fmt.Sprintf("expected message %s, got bool literal", md.FullName()),
			Code:     CodeTypeMismatch,
		}}

	case *structpb.Value_ListValue:
		return []LintDiagnostic{{
			Severity: SeverityError,
			Path:     path,
			Message:  fmt.Sprintf("expected message %s, got list", md.FullName()),
			Code:     CodeTypeMismatch,
		}}
	}

	return nil
}

// lintFieldValue validates a structpb.Value against a proto FieldDescriptor,
// checking type compatibility for literal values and recursing into nested
// messages and repeated fields.
func lintFieldValue(v *structpb.Value, fd protoreflect.FieldDescriptor, path string, env shared.Env) []LintDiagnostic {
	if v == nil {
		return nil
	}

	// CEL expressions: validate return type when env is available.
	if sv, ok := v.GetKind().(*structpb.Value_StringValue); ok {
		if _, celOK := shared.IsValidExpr(sv.StringValue); celOK {
			if env != nil {
				outType := checkCELOutputType(env, sv.StringValue)
				if outType != nil {
					kind := outType.Kind()
					if kind == cel.NullTypeKind {
						return nil
					}
					if fd.IsList() && kind != cel.ListKind {
						return []LintDiagnostic{{
							Severity: SeverityError,
							Path:     path,
							Message:  fmt.Sprintf("CEL expression returns %s, expected list for repeated field %q", cel.FormatCELType(outType), fd.Name()),
							Code:     CodeTypeMismatch,
						}}
					}
					if fd.IsMap() && kind != cel.MapKind {
						return []LintDiagnostic{{
							Severity: SeverityError,
							Path:     path,
							Message:  fmt.Sprintf("CEL expression returns %s, expected map for map field %q", cel.FormatCELType(outType), fd.Name()),
							Code:     CodeTypeMismatch,
						}}
					}
					if !fd.IsList() && !fd.IsMap() && !isCELTypeCompatibleWithProtoKind(kind, fd.Kind()) {
						return []LintDiagnostic{{
							Severity: SeverityError,
							Path:     path,
							Message:  fmt.Sprintf("CEL expression returns %s, incompatible with field type %s", cel.FormatCELType(outType), fd.Kind()),
							Code:     CodeTypeMismatch,
						}}
					}
				}
			}
			return nil
		}
	}

	// Repeated (non-map) fields expect a list value.
	if fd.IsList() {
		lv, ok := v.GetKind().(*structpb.Value_ListValue)
		if !ok {
			return []LintDiagnostic{{
				Severity: SeverityError,
				Path:     path,
				Message:  fmt.Sprintf("field %q is repeated; expected list, got %s", fd.Name(), valueKindName(v)),
				Code:     CodeTypeMismatch,
			}}
		}
		var diags []LintDiagnostic
		for i, elem := range lv.ListValue.GetValues() {
			diags = append(diags, lintScalarFieldValue(elem, fd, fmt.Sprintf("%s[%d]", path, i), env)...)
		}
		return diags
	}

	// Map fields expect a struct value.
	if fd.IsMap() {
		sv, ok := v.GetKind().(*structpb.Value_StructValue)
		if !ok {
			return []LintDiagnostic{{
				Severity: SeverityError,
				Path:     path,
				Message:  fmt.Sprintf("field %q is map; expected struct, got %s", fd.Name(), valueKindName(v)),
				Code:     CodeTypeMismatch,
			}}
		}
		valFD := fd.MapValue()
		var diags []LintDiagnostic
		for key, val := range sv.StructValue.GetFields() {
			diags = append(diags, lintScalarFieldValue(val, valFD, fmt.Sprintf("%s[%q]", path, key), env)...)
		}
		return diags
	}

	return lintScalarFieldValue(v, fd, path, env)
}

// lintScalarFieldValue validates a single (non-repeated, non-map) value against a
// proto FieldDescriptor.
func lintScalarFieldValue(v *structpb.Value, fd protoreflect.FieldDescriptor, path string, env shared.Env) []LintDiagnostic {
	if v == nil {
		return nil
	}

	// CEL expressions: validate return type when env is available.
	if sv, ok := v.GetKind().(*structpb.Value_StringValue); ok {
		if _, celOK := shared.IsValidExpr(sv.StringValue); celOK {
			if env != nil {
				outType := checkCELOutputType(env, sv.StringValue)
				if outType != nil && outType.Kind() != cel.NullTypeKind {
					if !isCELTypeCompatibleWithProtoKind(outType.Kind(), fd.Kind()) {
						return []LintDiagnostic{{
							Severity: SeverityError,
							Path:     path,
							Message:  fmt.Sprintf("CEL expression returns %s, incompatible with field type %s", cel.FormatCELType(outType), fd.Kind()),
							Code:     CodeTypeMismatch,
						}}
					}
				}
			}
			return nil
		}
	}

	kind := fd.Kind()

	switch v.GetKind().(type) {
	case *structpb.Value_StringValue:
		if !isStringCompatibleKind(kind) {
			return []LintDiagnostic{{
				Severity: SeverityError,
				Path:     path,
				Message:  fmt.Sprintf("string literal incompatible with field type %s", kind),
				Code:     CodeTypeMismatch,
			}}
		}

	case *structpb.Value_NumberValue:
		if !isNumberCompatibleKind(kind) {
			return []LintDiagnostic{{
				Severity: SeverityError,
				Path:     path,
				Message:  fmt.Sprintf("number literal incompatible with field type %s", kind),
				Code:     CodeTypeMismatch,
			}}
		}

	case *structpb.Value_BoolValue:
		if kind != protoreflect.BoolKind {
			return []LintDiagnostic{{
				Severity: SeverityError,
				Path:     path,
				Message:  fmt.Sprintf("bool literal incompatible with field type %s", kind),
				Code:     CodeTypeMismatch,
			}}
		}

	case *structpb.Value_StructValue:
		if kind != protoreflect.MessageKind && kind != protoreflect.GroupKind {
			return []LintDiagnostic{{
				Severity: SeverityError,
				Path:     path,
				Message:  fmt.Sprintf("struct incompatible with field type %s", kind),
				Code:     CodeTypeMismatch,
			}}
		}
		return lintRequestSchema(v, fd.Message(), path, env)

	case *structpb.Value_ListValue:
		return []LintDiagnostic{{
			Severity: SeverityError,
			Path:     path,
			Message:  fmt.Sprintf("unexpected list for singular field %q of type %s", fd.Name(), kind),
			Code:     CodeTypeMismatch,
		}}

	case *structpb.Value_NullValue:
		// null is acceptable for any optional/message field.
	}

	return nil
}

// isStringCompatibleKind returns true if a proto field kind can accept a string literal.
func isStringCompatibleKind(kind protoreflect.Kind) bool {
	switch kind {
	case protoreflect.StringKind,
		protoreflect.BytesKind,
		protoreflect.EnumKind:
		return true
	}
	return false
}

// isNumberCompatibleKind returns true if a proto field kind can accept a number literal.
func isNumberCompatibleKind(kind protoreflect.Kind) bool {
	switch kind {
	case protoreflect.FloatKind,
		protoreflect.DoubleKind,
		protoreflect.Int32Kind,
		protoreflect.Int64Kind,
		protoreflect.Uint32Kind,
		protoreflect.Uint64Kind,
		protoreflect.Sint32Kind,
		protoreflect.Sint64Kind,
		protoreflect.Fixed32Kind,
		protoreflect.Fixed64Kind,
		protoreflect.Sfixed32Kind,
		protoreflect.Sfixed64Kind,
		protoreflect.EnumKind:
		return true
	}
	return false
}

// valueKindName returns a human-readable name for a structpb.Value kind.
func valueKindName(v *structpb.Value) string {
	switch v.GetKind().(type) {
	case *structpb.Value_NullValue:
		return "null"
	case *structpb.Value_BoolValue:
		return "bool"
	case *structpb.Value_NumberValue:
		return "number"
	case *structpb.Value_StringValue:
		return "string"
	case *structpb.Value_StructValue:
		return "struct"
	case *structpb.Value_ListValue:
		return "list"
	}
	return "unknown"
}

// isCELTypeCompatibleWithProtoKind checks whether a CEL type kind is compatible
// with a proto field kind for assignment purposes.
func isCELTypeCompatibleWithProtoKind(celKind cel.Kind, protoKind protoreflect.Kind) bool {
	switch celKind {
	case cel.BoolKind:
		return protoKind == protoreflect.BoolKind
	case cel.IntKind, cel.UintKind, cel.DoubleKind:
		return isNumberCompatibleKind(protoKind)
	case cel.StringKind:
		return isStringCompatibleKind(protoKind)
	case cel.BytesKind:
		return protoKind == protoreflect.BytesKind || protoKind == protoreflect.StringKind
	case cel.StructKind, cel.MapKind:
		return protoKind == protoreflect.MessageKind || protoKind == protoreflect.GroupKind
	case cel.ListKind:
		return false // a list is not compatible with a singular field
	default:
		return true // unknown CEL kind, allow
	}
}

// lintProtoConflicts detects proto message types with the same fully-qualified
// name registered by different connections. When two connections provide
// different definitions for the same type, only one will be used in CEL
// evaluation -- which can cause subtle field resolution bugs at runtime.
func lintProtoConflicts(resolvers map[string]shared.Resolver) []LintDiagnostic {
	// Track which connection first registered each message FQN.
	type owner struct {
		connID string
		file   string
	}
	seen := make(map[protoreflect.FullName]owner)
	var diags []LintDiagnostic

	for connID, resolver := range resolvers {
		resolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
			collectMessages(fd.Messages(), func(fqn protoreflect.FullName) {
				prev, exists := seen[fqn]
				if !exists {
					seen[fqn] = owner{connID: connID, file: string(fd.Path())}
					return
				}
				if prev.connID == connID || prev.file == string(fd.Path()) {
					return
				}
				diags = append(diags, LintDiagnostic{
					Severity: SeverityWarning,
					Node:     "connections." + connID,
					Message: fmt.Sprintf(
						"proto type %q also registered by connection %q (from %s); first registration wins",
						fqn, prev.connID, prev.file,
					),
					Code: CodeProtoConflict,
				})
			})
			return true
		})
	}
	return diags
}

// collectMessages recursively walks a MessageDescriptors list and calls fn
// with the fully-qualified name of each message (including nested types).
func collectMessages(msgs protoreflect.MessageDescriptors, fn func(protoreflect.FullName)) {
	for i := range msgs.Len() {
		md := msgs.Get(i)
		fn(md.FullName())
		collectMessages(md.Messages(), fn)
	}
}
