package common_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
	sqlv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/sql/v1beta1"
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

func TestTrimEnumPrefixFor(t *testing.T) {
	enum := sqlv1beta1.OrderBy_Direction(0)
	if str := common.TrimEnumPrefixFor(enum); str != "" {
		t.Fatalf("expected empty string got %s", str)
	}

	enum += 1
	if str := common.TrimEnumPrefixFor(enum); str != "ASC" {
		t.Fatalf("expected ASC got %s", str)
	}

	enum += 1
	if str := common.TrimEnumPrefixFor(enum); str != "DESC" {
		t.Fatalf("expected DESC got %s", str)
	}

}
