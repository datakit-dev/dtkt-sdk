package graph

import (
	"testing"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild_EdgeInference(t *testing.T) {
	flow := &flowv1beta2.Flow{
		Inputs: []*flowv1beta2.Input{
			{Id: "x", Type: &flowv1beta2.Input_Int64{Int64: &flowv1beta2.Int64{}}},
		},
		Vars: []*flowv1beta2.Var{
			{Id: "doubled", Type: &flowv1beta2.Var_Value{Value: "= inputs.x.value"}},
		},
		Outputs: []*flowv1beta2.Output{
			{Id: "result", Value: "= vars.doubled.value"},
		},
	}

	g, err := Build(flow)
	require.NoError(t, err)

	edges := edgeSet(g)
	assert.Contains(t, edges, "inputs.x->vars.doubled")
	assert.Contains(t, edges, "vars.doubled->outputs.result")
	assert.Len(t, edges, 2)
}

func TestBuild_TransformEdgeInference(t *testing.T) {
	flow := &flowv1beta2.Flow{
		Inputs: []*flowv1beta2.Input{
			{Id: "x", Type: &flowv1beta2.Input_Int64{Int64: &flowv1beta2.Int64{}}},
		},
		Vars: []*flowv1beta2.Var{
			{
				Id:   "sum",
				Type: &flowv1beta2.Var_Value{Value: "= inputs.x.value"},
				Transforms: []*flowv1beta2.Transform{
					{Type: &flowv1beta2.Transform_Scan_{Scan: &flowv1beta2.Transform_Scan{
						Initial:     "= 0",
						Accumulator: "= this.accumulator + this.value",
					}}},
				},
			},
		},
		Outputs: []*flowv1beta2.Output{
			{Id: "result", Value: "= vars.sum.value"},
		},
	}

	g, err := Build(flow)
	require.NoError(t, err)

	edges := edgeSet(g)
	assert.Contains(t, edges, "inputs.x->vars.sum")
	assert.Contains(t, edges, "vars.sum->outputs.result")
}

func TestBuild_ReduceGroupByEdgeInference(t *testing.T) {
	flow := &flowv1beta2.Flow{
		Inputs: []*flowv1beta2.Input{
			{Id: "x", Type: &flowv1beta2.Input_Int64{Int64: &flowv1beta2.Int64{}}},
		},
		Vars: []*flowv1beta2.Var{
			{
				Id:   "grouped",
				Type: &flowv1beta2.Var_Value{Value: "= inputs.x.value"},
				Transforms: []*flowv1beta2.Transform{
					{Type: &flowv1beta2.Transform_Reduce_{Reduce: &flowv1beta2.Transform_Reduce{
						Initial:     "= 0",
						Accumulator: "= this.accumulator + this.value",
						GroupBy: &flowv1beta2.Transform_GroupBy{
							Key: "= this.value % 2",
							Window: &flowv1beta2.Transform_GroupBy_Window{
								Type: &flowv1beta2.Transform_GroupBy_Window_Event_{
									Event: &flowv1beta2.Transform_GroupBy_Window_Event{
										When: "= inputs.x.closed",
									},
								},
							},
						},
					}}},
				},
			},
		},
		Outputs: []*flowv1beta2.Output{
			{Id: "result", Value: "= vars.grouped.value"},
		},
	}

	g, err := Build(flow)
	require.NoError(t, err)

	edges := edgeSet(g)
	// The window's "when" expression references inputs.x, so vars.grouped should have an edge from inputs.x
	assert.Contains(t, edges, "inputs.x->vars.grouped")
}

func TestBuild_DuplicateNodeID(t *testing.T) {
	flow := &flowv1beta2.Flow{
		Inputs: []*flowv1beta2.Input{
			{Id: "x", Type: &flowv1beta2.Input_Int64{Int64: &flowv1beta2.Int64{}}},
			{Id: "x", Type: &flowv1beta2.Input_Int64{Int64: &flowv1beta2.Int64{}}},
		},
	}

	_, err := Build(flow)
	assert.ErrorContains(t, err, "duplicate node ID")
}

func TestBuild_UnknownNodeReference(t *testing.T) {
	flow := &flowv1beta2.Flow{
		Outputs: []*flowv1beta2.Output{
			{Id: "result", Value: "= vars.missing.value"},
		},
	}

	_, err := Build(flow)
	assert.ErrorContains(t, err, "unknown node")
}

func TestBuild_CycleDetection(t *testing.T) {
	// vars.a references vars.b and vars.b references vars.a
	flow := &flowv1beta2.Flow{
		Vars: []*flowv1beta2.Var{
			{Id: "a", Type: &flowv1beta2.Var_Value{Value: "= vars.b.value"}},
			{Id: "b", Type: &flowv1beta2.Var_Value{Value: "= vars.a.value"}},
		},
	}

	_, err := Build(flow)
	assert.ErrorContains(t, err, "cycle")
}

func TestDOT_Produces_Output(t *testing.T) {
	flow := &flowv1beta2.Flow{
		Inputs: []*flowv1beta2.Input{
			{Id: "x", Type: &flowv1beta2.Input_Int64{Int64: &flowv1beta2.Int64{}}},
		},
		Outputs: []*flowv1beta2.Output{
			{Id: "result", Value: "= inputs.x.value"},
		},
	}

	g, err := Build(flow)
	require.NoError(t, err)

	dot, err := DOT(g)
	require.NoError(t, err)
	assert.Contains(t, dot, "digraph")
	assert.Contains(t, dot, "inputs.x")
	assert.Contains(t, dot, "outputs.result")
}

// edgeSet converts graph edges to a set of "source->target" strings.
func edgeSet(g *flowv1beta2.Graph) map[string]bool {
	m := make(map[string]bool)
	for _, e := range g.GetEdges() {
		m[e.GetSource()+"->"+e.GetTarget()] = true
	}
	return m
}
