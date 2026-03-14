package spec

import (
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestInputValidator(t *testing.T) {
	now := timestamppb.Now()
	tests := []struct {
		input          *flowv1beta1.Input
		expectTypeName string
		expectTypeErr  bool
		value          any
		expectValueErr bool
	}{
		{
			input: &flowv1beta1.Input{
				Id: "testBool",
				Type: &flowv1beta1.Input_Bool{
					Bool: &flowv1beta1.Bool{},
				},
			},
			value: true,
		},
		{
			input: &flowv1beta1.Input{
				Id: "testBoolWrapper",
				Type: &flowv1beta1.Input_Bool{
					Bool: &flowv1beta1.Bool{},
				},
			},
			value: wrapperspb.Bool(true),
		},
		{
			input: &flowv1beta1.Input{
				Id: "testMessage",
				Type: &flowv1beta1.Input_Message{
					Message: &flowv1beta1.Message{
						Type: "test.v1.Config",
					},
				},
			},
			expectTypeName: "test.v1.Config",
			value:          &structpb.Struct{},
			expectValueErr: true,
		},
		{
			input: &flowv1beta1.Input{
				Id: "testAnyMessage",
				Type: &flowv1beta1.Input_Message{
					Message: &flowv1beta1.Message{
						Type: "google.protobuf.Any",
					},
				},
			},
			expectTypeName: "google.protobuf.Any",
			value:          &anypb.Any{},
		},
		{
			input: &flowv1beta1.Input{
				Id: "testInt64List",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items: "int64",
					},
				},
			},
			value: []any{
				wrapperspb.Int64(123),
				int64(456),
			},
		},
		{
			input: &flowv1beta1.Input{
				Id: "testMessageList",
				Type: &flowv1beta1.Input_List{
					List: &flowv1beta1.List{
						Items: "google.protobuf.Struct",
					},
				},
			},
			expectTypeName: "google.protobuf.Struct",
			value: []any{
				&structpb.Struct{
					Fields: map[string]*structpb.Value{
						"foo": structpb.NewStringValue("bar"),
					},
				},
				map[string]any{
					"bar": "baz",
				},
			},
		},
		{
			input: &flowv1beta1.Input{
				Id: "testStringDurationMap",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:   "string",
						Value: "google.protobuf.Duration",
						Default: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"foo": structpb.NewStringValue("10s"),
							},
						},
					},
				},
			},
			expectTypeName: "google.protobuf.Duration",
			value: map[string]any{
				"foo": "30s",
				"bar": durationpb.New(15 * time.Second),
			},
		},
		{
			input: &flowv1beta1.Input{
				Id: "testIntToTimestampMap",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:   "int32",
						Value: "google.protobuf.Timestamp",
						Default: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"123": structpb.NewStringValue(now.AsTime().Format(time.RFC3339)),
							},
						},
					},
				},
			},
			expectTypeName: "google.protobuf.Timestamp",
			value: map[any]any{
				123: now,
				456: "2026-02-23T07:05:55Z",
			},
		},
		{
			input: &flowv1beta1.Input{
				Id: "testIntToFloatMap",
				Type: &flowv1beta1.Input_Map{
					Map: &flowv1beta1.Map{
						Key:   "int32",
						Value: "float64",
						Default: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"123": structpb.NewNumberValue(123.456),
							},
						},
					},
				},
			},
			value: map[any]any{
				123: 456.789,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input.Id, func(t *testing.T) {
			validator := NewInputValidator(tt.input)
			inputType, err := NewInputTypeWithResolver(tt.input, validator)
			if err != nil {
				t.Error(err)
			} else {
				def, err := inputType.GetDefault()
				if err != nil {
					t.Error(err)
				} else {
					b, _ := encoding.ToJSONV2(def)
					t.Logf("%s default value: %s", tt.input.Id, string(b))
				}
			}

			if tt.expectTypeName != "" && string(validator.TypeName()) != tt.expectTypeName {
				t.Errorf("expected message name: %s, got: %s", tt.expectTypeName, validator.TypeName())
			} else if tt.expectTypeErr && validator.err == nil {
				t.Errorf("expected type error, got none")
			} else if !tt.expectTypeErr && validator.err != nil {
				t.Errorf("expected no type error, got: %v", validator.err)
			}

			value, err := inputType.Validate(tt.value)
			if tt.expectValueErr && err == nil {
				t.Errorf("expected value error, got none")
			} else if !tt.expectValueErr && err != nil {
				t.Errorf("expected no value error, got: %v", err)
			} else {
				b, _ := encoding.ToJSONV2(value)
				t.Logf("%s value (%T): %s", tt.input.Id, value, string(b))
			}
		})
	}
}
