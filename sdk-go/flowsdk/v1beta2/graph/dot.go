package graph

import (
	"bytes"

	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	graphlib "github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
)

// DOT exports a Graph to Graphviz DOT format for debugging/visualization.
func DOT(g *flowv1beta2.Graph) (string, error) {
	dag := graphlib.New(graphlib.StringHash, graphlib.Directed())
	for _, n := range g.GetNodes() {
		_ = dag.AddVertex(n.GetId(), vertexAttributes(n)...)
	}
	for _, e := range g.GetEdges() {
		_ = dag.AddEdge(e.GetSource(), e.GetTarget())
	}

	var buf bytes.Buffer
	if err := draw.DOT(dag, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// vertexAttributes returns graphlib vertex properties for DOT styling based on node type.
func vertexAttributes(node *flowv1beta2.Node) []func(*graphlib.VertexProperties) {
	switch node.WhichType() {
	case flowv1beta2.Node_Input_case:
		return []func(*graphlib.VertexProperties){
			graphlib.VertexAttribute("shape", "ellipse"),
			graphlib.VertexAttribute("style", "filled"),
			graphlib.VertexAttribute("fillcolor", "#e3f2fd"),
		}
	case flowv1beta2.Node_Generator_case:
		return []func(*graphlib.VertexProperties){
			graphlib.VertexAttribute("shape", "diamond"),
			graphlib.VertexAttribute("style", "filled"),
			graphlib.VertexAttribute("fillcolor", "#e8f5e9"),
		}
	case flowv1beta2.Node_Var_case:
		return []func(*graphlib.VertexProperties){
			graphlib.VertexAttribute("shape", "box"),
		}
	case flowv1beta2.Node_Action_case, flowv1beta2.Node_Stream_case:
		return []func(*graphlib.VertexProperties){
			graphlib.VertexAttribute("shape", "box"),
			graphlib.VertexAttribute("style", "rounded,filled"),
			graphlib.VertexAttribute("fillcolor", "#fff3e0"),
		}
	case flowv1beta2.Node_Output_case:
		return []func(*graphlib.VertexProperties){
			graphlib.VertexAttribute("shape", "ellipse"),
			graphlib.VertexAttribute("style", "filled"),
			graphlib.VertexAttribute("fillcolor", "#fce4ec"),
		}
	default:
		return nil
	}
}
