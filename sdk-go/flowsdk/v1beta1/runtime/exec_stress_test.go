package runtime_test

import (
	"bytes"
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/runtime"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test 1: Deep dependency chain - ensures sequential execution works
const deepChainSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: DeepChain
  description: Tests a long chain of sequential dependencies

  inputs:
    - id: seed
      int64:
        default: 1

  vars:
    - id: step1
      value: = inputs.seed.getValue() + 1
    - id: step2
      value: = vars.step1.getValue() + 1
    - id: step3
      value: = vars.step2.getValue() + 1
    - id: step4
      value: = vars.step3.getValue() + 1
    - id: step5
      value: = vars.step4.getValue() + 1
    - id: step6
      value: = vars.step5.getValue() + 1
    - id: step7
      value: = vars.step6.getValue() + 1
    - id: step8
      value: = vars.step7.getValue() + 1
    - id: step9
      value: = vars.step8.getValue() + 1
    - id: step10
      value: = vars.step9.getValue() + 1

  outputs:
    - id: result
      value: = vars.step10.getValue()
    - id: success
      value: '= vars.step10.getValue() == 11 ? Done{reason: "Deep chain successful"} : null'
`

// Test 2: Wide parallelism - many independent nodes in one group
const wideParallelSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: WideParallel
  description: Tests parallel execution of many independent nodes

  inputs:
    - id: multiplier
      int64:
        default: 2

  vars:
    - id: calc1
      value: = inputs.multiplier.getValue() * 1
    - id: calc2
      value: = inputs.multiplier.getValue() * 2
    - id: calc3
      value: = inputs.multiplier.getValue() * 3
    - id: calc4
      value: = inputs.multiplier.getValue() * 4
    - id: calc5
      value: = inputs.multiplier.getValue() * 5
    - id: calc6
      value: = inputs.multiplier.getValue() * 6
    - id: calc7
      value: = inputs.multiplier.getValue() * 7
    - id: calc8
      value: = inputs.multiplier.getValue() * 8
    - id: calc9
      value: = inputs.multiplier.getValue() * 9
    - id: calc10
      value: = inputs.multiplier.getValue() * 10
    - id: sum
      value: '= vars.calc1.getValue() + vars.calc2.getValue() + vars.calc3.getValue() +
  			vars.calc4.getValue() + vars.calc5.getValue() + vars.calc6.getValue() +
				vars.calc7.getValue() + vars.calc8.getValue() + vars.calc9.getValue() +
				vars.calc10.getValue()'

  outputs:
    - id: result
      value: = vars.sum.getValue()
    - id: success
      value: '= vars.sum.getValue() == 110 ? Done{reason: "Sum is correct"} : null'
`

// Test 3: Diamond pattern with multiple convergence points
const complexDiamondSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: ComplexDiamond
  description: Tests multiple diamond patterns converging

  inputs:
    - id: base
      int64:
        default: 10

  vars:
    - id: left1
      value: = inputs.base.getValue() * 2
    - id: right1
      value: = inputs.base.getValue() * 3
    - id: left2
      value: = vars.left1.getValue() + 5
    - id: right2
      value: = vars.right1.getValue() + 5
    - id: merge1
      value: = vars.left2.getValue() + vars.right2.getValue()
    - id: left3
      value: = vars.merge1.getValue() * 2
    - id: right3
      value: = vars.merge1.getValue() * 3
    - id: final
      value: = vars.left3.getValue() + vars.right3.getValue()

  outputs:
    - id: result
      value: = vars.final.getValue()
    - id: success
      value: '= vars.final.getValue() == 300 ? Done{reason: "Complex diamond works"} : null'
`

// Test 4: Null handling and conditional logic
const nullHandlingSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: NullHandling
  description: Tests null value handling and conditionals

  inputs:
    - id: trigger
      bool:
        default: true

  vars:
    - id: maybeValue
      value: '= inputs.trigger.getValue() ? 42 : 0'
    - id: conditional
      value: '= vars.maybeValue.getValue() > 0 ? "positive" : "zero or negative"'

  outputs:
    - id: result
      value: = vars.conditional.getValue()
    - id: success
      value: '= vars.conditional.getValue() == "positive" ? Done{reason: "Null handling works"} : null'
`

// Test 5: Error propagation test
const errorPropagationSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: ErrorPropagation
  description: Tests error handling and propagation

  inputs:
    - id: shouldError
      bool:
        default: false

  vars:
    - id: check
      value: '= inputs.shouldError.getValue() ? Done{reason: "Intentional error", is_error: true} : Done{reason: "ok"}'
    - id: dependent
      value: = vars.check.getValue()

  outputs:
    - id: result
      value: = vars.dependent.getValue()
`

// Test 6: Multiple iterations with state reset
const multiIterationSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: MultiIteration
  description: Tests that state resets properly between iterations

  inputs:
    - id: counter
      int64:
        default: 0

  vars:
    - id: squared
      value: = inputs.counter.getValue() * inputs.counter.getValue()

  outputs:
    - id: result
      value: = vars.squared.getValue()
    - id: success
      value: '= inputs.counter.getValue() >= 5 ? Done{reason: "Completed 5 iterations"} : null'
`

// Test 7: Complex expression evaluation (simplified - CEL doesn't have fold by default)
const complexExpressionSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: ComplexExpression
  description: Tests complex CEL expressions

  inputs:
    - id: numbers
      list:
        items: int64
        default: [1, 2, 3, 4, 5]

  vars:
    - id: filtered
      value: '= inputs.numbers.getValue().filter(x, x > 2)'
    - id: mapped
      value: '= vars.filtered.getValue().map(x, x * 2)'
    - id: sum
      value: '= size(vars.mapped.getValue()) > 0 ? vars.mapped.getValue()[0] + vars.mapped.getValue()[1] + vars.mapped.getValue()[2] : 0'

  outputs:
    - id: result
      value: = vars.sum.getValue()
    - id: success
      value: '= vars.sum.getValue() == 24 ? Done{reason: "Complex expressions work"} : null'
`

func TestDeepChain(t *testing.T) {
	values := runFlowSpec(t, deepChainSpec, 1)

	// Should have result = 11 (seed 1 + 10 steps)
	assert.Equal(t, int64(11), values["outputs.result"])

	// Check that we got a success Done
	done := findDone(values)
	require.NotNil(t, done, "should have a Done message")
	assert.Contains(t, done.Reason, "Deep chain successful")
}

func TestWideParallel(t *testing.T) {
	values := runFlowSpec(t, wideParallelSpec, 1)

	// Sum should be 2 * (1+2+3+4+5+6+7+8+9+10) = 2 * 55 = 110
	assert.Equal(t, int64(110), values["outputs.result"])

	done := findDone(values)
	require.NotNil(t, done)
	assert.Contains(t, done.Reason, "Sum is correct")
}

func TestComplexDiamond(t *testing.T) {
	values := runFlowSpec(t, complexDiamondSpec, 1)

	// Expected: base=10, left1=20, right1=30, left2=25, right2=35
	// merge1=60, left3=120, right3=180, final=300
	assert.Equal(t, int64(300), values["outputs.result"])

	done := findDone(values)
	require.NotNil(t, done)
	assert.Contains(t, done.Reason, "Complex diamond works")
}

func TestNullHandling(t *testing.T) {
	values := runFlowSpec(t, nullHandlingSpec, 1)

	assert.Equal(t, "positive", values["outputs.result"])

	done := findDone(values)
	require.NotNil(t, done)
	assert.Contains(t, done.Reason, "Null handling works")
}

func TestErrorPropagation(t *testing.T) {
	// Run with shouldError = false first
	values := runFlowSpec(t, errorPropagationSpec, 1)

	// Result is now a Done message
	done := findDone(values)
	require.NotNil(t, done)
	assert.Contains(t, done.Reason, "ok")
	assert.False(t, done.IsError)

	// TODO: Test with shouldError = true
	// This would require setting input values, which we'll need to add support for
}

func TestMultipleIterations(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(multiIterationSpec)

	spec, err := flowsdk.ReadSpec(encoding.YAML, &buf)
	require.NoError(t, err)

	ctx := t.Context()

	run := runtime.New(ctx, spec.GetFlow())
	graph, err := runtime.NewGraph(run)
	require.NoError(t, err)

	exec, err := runtime.NewExecutor(graph)
	require.NoError(t, err)

	// Run multiple iterations and verify state resets
	for i := range int64(6) {
		err = run.SetInputValues(map[string]any{"counter": i})
		require.NoError(t, err)

		values, err := exec.EvalAndReset(run)
		require.NoError(t, err)

		// Verify result is i^2
		expected := i * i
		actual := values["outputs.result"]
		assert.Equal(t, expected, actual, "iteration %d: expected %d^2=%d", i, i, expected)

		if i >= 5 {
			done := findDone(values)
			require.NotNil(t, done, "iteration %d should have Done", i)
			assert.Contains(t, done.Reason, "Completed 5 iterations")
			break
		}
	}
}

func TestComplexExpressions(t *testing.T) {
	values := runFlowSpec(t, complexExpressionSpec, 1)

	// [1,2,3,4,5].filter(x > 2) = [3,4,5]
	// [3,4,5].map(x * 2) = [6,8,10]
	// [6,8,10].fold(0, +) = 24
	assert.Equal(t, int64(24), values["outputs.result"])

	done := findDone(values)
	require.NotNil(t, done)
	assert.Contains(t, done.Reason, "Complex expressions work")
}

// Test for potential race condition: same node accessed by multiple paths
func TestRaceConditionDetection(t *testing.T) {
	const raceSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: PotentialRace
  description: Shared dependency accessed by multiple independent nodes

  inputs:
    - id: shared
      int64:
        default: 100

  vars:
    - id: user1
      value: = inputs.shared.getValue() * 2
    - id: user2
      value: = inputs.shared.getValue() * 3
    - id: user3
      value: = inputs.shared.getValue() * 4
    - id: user4
      value: = inputs.shared.getValue() * 5
    - id: sum
      value: '= vars.user1.getValue() + vars.user2.getValue() +
               vars.user3.getValue() + vars.user4.getValue()'

  outputs:
    - id: result
      value: = vars.sum.getValue()
`

	// Run this many times to try to expose race conditions
	for i := 0; i < 100; i++ {
		values := runFlowSpec(t, raceSpec, 1)

		// Expected: 100 * (2+3+4+5) = 1400
		assert.Equal(t, int64(1400), values["outputs.result"], "iteration %d failed", i)
	}
}

