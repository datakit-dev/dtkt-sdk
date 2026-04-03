package runtime

import (
	"testing"

	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	graphlib "github.com/dominikbraun/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildGraph constructs a graph from explicit node IDs and directed edges
// (source → target means target depends on source). No Runtime is involved.
func buildGraph(t *testing.T, ids []string, edges [][2]string) *Graph {
	t.Helper()

	g := &Graph{
		proto:   &flowv1beta1.Graph{},
		forward: make(map[string][]string),
		reverse: make(map[string][]string),
	}
	g.dag = graphlib.NewWithStore(
		GetNodeID,
		g,
		graphlib.Directed(),
		graphlib.PreventCycles(),
	)

	for _, id := range ids {
		node := &flowv1beta1.Node{
			Id: id,
			Type: &flowv1beta1.Node_Var{
				Var: &flowv1beta1.Var{Id: id, Value: `= "x"`},
			},
		}
		require.NoError(t, g.dag.AddVertex(node))
	}

	for _, e := range edges {
		require.NoError(t, g.dag.AddEdge(e[0], e[1]))
	}

	require.NoError(t, g.Build(&Env{}))
	return g
}

// groupOf returns the 0-based group index that nodeID appears in.
func groupOf(exec *Executor, nodeID string) (int, bool) {
	for i, grp := range exec.graph.groups {
		for _, id := range grp {
			if id == nodeID {
				return i, true
			}
		}
	}
	return -1, false
}

// TestGraph_SingleNode — one node, no edges → one group.
func TestGraph_SingleNode(t *testing.T) {
	g := buildGraph(t, []string{"vars.a"}, nil)
	exec, err := NewExecutor(&Runtime{
		env: func() (*Env, error) {
			return NewEnv(&flowv1beta1.Runtime{})
		},
	}, g)
	require.NoError(t, err)
	assert.Equal(t, 1, len(exec.graph.groups))
	idx, ok := groupOf(exec, "vars.a")
	assert.True(t, ok)
	assert.Equal(t, 0, idx)
}

// TestGraph_IndependentNodes — three nodes, no edges → all in group 0.
func TestGraph_IndependentNodes(t *testing.T) {
	g := buildGraph(t, []string{"vars.a", "vars.b", "vars.c"}, nil)
	exec, err := NewExecutor(&Runtime{}, g)
	require.NoError(t, err)
	assert.Equal(t, 1, len(exec.graph.groups))
	for _, id := range []string{"vars.a", "vars.b", "vars.c"} {
		_, ok := groupOf(exec, id)
		assert.True(t, ok, "%s should be in a group", id)
	}
}

// TestGraph_LinearChain — a→b→c must produce three separate groups in order.
func TestGraph_LinearChain(t *testing.T) {
	g := buildGraph(t,
		[]string{"vars.a", "vars.b", "vars.c"},
		[][2]string{{"vars.a", "vars.b"}, {"vars.b", "vars.c"}},
	)
	exec, err := NewExecutor(&Runtime{}, g)
	require.NoError(t, err)
	assert.Equal(t, 3, len(exec.graph.groups))

	ia, _ := groupOf(exec, "vars.a")
	ib, _ := groupOf(exec, "vars.b")
	ic, _ := groupOf(exec, "vars.c")
	assert.Less(t, ia, ib, "a before b")
	assert.Less(t, ib, ic, "b before c")
}

// TestGraph_Diamond — a→b, a→c, b→d, c→d: b and c are parallel; d is last.
func TestGraph_Diamond(t *testing.T) {
	g := buildGraph(t,
		[]string{"vars.a", "vars.b", "vars.c", "vars.d"},
		[][2]string{
			{"vars.a", "vars.b"}, {"vars.a", "vars.c"},
			{"vars.b", "vars.d"}, {"vars.c", "vars.d"},
		},
	)
	exec, err := NewExecutor(&Runtime{}, g)
	require.NoError(t, err)
	assert.Equal(t, 3, len(exec.graph.groups))

	ia, _ := groupOf(exec, "vars.a")
	ib, _ := groupOf(exec, "vars.b")
	ic, _ := groupOf(exec, "vars.c")
	id, _ := groupOf(exec, "vars.d")

	assert.Less(t, ia, ib)
	assert.Equal(t, ib, ic, "b and c should be in the same parallel group")
	assert.Less(t, ic, id)
}

// TestGraph_ComplexDAG — two roots fan out, merge, fan out and merge again.
func TestGraph_ComplexDAG(t *testing.T) {
	//   a   b
	//  / \ / \
	// c   d   e
	//  \ / \ /
	//   f   g
	//    \ /
	//     h
	g := buildGraph(t,
		[]string{"vars.a", "vars.b", "vars.c", "vars.d", "vars.e", "vars.f", "vars.g", "vars.h"},
		[][2]string{
			{"vars.a", "vars.c"}, {"vars.a", "vars.d"},
			{"vars.b", "vars.d"}, {"vars.b", "vars.e"},
			{"vars.c", "vars.f"}, {"vars.d", "vars.f"},
			{"vars.d", "vars.g"}, {"vars.e", "vars.g"},
			{"vars.f", "vars.h"}, {"vars.g", "vars.h"},
		},
	)
	exec, err := NewExecutor(&Runtime{}, g)
	require.NoError(t, err)

	ia, _ := groupOf(exec, "vars.a")
	ib, _ := groupOf(exec, "vars.b")
	ic, _ := groupOf(exec, "vars.c")
	id, _ := groupOf(exec, "vars.d")
	ie, _ := groupOf(exec, "vars.e")
	if_, _ := groupOf(exec, "vars.f")
	ig, _ := groupOf(exec, "vars.g")
	ih, _ := groupOf(exec, "vars.h")

	// roots are parallel
	assert.Equal(t, ia, ib)
	// c, d, e are all one level below roots
	assert.Equal(t, ic, id)
	assert.Equal(t, id, ie)
	// f and g are parallel
	assert.Equal(t, if_, ig)
	// h is last
	assert.Less(t, ig, ih)
}

// TestGraph_ForwardAndReverse verifies the helper maps are populated correctly.
func TestGraph_ForwardAndReverse(t *testing.T) {
	g := buildGraph(t,
		[]string{"vars.a", "vars.b", "vars.c"},
		[][2]string{{"vars.a", "vars.b"}, {"vars.a", "vars.c"}},
	)

	// b and c depend on a
	fwd := g.Forward("vars.b")
	assert.Contains(t, fwd, "vars.a")

	// a has two dependents
	rev := g.Reverse("vars.a")
	assert.Contains(t, rev, "vars.b")
	assert.Contains(t, rev, "vars.c")
}

// TestGraph_StartsAndEnds verifies root/leaf detection.
func TestGraph_StartsAndEnds(t *testing.T) {
	g := buildGraph(t,
		[]string{"vars.a", "vars.b", "vars.c"},
		[][2]string{{"vars.a", "vars.b"}, {"vars.a", "vars.c"}},
	)
	starts := g.Starts()
	ends := g.Ends()

	assert.Contains(t, starts, "vars.a")
	assert.NotContains(t, starts, "vars.b")
	assert.Contains(t, ends, "vars.b")
	assert.Contains(t, ends, "vars.c")
	assert.NotContains(t, ends, "vars.a")
}
