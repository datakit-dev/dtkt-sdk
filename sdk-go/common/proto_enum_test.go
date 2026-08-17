package common_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
)

func TestProtoEnumToString(t *testing.T) {
	str := common.ProtoEnumToString(corev1.AutomationRuntime_RUNNING, true)
	if str != "RUNNING" {
		t.Fatalf("expected: RUNNING, got: %s", str)
	}

	str = common.ProtoEnumToString(corev1.Auth_AUTH_METHOD_DEVICE, true)
	if str != "DEVICE" {
		t.Fatalf("expected: DEVICE, got: %s", str)
	}
}
