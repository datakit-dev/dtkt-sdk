package api

import (
	"testing"

	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
)

func TestInputValidator(t *testing.T) {
	req := &corev1.CreateConnectionRequest{
		Connection: &corev1.Connection{},
	}
	validator := ValidateInputFilter(req)

	if validator.ShouldValidate(req.Connection.ProtoReflect(), req.Connection.ProtoReflect().Descriptor().Fields().ByName("name")) {
		t.Error("expected name field to be skipped for validation on CreateConnectionRequest")
	} else if validator.ShouldValidate(req.Connection.ProtoReflect(), req.Connection.ProtoReflect().Descriptor().Fields().ByName("uid")) {
		t.Error("expected uid field to be skipped for validation on CreateConnectionRequest")
	}
}

func TestOutputValidator(t *testing.T) {
	resp := &corev1.CreateConnectionResponse{
		Connection: &corev1.Connection{},
	}
	validator := ValidateOutputFilter(resp)

	if validator.ShouldValidate(resp.Connection.ProtoReflect(), resp.Connection.ProtoReflect().Descriptor().Fields().ByName("config_uri")) {
		t.Error("expected config_uri field to be skipped for validation on CreateConnectionRequest")
	} else if validator.ShouldValidate(resp.Connection.ProtoReflect(), resp.Connection.ProtoReflect().Descriptor().Fields().ByName("decrypted_config")) {
		t.Error("expected decrypted_config field to be skipped for validation on CreateConnectionRequest")
	}
}