// Test 8: Circular dependency detection
func TestCircularDependency(t *testing.T) {
	const circularSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: CircularDependency
  description: Should fail - contains circular dependency

  inputs:
    - id: start
      int64:
        default: 1

  vars:
    - id: nodeA
      value: = vars.nodeB.getValue() + 1
    - id: nodeB
      value: = vars.nodeC.getValue() + 1
    - id: nodeC
      value: = vars.nodeA.getValue() + 1

  outputs:
    - id: result
      value: = vars.nodeC.getValue()
`

	var buf bytes.Buffer
	buf.WriteString(circularSpec)

	spec, err := flowsdk.ReadSpec(encoding.YAML, &buf)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	run := runtime.New(ctx, spec.GetFlow())

	// Should fail to build graph due to cycle
	_, err = runtime.NewGraph(run)
	require.Error(t, err, "expected error for circular dependency")
	assert.Contains(t, err.Error(), "cycle", "error should mention cycle")
}

// Test 9: Error in parallel group - verify one failure doesn't block other nodes
func TestErrorInParallelGroup(t *testing.T) {
	const errorParallelSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: ErrorInParallel
  description: Tests error handling when one node in parallel group fails

  inputs:
    - id: base
      int64:
        default: 10

  vars:
    - id: good1
      value: = inputs.base.getValue() * 2
    - id: bad
      value: '= Done{reason: "Intentional error", is_error: true}'
    - id: good2
      value: = inputs.base.getValue() * 3
    - id: good3
      value: = inputs.base.getValue() * 4
    - id: dependent
      value: = vars.good1.getValue() + vars.good2.getValue() + vars.good3.getValue()

  outputs:
    - id: result
      value: = vars.dependent.getValue()
    - id: error
      value: = vars.bad.getValue()
`

	values := runFlowSpec(t, errorParallelSpec, 1)

	// Good nodes should still execute and produce results
	assert.Equal(t, int64(90), values["outputs.result"], "good nodes should execute: 10*(2+3+4)=90")

	// Error should be propagated
	done := values["outputs.error"].(*flowv1beta1.Runtime_Done)
	require.NotNil(t, done)
	assert.True(t, done.IsError)
	assert.Contains(t, done.Reason, "Intentional error")
}

