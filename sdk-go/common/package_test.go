package common_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

func TestPackageIdentity(t *testing.T) {
	identSchema := common.PackageIdentity("").JSONSchema()

	schema, err := encoding.ToJSONV2(identSchema)
	if err != nil {
		t.Fatal(err)
	}

	js, err := jsonschema.CompileString(identSchema.ID.String(), string(schema))
	if err != nil {
		t.Fatal(err)
	}

	ident := common.PackageIdentity("OpenAI@1.2.3")
	err = js.Validate(ident.String())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if ident.Name() != "OpenAI" {
		t.Errorf("unexpected name: %s", ident.Name())
	} else if ident.Version() != "1.2.3" {
		t.Errorf("unexpected version: %s", ident.Version())
	}

	if ident.ProtoName() != "openai" {
		t.Errorf("expected proto name: openai, got: %s", ident.ProtoName())
	} else if ident.ProtoVersion() != "v1" {
		t.Errorf("expected proto version: v1, got: %s", ident.ProtoVersion())
	} else if ident.ProtoPackage() != "openai.v1" {
		t.Errorf("expected proto package: openai.v1, got: %s", ident.ProtoPackage())
	} else if ident.ProtoPackage("dtkt", "integration") != "dtkt.integration.openai.v1" {
		t.Errorf("expected proto package: dtkt.integration.openai.v1, got: %s", ident.ProtoPackage("dtkt", "integration"))
	}

	err = js.Validate("OpenAI@")
	if err == nil {
		t.Error("expected error for empty version")
	}

	err = js.Validate("OpenAI@latest@")
	if err == nil {
		t.Error("expected error for multiple versions")
	}
}
