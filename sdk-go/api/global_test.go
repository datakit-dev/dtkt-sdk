package api

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestSourceInfo(t *testing.T) {
	msgDesc, err := GlobalResolver().FindDescriptorByName(protoreflect.FullName("dtkt.shared.v1beta1.Package.identity"))
	if err != nil {
		t.Fatal(err)
	}

	desc := util.GetProtoDescription(msgDesc)
	if desc == "" {
		t.Fatalf("expected description for: %s", msgDesc.FullName())
	} else {
		t.Log(desc)
	}
}

func TestRangeServicesAndMethods(t *testing.T) {
	GlobalResolver().RangeServices(func(sd protoreflect.ServiceDescriptor) bool {
		t.Log(sd.FullName())
		return false
	})

	GlobalResolver().RangeMethods(func(md protoreflect.MethodDescriptor) bool {
		t.Log(md.FullName())
		return false
	})
}