// Test 10: Cache behavior validation
func TestCacheValidation(t *testing.T) {
	const cacheSpec = `
kind: Flow
apiVersion: v1beta1
spec:
  name: CacheBehavior
  description: Tests that cache=true actually caches values

  inputs:
    - id: counter
      int64:
        default: 0

  vars:
    - id: expensive
      cache: true
      value: = inputs.counter.getValue() + 100
    - id: user1
      value: = vars.expensive.getValue()
    - id: user2
      value: = vars.expensive.getValue()
    - id: user3
      value: = vars.expensive.getValue()
    - id: total
      value: = vars.user1.getValue() + vars.user2.getValue() + vars.user3.getValue()

  outputs:
    - id: result
      value: = vars.total.getValue()
    - id: cached
      value: = vars.expensive.getValue()
`

	var buf bytes.Buffer
	buf.WriteString(cacheSpec)

	spec, err := flowsdk.ReadSpec(encoding.YAML, &buf)
	require.NoError(t, err)

	ctx := t.Context()

	run := runtime.New(ctx, spec.GetFlow())
	graph, err := runtime.NewGraph(run)
	require.NoError(t, err)

	exec, err := runtime.NewExecutor(graph)
	require.NoError(t, err)

	// First iteration
	values1, err := exec.EvalAndReset(run)
	require.NoError(t, err)
	assert.Equal(t, int64(300), values1["outputs.result"], "first: 100*3=300")
	assert.Equal(t, int64(100), values1["outputs.cached"])

	err = run.SetInputValues(map[string]any{"counter": 100})
	require.NoError(t, err)

	// Second iteration
	values2, err := exec.EvalAndReset(run)
	require.NoError(t, err)
	assert.Equal(t, int64(300), values2["outputs.result"], "first: 100*3=300")
	assert.Equal(t, int64(100), values1["outputs.cached"])
}

