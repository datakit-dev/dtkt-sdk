package api_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestVersion(t *testing.T) {
	for _, version := range api.Versions() {
		version.RangeServices(func(sd protoreflect.ServiceDescriptor) bool {
			t.Log(version, sd.FullName())
			return true
		})
		version.RangeMessages(func(mt protoreflect.MessageType) bool {
			t.Log(version, mt.Descriptor().FullName())
			return true
		})
	}
}
