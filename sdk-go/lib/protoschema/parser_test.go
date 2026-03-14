package protoschema

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/invopop/jsonschema"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Test 1: Simple boolean field
func TestParser_SimpleBooleanField(t *testing.T) {
	schema := `{
		"title": "SimpleMessage",
		"type": "object",
		"properties": {
			"enabled": {
				"type": "boolean"
			}
		}
	}`

	parser := NewParser(ParserOptions{
		PackageName: "test",
		Syntax:      "proto3",
	})

	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify file descriptor
	if *fdp.Package != "test" {
		t.Errorf("Expected package 'test', got %q", *fdp.Package)
	}
	if *fdp.Syntax != "proto3" {
		t.Errorf("Expected syntax 'proto3', got %q", *fdp.Syntax)
	}

	// Verify message
	if len(fdp.MessageType) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(fdp.MessageType))
	}

	msg := fdp.MessageType[0]
	if *msg.Name != "SimpleMessage" {
		t.Errorf("Expected message name 'SimpleMessage', got %q", *msg.Name)
	}

	// Verify field
	if len(msg.Field) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(msg.Field))
	}

	field := msg.Field[0]
	if *field.Name != "enabled" {
		t.Errorf("Expected field name 'enabled', got %q", *field.Name)
	}
	if *field.Number != 1 {
		t.Errorf("Expected field number 1, got %d", *field.Number)
	}
	if *field.Type != descriptorpb.FieldDescriptorProto_TYPE_BOOL {
		t.Errorf("Expected TYPE_BOOL, got %v", *field.Type)
	}
	if field.Proto3Optional == nil || !*field.Proto3Optional {
		t.Error("Expected proto3_optional to be true for non-required field")
	}
}

// Test 2: Required field (should not have proto3_optional)
func TestParser_RequiredField(t *testing.T) {
	schema := `{
		"title": "RequiredMessage",
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			}
		},
		"required": ["name"]
	}`

	parser := NewParser(ParserOptions{
		PackageName: "test",
		Syntax:      "proto3",
	})

	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	msg := fdp.MessageType[0]
	field := msg.Field[0]

	if field.Proto3Optional != nil && *field.Proto3Optional {
		t.Error("Expected proto3_optional to be false/nil for required field")
	}
}

// Test 3: Multiple fields with stable ordering
func TestParser_MultipleFieldsStableOrdering(t *testing.T) {
	schema := `{
		"title": "User",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"},
			"email": {"type": "string"},
			"active": {"type": "boolean"}
		}
	}`

	parser := NewParser(ParserOptions{
		PackageName: "test",
	})

	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	msg := fdp.MessageType[0]
	if len(msg.Field) != 4 {
		t.Fatalf("Expected 4 fields, got %d", len(msg.Field))
	}

	// Fields should be ordered alphabetically: active, age, email, name
	expectedOrder := []string{"active", "age", "email", "name"}
	for i, field := range msg.Field {
		if *field.Name != expectedOrder[i] {
			t.Errorf("Field %d: expected %q, got %q", i, expectedOrder[i], *field.Name)
		}
		if *field.Number != int32(i+1) {
			t.Errorf("Field %q: expected number %d, got %d", *field.Name, i+1, *field.Number)
		}
	}
}

// Test 4: String type inference
func TestParser_StringTypes(t *testing.T) {
	tests := []struct {
		name           string
		schema         string
		expectedType   descriptorpb.FieldDescriptorProto_Type
		expectedFormat string
	}{
		{
			name: "plain string",
			schema: `{
				"title": "Test",
				"type": "object",
				"properties": {
					"field": {"type": "string"}
				}
			}`,
			expectedType: descriptorpb.FieldDescriptorProto_TYPE_STRING,
		},
		{
			name: "byte format",
			schema: `{
				"title": "Test",
				"type": "object",
				"properties": {
					"field": {"type": "string", "format": "byte"}
				}
			}`,
			expectedType: descriptorpb.FieldDescriptorProto_TYPE_BYTES,
		},
		{
			name: "binary format",
			schema: `{
				"title": "Test",
				"type": "object",
				"properties": {
					"field": {"type": "string", "format": "binary"}
				}
			}`,
			expectedType: descriptorpb.FieldDescriptorProto_TYPE_BYTES,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(ParserOptions{PackageName: "test"})
			fdp, err := parser.Parse([]byte(tt.schema))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			field := fdp.MessageType[0].Field[0]
			if *field.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, *field.Type)
			}
		})
	}
}

