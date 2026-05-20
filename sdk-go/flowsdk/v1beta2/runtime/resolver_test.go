package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc/mock"
)

// connectorFromFiles builds an *rpc.Connector wrapping a flowResolver
// (the test-harness shared.Resolver impl) over the given files. The
// client is a no-op mock - these tests exercise resolver behavior, not
// dispatch.
func connectorFromFiles(t *testing.T, files ...protoreflect.FileDescriptor) *rpc.Connector {
	t.Helper()
	return &rpc.Connector{
		Client:   mock.NewClient(),
		Resolver: newFlowResolver(mock.NewClient(), files...),
	}
}

func TestFlowUnionResolver_FindMessageByName_PrefersConnectorOverPlatform(t *testing.T) {
	// A connector file declares pkg.Custom; api.GlobalResolver does not.
	// FindMessageByName must return the connector's type.
	custom := buildSyntheticFile(t, syntheticFileSpec{
		fileName:    "custom.proto",
		packageName: "custompkg",
		messages: []syntheticMessage{
			{name: "Custom", fields: []syntheticField{
				{name: "payload", number: 1, fieldType: descriptorpb.FieldDescriptorProto_TYPE_STRING},
			}},
		},
	})
	r := newFlowUnionResolver(
		[]*rpc.Connector{connectorFromFiles(t, custom)},
		api.GlobalResolver(),
	)

	mt, err := r.FindMessageByName("custompkg.Custom")
	require.NoError(t, err)
	require.NotNil(t, mt)
	assert.Equal(t, protoreflect.FullName("custompkg.Custom"), mt.Descriptor().FullName())
}

func TestFlowUnionResolver_FindMessageByName_FallsBackToPlatform(t *testing.T) {
	// google.protobuf.StringValue lives in the SDK platform layer, not
	// in any connector. With no connectors AND the platform, lookup must
	// still succeed.
	r := newFlowUnionResolver(nil, api.GlobalResolver())

	mt, err := r.FindMessageByName("google.protobuf.StringValue")
	require.NoError(t, err)
	require.NotNil(t, mt)
	assert.Equal(t, protoreflect.FullName("google.protobuf.StringValue"), mt.Descriptor().FullName())
}

func TestFlowUnionResolver_FindMessageByURL_StripsPrefixAndDispatches(t *testing.T) {
	r := newFlowUnionResolver(nil, api.GlobalResolver())

	mt, err := r.FindMessageByURL("type.googleapis.com/google.protobuf.StringValue")
	require.NoError(t, err)
	require.NotNil(t, mt)
	assert.Equal(t, protoreflect.FullName("google.protobuf.StringValue"), mt.Descriptor().FullName())
}

func TestFlowUnionResolver_FindMessageByName_NotFoundReturnsNotFoundError(t *testing.T) {
	r := newFlowUnionResolver(nil, api.GlobalResolver())

	_, err := r.FindMessageByName("nonexistent.Type")
	require.Error(t, err)
	// Either the platform's NotFound or an equivalent - just confirm the
	// caller gets an error they can branch on.
	assert.True(t, errIsNotFound(err) || err == protoregistry.NotFound,
		"expected NotFound-equivalent error, got %v", err)
}

func TestFlowUnionResolver_FindMethodByName_PrefersConnector(t *testing.T) {
	custom := buildSyntheticFile(t, syntheticFileSpec{
		fileName:    "svc.proto",
		packageName: "svcpkg",
		messages: []syntheticMessage{
			{name: "Req"},
			{name: "Resp"},
		},
		services: []syntheticService{
			{
				name: "Service",
				methods: []syntheticMethod{
					{name: "Call", inputType: ".svcpkg.Req", outputType: ".svcpkg.Resp"},
				},
			},
		},
	})
	r := newFlowUnionResolver(
		[]*rpc.Connector{connectorFromFiles(t, custom)},
		api.GlobalResolver(),
	)

	md, err := r.FindMethodByName("svcpkg.Service.Call")
	require.NoError(t, err)
	require.NotNil(t, md)
	assert.Equal(t, protoreflect.FullName("svcpkg.Service.Call"), md.FullName())
}

