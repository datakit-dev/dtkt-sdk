package protoschema_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/protoschema"
	"github.com/invopop/jsonschema"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TestGoTypeRoundTrip verifies that Go type information is preserved through
// the JSON Schema generation and parsing pipeline using x-dtkt-format annotations
func TestGoTypeRoundTrip(t *testing.T) {
	type TestStruct struct {
		Int8Field    int8    `json:"int8_field"`
		Int16Field   int16   `json:"int16_field"`
		Int32Field   int32   `json:"int32_field"`
		Int64Field   int64   `json:"int64_field"`
		UInt8Field   uint8   `json:"uint8_field"`
		UInt16Field  uint16  `json:"uint16_field"`
		UInt32Field  uint32  `json:"uint32_field"`
		UInt64Field  uint64  `json:"uint64_field"`
		Float32Field float32 `json:"float32_field"`
		Float64Field float64 `json:"float64_field"`
		StringField  string  `json:"string_field"`
		BoolField    bool    `json:"bool_field"`
	}

	// Step 1: Generate JSON Schema from Go struct with type annotations
	reflector := &jsonschema.Reflector{
		Mapper: common.GoTypeAnnotationMapper,
	}
	schema := reflector.Reflect(&TestStruct{})

	// Add title to ensure consistent schema structure
	schema.Title = "TestStruct"

	schemaBytes, err := schema.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	// Step 2: Parse JSON Schema to Proto descriptor
	parser := protoschema.NewParser(protoschema.ParserOptions{
		PackageName: "test",
	})
	fdp, err := parser.Parse(schemaBytes)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Step 3: Create file descriptor
	fd, err := protodesc.NewFile(fdp, nil)
	if err != nil {
		t.Fatalf("Failed to create file descriptor: %v", err)
	}

	// Step 4: Verify message structure
	messages := fd.Messages()
	// Find the TestStruct message (there may be nested definitions)
	var msg protoreflect.MessageDescriptor
	for i := 0; i < messages.Len(); i++ {
		m := messages.Get(i)
		if m.Name() == "TestStruct" {
			msg = m
			break
		}
	}
	if msg == nil {
		t.Fatalf("TestStruct message not found")
	}

	// Step 5: Verify field types match expectations
	// Note: Smaller types (int8, int16, uint8, uint16) map to protobuf's 32-bit equivalents
	expectedTypes := map[string]protoreflect.Kind{
		"int8_field":    protoreflect.Int32Kind, // int8 → int32
		"int16_field":   protoreflect.Int32Kind, // int16 → int32
		"int32_field":   protoreflect.Int32Kind,
		"int64_field":   protoreflect.Int64Kind,
		"uint8_field":   protoreflect.Uint32Kind, // uint8 → uint32
		"uint16_field":  protoreflect.Uint32Kind, // uint16 → uint32
		"uint32_field":  protoreflect.Uint32Kind,
		"uint64_field":  protoreflect.Uint64Kind,
		"float32_field": protoreflect.FloatKind,
		"float64_field": protoreflect.DoubleKind,
		"string_field":  protoreflect.StringKind,
		"bool_field":    protoreflect.BoolKind,
	}

	fields := msg.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		fieldName := string(field.Name())

		expectedKind, ok := expectedTypes[fieldName]
		if !ok {
			t.Errorf("Unexpected field: %s", fieldName)
			continue
		}

		if field.Kind() != expectedKind {
			t.Errorf("Field %s: expected kind %v, got %v", fieldName, expectedKind, field.Kind())
		}

		// Verify field is not repeated
		if field.IsList() {
			t.Errorf("Field %s should not be repeated", fieldName)
		}

		delete(expectedTypes, fieldName)
	}

	// Ensure all expected fields were found
	if len(expectedTypes) > 0 {
		t.Errorf("Missing fields: %v", expectedTypes)
	}
}