// Test 5: Integer type inference
func TestParser_IntegerTypes(t *testing.T) {
	tests := []struct {
		name         string
		schema       map[string]any
		expectedType descriptorpb.FieldDescriptorProto_Type
	}{
		{
			name: "default int32",
			schema: map[string]any{
				"type": "integer",
			},
			expectedType: descriptorpb.FieldDescriptorProto_TYPE_INT32,
		},
		{
			name: "int32 format",
			schema: map[string]any{
				"type":   "integer",
				"format": "int32",
			},
			expectedType: descriptorpb.FieldDescriptorProto_TYPE_INT32,
		},
		{
			name: "int64 format",
			schema: map[string]any{
				"type":   "integer",
				"format": "int64",
			},
			expectedType: descriptorpb.FieldDescriptorProto_TYPE_INT64,
		},
		{
			name: "uint32 format",
			schema: map[string]any{
				"type":   "integer",
				"format": "uint32",
			},
			expectedType: descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		},
		{
			name: "uint64 format",
			schema: map[string]any{
				"type":   "integer",
				"format": "uint64",
			},
			expectedType: descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		},
		{
			name: "infer uint32 from range",
			schema: map[string]any{
				"type":    "integer",
				"minimum": float64(0),
				"maximum": float64(1000),
			},
			expectedType: descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		},
		{
			name: "infer uint64 from large range",
			schema: map[string]any{
				"type":    "integer",
				"minimum": float64(0),
				"maximum": float64(5000000000), // > uint32 max
			},
			expectedType: descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(ParserOptions{PackageName: "test"})
			// Convert map to jsonschema.Schema
			b, _ := json.Marshal(tt.schema)
			var schema jsonschema.Schema
			//nolint:errcheck
			json.Unmarshal(b, &schema)
			actualType := parser.inferIntegerType(&schema)
			if actualType != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, actualType)
			}
		})
	}
}

// Test 6: Number (float/double) type inference
func TestParser_NumberTypes(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		expectedType descriptorpb.FieldDescriptorProto_Type
	}{
		{
			name:         "default double",
			format:       "",
			expectedType: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
		},
		{
			name:         "float format",
			format:       "float",
			expectedType: descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		},
		{
			name:         "double format",
			format:       "double",
			expectedType: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(ParserOptions{PackageName: "test"})
			schema := map[string]any{"type": "number"}
			if tt.format != "" {
				schema["format"] = tt.format
			}
			// Convert map to jsonschema.Schema
			b, _ := json.Marshal(schema)
			var s jsonschema.Schema
			//nolint:errcheck
			json.Unmarshal(b, &s)
			actualType := parser.inferNumberType(&s)
			if actualType != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, actualType)
			}
		})
	}
}

// Test 7: Field name sanitization
func TestParser_FieldNameSanitization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"validName", "validName"},
		{"Valid_Name_123", "Valid_Name_123"},
		{"invalid-name", "invalid_name"},
		{"invalid.name", "invalid_name"},
		{"invalid name", "invalid_name"},
		{"123invalid", "_123invalid"},
		{"special!@#chars", "special___chars"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			actual := SanitizeFieldName(tt.input)
			if actual != tt.expected {
				t.Errorf("sanitizeFieldName(%q) = %q, want %q", tt.input, actual, tt.expected)
			}
		})
	}
}

