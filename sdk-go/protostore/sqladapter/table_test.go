package sqladapter

import (
	"fmt"
	"testing"

	"buf.build/go/protovalidate"
	"buf.build/go/protoyaml"
	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
)

func TestTableFromProto(t *testing.T) {
	name := "schema.connections"
	table, err := TableOfProto[corev1.Connection](
		WithTableName(name),
	)
	if err != nil {
		t.Fatal(err)
	}

	if table.GetName() != name {
		t.Fatalf("expected table name: %q, got: %q", name, table.GetName())
	}

	err = protovalidate.Validate(table)
	if err != nil {
		t.Fatalf("table validation failed: %v", err)
	}

	b, err := protoyaml.MarshalOptions{}.Marshal(table)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(b))
}