// TestGoTypeAnnotationMapper verifies that the mapper correctly adds x-dtkt-format
func TestGoTypeAnnotationMapper(t *testing.T) {
	type Sample struct {
		Int8Value    int8    `json:"int8_value"`
		Uint32Value  uint32  `json:"uint32_value"`
		Float32Value float32 `json:"float32_value"`
		StringValue  string  `json:"string_value"`
	}

	// Use the common package to create a complete JSON schema with annotations
	schema, err := common.NewJSONSchema(Sample{})
	if err != nil {
		t.Fatalf("Failed to create JSON schema: %v", err)
	}

	jsonSchema := schema.JSONSchema()
	if jsonSchema.Properties == nil {
		t.Fatal("schema.Properties is nil")
	}

	// Check that x-dtkt-format was added to integer/number fields
	for pair := jsonSchema.Properties.Oldest(); pair != nil; pair = pair.Next() {
		propName := pair.Key
		propSchema := pair.Value

		t.Logf("Property %s: type=%s, format=%s, x-dtkt-format=%v",
			propName, propSchema.Type, propSchema.Format, propSchema.Extras["x-dtkt-format"])

		switch propName {
		case "int8_value":
			if format, ok := propSchema.Extras["x-dtkt-format"].(string); !ok || format != "int8" {
				t.Errorf("int8_value: expected x-dtkt-format='int8', got %v", propSchema.Extras["x-dtkt-format"])
			}
		case "uint32_value":
			if format, ok := propSchema.Extras["x-dtkt-format"].(string); !ok || format != "uint32" {
				t.Errorf("uint32_value: expected x-dtkt-format='uint32', got %v", propSchema.Extras["x-dtkt-format"])
			}
		case "float32_value":
			if format, ok := propSchema.Extras["x-dtkt-format"].(string); !ok || format != "float32" {
				t.Errorf("float32_value: expected x-dtkt-format='float32', got %v", propSchema.Extras["x-dtkt-format"])
			}
		case "string_value":
			if _, ok := propSchema.Extras["x-dtkt-format"]; ok {
				t.Errorf("string_value should not have x-dtkt-format, got %v", propSchema.Extras["x-dtkt-format"])
			}
		}
	}
}

// TestScalarTypeAnnotations verifies that scalar types get x-dtkt-format annotations
func TestScalarTypeAnnotations(t *testing.T) {
	// Test int64 scalar
	schemaInt64, err := common.NewJSONSchema(int64(0))
	if err != nil {
		t.Fatalf("Failed to create int64 schema: %v", err)
	}

	jsonSchemaInt64 := schemaInt64.JSONSchema()
	if jsonSchemaInt64.Type != "integer" {
		t.Errorf("Expected type 'integer', got %s", jsonSchemaInt64.Type)
	}
	if format, ok := jsonSchemaInt64.Extras["x-dtkt-format"].(string); !ok || format != "int64" {
		t.Errorf("int64: expected x-dtkt-format='int64', got %v", jsonSchemaInt64.Extras["x-dtkt-format"])
	}

	// Test float32 scalar
	schemaFloat32, err := common.NewJSONSchema(float32(0))
	if err != nil {
		t.Fatalf("Failed to create float32 schema: %v", err)
	}

	jsonSchemaFloat32 := schemaFloat32.JSONSchema()
	if jsonSchemaFloat32.Type != "number" {
		t.Errorf("Expected type 'number', got %s", jsonSchemaFloat32.Type)
	}
	if format, ok := jsonSchemaFloat32.Extras["x-dtkt-format"].(string); !ok || format != "float32" {
		t.Errorf("float32: expected x-dtkt-format='float32', got %v", jsonSchemaFloat32.Extras["x-dtkt-format"])
	}

	// Test uint32 scalar
	schemaUint32, err := common.NewJSONSchema(uint32(0))
	if err != nil {
		t.Fatalf("Failed to create uint32 schema: %v", err)
	}

	jsonSchemaUint32 := schemaUint32.JSONSchema()
	if jsonSchemaUint32.Type != "integer" {
		t.Errorf("Expected type 'integer', got %s", jsonSchemaUint32.Type)
	}
	if format, ok := jsonSchemaUint32.Extras["x-dtkt-format"].(string); !ok || format != "uint32" {
		t.Errorf("uint32: expected x-dtkt-format='uint32', got %v", jsonSchemaUint32.Extras["x-dtkt-format"])
	}
}