// Test 8: Message name sanitization
func TestParser_MessageNameSanitization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Message", "SimpleMessage"},
		{"user-profile", "UserProfile"},
		{"API Response", "APIResponse"},
		{"some_snake_case", "SomeSnakeCase"},
		{"", "Message"},
		{"123 Numbers", "123Numbers"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			actual := SanitizeMessageName(tt.input)
			if actual != tt.expected {
				t.Errorf("sanitizeMessageName(%q) = %q, want %q", tt.input, actual, tt.expected)
			}
		})
	}
}

// Test 9: JSON names preservation
func TestParser_JSONNames(t *testing.T) {
	schema := `{
		"title": "Test",
		"type": "object",
		"properties": {
			"userName": {"type": "string"},
			"user-email": {"type": "string"}
		}
	}`

	parser := NewParser(ParserOptions{
		PackageName:  "test",
		UseJSONNames: true,
	})

	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	msg := fdp.MessageType[0]

	// Find userName field (should be sanitized to user_email after sorting)
	var userEmailField *descriptorpb.FieldDescriptorProto
	for _, field := range msg.Field {
		if *field.Name == "user_email" {
			userEmailField = field
			break
		}
	}

	if userEmailField == nil {
		t.Fatal("user_email field not found")
	}

	if userEmailField.JsonName == nil {
		t.Error("Expected JsonName to be set")
	} else if *userEmailField.JsonName != "user-email" {
		t.Errorf("Expected JsonName 'user-email', got %q", *userEmailField.JsonName)
	}
}

// Test 10: Empty message (no properties)
func TestParser_EmptyMessage(t *testing.T) {
	schema := `{
		"title": "EmptyMessage",
		"type": "object"
	}`

	parser := NewParser(ParserOptions{PackageName: "test"})
	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	msg := fdp.MessageType[0]
	if len(msg.Field) != 0 {
		t.Errorf("Expected 0 fields, got %d", len(msg.Field))
	}
}

// Test 11: Dynamic message creation and validation
func TestParser_DynamicMessageCreation(t *testing.T) {
	schema := `{
		"title": "Person",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "format": "int32"},
			"active": {"type": "boolean"}
		},
		"required": ["name"]
	}`

	parser := NewParser(ParserOptions{
		PackageName: "test",
		Syntax:      "proto3",
	})

	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Create a file descriptor from the proto
	fd, err := protodesc.NewFile(fdp, nil)
	if err != nil {
		t.Fatalf("Failed to create file descriptor: %v", err)
	}

	// Get the message descriptor
	md := fd.Messages().Get(0)

	// Create a dynamic message
	msg := dynamicpb.NewMessage(md)

	// Verify we can set fields
	fields := md.Fields()

	// Set name (field 0 after sorting: active, age, name)
	nameField := fields.ByName("name")
	if nameField == nil {
		t.Fatal("name field not found")
	}
	msg.Set(nameField, protoreflect.ValueOfString("John Doe"))

	// Set age
	ageField := fields.ByName("age")
	if ageField == nil {
		t.Fatal("age field not found")
	}
	msg.Set(ageField, protoreflect.ValueOfInt32(30))

	// Set active
	activeField := fields.ByName("active")
	if activeField == nil {
		t.Fatal("active field not found")
	}
	msg.Set(activeField, protoreflect.ValueOfBool(true))

	// Verify values
	if msg.Get(nameField).String() != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %q", msg.Get(nameField).String())
	}
	if msg.Get(ageField).Int() != 30 {
		t.Errorf("Expected age 30, got %d", msg.Get(ageField).Int())
	}
	if !msg.Get(activeField).Bool() {
		t.Error("Expected active to be true")
	}

	// Verify marshaling works
	data, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify unmarshaling works
	msg2 := dynamicpb.NewMessage(md)
	if err := proto.Unmarshal(data, msg2); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if msg2.Get(nameField).String() != "John Doe" {
		t.Error("Value lost after marshal/unmarshal round trip")
	}
}