// Test 11: Large graph scalability (100+ nodes)
func TestLargeGraphScalability(t *testing.T) {
	// Build a flow spec with 120 nodes: 1 input + 100 vars + 19 outputs
	// Structure: fan-out from input to 10 branches, each branch has 10 sequential nodes

	var buf bytes.Buffer
	buf.WriteString(`
kind: Flow
apiVersion: v1beta1
spec:
  name: LargeGraph
  description: Scalability test with 100+ nodes

  inputs:
    - id: base
      int64:
        default: 1

  vars:
`)

	// Create 10 branches of 10 nodes each
	for branch := 0; branch < 10; branch++ {
		for step := 0; step < 10; step++ {
			nodeID := branch*10 + step
			if step == 0 {
				// First node in branch depends on input
				buf.WriteString("    - id: n" + string(rune('0'+nodeID/10)) + string(rune('0'+nodeID%10)) + "\n")
				buf.WriteString("      value: = inputs.base.getValue() + " + string(rune('0'+branch)) + "\n")
			} else {
				// Subsequent nodes depend on previous in chain
				prevID := nodeID - 1
				buf.WriteString("    - id: n" + string(rune('0'+nodeID/10)) + string(rune('0'+nodeID%10)) + "\n")
				buf.WriteString("      value: = vars.n" + string(rune('0'+prevID/10)) + string(rune('0'+prevID%10)) + ".getValue() + 1\n")
			}
		}
	}

	// Add outputs that sum various branches
	buf.WriteString("\n  outputs:\n")
	for i := 0; i < 10; i++ {
		lastNodeInBranch := i*10 + 9
		buf.WriteString("    - id: branch" + string(rune('0'+i)) + "\n")
		buf.WriteString("      value: = vars.n" + string(rune('0'+lastNodeInBranch/10)) + string(rune('0'+lastNodeInBranch%10)) + ".getValue()\n")
	}

	spec, err := flowsdk.ReadSpec(encoding.YAML, &buf)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	run := runtime.New(ctx, spec.GetFlow())

	// Build graph - should handle 120 nodes
	graph, err := runtime.NewGraph(run)
	require.NoError(t, err)

	// Create executor and verify grouping
	executor, err := runtime.NewExecutor(graph)
	require.NoError(t, err)

	// Execute and verify results
	values, err := executor.EvalAndReset(run)
	require.NoError(t, err)

	// Each branch should be: base(1) + branch_num + 9 steps = 1 + branch + 9
	for i := 0; i < 10; i++ {
		expected := int64(1 + i + 9) // base + branch offset + 9 steps
		actual := values["outputs.branch"+string(rune('0'+i))]
		assert.Equal(t, expected, actual, "branch %d should be %d", i, expected)
	}

	// Verify graph structure - should have multiple groups
	nodeCount, err := graph.Order()
	require.NoError(t, err)
	require.Greater(t, len(executor.Proto().Groups), 1, "should have multiple execution groups")

	t.Logf("Large graph: %d nodes organized into %d execution groups",
		nodeCount, len(executor.Proto().Groups))
}

