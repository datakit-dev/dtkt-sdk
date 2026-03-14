package api_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestVersionContainsName(t *testing.T) {
	pkg := sharedv1beta1.File_dtkt_shared_v1beta1_messages_proto.FullName()
	if !api.VersionContainsName(api.V1Beta1, pkg) {
		t.Fatalf("expected: %s to be version: %s", pkg, api.V1Beta1)
	}
}

func TestIsWellKnownName(t *testing.T) {
	names := []protoreflect.FullName{
		"google.protobuf.Struct",
		"google.api.Http",
		"google.type.Foo",
	}

	for _, name := range names {
		if !api.IsWellKnownName(name) {
			t.Fatalf("expected: %s to be well known name", name)
		}
	}
}