// Test 12: Default package name
func TestParser_DefaultPackageName(t *testing.T) {
	schema := `{
		"title": "Test",
		"type": "object",
		"properties": {
			"field": {"type": "string"}
		}
	}`

	parser := NewParser(ParserOptions{}) // No package name specified
	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if *fdp.Package != "generated" {
		t.Errorf("Expected default package 'generated', got %q", *fdp.Package)
	}
}

// Test 13: Proto2 syntax
func TestParser_Proto2Syntax(t *testing.T) {
	schema := `{
		"title": "Test",
		"type": "object",
		"properties": {
			"field": {"type": "string"}
		}
	}`

	parser := NewParser(ParserOptions{
		PackageName: "test",
		Syntax:      "proto2",
	})

	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if *fdp.Syntax != "proto2" {
		t.Errorf("Expected syntax 'proto2', got %q", *fdp.Syntax)
	}

	// In proto2, optional fields should not have proto3_optional set
	field := fdp.MessageType[0].Field[0]
	if field.Proto3Optional != nil && *field.Proto3Optional {
		t.Error("proto3_optional should not be set for proto2")
	}
}

// Test 14: Invalid JSON
func TestParser_InvalidJSON(t *testing.T) {
	schema := `{invalid json`

	parser := NewParser(ParserOptions{PackageName: "test"})
	_, err := parser.Parse([]byte(schema))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// Test 15: Complex schema with all basic types
func TestParser_AllBasicTypes(t *testing.T) {
	schema := `{
		"title": "AllTypes",
		"type": "object",
		"properties": {
			"bool_field": {"type": "boolean"},
			"string_field": {"type": "string"},
			"int32_field": {"type": "integer", "format": "int32"},
			"int64_field": {"type": "integer", "format": "int64"},
			"uint32_field": {"type": "integer", "format": "uint32"},
			"uint64_field": {"type": "integer", "format": "uint64"},
			"float_field": {"type": "number", "format": "float"},
			"double_field": {"type": "number", "format": "double"},
			"bytes_field": {"type": "string", "format": "byte"}
		}
	}`

	parser := NewParser(ParserOptions{PackageName: "test"})
	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	msg := fdp.MessageType[0]
	if len(msg.Field) != 9 {
		t.Fatalf("Expected 9 fields, got %d", len(msg.Field))
	}

	expectedTypes := map[string]descriptorpb.FieldDescriptorProto_Type{
		"bool_field":   descriptorpb.FieldDescriptorProto_TYPE_BOOL,
		"string_field": descriptorpb.FieldDescriptorProto_TYPE_STRING,
		"int32_field":  descriptorpb.FieldDescriptorProto_TYPE_INT32,
		"int64_field":  descriptorpb.FieldDescriptorProto_TYPE_INT64,
		"uint32_field": descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		"uint64_field": descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		"float_field":  descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		"double_field": descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
		"bytes_field":  descriptorpb.FieldDescriptorProto_TYPE_BYTES,
	}

	for _, field := range msg.Field {
		expectedType, ok := expectedTypes[*field.Name]
		if !ok {
			t.Errorf("Unexpected field: %s", *field.Name)
			continue
		}
		if *field.Type != expectedType {
			t.Errorf("Field %s: expected type %v, got %v", *field.Name, expectedType, *field.Type)
		}
	}
}

// Test 16: Arrays (repeated fields)
func TestParser_ArrayFields(t *testing.T) {
	schema := `{
		"title": "ArrayTest",
		"type": "object",
		"properties": {
			"tags": {
				"type": "array",
				"items": {
					"type": "string"
				}
			},
			"scores": {
				"type": "array",
				"items": {
					"type": "integer",
					"format": "int32"
				}
			}
		}
	}`

	parser := NewParser(ParserOptions{PackageName: "test"})
	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	msg := fdp.MessageType[0]
	if len(msg.Field) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(msg.Field))
	}

	// Find scores field (alphabetically after tags)
	var scoresField, tagsField *descriptorpb.FieldDescriptorProto
	for _, field := range msg.Field {
		switch field.GetName() {
		case "scores":
			scoresField = field
		case "tags":
			tagsField = field
		}
	}

	// Verify tags field
	if tagsField == nil {
		t.Fatal("tags field not found")
	}
	if *tagsField.Label != descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		t.Error("Expected tags to be LABEL_REPEATED")
	}
	if *tagsField.Type != descriptorpb.FieldDescriptorProto_TYPE_STRING {
		t.Errorf("Expected tags type STRING, got %v", *tagsField.Type)
	}

	// Verify scores field
	if scoresField == nil {
		t.Fatal("scores field not found")
	}
	if *scoresField.Label != descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		t.Error("Expected scores to be LABEL_REPEATED")
	}
	if *scoresField.Type != descriptorpb.FieldDescriptorProto_TYPE_INT32 {
		t.Errorf("Expected scores type INT32, got %v", *scoresField.Type)
	}
}

