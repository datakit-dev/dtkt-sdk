package spec_test

import (
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestInputScalarTypes tests all scalar input types
func TestInputScalarTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    *flowv1beta1.Input
		testFunc func(t *testing.T, inputType spec.InputType)
	}{
		{
			name: "bool_required",
			input: &flowv1beta1.Input{
				Id: "test_bool",
				Type: &flowv1beta1.Input_Bool{
					Bool: &flowv1beta1.Bool{
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if !inputType.IsRequired() {
					t.Error("expected required")
				}
				if inputType.HasDefault() {
					t.Error("expected no default")
				}
				if inputType.GetNullable() {
					t.Error("expected not nullable")
				}

				// Test validation with valid value
				result, err := inputType.Validate(true)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != true {
					t.Errorf("expected true, got %v", result)
				}

				// Test validation with nil should error
				_, err = inputType.Validate(nil)
				if err == nil {
					t.Error("expected error for nil value")
				}
			},
		},
		{
			name: "bool_nullable",
			input: &flowv1beta1.Input{
				Id: "test_bool_nullable",
				Type: &flowv1beta1.Input_Bool{
					Bool: &flowv1beta1.Bool{
						Nullable: true,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if inputType.IsRequired() {
					t.Error("expected not required")
				}
				if !inputType.GetNullable() {
					t.Error("expected nullable")
				}

				defVal, err := inputType.GetDefault()
				if err != nil {
					t.Error(err)
				}

				// Test validation with nil
				result, err := inputType.Validate(nil)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != defVal {
					t.Errorf("expected zero, got %v", result)
				}
			},
		},
		{
			name: "bool_with_default",
			input: &flowv1beta1.Input{
				Id: "test_bool_default",
				Type: &flowv1beta1.Input_Bool{
					Bool: &flowv1beta1.Bool{
						Nullable: false,
						Default:  boolPtr(true),
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if inputType.IsRequired() {
					t.Error("expected not required")
				}
				if !inputType.HasDefault() {
					t.Error("expected to have default")
				}

				// Test getting default
				def, err := inputType.GetDefault()
				if err != nil {
					t.Errorf("get default failed: %v", err)
				}
				if def != true {
					t.Errorf("expected default true, got %v", def)
				}

				// Test validation with nil returns default
				result, err := inputType.Validate(nil)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != true {
					t.Errorf("expected default true, got %v", result)
				}
			},
		},
		{
			name: "string_required",
			input: &flowv1beta1.Input{
				Id: "test_string",
				Type: &flowv1beta1.Input_String_{
					String_: &flowv1beta1.String{
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if !inputType.IsRequired() {
					t.Error("expected required")
				}

				// Test validation with valid value
				result, err := inputType.Validate("hello")
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != "hello" {
					t.Errorf("expected 'hello', got %v", result)
				}

				// Test validation with nil should error
				_, err = inputType.Validate(nil)
				if err == nil {
					t.Error("expected error for nil value")
				}
			},
		},
		{
			name: "string_with_default",
			input: &flowv1beta1.Input{
				Id: "test_string_default",
				Type: &flowv1beta1.Input_String_{
					String_: &flowv1beta1.String{
						Nullable: false,
						Default:  stringPtr("default_value"),
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if !inputType.HasDefault() {
					t.Error("expected to have default")
				}

				// Test getting default
				def, err := inputType.GetDefault()
				if err != nil {
					t.Errorf("get default failed: %v", err)
				}
				if def != "default_value" {
					t.Errorf("expected default 'default_value', got %v", def)
				}
			},
		},
		{
			name: "int32_required",
			input: &flowv1beta1.Input{
				Id: "test_int32",
				Type: &flowv1beta1.Input_Int32{
					Int32: &flowv1beta1.Int32{
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if !inputType.IsRequired() {
					t.Error("expected required")
				}

				// Test validation
				result, err := inputType.Validate(int32(42))
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != int32(42) {
					t.Errorf("expected 42, got %v", result)
				}
			},
		},
		{
			name: "int32_with_default",
			input: &flowv1beta1.Input{
				Id: "test_int32_default",
				Type: &flowv1beta1.Input_Int32{
					Int32: &flowv1beta1.Int32{
						Default: int32Ptr(100),
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				def, err := inputType.GetDefault()
				if err != nil {
					t.Errorf("get default failed: %v", err)
				}
				if def != int32(100) {
					t.Errorf("expected 100, got %v", def)
				}
			},
		},
		{
			name: "int64_required",
			input: &flowv1beta1.Input{
				Id: "test_int64",
				Type: &flowv1beta1.Input_Int64{
					Int64: &flowv1beta1.Int64{
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				result, err := inputType.Validate(int64(12345))
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != int64(12345) {
					t.Errorf("expected 12345, got %v", result)
				}
			},
		},
		{
			name: "uint32_required",
			input: &flowv1beta1.Input{
				Id: "test_uint32",
				Type: &flowv1beta1.Input_Uint32{
					Uint32: &flowv1beta1.Uint32{
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				result, err := inputType.Validate(uint32(99))
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != uint32(99) {
					t.Errorf("expected 99, got %v", result)
				}
			},
		},
		{
			name: "uint64_required",
			input: &flowv1beta1.Input{
				Id: "test_uint64",
				Type: &flowv1beta1.Input_Uint64{
					Uint64: &flowv1beta1.Uint64{
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				result, err := inputType.Validate(uint64(88888))
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != uint64(88888) {
					t.Errorf("expected 88888, got %v", result)
				}
			},
		},
		{
			name: "float_required",
			input: &flowv1beta1.Input{
				Id: "test_float",
				Type: &flowv1beta1.Input_Float{
					Float: &flowv1beta1.Float{
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				result, err := inputType.Validate(float32(3.14))
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != float32(3.14) {
					t.Errorf("expected 3.14, got %v", result)
				}
			},
		},
		{
			name: "double_required",
			input: &flowv1beta1.Input{
				Id: "test_double",
				Type: &flowv1beta1.Input_Double{
					Double: &flowv1beta1.Double{
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				result, err := inputType.Validate(float64(2.71828))
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != float64(2.71828) {
					t.Errorf("expected 2.71828, got %v", result)
				}
			},
		},
		{
			name: "bytes_required",
			input: &flowv1beta1.Input{
				Id: "test_bytes",
				Type: &flowv1beta1.Input_Bytes{
					Bytes: &flowv1beta1.Bytes{
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testBytes := []byte("hello world")
				result, err := inputType.Validate(testBytes)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				resultBytes, ok := result.([]byte)
				if !ok {
					t.Errorf("expected []byte, got %T", result)
				}
				if string(resultBytes) != string(testBytes) {
					t.Errorf("expected %v, got %v", testBytes, resultBytes)
				}
			},
		},
		{
			name: "bytes_with_default",
			input: &flowv1beta1.Input{
				Id: "test_bytes_default",
				Type: &flowv1beta1.Input_Bytes{
					Bytes: &flowv1beta1.Bytes{
						Default: []byte("default"),
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				def, err := inputType.GetDefault()
				if err != nil {
					t.Errorf("get default failed: %v", err)
				}
				defBytes, ok := def.([]byte)
				if !ok {
					t.Errorf("expected []byte, got %T", def)
				}
				if string(defBytes) != "default" {
					t.Errorf("expected 'default', got %v", string(defBytes))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputType, err := spec.NewInputTypeWithResolver(tt.input, protoregistry.GlobalTypes)
			if err != nil {
				t.Fatalf("GetInputType failed: %v", err)
			}
			tt.testFunc(t, inputType)
		})
	}
}

// TestInputListTypes tests list input types
func TestInputListTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    *flowv1beta1.Input
		testFunc func(t *testing.T, inputType spec.InputType)
	}{
		{
			name: "list_string_required",
			input: &flowv1beta1.Input{
				Id: "test_list_string",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items:    "string",
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if !inputType.IsRequired() {
					t.Error("expected required")
				}
				if inputType.GetNullable() {
					t.Error("expected not nullable")
				}

				// Test validation with valid value
				testList := []string{"a", "b", "c"}
				result, err := inputType.Validate(testList)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				resultList, ok := result.([]string)
				if !ok {
					t.Errorf("expected []string, got %T", result)
				}
				if len(resultList) != 3 {
					t.Errorf("expected length 3, got %d", len(resultList))
				}

				// Test validation with nil should error
				_, err = inputType.Validate(nil)
				if err == nil {
					t.Error("expected error for nil value")
				}
			},
		},
		{
			name: "list_int32_nullable",
			input: &flowv1beta1.Input{
				Id: "test_list_int32",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items:    "int32",
						Nullable: true,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if inputType.IsRequired() {
					t.Error("expected not required")
				}
				if !inputType.GetNullable() {
					t.Error("expected nullable")
				}

				// Test validation with nil
				result, err := inputType.Validate(nil)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}

				// Test validation with valid value
				testList := []int32{1, 2, 3}
				result, err = inputType.Validate(testList)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				resultList, ok := result.([]int32)
				if !ok {
					t.Errorf("expected []int32, got %T", result)
				}
				if len(resultList) != 3 {
					t.Errorf("expected length 3, got %d", len(resultList))
				}
			},
		},
		{
			name: "list_bool",
			input: &flowv1beta1.Input{
				Id: "test_list_bool",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items:    "bool",
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testList := []bool{true, false, true}
				result, err := inputType.Validate(testList)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				resultList, ok := result.([]bool)
				if !ok {
					t.Errorf("expected []bool, got %T", result)
				}
				if len(resultList) != 3 {
					t.Errorf("expected length 3, got %d", len(resultList))
				}
			},
		},
		{
			name: "list_int64",
			input: &flowv1beta1.Input{
				Id: "test_list_int64",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items: "int64",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testList := []int64{100, 200, 300}
				result, err := inputType.Validate(testList)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if _, ok := result.([]int64); !ok {
					t.Errorf("expected []int64, got %T", result)
				}
			},
		},
		{
			name: "list_uint32",
			input: &flowv1beta1.Input{
				Id: "test_list_uint32",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items: "uint32",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testList := []uint32{10, 20, 30}
				result, err := inputType.Validate(testList)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if _, ok := result.([]uint32); !ok {
					t.Errorf("expected []uint32, got %T", result)
				}
			},
		},
		{
			name: "list_uint64",
			input: &flowv1beta1.Input{
				Id: "test_list_uint64",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items: "uint64",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testList := []uint64{1000, 2000, 3000}
				result, err := inputType.Validate(testList)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if _, ok := result.([]uint64); !ok {
					t.Errorf("expected []uint64, got %T", result)
				}
			},
		},
		{
			name: "list_float",
			input: &flowv1beta1.Input{
				Id: "test_list_float",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items: "float",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testList := []float32{1.1, 2.2, 3.3}
				result, err := inputType.Validate(testList)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if _, ok := result.([]float32); !ok {
					t.Errorf("expected []float32, got %T", result)
				}
			},
		},
		{
			name: "list_double",
			input: &flowv1beta1.Input{
				Id: "test_list_double",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items: "double",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testList := []float64{1.11, 2.22, 3.33}
				result, err := inputType.Validate(testList)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if _, ok := result.([]float64); !ok {
					t.Errorf("expected []float64, got %T", result)
				}
			},
		},
		{
			name: "list_bytes",
			input: &flowv1beta1.Input{
				Id: "test_list_bytes",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items: "bytes",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testList := [][]byte{[]byte("a"), []byte("b")}
				result, err := inputType.Validate(testList)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if _, ok := result.([][]byte); !ok {
					t.Errorf("expected [][]byte, got %T", result)
				}
			},
		},
		{
			name: "list_string_with_default",
			input: &flowv1beta1.Input{
				Id: "test_list_string_default",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items: "string",
						Default: &structpb.ListValue{
							Values: []*structpb.Value{
								structpb.NewStringValue("default1"),
								structpb.NewStringValue("default2"),
							},
						},
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if inputType.IsRequired() {
					t.Error("expected not required due to default")
				}
				if !inputType.HasDefault() {
					t.Error("expected to have default")
				}

				// Test GetDefault
				def, err := inputType.GetDefault()
				if err != nil {
					t.Errorf("get default failed: %v", err)
				}
				if def != nil {
					t.Logf("Got default value: %v", def)
				}

				// Test validation with nil should use default
				result, err := inputType.Validate(nil)
				if err != nil {
					t.Errorf("validation with nil failed: %v", err)
				}
				t.Logf("Validation with nil returned: %v", result)
			},
		},
		{
			name: "list_message_timestamp",
			input: &flowv1beta1.Input{
				Id: "test_list_timestamp",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items: "google.protobuf.Timestamp",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				// Create valid timestamp list
				now := timestamppb.Now()
				later := timestamppb.New(now.AsTime().Add(time.Hour))
				testList := []*timestamppb.Timestamp{now, later}
				result, err := inputType.Validate(testList)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != nil {
					// Message lists return []proto.Message to support dynamicpb.Message
					if msgList, ok := result.([]proto.Message); !ok {
						t.Errorf("expected []proto.Message, got %T", result)
					} else if len(msgList) != 2 {
						t.Errorf("expected 2 messages, got %d", len(msgList))
					} else {
						// Verify using protoreflect that they are Timestamps
						for i, msg := range msgList {
							if msg.ProtoReflect().Descriptor().FullName() != "google.protobuf.Timestamp" {
								t.Errorf("message %d: expected Timestamp, got %s", i, msg.ProtoReflect().Descriptor().FullName())
							}
						}
					}
				}
			},
		},
		{
			name: "list_message",
			input: &flowv1beta1.Input{
				Id: "test_list_message",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items: "google.protobuf.Struct",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				// Create valid struct list
				s1, _ := structpb.NewStruct(map[string]interface{}{"key": "value1"})
				s2, _ := structpb.NewStruct(map[string]interface{}{"key": "value2"})
				testList := []*structpb.Struct{s1, s2}
				result, err := inputType.Validate(testList)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != nil {
					// Message lists return []proto.Message to support dynamicpb.Message
					if msgList, ok := result.([]proto.Message); !ok {
						t.Errorf("expected []proto.Message, got %T", result)
					} else if len(msgList) != 2 {
						t.Errorf("expected 2 messages, got %d", len(msgList))
					} else {
						// Verify using protoreflect that they are Structs
						for i, msg := range msgList {
							if msg.ProtoReflect().Descriptor().FullName() != "google.protobuf.Struct" {
								t.Errorf("message %d: expected Struct, got %s", i, msg.ProtoReflect().Descriptor().FullName())
							}
						}
					}
				}

				// Test with []any conversion
				s3, _ := structpb.NewStruct(map[string]interface{}{"key": "value3"})
				testListAny := []any{s3}
				result2, err := inputType.Validate(testListAny)
				if err != nil {
					t.Errorf("validation with []any failed: %v", err)
				}
				if result2 != nil {
					if msgList, ok := result2.([]proto.Message); !ok {
						t.Errorf("expected []proto.Message, got %T", result2)
					} else if len(msgList) != 1 {
						t.Errorf("expected 1 message, got %d", len(msgList))
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputType, err := spec.NewInputTypeWithResolver(tt.input, protoregistry.GlobalTypes)
			if err != nil {
				t.Fatalf("GetInputType failed: %v", err)
			}
			tt.testFunc(t, inputType)
		})
	}
}

// TestInputMapTypes tests map input types
func TestInputMapTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    *flowv1beta1.Input
		testFunc func(t *testing.T, inputType spec.InputType)
	}{
		{
			name: "map_string_string_required",
			input: &flowv1beta1.Input{
				Id: "test_map_string_string",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:      "string",
						Value:    "string",
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if !inputType.IsRequired() {
					t.Error("expected required")
				}
				if inputType.GetNullable() {
					t.Error("expected not nullable")
				}

				// Test validation with valid value
				testMap := map[string]string{"key1": "value1", "key2": "value2"}
				result, err := inputType.Validate(testMap)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				resultMap, ok := result.(map[string]string)
				if !ok {
					t.Errorf("expected map[string]string, got %T", result)
				}
				if len(resultMap) != 2 {
					t.Errorf("expected length 2, got %d", len(resultMap))
				}

				// Test validation with nil should error
				_, err = inputType.Validate(nil)
				if err == nil {
					t.Error("expected error for nil value")
				}
			},
		},
		{
			name: "map_string_int32_nullable",
			input: &flowv1beta1.Input{
				Id: "test_map_string_int32",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:      "string",
						Value:    "int32",
						Nullable: true,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if inputType.IsRequired() {
					t.Error("expected not required")
				}
				if !inputType.GetNullable() {
					t.Error("expected nullable")
				}

				// Test validation with nil
				result, err := inputType.Validate(nil)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}

				// Test validation with valid value
				testMap := map[string]int32{"a": 1, "b": 2}
				result, err = inputType.Validate(testMap)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if _, ok := result.(map[string]int32); !ok {
					t.Errorf("expected map[string]int32, got %T", result)
				}
			},
		},
		{
			name: "map_int32_string",
			input: &flowv1beta1.Input{
				Id: "test_map_int32_string",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:   "int32",
						Value: "string",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testMap := map[int32]string{1: "one", 2: "two"}
				result, err := inputType.Validate(testMap)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if _, ok := result.(map[int32]string); !ok {
					t.Errorf("expected map[int32]string, got %T", result)
				}
			},
		},
		{
			name: "map_int64_int64",
			input: &flowv1beta1.Input{
				Id: "test_map_int64_int64",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:   "int64",
						Value: "int64",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testMap := map[int64]int64{100: 200, 300: 400}
				result, err := inputType.Validate(testMap)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if _, ok := result.(map[int64]int64); !ok {
					t.Errorf("expected map[int64]int64, got %T", result)
				}
			},
		},
		{
			name: "map_uint32_bool",
			input: &flowv1beta1.Input{
				Id: "test_map_uint32_bool",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:   "uint32",
						Value: "bool",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testMap := map[uint32]bool{1: true, 2: false}
				result, err := inputType.Validate(testMap)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if _, ok := result.(map[uint32]bool); !ok {
					t.Errorf("expected map[uint32]bool, got %T", result)
				}
			},
		},
		{
			name: "map_uint64_float",
			input: &flowv1beta1.Input{
				Id: "test_map_uint64_float",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:   "uint64",
						Value: "float",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testMap := map[uint64]float32{1: 1.1, 2: 2.2}
				result, err := inputType.Validate(testMap)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if _, ok := result.(map[uint64]float32); !ok {
					t.Errorf("expected map[uint64]float32, got %T", result)
				}
			},
		},
		{
			name: "map_bool_double",
			input: &flowv1beta1.Input{
				Id: "test_map_bool_double",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:   "bool",
						Value: "double",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testMap := map[bool]float64{true: 1.11, false: 2.22}
				result, err := inputType.Validate(testMap)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if _, ok := result.(map[bool]float64); !ok {
					t.Errorf("expected map[bool]float64, got %T", result)
				}
			},
		},
		{
			name: "map_string_bytes",
			input: &flowv1beta1.Input{
				Id: "test_map_string_bytes",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:   "string",
						Value: "bytes",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				testMap := map[string][]byte{"a": []byte("data1"), "b": []byte("data2")}
				result, err := inputType.Validate(testMap)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if _, ok := result.(map[string][]byte); !ok {
					t.Errorf("expected map[string][]byte, got %T", result)
				}
			},
		},
		{
			name: "map_string_int32_with_default",
			input: &flowv1beta1.Input{
				Id: "test_map_string_int32_default",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:   "string",
						Value: "int32",
						Default: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"key1": structpb.NewNumberValue(42),
								"key2": structpb.NewNumberValue(99),
							},
						},
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if inputType.IsRequired() {
					t.Error("expected not required due to default")
				}
				if !inputType.HasDefault() {
					t.Error("expected to have default")
				}

				// Test GetDefault
				def, err := inputType.GetDefault()
				if err != nil {
					t.Errorf("get default failed: %v", err)
				}
				if def != nil {
					t.Logf("Got default value: %v", def)
				}

				// Test validation with nil should use default
				result, err := inputType.Validate(nil)
				if err != nil {
					t.Errorf("validation with nil failed: %v", err)
				}
				t.Logf("Validation with nil returned: %v", result)
			},
		},
		{
			name: "map_string_timestamp",
			input: &flowv1beta1.Input{
				Id: "test_map_string_timestamp",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:   "string",
						Value: "google.protobuf.Timestamp",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				now := timestamppb.Now()
				testMap := map[string]*timestamppb.Timestamp{"now": now}
				result, err := inputType.Validate(testMap)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != nil {
					// Message maps return map[K]proto.Message to support dynamicpb.Message
					if msgMap, ok := result.(map[string]proto.Message); !ok {
						t.Errorf("expected map[string]proto.Message, got %T", result)
					} else if len(msgMap) != 1 {
						t.Errorf("expected 1 message, got %d", len(msgMap))
					} else {
						// Verify using protoreflect that it's a Timestamp
						for key, msg := range msgMap {
							if msg.ProtoReflect().Descriptor().FullName() != "google.protobuf.Timestamp" {
								t.Errorf("message for key %s: expected Timestamp, got %s", key, msg.ProtoReflect().Descriptor().FullName())
							}
						}
					}
				}
			},
		},
		{
			name: "map_string_message",
			input: &flowv1beta1.Input{
				Id: "test_map_string_message",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:   "string",
						Value: "google.protobuf.Struct",
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				s1, _ := structpb.NewStruct(map[string]interface{}{"key": "value1"})
				testMap := map[string]*structpb.Struct{"item1": s1}
				result, err := inputType.Validate(testMap)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != nil {
					// Message maps return map[K]proto.Message to support dynamicpb.Message
					if msgMap, ok := result.(map[string]proto.Message); !ok {
						t.Errorf("expected map[string]proto.Message, got %T", result)
					} else if len(msgMap) != 1 {
						t.Errorf("expected 1 message, got %d", len(msgMap))
					} else {
						// Verify using protoreflect that it's a Struct
						for key, msg := range msgMap {
							if msg.ProtoReflect().Descriptor().FullName() != "google.protobuf.Struct" {
								t.Errorf("message for key %s: expected Struct, got %s", key, msg.ProtoReflect().Descriptor().FullName())
							}
						}
					}
				}

				// Test with map[string]any conversion
				s2, _ := structpb.NewStruct(map[string]interface{}{"key": "value2"})
				testMapAny := map[string]any{"item2": s2}
				result2, err := inputType.Validate(testMapAny)
				if err != nil {
					t.Errorf("validation with map[string]any failed: %v", err)
				}
				if result2 != nil {
					if msgMap, ok := result2.(map[string]proto.Message); !ok {
						t.Errorf("expected map[string]proto.Message, got %T", result2)
					} else if len(msgMap) != 1 {
						t.Errorf("expected 1 message, got %d", len(msgMap))
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputType, err := spec.NewInputTypeWithResolver(tt.input, protoregistry.GlobalTypes)
			if err != nil {
				t.Fatalf("GetInputType failed: %v", err)
			}
			tt.testFunc(t, inputType)
		})
	}
}

// TestInputMessageTypes tests message input types
func TestInputMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    *flowv1beta1.Input
		testFunc func(t *testing.T, inputType spec.InputType)
	}{
		{
			name: "message_struct_required",
			input: &flowv1beta1.Input{
				Id: "test_message_struct",
				Type: &flowv1beta1.Input_Message{
					Message: &flowv1beta1.Message{
						Type:     "google.protobuf.Struct",
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if !inputType.IsRequired() {
					t.Error("expected required")
				}
				if inputType.GetNullable() {
					t.Error("expected not nullable")
				}

				// Test validation with valid message
				testStruct, _ := structpb.NewStruct(map[string]interface{}{"key": "value"})
				result, err := inputType.Validate(testStruct)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				resultStruct, ok := result.(*structpb.Struct)
				if !ok {
					t.Errorf("expected *structpb.Struct, got %T", result)
				}
				if resultStruct.Fields["key"].GetStringValue() != "value" {
					t.Error("struct field mismatch")
				}

				// Test validation with nil should error
				_, err = inputType.Validate(nil)
				if err == nil {
					t.Error("expected error for nil value")
				}
			},
		},
		{
			name: "message_struct_nullable",
			input: &flowv1beta1.Input{
				Id: "test_message_struct_nullable",
				Type: &flowv1beta1.Input_Message{
					Message: &flowv1beta1.Message{
						Type:     "google.protobuf.Struct",
						Nullable: true,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if inputType.IsRequired() {
					t.Error("expected not required")
				}
				if !inputType.GetNullable() {
					t.Error("expected nullable")
				}

				// Test validation with nil
				result, err := inputType.Validate(nil)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			},
		},
		{
			name: "message_struct_with_default",
			input: &flowv1beta1.Input{
				Id: "test_message_struct_default",
				Type: &flowv1beta1.Input_Message{
					Message: &flowv1beta1.Message{
						Type:     "google.protobuf.Struct",
						Nullable: false,
						Default: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"default_key": structpb.NewStringValue("default_value"),
							},
						},
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				if inputType.IsRequired() {
					t.Error("expected not required")
				}
				if !inputType.HasDefault() {
					t.Error("expected to have default")
				}

				// Test getting default
				def, err := inputType.GetDefault()
				if err != nil {
					t.Errorf("get default failed: %v", err)
				}
				defStruct, ok := def.(*structpb.Struct)
				if !ok {
					t.Errorf("expected *structpb.Struct, got %T", def)
				}
				if defStruct.Fields["default_key"].GetStringValue() != "default_value" {
					t.Errorf("default struct field mismatch: %v", defStruct)
				}

				// Test validation with nil returns default
				result, err := inputType.Validate(nil)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				resultStruct, ok := result.(*structpb.Struct)
				if !ok {
					t.Errorf("expected *structpb.Struct, got %T", result)
				}
				if resultStruct.Fields["default_key"].GetStringValue() != "default_value" {
					t.Error("validated struct field mismatch")
				}
			},
		},
		{
			name: "message_timestamp",
			input: &flowv1beta1.Input{
				Id: "test_message_timestamp",
				Type: &flowv1beta1.Input_Message{
					Message: &flowv1beta1.Message{
						Type:     "google.protobuf.Timestamp",
						Nullable: false,
					},
				},
			},
			testFunc: func(t *testing.T, inputType spec.InputType) {
				// Test validation with valid timestamp
				testTimestamp := timestamppb.Now()
				result, err := inputType.Validate(testTimestamp)
				if err != nil {
					t.Errorf("validation failed: %v", err)
				}
				resultTimestamp, ok := result.(*timestamppb.Timestamp)
				if !ok {
					t.Errorf("expected *timestamppb.Timestamp, got %T", result)
				}
				if resultTimestamp.Seconds != testTimestamp.Seconds {
					t.Error("timestamp mismatch")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputType, err := spec.NewInputTypeWithResolver(tt.input, protoregistry.GlobalTypes)
			if err != nil {
				t.Fatalf("GetInputType failed: %v", err)
			}
			tt.testFunc(t, inputType)
		})
	}
}

// TestGetInputTypeErrors tests error cases for GetInputType
func TestGetInputTypeErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       *flowv1beta1.Input
		expectError bool
	}{
		{
			name: "invalid_type_empty",
			input: &flowv1beta1.Input{
				Id: "test_invalid",
			},
			expectError: true,
		},
		{
			name: "list_invalid_message_type",
			input: &flowv1beta1.Input{
				Id: "test_list_invalid",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items: "com.invalid.NonExistent",
					},
				},
			},
			expectError: true,
		},
		{
			name: "map_invalid_message_type",
			input: &flowv1beta1.Input{
				Id: "test_map_invalid",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:   "string",
						Value: "com.invalid.NonExistent",
					},
				},
			},
			expectError: true,
		},
		{
			name: "message_invalid_type",
			input: &flowv1beta1.Input{
				Id: "test_message_invalid",
				Type: &flowv1beta1.Input_Message{
					Message: &flowv1beta1.Message{
						Type: "com.invalid.NonExistent",
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := spec.NewInputTypeWithResolver(tt.input, protoregistry.GlobalTypes)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Helper functions for pointer values
func boolPtr(v bool) *bool {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func int32Ptr(v int32) *int32 {
	return &v
}
