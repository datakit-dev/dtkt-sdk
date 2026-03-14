package protoschema_test

import (
	"encoding/json"
	"fmt"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/protoschema"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Example_basicUsage demonstrates how to convert a JSON Schema to a protobuf descriptor
// and use it to create dynamic messages.
func Example_basicUsage() {
	schema := `{
		"title": "User",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "format": "int32"},
			"email": {"type": "string"},
			"active": {"type": "boolean"}
		},
		"required": ["name", "email"]
	}`

	// Create parser
	parser := protoschema.NewParser(protoschema.ParserOptions{
		PackageName: "example",
		Syntax:      "proto3",
	})

	// Parse JSON Schema to protobuf descriptor
	fdp, err := parser.Parse([]byte(schema))
	if err != nil {
		panic(err)
	}

	// Create a file descriptor
	fd, err := protodesc.NewFile(fdp, nil)
	if err != nil {
		panic(err)
	}

	// Get the message descriptor
	md := fd.Messages().Get(0)

	// Create a dynamic message
	msg := dynamicpb.NewMessage(md)

	// Set field values
	msg.Set(md.Fields().ByName("name"), protoreflect.ValueOfString("Alice"))
	msg.Set(md.Fields().ByName("age"), protoreflect.ValueOfInt32(30))
	msg.Set(md.Fields().ByName("email"), protoreflect.ValueOfString("alice@example.com"))
	msg.Set(md.Fields().ByName("active"), protoreflect.ValueOfBool(true))

	// Marshal to bytes
	data, _ := proto.Marshal(msg)

	// Unmarshal back
	msg2 := dynamicpb.NewMessage(md)
	//nolint:errcheck
	proto.Unmarshal(data, msg2)

	fmt.Printf("Name: %s\n", msg2.Get(md.Fields().ByName("name")).String())
	fmt.Printf("Email: %s\n", msg2.Get(md.Fields().ByName("email")).String())
	// Output:
	// Name: Alice
	// Email: alice@example.com
}

// Example_nestedMessages demonstrates handling nested objects.
func Example_nestedMessages() {
	schema := `{
		"title": "Company",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
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

	parser := protoschema.NewParser(protoschema.ParserOptions{
		PackageName: "example",
	})

	fdp, _ := parser.Parse([]byte(schema))
	fd, _ := protodesc.NewFile(fdp, nil)

	// The file descriptor now contains both Company and Address messages
	fmt.Printf("Number of messages: %d\n", fd.Messages().Len())
	fmt.Printf("Message 0: %s\n", fd.Messages().Get(0).Name())
	fmt.Printf("Message 1: %s\n", fd.Messages().Get(1).Name())
	// Output:
	// Number of messages: 2
	// Message 0: Company
	// Message 1: Address
}

// Example_arrayFields demonstrates handling array types (repeated fields).
func Example_arrayFields() {
	schema := `{
		"title": "Playlist",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"songs": {
				"type": "array",
				"items": {"type": "string"}
			},
			"ratings": {
				"type": "array",
				"items": {"type": "integer", "format": "int32"}
			}
		}
	}`

	parser := protoschema.NewParser(protoschema.ParserOptions{
		PackageName: "example",
	})

	fdp, _ := parser.Parse([]byte(schema))
	fd, _ := protodesc.NewFile(fdp, nil)
	md := fd.Messages().Get(0)

	// Check that songs is a repeated field
	songsField := md.Fields().ByName("songs")
	fmt.Printf("songs is repeated: %v\n", songsField.IsList())
	fmt.Printf("songs type: %s\n", songsField.Kind())

	ratingsField := md.Fields().ByName("ratings")
	fmt.Printf("ratings is repeated: %v\n", ratingsField.IsList())
	fmt.Printf("ratings type: %s\n", ratingsField.Kind())
	// Output:
	// songs is repeated: true
	// songs type: string
	// ratings is repeated: true
	// ratings type: int32
}

// Example_enumFields demonstrates handling enums.
// func Example_enumFields() {
// 	schema := `{
// 		"title": "Task",
// 		"type": "object",
// 		"properties": {
// 			"status": {
// 				"type": "string",
// 				"enum": ["pending", "in_progress", "completed"]
// 			}
// 		}
// 	}`

// 	parser := protoschema.NewParser(protoschema.ParserOptions{
// 		PackageName: "example",
// 	})

// 	fdp, _ := parser.Parse([]byte(schema))
// 	fd, _ := protodesc.NewFile(fdp, nil)

// 	// Check enum type
// 	fmt.Printf("Number of enums: %d\n", fd.Enums().Len())
// 	enumDesc := fd.Enums().Get(0)
// 	fmt.Printf("Enum name: %s\n", enumDesc.Name())
// 	fmt.Printf("Enum values: %d\n", enumDesc.Values().Len())

// 	// List enum values
// 	for i := 0; i < enumDesc.Values().Len(); i++ {
// 		val := enumDesc.Values().Get(i)
// 		fmt.Printf("  %s = %d\n", val.Name(), val.Number())
// 	}
// 	// Output:
// 	// Number of enums: 1
// 	// Enum name: Status
// 	// Enum values: 4
// 	//   STATUS_UNSPECIFIED = 0
// 	//   STATUS_PENDING = 1
// 	//   STATUS_IN_PROGRESS = 2
// 	//   STATUS_COMPLETED = 3
// }

// Example_openAPIIntegration shows how this could be used with OpenAPI schemas.
func Example_openAPIIntegration() {
	// Simplified OpenAPI component schema
	openAPISchema := map[string]any{
		"title": "Pet",
		"type":  "object",
		"properties": map[string]any{
			"id":   map[string]any{"type": "integer", "format": "int64"},
			"name": map[string]any{"type": "string"},
			"tag":  map[string]any{"type": "string"},
		},
		"required": []any{"id", "name"},
	}

	schemaJSON, _ := json.Marshal(openAPISchema)

	parser := protoschema.NewParser(protoschema.ParserOptions{
		PackageName: "petstore",
		Syntax:      "proto3",
	})

	fdp, _ := parser.Parse(schemaJSON)
	fd, _ := protodesc.NewFile(fdp, nil)
	md := fd.Messages().Get(0)

	fmt.Printf("Message: %s\n", md.FullName())
	fmt.Printf("Package: %s\n", fd.Package())
	fmt.Printf("Fields:\n")
	for i := 0; i < md.Fields().Len(); i++ {
		field := md.Fields().Get(i)
		fmt.Printf("  %d: %s (%s)\n", field.Number(), field.Name(), field.Kind())
	}
	// Output:
	// Message: petstore.Pet
	// Package: petstore
	// Fields:
	//   1: id (int64)
	//   2: name (string)
	//   3: tag (string)
}

// Example_goTypeRoundTrip demonstrates preserving Go type information through
// the JSON Schema → Proto descriptor conversion using x-dtkt-format annotations.
func Example_goTypeRoundTrip() {
	// type Metrics struct {
	// 	Count    uint32  `json:"count"`
	// 	Average  float32 `json:"average"`
	// 	Total    int64   `json:"total"`
	// 	SmallNum int8    `json:"small_num"`
	// }

	// Generate JSON Schema with type annotations using common package
	schema := `{
		"title": "Metrics",
		"type": "object",
		"properties": {
			"count": {
				"type": "integer",
				"x-dtkt-format": "uint32"
			},
			"average": {
				"type": "number",
				"x-dtkt-format": "float32"
			},
			"total": {
				"type": "integer",
				"x-dtkt-format": "int64"
			},
			"small_num": {
				"type": "integer",
				"x-dtkt-format": "int8"
			}
		}
	}`

	parser := protoschema.NewParser(protoschema.ParserOptions{
		PackageName: "example",
	})

	fdp, _ := parser.Parse([]byte(schema))
	fd, _ := protodesc.NewFile(fdp, nil)
	md := fd.Messages().Get(0)

	// Verify types are correctly preserved
	fmt.Printf("count: %s (from uint32)\n", md.Fields().ByName("count").Kind())
	fmt.Printf("average: %s (from float32)\n", md.Fields().ByName("average").Kind())
	fmt.Printf("total: %s (from int64)\n", md.Fields().ByName("total").Kind())
	fmt.Printf("small_num: %s (from int8, mapped to int32)\n", md.Fields().ByName("small_num").Kind())
	// Output:
	// count: uint32 (from uint32)
	// average: float (from float32)
	// total: int64 (from int64)
	// small_num: int32 (from int8, mapped to int32)
}

// Example_parseMessage demonstrates the ergonomic ParseMessage API that returns
// a message descriptor directly without needing to navigate FileDescriptorProto.
func Example_parseMessage() {
	schema := `{
		"title": "User",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "format": "int32"},
			"email": {"type": "string"},
			"active": {"type": "boolean"}
		},
		"required": ["name", "email"]
	}`

	parser := protoschema.NewParser(protoschema.ParserOptions{
		PackageName: "example",
	})

	// Get the message descriptor directly - no need to navigate FileDescriptorProto
	md, err := parser.ParseMessage([]byte(schema))
	if err != nil {
		panic(err)
	}

	// Immediately usable for creating dynamic messages
	msg := dynamicpb.NewMessage(md)

	// Set field values
	msg.Set(md.Fields().ByName("name"), protoreflect.ValueOfString("Alice"))
	msg.Set(md.Fields().ByName("age"), protoreflect.ValueOfInt32(30))
	msg.Set(md.Fields().ByName("email"), protoreflect.ValueOfString("alice@example.com"))
	msg.Set(md.Fields().ByName("active"), protoreflect.ValueOfBool(true))

	fmt.Printf("Message: %s\n", md.FullName())
	fmt.Printf("Fields: %d\n", md.Fields().Len())
	fmt.Printf("Name: %s\n", msg.Get(md.Fields().ByName("name")).String())
	// Output:
	// Message: example.User
	// Fields: 4
	// Name: Alice
}
