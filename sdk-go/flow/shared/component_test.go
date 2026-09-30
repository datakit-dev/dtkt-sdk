package shared_test

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/shared"
)

// testNode is a minimal shared.Node implementation used to exercise
// shared.Graph/shared.Executor topology and scheduling behavior directly,
// without going through the CEL/proto machinery in runtime.Node.
type testNode struct {
	ident string
	delay time.Duration

	send chan any
	recv chan any

	mut  sync.RWMutex
	curr any
}

func newTestNode(ident string, delay time.Duration) *testNode {
	return &testNode{
		ident: ident,
		delay: delay,
		send:  make(chan any),
		recv:  make(chan any),
	}
}

func (n *testNode) Ident() string    { return n.ident }
func (n *testNode) Send() chan<- any { return n.send }
func (n *testNode) Recv() <-chan any { return n.recv }

func (n *testNode) Value() any {
	n.mut.RLock()
	defer n.mut.RUnlock()
	return n.curr
}

// Run echoes whatever it receives on send back out on recv, optionally
// after a fixed per-round delay (used to simulate a throttled/paced branch).
func (n *testNode) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case v := <-n.send:
			if n.delay > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(n.delay):
				}
			}

			n.mut.Lock()
			n.curr = v
			n.mut.Unlock()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case n.recv <- v:
			}
		}
	}
}

// buildGraph constructs a shared.Graph[*testNode] from a node list and a set
// of source -> target edges, bypassing Graph.Parse (which requires CEL/proto
// SpecNodes) since these tests only care about topology and scheduling.
func buildGraph(t *testing.T, nodes []*testNode, edges [][2]string) *shared.Graph[*testNode] {
	t.Helper()

	var sharedEdges shared.Edges
	for _, e := range edges {
		edge, err := shared.NewEdge(e[0], e[1])
		if err != nil {
			t.Fatal(err)
		}
		sharedEdges = append(sharedEdges, edge)
	}

	g, err := shared.NewGraph(
		shared.WithGraphNodes(nodes),
		shared.WithGraphEdges[*testNode](sharedEdges),
		shared.WithGraphTraits[*testNode](shared.DAGTraits()),
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, node := range nodes {
		if err := g.AddVertex(node); err != nil {
			t.Fatalf("add vertex %q: %v", node.Ident(), err)
		}
	}
	for _, e := range edges {
		if err := g.AddEdge(e[0], e[1]); err != nil {
			t.Fatalf("add edge %s -> %s: %v", e[0], e[1], err)
		}
	}

	return g
}

func identSet(nodes []*testNode) []string {
	idents := make([]string, len(nodes))
	for i, node := range nodes {
		idents[i] = node.Ident()
	}
	slices.Sort(idents)
	return idents
}

func TestComponents(t *testing.T) {
	t.Run("fully connected graph is a single, unmodified component", func(t *testing.T) {
		a, b, c := newTestNode("vars.a", 0), newTestNode("vars.b", 0), newTestNode("vars.c", 0)
		g := buildGraph(t, []*testNode{a, b, c}, [][2]string{
			{"vars.a", "vars.b"},
			{"vars.b", "vars.c"},
		})

		components, err := shared.Components(g)
		if err != nil {
			t.Fatal(err)
		}
		if len(components) != 1 {
			t.Fatalf("expected 1 component, got %d", len(components))
		}
		if components[0] != g {
			t.Fatalf("expected the single-component fast path to return the original graph unchanged")
		}
	})

	t.Run("disjoint branches split into separate components", func(t *testing.T) {
		a, b := newTestNode("vars.a", 0), newTestNode("vars.b", 0)
		c, d := newTestNode("vars.c", 0), newTestNode("vars.d", 0)
		g := buildGraph(t, []*testNode{a, b, c, d}, [][2]string{
			{"vars.a", "vars.b"},
			{"vars.c", "vars.d"},
		})

		components, err := shared.Components(g)
		if err != nil {
			t.Fatal(err)
		}
		if len(components) != 2 {
			t.Fatalf("expected 2 components, got %d", len(components))
		}

		var got [][]string
		for _, component := range components {
			got = append(got, identSet(component.Nodes()))
		}
		slices.SortFunc(got, func(a, b []string) int {
			if len(a) == 0 || len(b) == 0 {
				return len(a) - len(b)
			}
			switch {
			case a[0] < b[0]:
				return -1
			case a[0] > b[0]:
				return 1
			default:
				return 0
			}
		})

		want := [][]string{
			{"vars.a", "vars.b"},
			{"vars.c", "vars.d"},
		}
		if !slices.EqualFunc(got, want, slices.Equal) {
			t.Fatalf("expected components %v, got %v", want, got)
		}
	})

	t.Run("fully isolated nodes are all singleton components", func(t *testing.T) {
		a, b, c := newTestNode("vars.a", 0), newTestNode("vars.b", 0), newTestNode("vars.c", 0)
		g := buildGraph(t, []*testNode{a, b, c}, nil)

		components, err := shared.Components(g)
		if err != nil {
			t.Fatal(err)
		}
		if len(components) != 3 {
			t.Fatalf("expected 3 components, got %d", len(components))
		}
		for _, component := range components {
			if len(component.Nodes()) != 1 {
				t.Fatalf("expected each component to have exactly 1 node, got %d", len(component.Nodes()))
			}
		}
	})
}

// TestRunPartitioned verifies that RunPartitioned schedules independent
// weakly-connected components on independent cadences: a component whose
// operators are individually throttled must not throttle an unrelated,
// disconnected component sharing the same graph.
func TestRunPartitioned(t *testing.T) {
	slowSrc := newTestNode("vars.slowSrc", 20*time.Millisecond)
	slowSink := newTestNode("vars.slowSink", 0)
	fastSrc := newTestNode("vars.fastSrc", 0)
	fastSink := newTestNode("vars.fastSink", 0)

	g := buildGraph(t,
		[]*testNode{slowSrc, slowSink, fastSrc, fastSink},
		[][2]string{
			{"vars.slowSrc", "vars.slowSink"},
			{"vars.fastSrc", "vars.fastSink"},
		},
	)

	components, err := shared.Components(g)
	if err != nil {
		t.Fatal(err)
	}
	if len(components) != 2 {
		t.Fatalf("expected 2 weakly-connected components, got %d", len(components))
	}

	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()

	var slowRounds, fastRounds atomic.Uint64

	err = shared.RunPartitioned(ctx, g,
		shared.WithAfterEnd(func(_ context.Context, nodes shared.GraphNodes[*testNode]) error {
			for _, node := range nodes {
				switch node.Ident() {
				case "vars.slowSink":
					slowRounds.Add(1)
				case "vars.fastSink":
					fastRounds.Add(1)
				}
			}
			return nil
		}),
	)
	if err != nil && ctx.Err() == nil {
		t.Fatal(err)
	}

	t.Logf("slow rounds: %d, fast rounds: %d", slowRounds.Load(), fastRounds.Load())

	if fastRounds.Load() < slowRounds.Load()*5 {
		t.Fatalf(
			"expected the fast component to run at its own, unthrottled cadence: fast=%d slow=%d",
			fastRounds.Load(), slowRounds.Load(),
		)
	}
}
