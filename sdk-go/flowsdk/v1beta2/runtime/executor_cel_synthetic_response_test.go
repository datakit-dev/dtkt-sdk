package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc/mock"
)

// syntheticResponseMockOptions wires an "echo" connection whose
// echo.Service.Echo unary returns a *dynamicpb.Message of a SYNTHETIC
// EchoResponse{payload string}. The type is built at test time via
// buildSyntheticFile, so its Any type URL (type.googleapis.com/echo.
// EchoResponse) is NOT in protoregistry.GlobalTypes. This is the exact
// shape of a real connector response (runtime dynamicpb type loaded from
// a deployment descriptor set) and is the hermetic reproduction of U10:
// cel-go ProtoAsValue's resolver-less anypb.UnmarshalNew fails to resolve
// it. Existing helpers (packageMockOptions returns a generated Package;
// echoRequestCaptureOptions returns a WKT StringValue) deliberately avoid
// this path, which is why U10 went unnoticed.
func syntheticResponseMockOptions(t *testing.T) []Option {
	file := buildSyntheticFile(t, syntheticFileSpec{
		fileName:    "echobench.proto",
		packageName: "echo",
		messages: []syntheticMessage{
			{name: "EchoRequest", fields: []syntheticField{
				{name: "payload", number: 1, fieldType: descriptorpb.FieldDescriptorProto_TYPE_STRING},
			}},
			{name: "EchoResponse", fields: []syntheticField{
				{name: "payload", number: 1, fieldType: descriptorpb.FieldDescriptorProto_TYPE_STRING},
			}},
		},
		services: []syntheticService{
			{
				name: "Service",
				methods: []syntheticMethod{
					{name: "Echo", inputType: ".echo.EchoRequest", outputType: ".echo.EchoResponse"},
				},
			},
		},
	})
	respMD := file.Messages().ByName("EchoResponse")

	c := mock.NewClient()
	c.RegisterUnary("echo.Service.Echo", func(_ context.Context, _ proto.Message) (proto.Message, error) {
		m := dynamicpb.NewMessage(respMD)
		m.Set(respMD.Fields().ByName("payload"), protoreflect.ValueOfString("hello-u10"))
		return m, nil
	})

	return []Option{withMockConnection("echo", c, file)}
}

// TestGraph_CEL_ActionResponse_SyntheticType is the hermetic U10 regression
// test. An action returns a synthetic (non-global) typed message and an
// output reads a field off it via CEL. Pre-fix this fails because
// exprToRefVal -> cel.ProtoAsValue -> anypb.UnmarshalNew uses
// protoregistry.GlobalTypes (no Resolver) and the synthetic type is not
// there ("proto: not found"). Post-fix exprToRefVal decodes the Any via
// the connector resolver (shared.ExprValueToNative) and this passes.
func TestGraph_CEL_ActionResponse_SyntheticType(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_action_response_synthetic.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", "hello-u10")

		ctx := testContext(t)
		opts := append(extraOpts, syntheticResponseMockOptions(t)...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		require.NoError(t, err)

		assertSingleString(t, ctx, ps, "outputs.payload", "hello-u10")
	})
}

