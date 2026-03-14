package v1beta1_test

import (
	"log"
	"net/url"
	"path"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
	basev1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/base/v1beta1"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/emptypb"
)

var types = v1beta1.DefaultTypeRegistry()

type (
	StructType struct {
		Foo string `json:"foo,omitzero"`
	}
)

func TestNewTypeSchemaForProto(t *testing.T) {
	// 1. Test concrete proto type
	proto := &sharedv1beta1.Package{
		Identity: &sharedv1beta1.Package_Identity{
			Name:    "FooBar",
			Version: "0.1.0",
		},
		Type: 1,
	}

	ts, err := v1beta1.NewTypeSchemaForProto(types, proto)
	if err != nil {
		t.Fatal(err)
	}

	err = ts.Validate(proto)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Test registry resolved type
	mt, err := protoregistry.GlobalTypes.FindMessageByName("dtkt.shared.v1beta1.StringList")
	if err != nil {
		t.Fatal(err)
	}

	proto2 := mt.New().Interface()

	t.Logf("resolved proto: %s(%T)", mt.Descriptor().FullName(), proto2)

	ts2, err := v1beta1.NewTypeSchemaForProto(types, proto2)
	if err != nil {
		t.Fatal(err)
	}

	err = ts2.Validate(proto2)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewTypeSchemaFor(t *testing.T) {
	// 1. Test with no base path set:
	testTypeBase(t)

	// 2. Test with deploy name set in env:
	uri := (&url.URL{
		Scheme: "dtkt",
	})
	uri.Path = path.Join("/", "users/foo/integrations/bar/deployments/baz", "types")
	types = v1beta1.NewTypeRegistry(&v1beta1.MemoryTypeSyncer{}, uri)
	testTypeBase(t)

	// 3. Test with network and address set in env:
	uri = network.Addr(
		network.Type("tcp"),
		"127.0.0.1:9090",
	).URL()
	uri.Path = "/types"
	types = v1beta1.NewTypeRegistry(&v1beta1.MemoryTypeSyncer{}, uri)
	testTypeBase(t)
}

func newStructTypeSchema() *v1beta1.TypeSchema[*StructType] {
	s, err := v1beta1.NewTypeSchemaFor[*StructType](types, "testStruct")
	if err != nil {
		log.Fatal(err)
	}
	return s
}

func newProtoNativeSchema() *v1beta1.TypeSchema[*corev1.Connection] {
	s, err := v1beta1.NewTypeSchemaFor[*corev1.Connection](types, "dtkt.core.v1.Connection")
	if err != nil {
		log.Fatal(err)
	}
	return s
}

func testTypeBase(t *testing.T) {
	protoName := "dtkt.core.v1.Connection"
	typ := newProtoNativeSchema()
	if typ.ToProto().GetProtoName() != protoName {
		t.Fatalf("expected proto name: %s, got: %s", typ.ToProto().GetProtoName(), protoName)
	}

	getResp, err := types.GetType(&basev1beta1.GetTypeRequest{Name: typ.ToProto().GetUri()})
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(getResp.GetType())
	}

	protoName = "google.protobuf.Empty"
	emptyProto, err := v1beta1.NewTypeSchemaFor[*emptypb.Empty](types, "emptyProto")
	if err != nil {
		t.Fatal(err)
	} else if emptyProto.ToProto().GetProtoName() != protoName {
		t.Fatalf("expected proto name: %s, got: %s", emptyProto.ToProto().GetProtoName(), protoName)
	}

	empty, err := v1beta1.NewTypeSchemaFor[v1beta1.Empty](types, "emptyType")
	if err != nil {
		t.Fatal(err)
	} else if empty.ToProto().GetProtoName() != protoName {
		t.Fatalf("expected proto name: %s, got: %s", empty.ToProto().GetProtoName(), protoName)
	} else {
		t.Log(empty.ToProto())
	}

	protoName = "google.protobuf.Struct"
	schema1 := newStructTypeSchema()
	if schema1.ToProto().GetProtoName() != protoName {
		t.Fatalf("expected proto name: %s, got: %s", schema1.ToProto().GetProtoName(), protoName)
	}

	schema2, err := v1beta1.NewTypeSchemaFor[map[string]any](types, "testMap")
	if err != nil {
		t.Fatal(err)
	} else if schema2.ToProto().GetProtoName() != protoName {
		t.Fatalf("expected proto name: %s, got: %s", schema2.ToProto().GetProtoName(), protoName)
	}

	protoName = "google.protobuf.ListValue"
	schema3, err := v1beta1.NewTypeSchemaFor[[]any](types, "testSlice")
	if err != nil {
		t.Fatal(err)
	} else if schema3.ToProto().GetProtoName() != protoName {
		t.Fatalf("expected proto name: %s, got: %s", schema3.ToProto().GetProtoName(), protoName)
	}

	protoName = "google.protobuf.Int64Value"
	schema4, err := v1beta1.NewTypeSchemaFor[int64](types, "testInt")
	if err != nil {
		t.Fatal(err)
	} else if schema4.ToProto().GetProtoName() != protoName {
		t.Fatalf("expected proto name: %s, got: %s", schema4.ToProto().GetProtoName(), protoName)
	} else {
		t.Log(schema4.JSONSchema().String())
	}

	protoName = "google.protobuf.Duration"
	schema5, err := v1beta1.NewTypeSchemaFor[time.Duration](types, "testDuration")
	if err != nil {
		t.Fatal(err)
	} else if schema5.ToProto().GetProtoName() != protoName {
		t.Fatalf("expected proto name: %s, got: %s", schema5.ToProto().GetProtoName(), protoName)
	}

	protoName = "google.protobuf.Timestamp"
	schema6, err := v1beta1.NewTypeSchemaFor[time.Time](types, "testTime")
	if err != nil {
		t.Fatal(err)
	} else if schema6.ToProto().GetProtoName() != protoName {
		t.Fatalf("expected proto name: %s, got: %s", schema6.ToProto().GetProtoName(), protoName)
	}

	listResp, err := types.ListTypes(&basev1beta1.ListTypesRequest{})
	if err != nil {
		t.Fatal(err)
	}

	for _, typ := range listResp.GetTypes() {
		t.Log(typ.GetUri())
	}
}
