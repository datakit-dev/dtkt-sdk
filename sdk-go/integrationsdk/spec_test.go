package integrationsdk_test

import (
	"testing"

	"embed"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk"
)

//go:embed test_data/*
var testdata embed.FS

type Config struct {
	Name string `json:"name"`
}

func TestSpec_Valid(t *testing.T) {
	reader, err := testdata.Open("test_data/package_valid.dtkt.yaml")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	_, err = integrationsdk.ReadSpec(encoding.YAML, reader)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestSpec_Invalid(t *testing.T) {
	reader, err := testdata.Open("test_data/package_invalid.dtkt.yaml")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	_, err = integrationsdk.ReadSpec(encoding.YAML, reader)
	if err == nil {
		t.Fatalf("Load() error = %v, want error", err)
	}
}
