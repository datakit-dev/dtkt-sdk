package runtime

import (
	"fmt"
	"testing"

	"buf.build/go/protovalidate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
)

func TestLint_ValidGraph(t *testing.T) {
	graph := loadFlow(t, "lint_valid.yaml")

	result := Lint(graph)
	require.Empty(t, result.Diagnostics)
}

func TestLint_InvalidVarCEL(t *testing.T) {
	graph := loadFlow(t, "lint_invalid_var_cel.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	assert.Contains(t, result.Error(), "vars.bad")
}

func TestLint_InvalidOutputCEL(t *testing.T) {
	graph := loadFlow(t, "lint_invalid_output_cel.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	assert.Contains(t, result.Error(), "outputs.bad")
}

func TestLint_InvalidTransformCEL(t *testing.T) {
	graph := loadFlow(t, "lint_invalid_transform_cel.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	assert.Contains(t, result.Error(), "vars.bad")
}

func TestLint_InvalidSwitchCEL(t *testing.T) {
	graph := loadFlow(t, "lint_invalid_switch_cel.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	assert.Contains(t, result.Error(), "switch.value")
}

func TestLint_MultipleErrors(t *testing.T) {
	graph := loadFlow(t, "lint_multiple_errors.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	assert.Contains(t, result.Error(), "vars.bad1")
	assert.Contains(t, result.Error(), "outputs.bad2")
}

func TestLint_InputConstantAndDefaultMutuallyExclusive(t *testing.T) {
	graph := loadFlow(t, "lint_constant_and_default.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	assert.Contains(t, result.Error(), "constant and default are mutually exclusive")
}

func TestLint_OrphanedNodeWarning(t *testing.T) {
	graph := loadFlow(t, "lint_orphaned_node.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
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
	assert.Contains(t, result.Error(), "actions.bad")
	assert.Contains(t, result.Error(), "when")
}

func TestLint_InvalidStreamCloseRequestWhen(t *testing.T) {
	graph := loadFlow(t, "lint_invalid_stream_close_when.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	assert.Contains(t, result.Error(), "streams.bad")
	assert.Contains(t, result.Error(), "close_request_when")
}

func TestLint_NoUpstreamDependencies(t *testing.T) {
	graph := loadFlow(t, "lint_no_upstream.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	assert.Contains(t, result.Error(), "streams.echo")
	assert.Contains(t, result.Error(), "has no upstream dependencies")
}

func TestLint_InvalidRetryStrategyCEL(t *testing.T) {
	graph := loadFlow(t, "lint_invalid_retry_cel.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
	assert.Contains(t, result.Error(), "actions.bad")
	assert.Contains(t, result.Error(), "retry_strategy.skip_when")
}

func TestLint_ValidConnection(t *testing.T) {
	graph := loadFlow(t, "lint_valid_connection.yaml")

	result := Lint(graph)
	require.Empty(t, result.Diagnostics)
}

func TestLint_UndeclaredConnection(t *testing.T) {
	graph := loadFlow(t, "lint_undeclared_connection.yaml")

	result := Lint(graph)
	require.NotEmpty(t, result.Diagnostics)
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
	assert.Contains(t, result.Error(), "unknown field")
	assert.Contains(t, result.Error(), "nonexistent")
}

func TestLint_SchemaTypeMismatch(t *testing.T) {
	graph := loadFlow(t, "lint_schema_type_mismatch.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
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

func TestLint_SchemaCELTypeMismatch(t *testing.T) {
	graph := loadFlow(t, "lint_schema_cel_type_mismatch.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.NotEmpty(t, result.Diagnostics)
	assert.Contains(t, result.Error(), "request.name")
	assert.Contains(t, result.Error(), "CEL expression returns int")
	assert.Contains(t, result.Error(), "request.count")
	assert.Contains(t, result.Error(), "CEL expression returns bool")
}

func TestLint_SchemaEOFInRequest(t *testing.T) {
	// EOF() is a CEL expression -- schema validation should skip it.
	graph := loadFlow(t, "lint_schema_eof_request.yaml")
	resolvers := map[string]shared.Resolver{"myconn": newTestResolver(t)}

	result := Lint(graph, resolvers)
	require.Empty(t, result.Diagnostics)
}

// --- Test resolver ---

// newConflictResolver builds a test resolver with a "test.Shared" message type
// from a uniquely-named file. Two resolvers built with different filenames but
// the same package/message name simulate a proto conflict.
func newConflictResolver(t *testing.T, fileName string) *testResolver {
	t.Helper()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    proto.String(fileName),
		Syntax:  proto.String("proto3"),
		Package: proto.String("test"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("Shared")},
		},
	}
	file, err := protodesc.NewFile(fd, nil)
	if err != nil {
		t.Fatalf("building conflict descriptor: %v", err)
	}
	return &testResolver{file: file}
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

// testResolver is a minimal shared.Resolver for lint tests. It builds a proto
// descriptor with a known message schema: TestRequest{name string, count int32,
// active bool, tags repeated string, nested TestNested{value string}}.
type testResolver struct {
	file protoreflect.FileDescriptor
}

func newTestResolver(t *testing.T) *testResolver {
	t.Helper()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Syntax:  proto.String("proto3"),
		Package: proto.String("test"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("TestRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: proto.String("name"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()},
					{Name: proto.String("count"), Number: proto.Int32(2), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()},
					{Name: proto.String("active"), Number: proto.Int32(3), Type: descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()},
					{Name: proto.String("tags"), Number: proto.Int32(4), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()},
					{Name: proto.String("nested"), Number: proto.Int32(5), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".test.TestNested"), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()},
				},
			},
			{
				Name: proto.String("TestNested"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: proto.String("value"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()},
				},
			},
			{Name: proto.String("TestResponse")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("Service"),
			Method: []*descriptorpb.MethodDescriptorProto{
				{
					Name:            proto.String("Do"),
					InputType:       proto.String(".test.TestRequest"),
					OutputType:      proto.String(".test.TestResponse"),
					ClientStreaming: proto.Bool(false),
					ServerStreaming: proto.Bool(false),
				},
			},
		}},
	}
	file, err := protodesc.NewFile(fd, nil)
	if err != nil {
		t.Fatalf("building test descriptor: %v", err)
	}
	return &testResolver{file: file}
}

func (r *testResolver) FindMethodByName(name protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
	for i := range r.file.Services().Len() {
		svc := r.file.Services().Get(i)
		for j := range svc.Methods().Len() {
			md := svc.Methods().Get(j)
			if md.FullName() == name {
				return md, nil
			}
		}
	}
	return nil, fmt.Errorf("method %q not found", name)
}

func (r *testResolver) FindMessageByName(protoreflect.FullName) (protoreflect.MessageType, error) {
	return nil, protoregistry.NotFound
}
func (r *testResolver) FindMessageByURL(string) (protoreflect.MessageType, error) {
	return nil, protoregistry.NotFound
}
func (r *testResolver) FindExtensionByName(protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}
func (r *testResolver) FindExtensionByNumber(protoreflect.FullName, protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}
func (r *testResolver) RangeServices(func(protoreflect.ServiceDescriptor) bool) {}
func (r *testResolver) RangeMethods(func(protoreflect.MethodDescriptor) bool)   {}
func (r *testResolver) RangeFiles(f func(protoreflect.FileDescriptor) bool) {
	f(r.file)
}
func (r *testResolver) GetValidator() (protovalidate.Validator, error) {
	return nil, fmt.Errorf("test: validator not available")
}
