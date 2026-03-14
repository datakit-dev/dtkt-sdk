package spec_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
)

func TestGetID(t *testing.T) {
	tests := []struct {
		expected string
		returned string
	}{
		{
			returned: spec.GetID(&flowv1beta1.Connection{
				Id: "conn1",
			}),
			expected: "connections.conn1",
		},
		{
			returned: spec.GetID(&flowv1beta1.Input{
				Id: "input1",
			}),
			expected: "inputs.input1",
		},
		{
			returned: spec.GetID(&flowv1beta1.Var{
				Id: "var1",
			}),
			expected: "vars.var1",
		},
		{
			returned: spec.GetID(&flowv1beta1.Action{
				Id: "action1",
			}),
			expected: "actions.action1",
		},
		{
			returned: spec.GetID(&flowv1beta1.Stream{
				Id: "stream1",
			}),
			expected: "streams.stream1",
		},
		{
			returned: spec.GetID(&flowv1beta1.Output{
				Id: "output1",
			}),
			expected: "outputs.output1",
		},
	}
	for _, test := range tests {
		if test.expected != test.returned {
			t.Fatalf("expected: %s, got: %s", test.expected, test.returned)
		}
	}
}
