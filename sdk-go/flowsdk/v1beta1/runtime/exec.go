package runtime

import (
	"fmt"

	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	graphlib "github.com/dominikbraun/graph"
	"golang.org/x/sync/errgroup"
)

type Executor struct {
	proto  *flowv1beta1.Groups
	values util.SyncMap[string, any]
}

func NewExecutor(graph *Graph) (*Executor, error) {
	// Use topological ordering to group independent nodes
	// Get all nodes in topological order
	order, err := graphlib.TopologicalSort(graph.Graph)
	if err != nil {
		return nil, fmt.Errorf("topological sort error: %w", err)
	}

	// Track which nodes have been processed
	processed := make(map[string]bool)
	groups := []*flowv1beta1.Groups_Group{}

	// Process nodes in topological order, grouping independent nodes
	// We use the forward/reverse maps that were saved before transitive reduction
	for len(processed) < len(order) {
		nodeIDs := []string{}

		// Find all nodes whose dependencies have been processed
		for _, nodeID := range order {
			if processed[nodeID] {
				continue
			}

			// Check if all dependencies (predecessors) are processed
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
				_, _, err := graph.Vertex(nodeID)
				if err != nil {
					return nil, err
				}

				// _, ok := run.nodes.Load(nodeID)
				// if !ok {
				// 	return nil, fmt.Errorf("node not found: %s", nodeID)
				// }

				nodeIDs = append(nodeIDs, nodeID)
			}
		}

		// Mark all nodes in this group as processed AFTER building the group
		// This ensures nodes in the same group don't see each other as already processed
		for _, nodeID := range nodeIDs {
			processed[nodeID] = true
		}

		if len(nodeIDs) > 0 {
			groups = append(groups, &flowv1beta1.Groups_Group{
				Ids: nodeIDs,
			})
		} else {
			// Should never happen with a valid DAG
			break
		}
	}

	return &Executor{
		proto: &flowv1beta1.Groups{
			Groups: groups,
		},
	}, nil
}

func (g *Executor) Eval(run *Runtime) (map[string]any, error) {
	// Execute groups SEQUENTIALLY to respect dependencies
	// but execute nodes WITHIN each group in parallel
	for _, group := range g.proto.Groups {
		ids := group.GetIds()

		// Execute all nodes in this group in parallel
		errGroup, _ := errgroup.WithContext(run.Context())
		for _, id := range ids {
			errGroup.Go(func() error {
				value, err := run.GetNodeValue(id)
				if err != nil {
					return err
				}
				g.values.Store(id, value)
				return nil
			})
		}

		// Wait for all nodes in this group to complete before moving to next group
		if err := errGroup.Wait(); err != nil {
			return nil, err
		}
	}

	return g.values.ToNativeMap(), nil
}

func (g *Executor) EvalAndReset(run *Runtime) (map[string]any, error) {
	// Reset all nodes after evaluation to ensure clean state
	defer run.Reset()
	return g.Eval(run)
}

func (g *Executor) LoadGroup(idx int) ([]string, bool) {
	if idx >= 0 && idx < len(g.proto.Groups) {
		return g.proto.Groups[idx].GetIds(), true
	}
	return nil, false
}

func (g *Executor) RangeGroups(callback func(int, []string) bool) {
	for idx, group := range g.proto.Groups {
		if !callback(idx, group.Ids) {
			return
		}
	}
}

func (g *Executor) Proto() *flowv1beta1.Groups {
	return g.proto
}
