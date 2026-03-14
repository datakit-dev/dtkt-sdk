package runtime

import (
	"fmt"
	"slices"
	"testing"

	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	graphlib "github.com/dominikbraun/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a simple node
func createSimpleNode(id string) *flowv1beta1.Node {
	return &flowv1beta1.Node{
		Id: id,
		Type: &flowv1beta1.Node_Var{
			Var: &flowv1beta1.Var{
				Id: id,
				Type: &flowv1beta1.Var_Value{
					Value: fmt.Sprintf(`= "%s"`, id),
				},
			},
		},
	}
}

// Helper function to create an ExecGroups from manually constructed edges
func newTestExecGroupsFromEdges(t *testing.T, nodes []*flowv1beta1.Node, edges [][2]string) *Executor {
	t.Helper()

	syncMap := util.NewSyncMap[string, *Node]()
	for _, protoNode := range nodes {
		node := &Node{proto: protoNode}
		syncMap.Store(protoNode.GetId(), node)
	}

	run := &Runtime{
		// ctx:   context.Background(),
		nodes: syncMap,
		// proto: &flowv1beta1.Runtime{},
	}

	// Create graph manually without calling GraphFromProto to avoid automatic Build()
	// This allows us to add edges and then call Build() explicitly for testing
	graph := &Graph{
		proto:   &flowv1beta1.Graph{},
		forward: make(map[string][]string),
		reverse: make(map[string][]string),
	}
	graph.Graph = graphlib.NewWithStore(
		GetNodeID,
		graph,
		graphlib.Directed(),
		graphlib.PreventCycles(),
	)

	for _, node := range nodes {
		err := graph.Graph.AddVertex(node)
		require.NoError(t, err)
	}

	for _, edge := range edges {
		err := graph.Graph.AddEdge(edge[0], edge[1])
		require.NoError(t, err)
	}

	err := graph.Build()
	require.NoError(t, err)

	// Now create proto using the same logic as NewExecGroups
	proto := &flowv1beta1.Groups{}

	order, err := graphlib.TopologicalSort(graph.Graph)
	require.NoError(t, err)

	processed := make(map[string]bool)

	// Use the forward/reverse maps that were saved before transitive reduction
	for len(processed) < len(order) {
		var nodeIDs []string

		for _, nodeID := range order {
			if processed[nodeID] {
				continue
			}

			// Use the forward map which contains predecessors
			deps := graph.Forward(nodeID)

			allDepsProcessed := true
			for _, depID := range deps {
				if !processed[depID] {
					allDepsProcessed = false
					break
				}
			}

			if allDepsProcessed {
				_, ok := run.nodes.Load(nodeID)
				require.True(t, ok, "node %s should exist", nodeID)
				nodeIDs = append(nodeIDs, nodeID)
			}
		}

		// Mark all nodes in this group as processed AFTER building the group
		for _, nodeID := range nodeIDs {
			processed[nodeID] = true
		}

		if len(nodeIDs) > 0 {
			proto.Groups = append(proto.Groups, &flowv1beta1.Groups_Group{
				Ids: nodeIDs,
			})
		} else {
			break
		}
	}

	groups, err := NewExecutor(graph)
	require.NoError(t, err)

	return groups
}

// Helper to verify node is in a specific group
func assertNodeInGroup(t *testing.T, groups *Executor, nodeID string, groupIndex int) {
	t.Helper()

	group, ok := groups.LoadGroup(groupIndex)
	require.True(t, ok, "group %d should exist", groupIndex)

	found := slices.Contains(group, nodeID)

	t.Logf("assertNodeInGroup: %d %s %t %#v", groupIndex, nodeID, found, group)

	assert.True(t, found, "node %s should be in %d", nodeID, groupIndex)
}

// Helper to verify nodes are NOT in the same group
func assertNodesInDifferentGroups(t *testing.T, graph *Executor, nodeID1, nodeID2 string) {
	t.Helper()

	var group1ID, group2ID int
	graph.RangeGroups(func(idx int, ids []string) bool {
		for _, id := range ids {
			if id == nodeID1 {
				group1ID = idx
			}
			if id == nodeID2 {
				group2ID = idx
			}
		}
		return true
	})

	t.Log("assertNodesInDifferentGroups", nodeID1, group1ID)
	t.Log("assertNodesInDifferentGroups", nodeID2, group2ID)

	// require.NotZero(t, group1ID, "node1 %d should be in a group", nodeID1)
	// require.NotZero(t, group2ID, "node2 %d should be in a group", nodeID2)
	assert.NotEqual(t, group1ID, group2ID, "nodes %d and %d should be in different groups", nodeID1, nodeID2)
}

// Helper to verify nodes ARE in the same group
func assertNodesInSameGroup(t *testing.T, graph *Executor, nodeID1, nodeID2 string) {
	t.Helper()

	var group1ID, group2ID int
	graph.RangeGroups(func(groupID int, ids []string) bool {
		for _, id := range ids {
			if id == nodeID1 {
				group1ID = groupID
			}
			if id == nodeID2 {
				group2ID = groupID
			}
		}
		return true
	})

	t.Log(nodeID1, group1ID)
	t.Log(nodeID2, group2ID)

	// require.NotEmpty(t, group1ID, "node1 %d should be in a group", nodeID1)
	// require.NotEmpty(t, group2ID, "node2 %d should be in a group", nodeID2)
	assert.Equal(t, group1ID, group2ID, "nodes %d and %d should be in same group", nodeID1, nodeID2)
}

// Helper to count total groups
func countGroups(groups *Executor) int {
	return len(groups.proto.Groups)
}

// === TESTS ===

func TestNewExecGroups_SingleNode(t *testing.T) {
	nodes := []*flowv1beta1.Node{
		createSimpleNode("vars.a"),
	}
	edges := [][2]string{}

	graph := newTestExecGroupsFromEdges(t, nodes, edges)

	require.NotNil(t, graph)
	assert.Equal(t, 1, countGroups(graph), "should have exactly 1 group")
	assertNodeInGroup(t, graph, "vars.a", 0)
}

func TestNewExecGroups_IndependentNodes(t *testing.T) {
	// Three nodes with no dependencies should all be in the same group
	nodes := []*flowv1beta1.Node{
		createSimpleNode("vars.a"),
		createSimpleNode("vars.b"),
		createSimpleNode("vars.c"),
	}
	edges := [][2]string{}

	graph := newTestExecGroupsFromEdges(t, nodes, edges)

	require.NotNil(t, graph)
	assert.Equal(t, 1, countGroups(graph), "independent nodes should be in one group")

	// All nodes should be in group 0
	assertNodeInGroup(t, graph, "vars.a", 0)
	assertNodeInGroup(t, graph, "vars.b", 0)
	assertNodeInGroup(t, graph, "vars.c", 0)
}

func TestNewExecGroups_LinearChain(t *testing.T) {
	// Create a linear dependency chain: a -> b -> c
	nodes := []*flowv1beta1.Node{
		createSimpleNode("vars.a"),
		createSimpleNode("vars.b"),
		createSimpleNode("vars.c"),
	}
	edges := [][2]string{
		{"vars.a", "vars.b"}, // b depends on a
		{"vars.b", "vars.c"}, // c depends on b
	}

	groups := newTestExecGroupsFromEdges(t, nodes, edges)

	require.NotNil(t, groups)
	assert.Equal(t, 3, countGroups(groups), "linear chain should have 3 groups")

	// Verify execution order: a (group 0), b (group 1), c (group 2)
	assertNodeInGroup(t, groups, "vars.a", 0)
	assertNodeInGroup(t, groups, "vars.b", 1)
	assertNodeInGroup(t, groups, "vars.c", 2)

	// Verify dependent nodes are in different groups
	assertNodesInDifferentGroups(t, groups, "vars.a", "vars.b")
	assertNodesInDifferentGroups(t, groups, "vars.b", "vars.c")
}

func TestNewExecGroups_DiamondPattern(t *testing.T) {
	// Diamond dependency: a -> b, a -> c, b -> d, c -> d
	//       a
	//      / \
	//     b   c
	//      \ /
	//       d
	nodes := []*flowv1beta1.Node{
		createSimpleNode("vars.a"),
		createSimpleNode("vars.b"),
		createSimpleNode("vars.c"),
		createSimpleNode("vars.d"),
	}
	edges := [][2]string{
		{"vars.a", "vars.b"},
		{"vars.a", "vars.c"},
		{"vars.b", "vars.d"},
		{"vars.c", "vars.d"},
	}

	graph := newTestExecGroupsFromEdges(t, nodes, edges)

	require.NotNil(t, graph)
	assert.Equal(t, 3, countGroups(graph), "diamond should have 3 groups")

	// a should be in first group (group 0)
	assertNodeInGroup(t, graph, "vars.a", 0)

	// b and c should be in same group (group 1) since they only depend on a
	assertNodesInSameGroup(t, graph, "vars.b", "vars.c")
	assertNodeInGroup(t, graph, "vars.b", 1)
	assertNodeInGroup(t, graph, "vars.c", 1)

	// d should be in last group (group 2) since it depends on b and c
	assertNodeInGroup(t, graph, "vars.d", 2)

	// Verify dependencies are in different groups
	assertNodesInDifferentGroups(t, graph, "vars.a", "vars.b")
	assertNodesInDifferentGroups(t, graph, "vars.b", "vars.d")
}

func TestNewExecGroups_ComplexDAG(t *testing.T) {
	// More complex DAG:
	//     a    b
	//    / \  / \
	//   c   d   e
	//    \ / \ /
	//     f   g
	//      \ /
	//       h
	nodes := []*flowv1beta1.Node{
		createSimpleNode("vars.a"),
		createSimpleNode("vars.b"),
		createSimpleNode("vars.c"),
		createSimpleNode("vars.d"),
		createSimpleNode("vars.e"),
		createSimpleNode("vars.f"),
		createSimpleNode("vars.g"),
		createSimpleNode("vars.h"),
	}
	edges := [][2]string{
		{"vars.a", "vars.c"},
		{"vars.a", "vars.d"},
		{"vars.b", "vars.d"},
		{"vars.b", "vars.e"},
		{"vars.c", "vars.f"},
		{"vars.d", "vars.f"},
		{"vars.d", "vars.g"},
		{"vars.e", "vars.g"},
		{"vars.f", "vars.h"},
		{"vars.g", "vars.h"},
	}

	graph := newTestExecGroupsFromEdges(t, nodes, edges)

	require.NotNil(t, graph)

	// a and b should be in same group (no dependencies)
	assertNodesInSameGroup(t, graph, "vars.a", "vars.b")

	// c, d, e should be in same group (all depend only on group 0)
	assertNodesInSameGroup(t, graph, "vars.c", "vars.d")
	assertNodesInSameGroup(t, graph, "vars.d", "vars.e")

	// f and g should be in same group
	assertNodesInSameGroup(t, graph, "vars.f", "vars.g")

	// h should be alone in final group
	assertNodesInDifferentGroups(t, graph, "vars.f", "vars.h")
	assertNodesInDifferentGroups(t, graph, "vars.g", "vars.h")
}

func TestNewExecGroups_DependentNodesMustBeInDifferentGroups(t *testing.T) {
	// This is the KEY test for the bug fix:
	// Ensure that if node B depends on node A, they are NEVER in the same group
	//
	// Pattern:
	//   a -> b
	//   c -> d
	//   b -> e
	//   d -> e
	nodes := []*flowv1beta1.Node{
		createSimpleNode("vars.a"),
		createSimpleNode("vars.b"),
		createSimpleNode("vars.c"),
		createSimpleNode("vars.d"),
		createSimpleNode("vars.e"),
	}
	edges := [][2]string{
		{"vars.a", "vars.b"},
		{"vars.c", "vars.d"},
		{"vars.b", "vars.e"},
		{"vars.d", "vars.e"},
	}

	graph := newTestExecGroupsFromEdges(t, nodes, edges)

	require.NotNil(t, graph)

	// a and c should be in same group (independent roots)
	assertNodesInSameGroup(t, graph, "vars.a", "vars.c")

	// b and d should be in same group (both depend only on group 0)
	assertNodesInSameGroup(t, graph, "vars.b", "vars.d")

	// e should be in its own group
	assertNodesInDifferentGroups(t, graph, "vars.b", "vars.e")
	assertNodesInDifferentGroups(t, graph, "vars.d", "vars.e")

	// Most importantly: dependent nodes must be in different groups
	assertNodesInDifferentGroups(t, graph, "vars.a", "vars.b")
	assertNodesInDifferentGroups(t, graph, "vars.c", "vars.d")
}

func TestNewExecGroups_EmptyGraph(t *testing.T) {
	nodes := []*flowv1beta1.Node{}
	edges := [][2]string{}

	graph := newTestExecGroupsFromEdges(t, nodes, edges)

	require.NotNil(t, graph)
	assert.Equal(t, 0, countGroups(graph), "empty graph should have no groups")
}

func TestNewExecGroups_GroupOrdering(t *testing.T) {
	// Test that groups are properly ordered for execution
	// a -> b -> c
	// Verify that when we iterate through groups, we get them in correct order
	nodes := []*flowv1beta1.Node{
		createSimpleNode("vars.a"),
		createSimpleNode("vars.b"),
		createSimpleNode("vars.c"),
	}
	edges := [][2]string{
		{"vars.a", "vars.b"},
		{"vars.b", "vars.c"},
	}

	graph := newTestExecGroupsFromEdges(t, nodes, edges)

	require.NotNil(t, graph)

	// Collect groups in order
	var orderedNodeIDs []string
	graph.RangeGroups(func(_ int, ids []string) bool {
		orderedNodeIDs = append(orderedNodeIDs, ids...)
		return true
	})

	require.Len(t, orderedNodeIDs, 3)

	// Verify order: a must come before b, b must come before c
	aIdx := slices.Index(orderedNodeIDs, "vars.a")
	bIdx := slices.Index(orderedNodeIDs, "vars.b")
	cIdx := slices.Index(orderedNodeIDs, "vars.c")

	assert.Less(t, aIdx, bIdx, "a should come before b")
	assert.Less(t, bIdx, cIdx, "b should come before c")
}

func TestExecGroups_LoadGroup(t *testing.T) {
	nodes := []*flowv1beta1.Node{
		createSimpleNode("vars.a"),
		createSimpleNode("vars.b"),
	}
	edges := [][2]string{
		{"vars.a", "vars.b"},
	}

	graph := newTestExecGroupsFromEdges(t, nodes, edges)

	// Test loading existing group
	group0, ok := graph.LoadGroup(0)
	assert.True(t, ok)
	assert.Len(t, group0, 1)
	assert.Equal(t, "vars.a", group0[0])

	// Test loading non-existent group
	_, ok = graph.LoadGroup(999)
	assert.False(t, ok)
}

func TestExecGroups_RangeGroups(t *testing.T) {
	nodes := []*flowv1beta1.Node{
		createSimpleNode("vars.a"),
		createSimpleNode("vars.b"),
		createSimpleNode("vars.c"),
	}
	edges := [][2]string{
		{"vars.a", "vars.b"},
		{"vars.b", "vars.c"},
	}

	graph := newTestExecGroupsFromEdges(t, nodes, edges)

	visitedGroups := make(map[int]bool)
	totalNodes := 0

	graph.RangeGroups(func(groupID int, ids []string) bool {
		visitedGroups[groupID] = true
		totalNodes += len(ids)
		return true
	})

	assert.Len(t, visitedGroups, 3, "should visit 3 groups")
	assert.Equal(t, 3, totalNodes, "should visit 3 nodes total")
	assert.True(t, visitedGroups[0])
	assert.True(t, visitedGroups[1])
	assert.True(t, visitedGroups[2])
}

func TestExecGroups_RangeGroups_EarlyTermination(t *testing.T) {
	nodes := []*flowv1beta1.Node{
		createSimpleNode("vars.a"),
		createSimpleNode("vars.b"),
		createSimpleNode("vars.c"),
	}
	edges := [][2]string{
		{"vars.a", "vars.b"},
		{"vars.b", "vars.c"},
	}

	graph := newTestExecGroupsFromEdges(t, nodes, edges)

	visitCount := 0
	graph.RangeGroups(func(_ int, ids []string) bool {
		visitCount++
		return visitCount < 2 // Stop after visiting 2 groups
	})

	assert.Equal(t, 2, visitCount, "should stop after 2 groups")
}

func TestNewExecGroups_RealWorldFirmwareFlashScenario(t *testing.T) {
	// This reproduces the actual issue from the user's firmware flashing flow
	nodes := []*flowv1beta1.Node{
		createSimpleNode("inputs.firmwareFile"),
		createSimpleNode("connections.device"),
		createSimpleNode("inputs.commandWorkDir"),
		createSimpleNode("connections.storage"),
		createSimpleNode("actions.probeDevice"),
		createSimpleNode("connections.command"),
		createSimpleNode("inputs.firmwareBucket"),
		createSimpleNode("vars.isDeviceConnected"),
		createSimpleNode("actions.readFirmwareInfo"),
		createSimpleNode("streams.readFirmware"),
		createSimpleNode("vars.readFirmwareChecksum"),
		createSimpleNode("streams.writeFirmware"),
		createSimpleNode("actions.writeFirmwareInfo"),
		createSimpleNode("vars.writeFirmwareChecksum"),
		createSimpleNode("vars.verifyChecksum"),
		createSimpleNode("actions.flashFirmware"),
	}

	edges := [][2]string{
		{"inputs.firmwareFile", "actions.readFirmwareInfo"},
		{"inputs.firmwareBucket", "actions.readFirmwareInfo"},
		{"actions.writeFirmwareInfo", "vars.writeFirmwareChecksum"},
		{"inputs.commandWorkDir", "actions.writeFirmwareInfo"},
		{"vars.isDeviceConnected", "actions.writeFirmwareInfo"},
		{"streams.writeFirmware", "actions.writeFirmwareInfo"},
		{"actions.readFirmwareInfo", "vars.readFirmwareChecksum"},
		{"actions.probeDevice", "vars.isDeviceConnected"},
		{"streams.readFirmware", "streams.writeFirmware"},
		{"inputs.commandWorkDir", "streams.writeFirmware"},
		{"vars.isDeviceConnected", "streams.writeFirmware"},
		{"inputs.firmwareBucket", "streams.readFirmware"},
		{"inputs.firmwareFile", "streams.readFirmware"},
		{"vars.isDeviceConnected", "streams.readFirmware"},
		{"vars.readFirmwareChecksum", "vars.verifyChecksum"},
		{"vars.writeFirmwareChecksum", "vars.verifyChecksum"},
		{"vars.readFirmwareChecksum", "actions.flashFirmware"},
		{"vars.isDeviceConnected", "actions.flashFirmware"},
		{"vars.verifyChecksum", "actions.flashFirmware"},
	}

	groups := newTestExecGroupsFromEdges(t, nodes, edges)

	require.NotNil(t, groups)

	// The bug would put all nodes in 1 group, but we should have multiple groups
	groupCount := countGroups(groups)
	t.Logf("Group count: %d", groupCount)
	assert.Greater(t, groupCount, 1, "should have more than 1 group")

	// Verify some key dependencies are in different groups
	assertNodesInDifferentGroups(t, groups, "actions.probeDevice", "vars.isDeviceConnected")
	assertNodesInDifferentGroups(t, groups, "vars.isDeviceConnected", "streams.readFirmware")
	assertNodesInDifferentGroups(t, groups, "streams.readFirmware", "streams.writeFirmware")
	assertNodesInDifferentGroups(t, groups, "actions.readFirmwareInfo", "vars.readFirmwareChecksum")
	assertNodesInDifferentGroups(t, groups, "vars.writeFirmwareChecksum", "vars.verifyChecksum")
	assertNodesInDifferentGroups(t, groups, "vars.verifyChecksum", "actions.flashFirmware")

	// Log the groups for debugging
	groups.RangeGroups(func(idx int, ids []string) bool {
		t.Logf("Group %d: %v", idx, ids)
		return true
	})
}

func TestNewExecGroups_DebugForwardMaps(t *testing.T) {
	// Minimal test to check forward map population
	nodes := []*flowv1beta1.Node{
		createSimpleNode("inputs.a"),
		createSimpleNode("vars.b"),
	}
	edges := [][2]string{
		{"inputs.a", "vars.b"}, // b depends on a
	}

	syncMap := util.NewSyncMap[string, *Node]()
	for _, protoNode := range nodes {
		node := &Node{proto: protoNode}
		syncMap.Store(protoNode.GetId(), node)
	}

	// Create graph manually to test Build() behavior
	graph := &Graph{
		proto:   &flowv1beta1.Graph{},
		forward: make(map[string][]string),
		reverse: make(map[string][]string),
	}
	graph.Graph = graphlib.NewWithStore(
		GetNodeID,
		graph,
		graphlib.Directed(),
		graphlib.PreventCycles(),
	)

	for _, node := range nodes {
		err := graph.Graph.AddVertex(node)
		require.NoError(t, err)
	}

	for _, edge := range edges {
		err := graph.Graph.AddEdge(edge[0], edge[1])
		require.NoError(t, err)
	}

	// Check predecessors BEFORE Build
	predsBefore, err := graph.PredecessorMap()
	require.NoError(t, err)
	t.Logf("Predecessors BEFORE Build: %+v", predsBefore)

	err = graph.Build()
	require.NoError(t, err)

	// Check forward map AFTER Build
	t.Logf("Forward map for 'vars.b': %v", graph.Forward("vars.b"))
	t.Logf("Forward map for 'inputs.a': %v", graph.Forward("inputs.a"))

	// Now test grouping
	groups, err := NewExecutor(graph)
	require.NoError(t, err)

	groups.RangeGroups(func(idx int, ids []string) bool {
		t.Logf("Group %d: %v", idx, ids)
		return true
	})

	assert.Equal(t, 2, countGroups(groups), "should have 2 groups")
}

func TestNewExecGroups_LoadFromProtoWithAutoBuild(t *testing.T) {
	// Test that loading a graph proto automatically calls Build()
	// This ensures serialization/deserialization works correctly

	nodes := []*flowv1beta1.Node{
		createSimpleNode("inputs.a"),
		createSimpleNode("vars.b"),
	}

	proto := &flowv1beta1.Graph{
		Nodes: nodes,
		Edges: []*flowv1beta1.Edge{
			{Source: "inputs.a", Target: "vars.b"},
		},
	}

	// Load graph from proto - this now automatically calls Build()
	graph, err := GraphFromProto(proto)
	require.NoError(t, err)

	// Verify that forward maps are populated (Build was called)
	t.Logf("Forward map for 'vars.b': %v", graph.Forward("vars.b"))
	t.Logf("Forward map for 'inputs.a': %v", graph.Forward("inputs.a"))
	assert.Equal(t, []string{"inputs.a"}, graph.Forward("vars.b"), "vars.b should depend on inputs.a")
	assert.Empty(t, graph.Forward("inputs.a"), "inputs.a should have no dependencies")

	// Create groups and verify they work correctly
	groups, err := NewExecutor(graph)
	require.NoError(t, err)

	groupCount := countGroups(groups)
	t.Logf("Group count: %d", groupCount)

	groups.RangeGroups(func(idx int, ids []string) bool {
		t.Logf("Group %d: %v", idx, ids)
		return true
	})

	// Should have 2 groups since Build() was called automatically
	assert.Equal(t, 2, groupCount, "should have 2 groups with auto-Build")
	assertNodesInDifferentGroups(t, groups, "inputs.a", "vars.b")
}
