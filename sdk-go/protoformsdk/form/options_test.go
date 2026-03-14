package form_test

// import (
// 	"testing"

// 	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
// 	"github.com/datakit-dev/dtkt-sdk/sdk-go/protoformsdk/form"
// )

// func TestResolveValue(t *testing.T) {
// 	var (
// 		conn     = &corev1.Connection{}
// 		depl     = conn.ProtoReflect().Descriptor().Fields().ByName("deployment")
// 		deplElem = form.NewSelectElement(form.NewElement(depl))
// 		deplResp = &corev1.ListDeploymentsResponse{
// 			Deployments: []*corev1.Deployment{
// 				{
// 					Name:    "users/jordan/integrations/test/deployments/default",
// 					Address: &corev1.Address{Network: "tcp", Target: ":9090"},
// 				},
// 				{
// 					Name:    "users/jordan/integrations/test2/deployments/default",
// 					Address: &corev1.Address{Network: "unix", Target: "/tmp/test.sock"},
// 				},
// 			},
// 		}
// 	)

// 	err := form.BuildOptions(deplResp.ProtoReflect(), deplElem)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if deplElem.GetOptions().Len() != len(deplResp.Deployments) {
// 		t.Fatalf("expected: %d integrations, got: %d", len(deplResp.Deployments), deplElem.GetOptions().Len())
// 	}

// 	deplElem.GetOptions().Range(func(key string, value any) bool {
// 		t.Log(key, value)
// 		return true
// 	})
// }
