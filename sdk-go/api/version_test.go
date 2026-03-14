package api_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestVersion_RangeFiles(t *testing.T) {
	api.V1Beta1.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		t.Log(fd.FullName())
		return true
	})
}
