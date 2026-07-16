package runtime

import (
	"context"
	"maps"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
)

type extendEnv struct {
	*Env
	vars map[string]any
}

func (e extendEnv) Vars(context.Context) (cel.Activation, error) {
	vars := e.nodes.Vars(e.types)
	maps.Copy(vars, e.vars)
	return cel.NewActivation(vars)
}

func TestEnv_Extend(t *testing.T) {
	env, err := NewEnv(ProtoFromSpec(&flowv1beta1.Flow{
		Outputs: []*flowv1beta1.Output{
			{
				Id:    "test",
				Value: "= this.name",
			},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	node, ok := env.nodes.Load("outputs.test")
	if !ok {
		t.Fatal("expected node outputs.test")
	}

	extendedEnv, err := env.Extend(cel.Variable("this", cel.DynType))
	if err != nil {
		t.Fatal(err)
	}

	err = node.Parse(extendedEnv, func(id string, expr *cel.Ast) {})
	if err != nil {
		t.Fatal(err)
	}

	err = node.Compile(t.Context(), extendedEnv)
	if err != nil {
		t.Fatal(err)
	}

	if node, ok := node.exec.(*spec.Output); ok {
		val, err := node.Eval(t.Context(), extendEnv{
			Env: extendedEnv.(*Env),
			vars: map[string]any{
				"this": map[string]any{
					"name": "Jordan",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		} else {
			t.Log(val)
		}
	}
}