// Test 17: Nested objects (nested messages)
func TestParser_NestedObjects(t *testing.T) {
	schema := `{
		"title": "User",
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			},
			"address": {
				"type": "object",
				"properties": {
					"street": {"type": "string"},
					"city": {"type": "string"},
					"zipCode": {"type": "string"}
				}
			}
		}
	}`

	parser := NewParser(ParserOptions{PackageName: "test"})
	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have the main message and the nested Address message
	if len(fdp.MessageType) != 2 {
		t.Fatalf("Expected 2 messages (User and Address), got %d", len(fdp.MessageType))
	}

	// Find the User message
	var userMsg, addressMsg *descriptorpb.DescriptorProto
	for _, msg := range fdp.MessageType {
		switch msg.GetName() {
		case "User":
			userMsg = msg
		case "Address":
			addressMsg = msg
		}
	}

	if userMsg == nil {
		t.Fatal("User message not found")
	}
	if addressMsg == nil {
		t.Fatal("Address message not found")
	}

	// Find address field in User
	var addressField *descriptorpb.FieldDescriptorProto
	for _, field := range userMsg.Field {
		if *field.Name == "address" {
			addressField = field
			break
		}
	}

	if addressField == nil {
		t.Fatal("address field not found in User")
	}

	if *addressField.Type != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		t.Errorf("Expected address type MESSAGE, got %v", *addressField.Type)
	}

	if *addressField.TypeName != ".test.Address" {
		t.Errorf("Expected type name '.test.Address', got %q", *addressField.TypeName)
	}

	// Verify Address message has 3 fields
	if len(addressMsg.Field) != 3 {
		t.Fatalf("Expected Address to have 3 fields, got %d", len(addressMsg.Field))
	}

	// Verify fields are alphabetically ordered: city, street, zipCode
	expectedFields := []string{"city", "street", "zipCode"}
	for i, field := range addressMsg.Field {
		if *field.Name != expectedFields[i] {
			t.Errorf("Address field %d: expected %q, got %q", i, expectedFields[i], *field.Name)
		}
	}
}

// Test 18: Enum from string with enum constraint
// func TestParser_EnumFields(t *testing.T) {
// 	schema := `{
// 		"title": "Task",
// 		"type": "object",
// 		"properties": {
// 			"status": {
// 				"type": "string",
// 				"enum": ["pending", "in_progress", "completed", "cancelled"]
// 			}
// 		}
// 	}`

// 	parser := NewParser(ParserOptions{PackageName: "test"})
// 	fdp, err := parser.Parse([]byte(schema))
// 	if err != nil {
// 		t.Fatalf("Parse failed: %v", err)
// 	}

// 	// Should have enum type
// 	if len(fdp.EnumType) != 1 {
// 		t.Fatalf("Expected 1 enum type, got %d", len(fdp.EnumType))
// 	}

