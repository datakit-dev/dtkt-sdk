package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
)

// CEL access into nested values fed through inputs.
//
// These tests exercise the conversion pipeline at flow-runtime boundaries:
// Go native -> NativeToValue -> ValueAsProto -> expr.Value (wire form) ->
// ProtoAsValue -> ref.Val (CEL activation). The CEL expressions assert that
// nested fields, list elements, and well-known proto wrappers
// (structpb.Struct, structpb.Value) are reachable via dotted/indexed path
// syntax once unwrapped into the activation.

func TestGraph_CEL_Map_TopLevelField(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_struct_subfield.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", map[string]any{
			"name": "alice",
			"age":  int64(30),
		})

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, "alice", results[0].GetValue().GetStringValue())
	})
}

func TestGraph_CEL_Map_DeepNestedField(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_struct_nested.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", map[string]any{
			"name": "alice",
			"address": map[string]any{
				"city": "SF",
				"zip":  "94103",
			},
		})

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, "SF", results[0].GetValue().GetStringValue())
	})
}

func TestGraph_CEL_List_Index(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_list_index.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", []any{"a", "b", "c"})

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, "b", results[0].GetValue().GetStringValue())
	})
}

func TestGraph_CEL_Struct_WithListOfStructs(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_struct_with_list.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", map[string]any{
			"items": []any{
				map[string]any{"id": "first", "n": int64(1)},
				map[string]any{"id": "second", "n": int64(2)},
				map[string]any{"id": "third", "n": int64(3)},
			},
		})

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, "second", results[0].GetValue().GetStringValue())
	})
}

func TestGraph_CEL_Size_OfNestedList(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_size_nested.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", map[string]any{
			"tags": []any{"a", "b", "c", "d"},
		})

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(4), results[0].GetValue().GetInt64Value())
	})
}

func TestGraph_CEL_StructpbStruct_Subfield(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_struct_subfield.yaml")

		s, err := structpb.NewStruct(map[string]any{
			"name": "bob",
			"age":  42,
		})
		require.NoError(t, err)

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", s)

		ctx := testContext(t)
		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, "bob", results[0].GetValue().GetStringValue())
	})
}

func TestGraph_CEL_StructpbValue_Subfield(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_struct_subfield.yaml")

		s, err := structpb.NewStruct(map[string]any{
			"name": "carol",
		})
		require.NoError(t, err)
		// Wrap the Struct inside google.protobuf.Value to verify the Value
		// wrapper is unwrapped during conversion.
		v := structpb.NewStructValue(s)

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", v)

		ctx := testContext(t)
		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, "carol", results[0].GetValue().GetStringValue())
	})
}

func TestGraph_CEL_Has_OptionalField(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_has_optional.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Two values: one with the nickname field set, one without.
		feedInput(ps, "inputs.x",
			map[string]any{"name": "with-field", "nickname": "ace"},
			map[string]any{"name": "no-field"},
		)

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 2)
		assert.Equal(t, "ace", results[0].GetValue().GetStringValue())
		assert.Equal(t, "anonymous", results[1].GetValue().GetStringValue())
	})
}

// --- Numeric type pinning -----------------------------------------------------

func TestGraph_CEL_Numeric_GoIntStaysInt64(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_numeric_int.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", map[string]any{"count": int64(7)})

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		// Go int64 in a map -> CEL int -> wire Int64Value.
		assert.Equal(t, int64(7), results[0].GetValue().GetInt64Value())
	})
}

func TestGraph_CEL_Numeric_StructpbStaysDouble(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_numeric_int.yaml")

		s, err := structpb.NewStruct(map[string]any{"count": 7})
		require.NoError(t, err)

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", s)

		ctx := testContext(t)
		err = NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		// structpb.Value wraps all numbers as double; the wire form preserves
		// that.
		assert.Equal(t, 7.0, results[0].GetValue().GetDoubleValue())
	})
}

// --- Missing-field semantics --------------------------------------------------

func TestGraph_CEL_MissingField_NoHas_Errors(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_missing_field.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", map[string]any{"name": "alice"})

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no such key")
	})
}

// --- Well-known types ---------------------------------------------------------

func TestGraph_CEL_WKT_Timestamp_Comparison(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_wkt_timestamp.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		later := timestamppb.New(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
		earlier := timestamppb.New(time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC))

		feedInput(ps, "inputs.x", later, earlier)

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 2)
		assert.True(t, results[0].GetValue().GetBoolValue())
		assert.False(t, results[1].GetValue().GetBoolValue())
	})
}

