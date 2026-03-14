package spec_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
)

func TestValueNotFoundErr(t *testing.T) {
	err := fmt.Errorf("value: %w", spec.NewInputValueError("foo"))
	if !errors.Is(err, spec.InputValueMissingErr) {
		t.Fatal("expected value not found error")
	}
}
