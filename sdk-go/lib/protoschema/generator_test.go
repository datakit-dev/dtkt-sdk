package protoschema_test

import (
	"encoding/json"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/protoschema"
	basev1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/base/v1beta1"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
)

func Test_Generator(t *testing.T) {
	proto := &basev1beta1.GetPackageResponse{}
	gen := protoschema.NewGenerator(protoschema.WithJSONNames())
	err := gen.Add(proto.ProtoReflect().Descriptor())
	if err != nil {
		t.Fatal(err)
	}
}

func Test_Resolved(t *testing.T) {
	proto := &corev1.CreateConnectionRequest{}
	schema, err := protoschema.ResolvedJSONSchema(proto.ProtoReflect().Descriptor())
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.MarshalIndent(schema.Schema(), "", "  ")
	t.Log(string(b))
}