func TestFlowUnionResolver_RangeFiles_DedupesByPath(t *testing.T) {
	// Two connectors declaring DIFFERENT messages in files with the SAME
	// path -- RangeFiles must yield the path exactly once (the first
	// connector wins; the iteration in newFlowUnionResolver is
	// spec-ordered). If dedupe is broken, downstream consumers like
	// common.NewCELTypes' RegisterDescriptor would error on the
	// duplicate.
	fileA := buildSyntheticFile(t, syntheticFileSpec{
		fileName:    "shared.proto",
		packageName: "a",
		messages:    []syntheticMessage{{name: "FromA"}},
	})
	fileB := buildSyntheticFile(t, syntheticFileSpec{
		fileName:    "shared.proto", // SAME path as fileA on purpose
		packageName: "b",
		messages:    []syntheticMessage{{name: "FromB"}},
	})
	r := newFlowUnionResolver(
		[]*rpc.Connector{
			connectorFromFiles(t, fileA),
			connectorFromFiles(t, fileB),
		},
		api.GlobalResolver(),
	)

	paths := map[string]int{}
	r.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		paths[fd.Path()]++
		return true
	})

	assert.Equal(t, 1, paths["shared.proto"], "shared.proto should be yielded exactly once (got %d)", paths["shared.proto"])
}

func TestFlowUnionResolver_RangeFiles_VisitsConnectorsBeforePlatform(t *testing.T) {
	// Spec order matters: connector A's files come before connector B's
	// files come before the platform's files. We check by recording the
	// FIRST yielded file's package and ensuring it's the connector's.
	custom := buildSyntheticFile(t, syntheticFileSpec{
		fileName:    "first.proto",
		packageName: "firstpkg",
		messages:    []syntheticMessage{{name: "M"}},
	})
	r := newFlowUnionResolver(
		[]*rpc.Connector{connectorFromFiles(t, custom)},
		api.GlobalResolver(),
	)

	var firstPath string
	r.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		firstPath = fd.Path()
		return false // stop after first
	})

	assert.Equal(t, "first.proto", firstPath, "connector files must precede platform files in iteration order")
}

func TestFlowUnionResolver_EmptyConnectors_PlatformOnlyStillWorks(t *testing.T) {
	r := newFlowUnionResolver(nil, api.GlobalResolver())

	// FindMessageByName for a known platform type works.
	mt, err := r.FindMessageByName("google.protobuf.StringValue")
	require.NoError(t, err)
	require.NotNil(t, mt)

	// RangeFiles yields at least one file from the platform.
	found := false
	r.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Path() != "" {
			found = true
			return false
		}
		return true
	})
	assert.True(t, found, "platform-only resolver must yield platform files via RangeFiles")
}

func TestFlowUnionResolver_NilConnectorEntriesAreSkipped(t *testing.T) {
	// Defensive: a nil entry in orderedConnectors should not panic; nor
	// should a connector with a nil Resolver.
	custom := buildSyntheticFile(t, syntheticFileSpec{
		fileName:    "x.proto",
		packageName: "x",
		messages:    []syntheticMessage{{name: "M"}},
	})
	r := newFlowUnionResolver(
		[]*rpc.Connector{
			nil,
			{Client: mock.NewClient(), Resolver: nil},
			connectorFromFiles(t, custom),
		},
		api.GlobalResolver(),
	)

	mt, err := r.FindMessageByName("x.M")
	require.NoError(t, err)
	require.NotNil(t, mt)
}

func TestFlowUnionResolver_GetValidator_ValidatesPlatformTypes(t *testing.T) {
	// GetValidator must compose protovalidate with this resolver as the
	// extension type resolver. A no-rules google.protobuf.StringValue
	// must validate successfully through the validator we return.
	r := newFlowUnionResolver(nil, api.GlobalResolver())

	v, err := r.GetValidator()
	require.NoError(t, err)
	require.NotNil(t, v)

	mt, err := r.FindMessageByName("google.protobuf.StringValue")
	require.NoError(t, err)
	msg := mt.New().Interface() // empty StringValue, no validate rules -> must validate

	// protovalidate.Validate returns nil for valid messages.
	assert.NoError(t, v.Validate(msg))
}

func TestErrIsNotFound_MatchesGlobalNotFound(t *testing.T) {
	assert.True(t, errIsNotFound(protoregistry.NotFound))
}