// twoConnectorSyntheticOptions wires TWO synthetic connectors ("echoA"
// and "echoB"), each with its own distinct synthetic response type
// (echoa.Response{payload}, echob.Response{payload}). This is the
// scenario that would silently break under per-connection resolver
// scoping: an output that consumes responses from BOTH connections in
// the same env can only resolve types known to that env, so whichever
// connector's resolver is NOT the one feeding the env loses. With the
// flow-global union resolver (connectors + api.GlobalResolver()), both
// types resolve from the same shared resolver and the merge works.
func twoConnectorSyntheticOptions(t *testing.T) []Option {
	mkConnector := func(pkg, payloadValue string) *rpc.Connector {
		file := buildSyntheticFile(t, syntheticFileSpec{
			fileName:    pkg + ".proto",
			packageName: pkg,
			messages: []syntheticMessage{
				{name: "Request", fields: []syntheticField{
					{name: "payload", number: 1, fieldType: descriptorpb.FieldDescriptorProto_TYPE_STRING},
				}},
				{name: "Response", fields: []syntheticField{
					{name: "payload", number: 1, fieldType: descriptorpb.FieldDescriptorProto_TYPE_STRING},
				}},
			},
			services: []syntheticService{
				{
					name: "Service",
					methods: []syntheticMethod{
						{name: "Call", inputType: "." + pkg + ".Request", outputType: "." + pkg + ".Response"},
					},
				},
			},
		})
		respMD := file.Messages().ByName("Response")

		c := mock.NewClient()
		c.RegisterUnary(pkg+".Service.Call", func(_ context.Context, _ proto.Message) (proto.Message, error) {
			m := dynamicpb.NewMessage(respMD)
			m.Set(respMD.Fields().ByName("payload"), protoreflect.ValueOfString(payloadValue))
			return m, nil
		})
		return &rpc.Connector{Client: c, Resolver: newFlowResolver(c, file)}
	}
	// WithConnectors REPLACES the connectors map; multiple withMockConnection
	// calls would overwrite. Build one map with both entries.
	return []Option{WithConnectors(rpc.Connectors{
		"echoa": mkConnector("echoa", "from-A"),
		"echob": mkConnector("echob", "from-B"),
	})}
}

// TestGraph_CEL_ActionResponse_Aggregation is the recursion regression
// guard. The action's call.response transform wraps the connector
// response in a LIST so the action's published *expr.Value is a
// ListValue of Any-wrapped ObjectValues. A downstream output reads a
// nested element's field. This is the only configuration that exercises
// shared.ExprValueToNative's recursion through ListValue at the
// consuming side: the output's activation must walk the list and decode
// each nested Any via the flow-global resolver. If recursion is broken
// or per-element resolver routing fails, this trips with "proto: not
// found" or "converting expr to ref.Val" before the assertion runs.
//
// Why call.response instead of a var-scan-into-list: scan emits per-
// input intermediates (the first emit has length 1, so any test reading
// index [1] errors before the second input lands), and reduce emits
// only at EOF making timing the executor's full-graph completion fussy
// in the test harness. call.response giving a 2-element list per single
// invocation is the cleanest reproduction of "node value is a list of
// ObjectValues" the recursion needs.
func TestGraph_CEL_ActionResponse_Aggregation(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_action_response_aggregation.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", "hello-aggregation")

		ctx := testContext(t)
		opts := append(extraOpts, syntheticResponseMockOptions(t)...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		require.NoError(t, err)

		assertSingleString(t, ctx, ps, "outputs.second", "hello-u10")
	})
}

// TestGraph_CEL_ActionResponse_MultiConnector is the multi-connector
// regression guard. A single output expression consumes responses from
// TWO distinct synthetic connectors. The flow-global union resolver
// (connections + api.GlobalResolver()) makes both connector types
// resolvable from the same env; per-connection scoping (the broken
// shape that preceded this design) would have failed because the output
// handler's env can only see one connector's resolver.
func TestGraph_CEL_ActionResponse_MultiConnector(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "cel_action_response_multi_connector.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		feedInput(ps, "inputs.x", "tick")

		ctx := testContext(t)
		opts := append(extraOpts, twoConnectorSyntheticOptions(t)...)
		err := NewExecutor(ps, testTopics, opts...).Execute(ctx, graph)
		require.NoError(t, err)

		// The combined output references actions on BOTH connections; if
		// either type fails to resolve, the output errors and the test
		// fails. Per-connection scoping would surface as "proto: not
		// found" on the connector whose resolver is not the env's.
		assertSingleString(t, ctx, ps, "outputs.combined", "from-A:from-B")
	})
}
