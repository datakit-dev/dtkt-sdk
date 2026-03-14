package v1beta1_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/integrationsdk/v1beta1"
	actionv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/action/v1beta1"
	basev1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/base/v1beta1"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
)

var instMux = &testMux{
	registry: v1beta1.DefaultTypeRegistry(),
}

type (
	testMux struct {
		v1beta1.InstanceMux[*testInstance]
		registry *v1beta1.TypeRegistry
	}
	testInstance struct {
		basev1beta1.UnimplementedBaseServiceServer
	}
)

func (i *testMux) Types() *v1beta1.TypeRegistry {
	return i.registry
}

func (i *testInstance) Close() error {
	return nil
}

func testSimpleFunc(ctx context.Context, mux v1beta1.InstanceMux[*testInstance], input *structpb.Struct) (map[string]any, error) {
	return input.AsMap(), nil
}

func testLongRunning(ctx context.Context, mux v1beta1.InstanceMux[*testInstance], input *structpb.Struct) (map[string]any, error) {
	for {
		select {
		case <-ctx.Done():
			return input.AsMap(), nil
		case <-time.After(time.Second):
			fmt.Println("still waiting...")
		}
	}
}

func TestSimpleAction(t *testing.T) {
	action, err := v1beta1.NewAction(
		"Foo Bar", "Test action.",
		testSimpleFunc,
	)(instMux)
	if err != nil {
		t.Fatal(err)
	} else if action.Proto().GetInput().GetProtoName() == "" {
		t.Fatalf("expected input message: google.protobuf.Struct")
	}

	if action.Proto().GetName() != "actions/foo-bar" {
		t.Fatalf("expected: actions/foo-bar, got: %s", action.Proto().GetName())
	}

	expectedOutput := map[string]any{
		"foo": "bar",
	}

	input, err := common.WrapProtoAny(expectedOutput)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := action.Execute(context.Background(), &actionv1beta1.ExecuteActionRequest{
		Input: input,
	})
	if err != nil {
		t.Fatal(err)
	}

	output, err := common.UnwrapProtoAny(resp.GetOutput())
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(input, resp.GetOutput()) {
		t.Fatalf("expected matching input/output, expected: %v, got: %v", expectedOutput, output)
	} else {
		t.Logf("input: %v, output: %v", expectedOutput, output)
	}
}

func TestLongRunningAction(t *testing.T) {
	action, err := v1beta1.NewAction(
		"Long Boi", "Test long running action.",
		testLongRunning,
	)(instMux)
	if err != nil {
		t.Fatal(err)
	} else if action.Proto().GetInput().GetProtoName() == "" {
		t.Fatalf("expected input message: google.protobuf.Struct")
	}

	if action.Proto().GetName() != "actions/long-boi" {
		t.Fatalf("expected: actions/long-boi, got: %s", action.Proto().GetName())
	}

	expectedOutput := map[string]any{
		"foo": "bar",
	}

	input, err := common.WrapProtoAny(expectedOutput)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := action.Execute(context.Background(), &actionv1beta1.ExecuteActionRequest{
		Input:   input,
		Timeout: durationpb.New(5 * time.Second),
	})
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}

	output, err := common.UnwrapProtoAny(resp.GetOutput())
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(input, resp.GetOutput()) {
		t.Fatalf("expected matching input/output, expected: %v, got: %v", expectedOutput, output)
	} else {
		t.Logf("input: %v, output: %v", expectedOutput, output)
	}
}
