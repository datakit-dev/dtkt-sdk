package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc/mock"
)

// requireDiagnostic asserts that result contains at least one diagnostic
// with the given code. Use alongside assert.Contains on result.Error() so
// the suite guards BOTH the stable structured code AND the user-visible
// message text. The two layers fail independently if message phrasing or
// code-enum naming drifts.
func requireDiagnostic(t *testing.T, result *LintResult, code string) {
	t.Helper()
	for _, d := range result.Diagnostics {
		if d.Code == code {
			return
		}
	}
	t.Fatalf("expected diagnostic with code %q; got: %v", code, result.Diagnostics)
}

func TestLint_ValidGraph(t *testing.T) {
	graph := loadFlow(t, "lint_valid.yaml")

	result := Lint(graph)
	require.Empty(t, result.Diagnostics)
}

func TestLint_InvalidVarCEL(t *testing.T) {
	graph := loadFlow(t, "lint_invalid_var_cel.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeInvalidCEL)
	assert.Contains(t, result.Error(), "vars.bad")
}

func TestLint_InvalidOutputCEL(t *testing.T) {
	graph := loadFlow(t, "lint_invalid_output_cel.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeInvalidCEL)
	assert.Contains(t, result.Error(), "outputs.bad")
}

func TestLint_InvalidTransformCEL(t *testing.T) {
	graph := loadFlow(t, "lint_invalid_transform_cel.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeInvalidCEL)
	assert.Contains(t, result.Error(), "vars.bad")
}

// TestLint_TransformReferencesGraphNode rejects transform expressions
// that reference any graph-node category (inputs, vars, interactions,
// etc.). Transforms see only `this`; graph-aware logic belongs in the
// producer's main expression. Reproduces the user-reported pattern
// where an output filter referenced interactions.confirmDiscard.value
// and silently failed at eval time.
func TestLint_TransformReferencesGraphNode(t *testing.T) {
	graph := loadFlow(t, "lint_transform_references_graph.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeInvalidCEL)
	assert.Contains(t, result.Error(), "transforms[0].filter")
	assert.Contains(t, result.Error(), "may only reference `this`")
	assert.Contains(t, result.Error(), "interactions.confirmDiscard")
}

func TestLint_InvalidSwitchCEL(t *testing.T) {
	graph := loadFlow(t, "lint_invalid_switch_cel.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeInvalidCEL)
	assert.Contains(t, result.Error(), "switch.value")
}

func TestLint_MultipleErrors(t *testing.T) {
	graph := loadFlow(t, "lint_multiple_errors.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeInvalidCEL)
	assert.Contains(t, result.Error(), "vars.bad1")
	assert.Contains(t, result.Error(), "outputs.bad2")
}

func TestLint_OrphanedNodeWarning(t *testing.T) {
	graph := loadFlow(t, "lint_orphaned_node.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeOrphanedNode)
	assert.Contains(t, result.Error(), "vars.unused")
	assert.Contains(t, result.Error(), "orphaned node has no consumers")
}

func TestLint_SideEffectNodeNotOrphaned(t *testing.T) {
	graph := loadFlow(t, "lint_side_effect_not_orphaned.yaml")

	result := Lint(graph)
	require.Empty(t, result.Diagnostics)
}

func TestLint_InvalidActionWhen(t *testing.T) {
	graph := loadFlow(t, "lint_invalid_action_when.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeInvalidCEL)
	assert.Contains(t, result.Error(), "actions.bad")
	assert.Contains(t, result.Error(), "when")
}

func TestLint_NoUpstreamDependencies(t *testing.T) {
	graph := loadFlow(t, "lint_no_upstream.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeNoUpstream)
	assert.Contains(t, result.Error(), "streams.echo")
	assert.Contains(t, result.Error(), "has no upstream dependencies")
}

func TestLint_InvalidRetryStrategyCEL(t *testing.T) {
	graph := loadFlow(t, "lint_invalid_retry_cel.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeInvalidCEL)
	assert.Contains(t, result.Error(), "actions.bad")
	assert.Contains(t, result.Error(), "retry_strategy.skip_when")
}

func TestLint_ValidConnection(t *testing.T) {
	graph := loadFlow(t, "lint_valid_connection.yaml")

	result := Lint(graph)
	require.Empty(t, result.Diagnostics)
}

// Connection.package and Connection.services are mutually exclusive (oneof
// at the message level). This fixture covers the package branch.
func TestLint_ValidConnection_Package(t *testing.T) {
	graph := loadFlow(t, "lint_valid_connection_package.yaml")

	result := Lint(graph)
	require.Empty(t, result.Diagnostics)
}

func TestLint_UndeclaredConnection(t *testing.T) {
	graph := loadFlow(t, "lint_undeclared_connection.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeUndeclaredConnection)
	assert.Contains(t, result.Error(), "warning")
	assert.Contains(t, result.Error(), "mocked")
	assert.Contains(t, result.Error(), "not declared")
}

// --- Schema validation tests ---

func TestLint_SchemaValid(t *testing.T) {
	graph := loadFlow(t, "lint_schema_valid.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.Empty(t, result.Diagnostics)
}

func TestLint_SchemaUnknownField(t *testing.T) {
	graph := loadFlow(t, "lint_schema_unknown_field.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeUnknownField)
	assert.Contains(t, result.Error(), "unknown field")
	assert.Contains(t, result.Error(), "nonexistent")
}

func TestLint_SchemaTypeMismatch(t *testing.T) {
	graph := loadFlow(t, "lint_schema_type_mismatch.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeTypeMismatch)
	assert.Contains(t, result.Error(), "request.name")
	assert.Contains(t, result.Error(), "number literal incompatible")
	assert.Contains(t, result.Error(), "request.count")
	assert.Contains(t, result.Error(), "string literal incompatible")
}

func TestLint_SchemaRepeatedNotList(t *testing.T) {
	graph := loadFlow(t, "lint_schema_repeated_not_list.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeTypeMismatch)
	assert.Contains(t, result.Error(), "tags")
	assert.Contains(t, result.Error(), "repeated")
}

func TestLint_SchemaCELInField(t *testing.T) {
	graph := loadFlow(t, "lint_schema_cel_in_field.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.Empty(t, result.Diagnostics)
}

func TestLint_SchemaWithoutResolver(t *testing.T) {
	// Without resolvers, schema validation is skipped -- no error even with bad fields.
	graph := loadFlow(t, "lint_schema_unknown_field.yaml")

	result := Lint(graph)
	require.Empty(t, result.Diagnostics)
}

// --- Response-side schema validation (P6) ---

func TestLint_ResponseUnknownField(t *testing.T) {
	graph := loadFlow(t, "lint_response_unknown_field.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeUnknownField)
	assert.Contains(t, result.Error(), "response")
	assert.Contains(t, result.Error(), "bogus_field_xyz")
}

func TestLint_ResponseValidField(t *testing.T) {
	graph := loadFlow(t, "lint_response_valid_field.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.Empty(t, result.Diagnostics)
}

func TestLint_ResponseValidNestedField(t *testing.T) {
	graph := loadFlow(t, "lint_response_valid_nested.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.Empty(t, result.Diagnostics)
}

func TestLint_ResponseNestedUnknownField(t *testing.T) {
	graph := loadFlow(t, "lint_response_nested_unknown_field.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeUnknownField)
}

func TestLint_ResponseWithoutResolver(t *testing.T) {
	// Without resolvers, response schema validation is skipped -- no error
	// even with bogus field accesses. Matches the request-side
	// SchemaWithoutResolver contract.
	graph := loadFlow(t, "lint_response_unknown_field.yaml")

	result := Lint(graph)
	require.Empty(t, result.Diagnostics)
}

func TestLint_RequestTopLevelCELWrongType(t *testing.T) {
	// request: "= test.TestNested{...}" returns a concrete proto struct that
	// is NOT TestRequest -- the identity check (P6 item 6) fires.
	graph := loadFlow(t, "lint_request_toplevel_cel_wrong_type.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeTypeMismatch)
	assert.Contains(t, result.Error(), "test.TestNested")
	assert.Contains(t, result.Error(), "test.TestRequest")
}

func TestLint_RequestTopLevelCELRightType(t *testing.T) {
	// request: "= test.TestRequest{...}" - clean (false-positive guard).
	graph := loadFlow(t, "lint_request_toplevel_cel_right_type.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.Empty(t, result.Diagnostics)
}

func TestLint_SchemaCELTypeMismatch(t *testing.T) {
	graph := loadFlow(t, "lint_schema_cel_type_mismatch.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeTypeMismatch)
	assert.Contains(t, result.Error(), "request.name")
	assert.Contains(t, result.Error(), "CEL expression returns int")
	assert.Contains(t, result.Error(), "request.count")
	assert.Contains(t, result.Error(), "CEL expression returns bool")
}

// --- P14: Node-ref selector chain validation ---

func TestLint_NodeRef_ActionValueUnknownField(t *testing.T) {
	graph := loadFlow(t, "lint_noderef_action_value_unknown.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeUnknownField)
	assert.Contains(t, result.Error(), "bogus_field")
	assert.Contains(t, result.Error(), "test.TestResponse")
}

func TestLint_NodeRef_ActionValueValidChain(t *testing.T) {
	graph := loadFlow(t, "lint_noderef_action_value_valid.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.Empty(t, result.Diagnostics)
}

func TestLint_NodeRef_WrapperFieldUnknown(t *testing.T) {
	// `actions.run.bogus_wrapper` - bogus_wrapper isn't in the per-category
	// whitelist (only `value` and `eval_count` for actions).
	graph := loadFlow(t, "lint_noderef_wrapper_unknown.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeUnknownField)
	assert.Contains(t, result.Error(), "bogus_wrapper")
}

func TestLint_NodeRef_VarTransitiveChain(t *testing.T) {
	// `vars.passthrough = actions.run.value`, then a downstream
	// `vars.passthrough.value.bogus_field` should resolve transitively
	// through the var to the action's response descriptor.
	graph := loadFlow(t, "lint_noderef_var_chain.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeUnknownField)
	assert.Contains(t, result.Error(), "bogus_field")
}

func TestLint_NodeRef_RequestChain(t *testing.T) {
	// A downstream action's request leaf references an upstream action's
	// `.value.bogus_field`. P14 should catch this -- it's the same chain
	// pattern, just in a request site instead of an output.
	graph := loadFlow(t, "lint_noderef_request_chain.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeUnknownField)
	assert.Contains(t, result.Error(), "bogus_field")
}

func TestLint_NodeRef_WithoutResolver(t *testing.T) {
	// No resolvers -> no chain validation; the bogus reference passes.
	// Matches the P6 conns-absent contract; this is the "zero false
	// positives without ground truth" guardrail.
	graph := loadFlow(t, "lint_noderef_action_value_unknown.yaml")

	result := Lint(graph)
	require.Empty(t, result.Diagnostics)
}

func TestLint_NodeRef_StreamValueUnknownField(t *testing.T) {
	// Streams use the same callTypeInfo code path as actions; this
	// guards against the regression of "P14 quietly forgot streams."
	graph := loadFlow(t, "lint_noderef_stream_value_unknown.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeUnknownField)
	assert.Contains(t, result.Error(), "bogus_field")
	assert.Contains(t, result.Error(), "test.TestResponse")
}

func TestLint_NodeRef_InputMessageChain(t *testing.T) {
	// An input declared with `message: { type: test.TestRequest }` gets
	// its descriptor resolved via inputTypeInfo, so chains like
	// `inputs.req.value.<field>` are walkable.
	graph := loadFlow(t, "lint_noderef_input_message_chain.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeUnknownField)
	assert.Contains(t, result.Error(), "bogus_field")
	assert.Contains(t, result.Error(), "test.TestRequest")
}

func TestLint_NodeRef_BareValueAndWrapper(t *testing.T) {
	// `= actions.run.value` (no chain past .value) and
	// `= actions.run.eval_count` (valid non-value wrapper) must both
	// lint clean. Guards against the regression of "walker emits on
	// short chains" or "walker rejects valid non-value wrappers".
	graph := loadFlow(t, "lint_noderef_bare_value.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.Empty(t, result.Diagnostics)
}

func TestLint_NodeRef_ListFieldStopsWalk(t *testing.T) {
	// `inputs.req.value.tags.bogus_field` walks: req -> TestRequest;
	// .tags is REPEATED string -> walker MUST stop (no false positive).
	// The expression is nonsensical at runtime (you'd index .tags first)
	// but the zero-false-positives invariant says: stop walking when
	// type info is lost, even at the cost of missing a typo.
	graph := loadFlow(t, "lint_noderef_list_stop.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.Empty(t, result.Diagnostics)
}

func TestLint_NodeRef_CallResponseTransform(t *testing.T) {
	// When `call.response: "= this.response"` (identity transform), the
	// action's .value is still md.Output() shape; the walker should
	// continue to catch `.value.bogus_field`. This exercises the
	// responseExprTypeInfo path (compile call.response + read
	// OutputType + look up descriptor).
	graph := loadFlow(t, "lint_noderef_call_response_transform.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	requireDiagnostic(t, result, CodeUnknownField)
	assert.Contains(t, result.Error(), "bogus_field")
}

// --- Test resolver ---

// newConflictResolver builds a test resolver with a "test.Shared" message
// type from a uniquely-named file. Two resolvers built with different
// filenames but the same package/message name simulate a proto conflict.
func newConflictResolver(t *testing.T, fileName string) *flowResolver {
	t.Helper()
	file := buildSyntheticFile(t, syntheticFileSpec{
		fileName:    fileName,
		packageName: "test",
		messages:    []syntheticMessage{{name: "Shared"}},
	})
	return newFlowResolver(mock.NewClient(), file)
}

func TestLint_ProtoConflict(t *testing.T) {
	graph := loadFlow(t, "lint_valid.yaml")
	resolvers := map[string]shared.Resolver{
		"conn_a": newConflictResolver(t, "a/shared.proto"),
		"conn_b": newConflictResolver(t, "b/shared.proto"),
	}
	result := Lint(graph, resolvers)
	var found bool
	for _, d := range result.Diagnostics {
		if d.Code == CodeProtoConflict {
			found = true
			assert.Equal(t, SeverityWarning, d.Severity)
			assert.Contains(t, d.Message, "test.Shared")
		}
	}
	assert.True(t, found, "expected proto-conflict diagnostic for test.Shared")
}

func TestLint_ProtoNoConflictSameFile(t *testing.T) {
	graph := loadFlow(t, "lint_valid.yaml")
	r := newTestResolver(t)
	resolvers := map[string]shared.Resolver{
		"conn_a": r,
		"conn_b": r,
	}
	result := Lint(graph, resolvers)
	for _, d := range result.Diagnostics {
		assert.NotEqual(t, CodeProtoConflict, d.Code, "same resolver should not cause conflict")
	}
}

// newTestResolver builds a flowResolver with a synthetic test.Service
// descriptor: TestRequest{name, count, active, tags, nested} and
// TestResponse{payload, count, nested} (response gets fields so the P6
// response-schema lint can exercise valid + invalid selector chains).
// Used by lint tests that exercise schema validation against a known shape.
func newTestResolver(t *testing.T) *flowResolver {
	t.Helper()
	file := buildSyntheticFile(t, syntheticFileSpec{
		fileName:    "test.proto",
		packageName: "test",
		messages: []syntheticMessage{
			{
				name: "TestRequest",
				fields: []syntheticField{
					{name: "name", number: 1, fieldType: descriptorpb.FieldDescriptorProto_TYPE_STRING},
					{name: "count", number: 2, fieldType: descriptorpb.FieldDescriptorProto_TYPE_INT32},
					{name: "active", number: 3, fieldType: descriptorpb.FieldDescriptorProto_TYPE_BOOL},
					{name: "tags", number: 4, fieldType: descriptorpb.FieldDescriptorProto_TYPE_STRING, repeated: true},
					{name: "nested", number: 5, fieldType: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, typeName: ".test.TestNested"},
				},
			},
			{
				name: "TestNested",
				fields: []syntheticField{
					{name: "value", number: 1, fieldType: descriptorpb.FieldDescriptorProto_TYPE_STRING},
				},
			},
			{
				name: "TestResponse",
				fields: []syntheticField{
					{name: "payload", number: 1, fieldType: descriptorpb.FieldDescriptorProto_TYPE_STRING},
					{name: "count", number: 2, fieldType: descriptorpb.FieldDescriptorProto_TYPE_INT32},
					{name: "nested", number: 3, fieldType: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, typeName: ".test.TestNested"},
				},
			},
		},
		services: []syntheticService{
			{
				name: "Service",
				methods: []syntheticMethod{
					{name: "Do", inputType: ".test.TestRequest", outputType: ".test.TestResponse"},
				},
			},
		},
	})
	return newFlowResolver(mock.NewClient(), file)
}