// 	enumType := fdp.EnumType[0]
// 	if *enumType.Name != "Status" {
// 		t.Errorf("Expected enum name 'Status', got %q", *enumType.Name)
// 	}

// 	// Verify enum values
// 	expectedValues := map[string]int32{
// 		"STATUS_UNSPECIFIED": 0,
// 		"STATUS_PENDING":     1,
// 		"STATUS_IN_PROGRESS": 2,
// 		"STATUS_COMPLETED":   3,
// 		"STATUS_CANCELLED":   4,
// 	}

// 	if len(enumType.Value) != len(expectedValues) {
// 		t.Fatalf("Expected %d enum values, got %d", len(expectedValues), len(enumType.Value))
// 	}

// 	for _, val := range enumType.Value {
// 		expectedNum, ok := expectedValues[*val.Name]
// 		if !ok {
// 			t.Errorf("Unexpected enum value: %s", *val.Name)
// 			continue
// 		}
// 		if *val.Number != expectedNum {
// 			t.Errorf("Enum value %s: expected number %d, got %d", *val.Name, expectedNum, *val.Number)
// 		}
// 	}

// 	// Verify field uses enum type
// 	msg := fdp.MessageType[0]
// 	statusField := msg.Field[0]
// 	if *statusField.Type != descriptorpb.FieldDescriptorProto_TYPE_ENUM {
// 		t.Errorf("Expected TYPE_ENUM, got %v", *statusField.Type)
// 	}
// 	if *statusField.TypeName != ".test.Status" {
// 		t.Errorf("Expected type name '.test.Status', got %q", *statusField.TypeName)
// 	}
// }

// Test 19: Array of nested objects
func TestParser_ArrayOfObjects(t *testing.T) {
	schema := `{
		"title": "Team",
		"type": "object",
		"properties": {
			"members": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"name": {"type": "string"},
						"role": {"type": "string"}
					}
				}
			}
		}
	}`

	parser := NewParser(ParserOptions{PackageName: "test"})
	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have Team and Members messages
	if len(fdp.MessageType) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(fdp.MessageType))
	}

	var teamMsg, memberMsg *descriptorpb.DescriptorProto
	for _, msg := range fdp.MessageType {
		switch msg.GetName() {
		case "Team":
			teamMsg = msg
		case "Members":
			memberMsg = msg
		}
	}

	if teamMsg == nil {
		t.Fatal("Team message not found")
	}
	if memberMsg == nil {
		t.Fatal("Members message not found")
	}

	// Verify members field
	membersField := teamMsg.Field[0]
	if *membersField.Label != descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		t.Error("Expected members to be LABEL_REPEATED")
	}
	if *membersField.Type != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		t.Errorf("Expected members type MESSAGE, got %v", *membersField.Type)
	}
	if *membersField.TypeName != ".test.Members" {
		t.Errorf("Expected type name '.test.Members', got %q", *membersField.TypeName)
	}

	// Verify Member message
	if len(memberMsg.Field) != 2 {
		t.Fatalf("Expected Members to have 2 fields, got %d", len(memberMsg.Field))
	}
}

// Test 20: $ref references (if implemented)
func TestParser_RefReferences(t *testing.T) {
	schema := `{
		"title": "Order",
		"type": "object",
		"properties": {
			"customer": {
				"$ref": "#/$defs/Customer"
			}
		},
		"$defs": {
			"Customer": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"email": {"type": "string"}
				}
			}
		}
	}`

	parser := NewParser(ParserOptions{PackageName: "test"})
	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have Order and Customer messages
	if len(fdp.MessageType) < 2 {
		t.Fatalf("Expected at least 2 messages, got %d", len(fdp.MessageType))
	}

	var orderMsg, customerMsg *descriptorpb.DescriptorProto
	for _, msg := range fdp.MessageType {
		switch *msg.Name {
		case "Order":
			orderMsg = msg
		case "Customer":
			customerMsg = msg
		}
	}

	if orderMsg == nil {
		t.Fatal("Order message not found")
	}
	if customerMsg == nil {
		t.Fatal("Customer message not found")
	}

	// Verify customer field references Customer message
	customerField := orderMsg.Field[0]
	if *customerField.Type != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		t.Errorf("Expected TYPE_MESSAGE, got %v", *customerField.Type)
	}
	if *customerField.TypeName != ".test.Customer" {
		t.Errorf("Expected type name '.test.Customer', got %q", *customerField.TypeName)
	}
}