// Helper functions

func runFlowSpec(t *testing.T, spec string, iterations int) map[string]any {
	t.Helper()

	var buf bytes.Buffer
	buf.WriteString(spec)

	flowSpec, err := flowsdk.ReadSpec(encoding.YAML, &buf)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	run := runtime.New(ctx, flowSpec.GetFlow())
	graph, err := runtime.NewGraph(run)
	require.NoError(t, err)

	exec, err := runtime.NewExecutor(graph)
	require.NoError(t, err)

	var values map[string]any
	for i := range iterations {
		values, err = exec.EvalAndReset(run)
		if err != nil {
			t.Fatalf("iteration %d failed: %v", i, err)
		}
	}

	return values
}

func findDone(values map[string]any) *flowv1beta1.Runtime_Done {
	done := util.SliceReduce(slices.Collect(maps.Values(values)), func(v any) (any, bool) {
		done, ok := v.(*flowv1beta1.Runtime_Done)
		return done, ok
	})
	if len(done) > 0 {
		return done[0].(*flowv1beta1.Runtime_Done)
	}
	return nil
}

// Benchmark parallel execution efficiency
func BenchmarkWideParallel(b *testing.B) {
	var buf bytes.Buffer
	buf.WriteString(wideParallelSpec)

	spec, err := flowsdk.ReadSpec(encoding.YAML, &buf)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	run := runtime.New(ctx, spec.GetFlow())
	graph, err := runtime.NewGraph(run)
	if err != nil {
		b.Fatal(err)
	}

	exec, err := runtime.NewExecutor(graph)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := exec.EvalAndReset(run)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark deep chain to measure sequential overhead
func BenchmarkDeepChain(b *testing.B) {
	var buf bytes.Buffer
	buf.WriteString(deepChainSpec)

	spec, err := flowsdk.ReadSpec(encoding.YAML, &buf)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	run := runtime.New(ctx, spec.GetFlow())
	graph, err := runtime.NewGraph(run)
	if err != nil {
		b.Fatal(err)
	}

	exec, err := runtime.NewExecutor(graph)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := exec.EvalAndReset(run)
		if err != nil {
			b.Fatal(err)
		}
	}
}
