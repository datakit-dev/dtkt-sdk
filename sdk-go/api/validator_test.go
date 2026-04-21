package api

import (
	"testing"

	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
)

func TestRequestValidator(t *testing.T) {
	request := &corev1.CreateConnectionRequest{
		Connection: &corev1.Connection{},
	}
	validator := &RequestFieldValidator{request: request}

	if validator.ShouldValidate(request.Connection.ProtoReflect(), request.Connection.ProtoReflect().Descriptor().Fields().ByName("name")) {
		t.Error("expected name field to be skipped for validation on CreateConnectionRequest")
	} else if validator.ShouldValidate(request.Connection.ProtoReflect(), request.Connection.ProtoReflect().Descriptor().Fields().ByName("uid")) {
		t.Error("expected uid field to be skipped for validation on CreateConnectionRequest")
	}
}