// Test 21: x-dtkt-format extension for Go types
func TestParser_DtkFormatExtension(t *testing.T) {
	schema := `{
		"title": "GoTypes",
		"type": "object",
		"properties": {
			"int32_field": {
				"type": "integer",
				"x-dtkt-format": "int32"
			},
			"int64_field": {
				"type": "integer",
				"x-dtkt-format": "int64"
			},
			"uint32_field": {
				"type": "integer",
				"x-dtkt-format": "uint32"
			},
			"uint64_field": {
				"type": "integer",
				"x-dtkt-format": "uint64"
			},
			"float32_field": {
				"type": "number",
				"x-dtkt-format": "float32"
			},
			"float64_field": {
				"type": "number",
				"x-dtkt-format": "float64"
			}
		}
	}`

	parser := NewParser(ParserOptions{PackageName: "test"})
	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	msg := fdp.MessageType[0]

	expectedTypes := map[string]descriptorpb.FieldDescriptorProto_Type{
		"int32_field":   descriptorpb.FieldDescriptorProto_TYPE_INT32,
		"int64_field":   descriptorpb.FieldDescriptorProto_TYPE_INT64,
		"uint32_field":  descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		"uint64_field":  descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		"float32_field": descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		"float64_field": descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
	}

	for _, field := range msg.Field {
		expectedType, ok := expectedTypes[*field.Name]
		if !ok {
			t.Errorf("Unexpected field: %s", *field.Name)
			continue
		}
		if *field.Type != expectedType {
			t.Errorf("Field %s: expected type %v, got %v", *field.Name, expectedType, *field.Type)
		}
	}
}

// Test 22: x-dtkt-format takes precedence over standard format
func TestParser_DtkFormatPrecedence(t *testing.T) {
	schema := `{
		"title": "FormatPrecedence",
		"type": "object",
		"properties": {
			"field1": {
				"type": "integer",
				"format": "int64",
				"x-dtkt-format": "int32"
			},
			"field2": {
				"type": "number",
				"format": "double",
				"x-dtkt-format": "float32"
			}
		}
	}`

	parser := NewParser(ParserOptions{PackageName: "test"})
	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	msg := fdp.MessageType[0]

	// Find field1
	var field1, field2 *descriptorpb.FieldDescriptorProto
	for _, field := range msg.Field {
		switch field.GetName() {
		case "field1":
			field1 = field
		case "field2":
			field2 = field
		}
	}

	// x-dtkt-format should take precedence
	if field1 == nil {
		t.Fatal("field1 not found")
	}
	if *field1.Type != descriptorpb.FieldDescriptorProto_TYPE_INT32 {
		t.Errorf("field1: x-dtkt-format should take precedence, expected INT32, got %v", *field1.Type)
	}

	if field2 == nil {
		t.Fatal("field2 not found")
	}
	if *field2.Type != descriptorpb.FieldDescriptorProto_TYPE_FLOAT {
		t.Errorf("field2: x-dtkt-format should take precedence, expected FLOAT, got %v", *field2.Type)
	}
}

