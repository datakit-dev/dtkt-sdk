package runtime

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	flowsdkv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/graph"
	outboxmem "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox/memory"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	memorypubsub "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub/memory"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// loadFlow reads a YAML flow spec from testdata/, decodes it via
// flowsdk/v1beta2.ReadSpec, and builds a Graph via graph.Build().
func loadFlow(t *testing.T, name string) *flowv1beta2.Graph {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	spec, err := flowsdkv1beta2.ReadSpec(encoding.YAML, f)
	if err != nil {
		t.Fatal(err)
	}
	g, err := graph.Build(spec.GetFlow())
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func newPubSub() executor.PubSub {
	return memorypubsub.New(memorypubsub.WithPersistent())
}

// testTopics provides topic names scoped to a fixed "test" prefix, matching
// the Topics struct used in production code.
var testTopics = executor.NewTopics("test")

// feedInput publishes values to the input's PubSub topic, then publishes an EOF marker.
func feedInput(ps executor.PubSub, nodeID string, values ...any) {
	topic := testTopics.InputFor(nodeID)
	for _, v := range values {
		val, _ := nativeToExpr(v)
		ps.Publish(topic, pubsub.NewMessage(val)) //nolint:errcheck
	}
	ps.Publish(topic, pubsub.NewMessage(newEOFValue())) //nolint:errcheck
}

// sendInput publishes a single value to an input's PubSub topic without
// sending an EOF. The input channel remains open so the input handler
// continues resolving (e.g. falling back to cached/default on subsequent
// throttle cycles).
func sendInput(ps executor.PubSub, nodeID string, value any) {
	topic := testTopics.InputFor(nodeID)
	val, _ := nativeToExpr(value)
	ps.Publish(topic, pubsub.NewMessage(val)) //nolint:errcheck
}

// testContext returns a context with a 10-second timeout that fails the test on expiry.
func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// collectOutputs subscribes to an output topic and reads all messages until the
// EOF marker (Closed: true). Returns the non-EOF output nodes.
func collectOutputs(ctx context.Context, ps executor.PubSub, nodeID string) []*flowv1beta2.RunSnapshot_OutputNode {
	ch, _ := ps.Subscribe(ctx, testTopics.For(nodeID))
	var results []*flowv1beta2.RunSnapshot_OutputNode
	for {
		select {
		case <-ctx.Done():
			return results
		case msg := <-ch:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			outNode, ok := runtimeNodeFromEvent(evt).(*flowv1beta2.RunSnapshot_OutputNode)
			if !ok || outNode.GetClosed() {
				return results
			}
			results = append(results, outNode)
		}
	}
}

// outputInt64s extracts int64 values from output nodes.
func outputInt64s(events []*flowv1beta2.RunSnapshot_OutputNode) []int64 {
	vals := make([]int64, len(events))
	for i, e := range events {
		vals[i] = e.GetValue().GetInt64Value()
	}
	return vals
}

// outputStrings extracts string values from output nodes.
func outputStrings(events []*flowv1beta2.RunSnapshot_OutputNode) []string {
	vals := make([]string, len(events))
	for i, e := range events {
		vals[i] = e.GetValue().GetStringValue()
	}
	return vals
}

// outputsByID groups output nodes by their ID.
func outputsByID(events []*flowv1beta2.RunSnapshot_OutputNode) map[string][]int64 {
	m := make(map[string][]int64)
	for _, e := range events {
		m[e.GetId()] = append(m[e.GetId()], e.GetValue().GetInt64Value())
	}
	return m
}

// collectMultipleOutputs collects outputs from multiple output topics.
func collectMultipleOutputs(ctx context.Context, ps executor.PubSub, nodeIDs ...string) []*flowv1beta2.RunSnapshot_OutputNode {
	var all []*flowv1beta2.RunSnapshot_OutputNode
	for _, id := range nodeIDs {
		all = append(all, collectOutputs(ctx, ps, id)...)
	}
	return all
}

// withAndWithoutOutbox runs fn as two subtests: "direct" (no outbox) and
// "outbox" (with an in-memory outbox). extraOpts should be appended to
// whatever options the test already passes to NewExecutor.
// Each subtest also verifies that no goroutines leaked after fn returns.
func withAndWithoutOutbox(t *testing.T, fn func(t *testing.T, extraOpts []Option)) {
	t.Helper()
	t.Run("direct", func(t *testing.T) {
		before := runtime.NumGoroutine()
		// Register leak check BEFORE fn so it runs AFTER fn's cleanups (LIFO).
		t.Cleanup(func() { assertNoGoroutineLeak(t, before) })
		fn(t, nil)
	})
	t.Run("outbox", func(t *testing.T) {
		before := runtime.NumGoroutine()
		t.Cleanup(func() { assertNoGoroutineLeak(t, before) })
		fn(t, []Option{WithOutbox(outboxmem.New())})
	})
}

// assertNoGoroutineLeak verifies that the goroutine count settles back to the
// level observed before the test ran, catching leaked goroutines from handlers
// or pubsub subscribers that failed to exit.
func assertNoGoroutineLeak(t *testing.T, before int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("goroutine leak: started with %d goroutines, now have %d", before, runtime.NumGoroutine())
}

// nativeToExpr converts a Go native value to a cel.expr.Value using the
// standard CEL conversion pipeline: NativeToValue → cel.ValueAsProto.
func nativeToExpr(v any) (*expr.Value, error) {
	if v == nil {
		return &expr.Value{Kind: &expr.Value_NullValue{}}, nil
	}
	if ev, ok := v.(*expr.Value); ok {
		return ev, nil
	}
	refVal := types.DefaultTypeAdapter.NativeToValue(v)
	if types.IsError(refVal) {
		return nil, fmt.Errorf("NativeToValue: %v", refVal)
	}
	return cel.ValueAsProto(refVal)
}