func TestGraph_CEL_WKT_Duration_Comparison(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_wkt_duration.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", durationpb.New(90*time.Second), durationpb.New(30*time.Second))

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 2)
		assert.True(t, results[0].GetValue().GetBoolValue())
		assert.False(t, results[1].GetValue().GetBoolValue())
	})
}

// --- Comprehension macros over nested values ---------------------------------

func TestGraph_CEL_Comprehension_All(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_comprehension_all.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x",
			map[string]any{"items": []any{
				map[string]any{"n": int64(1)},
				map[string]any{"n": int64(2)},
				map[string]any{"n": int64(3)},
			}},
			map[string]any{"items": []any{
				map[string]any{"n": int64(1)},
				map[string]any{"n": int64(-1)},
			}},
		)

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 2)
		assert.True(t, results[0].GetValue().GetBoolValue())
		assert.False(t, results[1].GetValue().GetBoolValue())
	})
}

func TestGraph_CEL_Comprehension_Exists(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_comprehension_exists.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x",
			map[string]any{"items": []any{
				map[string]any{"name": "alpha"},
				map[string]any{"name": "needle"},
			}},
			map[string]any{"items": []any{
				map[string]any{"name": "alpha"},
				map[string]any{"name": "beta"},
			}},
		)

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 2)
		assert.True(t, results[0].GetValue().GetBoolValue())
		assert.False(t, results[1].GetValue().GetBoolValue())
	})
}

func TestGraph_CEL_Comprehension_Filter(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_comprehension_filter.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", map[string]any{"items": []any{
			map[string]any{"name": "a", "n": int64(1)},
			map[string]any{"name": "b", "n": int64(2)},
			map[string]any{"name": "c", "n": int64(3)},
		}})

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		got := results[0].GetValue().GetListValue().GetValues()
		require.Len(t, got, 2)
		assert.Equal(t, "b", got[0].GetStringValue())
		assert.Equal(t, "c", got[1].GetStringValue())
	})
}

func TestGraph_CEL_Comprehension_Map(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_comprehension_map.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", map[string]any{"items": []any{
			map[string]any{"n": int64(10)},
			map[string]any{"n": int64(20)},
			map[string]any{"n": int64(30)},
		}})

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		got := results[0].GetValue().GetListValue().GetValues()
		require.Len(t, got, 3)
		assert.Equal(t, int64(20), got[0].GetInt64Value())
		assert.Equal(t, int64(40), got[1].GetInt64Value())
		assert.Equal(t, int64(60), got[2].GetInt64Value())
	})
}

// --- vars namespace -----------------------------------------------------------

func TestGraph_CEL_VarsNamespace_NestedField(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_var_namespace.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", "hi")

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, "hi!", results[0].GetValue().GetStringValue())
	})
}

// --- Typed proto: dtkt.shared.v1beta1.Package --------------------------------

// TestGraph_CEL_TypedProto_Package_NestedAccess covers the breadth of
// CEL access patterns against a typed proto: nested message field, scalar,
// enum scalar, repeated enum, repeated message, map<string,string>,
// repeated nested message, and size() over a repeated.
//
// The Package is delivered via an action's response (mock pkg.Fixed).
// Connection-resolver-driven type registration only fires for connections
// referenced by Action/Stream nodes (see resolveConnections); a flow that
// declared a connection but never used it would not register Package and
// CEL would see the value as a generic map. Real flows always reach typed
// protos via RPC anyway, so this matches the production path.
func TestGraph_CEL_TypedProto_Package_NestedAccess(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_package_access.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.trigger", int64(1))

		ctx := testContext(t)
		opts := append(extraOpts, packageMockOptions()...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		require.NoError(t, err)

		assertSingleString(t, ctx, ps, "outputs.identity_name", "samplepkg")
		assertSingleString(t, ctx, ps, "outputs.identity_version", "1.2.3")
		assertSingleString(t, ctx, ps, "outputs.description", "a sample package")
		assertSingleInt64(t, ctx, ps, "outputs.type", int64(sharedv1beta1.PackageType_PACKAGE_TYPE_GO))
		assertSingleInt64(t, ctx, ps, "outputs.first_runtime", int64(sharedv1beta1.Runtime_RUNTIME_NATIVE))
		assertSingleInt64(t, ctx, ps, "outputs.second_runtime", int64(sharedv1beta1.Runtime_RUNTIME_DOCKER))
		assertSingleInt64(t, ctx, ps, "outputs.first_platform_os", int64(sharedv1beta1.OS_OS_LINUX))
		assertSingleInt64(t, ctx, ps, "outputs.second_platform_arch", int64(sharedv1beta1.Arch_ARCH_ARM64))
		assertSingleString(t, ctx, ps, "outputs.build_env_foo", "bar")
		assertSingleString(t, ctx, ps, "outputs.build_env_baz", "qux")
		assertSingleInt64(t, ctx, ps, "outputs.ports_count", 2)
		assertSingleString(t, ctx, ps, "outputs.first_port_name", "grpc")
		assertSingleString(t, ctx, ps, "outputs.second_port_protocol", "tcp")
	})
}