// Test 23: Smaller int types map to protobuf compatible types
func TestParser_SmallerIntTypes(t *testing.T) {
	schema := `{
		"title": "SmallerInts",
		"type": "object",
		"properties": {
			"int8_field": {
				"type": "integer",
				"x-dtkt-format": "int8"
			},
			"int16_field": {
				"type": "integer",
				"x-dtkt-format": "int16"
			},
			"uint8_field": {
				"type": "integer",
				"x-dtkt-format": "uint8"
			},
			"uint16_field": {
				"type": "integer",
				"x-dtkt-format": "uint16"
			}
		}
	}`

	parser := NewParser(ParserOptions{PackageName: "test"})
	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	msg := fdp.MessageType[0]

	// All smaller types should map to 32-bit equivalents
	expectedTypes := map[string]descriptorpb.FieldDescriptorProto_Type{
		"int8_field":   descriptorpb.FieldDescriptorProto_TYPE_INT32,
		"int16_field":  descriptorpb.FieldDescriptorProto_TYPE_INT32,
		"uint8_field":  descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		"uint16_field": descriptorpb.FieldDescriptorProto_TYPE_UINT32,
	}

	for _, field := range msg.Field {
		expectedType, ok := expectedTypes[*field.Name]
		if !ok {
			t.Errorf("Unexpected field: %s", *field.Name)
			continue
		}
		if *field.Type != expectedType {
			t.Errorf("Field %s: expected type %v, got %v", *field.Name, expectedType, *field.Type)
		}
	}
}

// Test 24: ParseMessage convenience method
func TestParser_ParseMessage(t *testing.T) {
	schema := `{
		"title": "User",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "format": "int32"},
			"email": {"type": "string"}
		},
		"required": ["name", "email"]
	}`

	parser := NewParser(ParserOptions{PackageName: "test"})
	md, err := parser.ParseMessage([]byte(schema))
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	// Verify message descriptor properties
	if md.Name() != "User" {
		t.Errorf("Expected message name 'User', got %s", md.Name())
	}

	if md.FullName() != "test.User" {
		t.Errorf("Expected full name 'test.User', got %s", md.FullName())
	}

	if md.Fields().Len() != 3 {
		t.Errorf("Expected 3 fields, got %d", md.Fields().Len())
	}

	// Verify we can create a dynamic message from it
	msg := dynamicpb.NewMessage(md)
	if msg == nil {
		t.Error("Failed to create dynamic message from descriptor")
	}
}

// Test 25: ParseMessageMap convenience method
func TestParser_ParseMessageMap(t *testing.T) {
	schemaMap := map[string]any{
		"title": "Product",
		"type":  "object",
		"properties": map[string]any{
			"id":    map[string]any{"type": "integer", "format": "int64"},
			"name":  map[string]any{"type": "string"},
			"price": map[string]any{"type": "number", "format": "float"},
		},
		"required": []any{"id", "name"},
	}

	parser := NewParser(ParserOptions{PackageName: "store"})
	md, err := parser.ParseMessageMap(schemaMap)
	if err != nil {
		t.Fatalf("ParseMessageMap failed: %v", err)
	}

	if md.Name() != "Product" {
		t.Errorf("Expected message name 'Product', got %s", md.Name())
	}

	if md.Fields().Len() != 3 {
		t.Errorf("Expected 3 fields, got %d", md.Fields().Len())
	}
}

// Benchmark: Parse simple schema
func BenchmarkParser_SimpleSchema(b *testing.B) {
	schema := []byte(`{
		"title": "User",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"},
			"email": {"type": "string"}
		}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewParser(ParserOptions{PackageName: "test"})
		_, err := parser.Parse(schema)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark: Parse complex schema
func BenchmarkParser_ComplexSchema(b *testing.B) {
	// Build a schema with many fields
	props := make(map[string]any)
	for i := 0; i < 100; i++ {
		props[fmt.Sprintf("field_%c_%d", 'a'+i%26, i)] = map[string]any{
			"type": "string",
		}
	}

	schemaMap := map[string]any{
		"title":      "LargeMessage",
		"type":       "object",
		"properties": props,
	}

	schema, _ := json.Marshal(schemaMap)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewParser(ParserOptions{PackageName: "test"})
		_, err := parser.Parse(schema)
		if err != nil {
			b.Fatal(err)
		}
	}
}
