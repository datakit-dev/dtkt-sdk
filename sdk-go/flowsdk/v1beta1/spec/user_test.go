package spec_test

// import (
// 	"context"
// 	"testing"

// 	"github.com/datakit-dev/dtkt-sdk/sdk-go/api"
// 	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
// 	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
// 	protoformv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/protoform/v1beta1"
// 	"github.com/datakit-dev/dtkt-sdk/sdk-go/protoformsdk"
// 	"github.com/datakit-dev/dtkt-sdk/sdk-go/protoformsdk/form"
// 	"github.com/datakit-dev/dtkt-sdk/sdk-go/protoformsdk/protos"
// 	"google.golang.org/protobuf/proto"
// 	"google.golang.org/protobuf/reflect/protoreflect"
// )

// var _ protoformsdk.Driver = (*testDriver)(nil)

// type testDriver struct {
// 	t         *testing.T
// 	fieldName string
// 	isConfirm bool
// }

// func (d *testDriver) Run(ctx context.Context, msg proto.Message) error {
// 	d.t.Logf("testDriver.Run called for message: %s", msg.ProtoReflect().Descriptor().FullName())
// 	return nil
// }

// func (d *testDriver) Resolver() bind.Resolver {
// 	return api.V1Beta1
// }

// func (d *testDriver) Caller() bind.Caller {
// 	return bind.InvokeMethodFunc(func(ctx context.Context, method string, req, resp proto.Message) error {
// 		return nil
// 	})
// }

// func (d *testDriver) VisitGroup(group bind.Group) {
// 	d.t.Logf("testDriver.VisitGroup called for group: %s(%T)", group.String(), group)

// 	// if len(group.GetFields()) == 0 {
// 	// 	d.t.Fatal("group has no fields")
// 	// }

// 	for _, field := range group.GetFields() {
// 		if field.Descriptor().Name() == protoreflect.Name(d.fieldName) {
// 			d.t.Log(field.String(), field.GetElements())
// 			if d.isConfirm {
// 				// if confirmElem, ok := field.IsConfirm(); !ok {
// 				// 	d.t.Fatalf("expected confirm element for field: %s", d.fieldName)
// 				// } else if binding, ok := confirmElem.GetBinding(); !ok || binding == nil {
// 				// 	d.t.Fatalf("expected confirm element binding for field: %s", d.fieldName)
// 				// } else {
// 				// 	binding.Set(true)
// 				// }
// 			} else {
// 				d.t.Fatalf("expected: confirm, got: %T", field.GetElements())
// 			}
// 		}
// 	}
// }

// func TestGetUserActionBinding_Confirm(t *testing.T) {
// 	var (
// 		input = &flowv1beta1.UserAction_Input{
// 			Title: "Test Input",
// 			Element: &flowv1beta1.UserAction_Input_Confirm{
// 				Confirm: &protoformv1beta1.ConfirmElement{
// 					Approve: "Yes",
// 					Decline: "No",
// 				},
// 			},
// 		}
// 	)

// 	elem, binding, ok := spec.GetUserActionBinding(input)
// 	if !ok {
// 		t.Fatal("binding not found for input")
// 	} else if elem.GetConfirm() == nil || input.GetConfirm() == nil || !proto.Equal(elem.GetConfirm(), input.GetConfirm()) {
// 		t.Fatal("expected element confirm == input confirm")
// 	}

// 	cache, err := protos.Build(binding)
// 	if err != nil {
// 		t.Fatal(err)
// 	} else if cache.Len() == 0 {
// 		t.Fatal("protoform build len=0")
// 	}

// 	var (
// 		ctx = context.Background()
// 		drv = &testDriver{
// 			t:         t,
// 			fieldName: "value",
// 			isConfirm: true,
// 		}
// 	)

// 	if err := protoformsdk.RunWithMessage(ctx, drv, binding); err != nil {
// 		t.Fatal(err)
// 	} else if val, ok := binding.GetValue().(bool); !ok {
// 		t.Fatalf("expected: bool binding, got: %T", binding.GetValue())
// 	} else if val == false {
// 		t.Fatalf("expected: true, got: %v", binding.GetValue())
// 	}
// }