// --- Action response field access (typed Package) ----------------------------

func TestGraph_CEL_ActionResponse_PackageField(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_action_response_package.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", int64(1))

		ctx := testContext(t)
		opts := append(extraOpts, packageMockOptions()...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		require.NoError(t, err)

		assertSingleString(t, ctx, ps, "outputs.name", "samplepkg")
		assertSingleString(t, ctx, ps, "outputs.version", "1.2.3")
		assertSingleInt64(t, ctx, ps, "outputs.type", int64(sharedv1beta1.PackageType_PACKAGE_TYPE_GO))
	})
}

// --- Streams namespace with typed proto elements -----------------------------

// TestGraph_CEL_StreamsNamespace_TypedProto verifies that CEL can access
// nested fields of typed proto messages flowing through a server stream.
// Each element yielded by the stream is a distinct Package; the output
// expression `streams.pkgs.value.identity.name` resolves once per element.
func TestGraph_CEL_StreamsNamespace_TypedProto(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_stream_namespace.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.trigger", int64(1))

		ctx := testContext(t)
		opts := append(extraOpts, packageMockOptions()...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		require.NoError(t, err)

		names := outputStrings(collectOutputs(ctx, ps, "outputs.name"))
		assert.Equal(t, []string{"alpha", "beta", "gamma"}, names)

		versions := outputStrings(collectOutputs(ctx, ps, "outputs.version"))
		assert.Equal(t, []string{"0.1.0", "0.2.0", "0.3.0"}, versions)
	})
}

// --- Action request: CEL navigates into nested input -------------------------

// TestGraph_CEL_ActionRequest_FromNestedInput verifies that an action's
// request tree, with CEL leaves that traverse into a nested input value,
// produces a typed request proto with the right field values populated.
//
// The synthetic echo.Service.Echo method has an EchoRequest{name, age, city}
// schema. exprToMessage materializes a *dynamicpb.Message from the CEL
// output, which the mock handler captures.
func TestGraph_CEL_ActionRequest_FromNestedInput(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_action_request_from_input.yaml")

		var captured proto.Message
		opts := append(extraOpts, echoRequestCaptureOptions(t, &captured)...)

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", map[string]any{
			"user": map[string]any{
				"name": "alice",
				"age":  int64(30),
				"address": map[string]any{
					"city": "SF",
				},
			},
		})

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		require.NoError(t, err)

		require.NotNil(t, captured, "mock handler did not receive a request")
		msg := captured.ProtoReflect()
		fields := msg.Descriptor().Fields()
		assert.Equal(t, "alice", msg.Get(fields.ByName("name")).String())
		assert.Equal(t, int64(30), msg.Get(fields.ByName("age")).Int())
		assert.Equal(t, "SF", msg.Get(fields.ByName("city")).String())
	})
}

// --- helpers ------------------------------------------------------------------

func assertSingleString(t *testing.T, ctx context.Context, ps executor.PubSub, topic, want string) {
	t.Helper()
	results := collectOutputs(ctx, ps, topic)
	require.Len(t, results, 1, "topic %s", topic)
	assert.Equal(t, want, results[0].GetValue().GetStringValue(), "topic %s", topic)
}

func assertSingleInt64(t *testing.T, ctx context.Context, ps executor.PubSub, topic string, want int64) {
	t.Helper()
	results := collectOutputs(ctx, ps, topic)
	require.Len(t, results, 1, "topic %s", topic)
	assert.Equal(t, want, results[0].GetValue().GetInt64Value(), "topic %s", topic)
}

