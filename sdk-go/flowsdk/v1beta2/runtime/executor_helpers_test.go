package runtime

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"google.golang.org/protobuf/proto"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/encoding"
	flowsdkv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/graph"
	outboxmem "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox/memory"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
	memorypubsub "github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub/memory"
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
	defer f.Close() //nolint:errcheck // fixture opened read-only and fully read below; close error cannot affect the parsed result
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
		ps.Publish(topic, pubsub.NewMessage(val)) //nolint:errcheck // test fixture feed to in-memory pubsub; a real failure surfaces as a downstream assertion
	}
	ps.Publish(topic, pubsub.NewMessage(newEOFValue())) //nolint:errcheck // test fixture feed to in-memory pubsub; a real failure surfaces as a downstream assertion
}

// sendInput publishes a single value to an input's PubSub topic without
// sending an EOF. The input channel remains open so the input handler
// continues resolving (e.g. falling back to cached/default on subsequent
// throttle cycles).
func sendInput(ps executor.PubSub, nodeID string, value any) {
	topic := testTopics.InputFor(nodeID)
	val, _ := nativeToExpr(value)
	ps.Publish(topic, pubsub.NewMessage(val)) //nolint:errcheck // test fixture feed to in-memory pubsub; a real failure surfaces as a downstream assertion
}

// testContext returns a context with a 10-second timeout that fails the test on expiry.
func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// collectOutputs subscribes to an output topic and reads NODE_OUTPUT events
// until the stream ends. NODE_UPDATE state events (phase transitions like
// PHASE_STOPPING / PHASE_SUSPENDED) are skipped, matching how downstream
// handlers' `recv` ignores them. End-of-stream is signalled by either an
// EOF sentinel value or the OutputNode's Closed:true flag.
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
			if evt.GetEventType() != flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
				continue
			}
			outNode, ok := runtimeNodeFromEvent(evt).(*flowv1beta2.RunSnapshot_OutputNode)
			if !ok || outNode.GetClosed() || isEOFValue(outNode.GetValue()) {
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

// parallelByDefault enables t.Parallel() for the calling test in the default
// build, and installs a goroutine-leak check (with serial execution) when
// the test binary is built with `-tags=leakcheck`.
//
// Use this in any top-level test that doesn't already go through
// `withAndWithoutOutbox`. The two helpers share the same gating logic;
// `withAndWithoutOutbox` simply applies it inside its two subtests.
//
// Goroutine-leak detection requires serial execution because
// `runtime.NumGoroutine()` is process-wide -- under parallel mode the
// counts are polluted by other in-flight tests.
//
// Run leak detection via `task test:leakcheck` (or `go test -tags=leakcheck
// ./flowsdk/v1beta2/runtime/`).
func parallelByDefault(t *testing.T) {
	t.Helper()
	if leakCheckEnabled {
		before := runtime.NumGoroutine()
		// Register the leak check before fn body executes so it runs
		// AFTER any t.Cleanup the test installs (LIFO order).
		t.Cleanup(func() { assertNoGoroutineLeak(t, before) })
		return
	}
	t.Parallel()
}

// withAndWithoutOutbox runs fn as two subtests: "direct" (no outbox) and
// "outbox" (with an in-memory outbox). extraOpts should be appended to
// whatever options the test already passes to NewExecutor.
//
// Each subtest applies `parallelByDefault` (parallel by default,
// serial+leak-check under `-tags=leakcheck`).
func withAndWithoutOutbox(t *testing.T, fn func(t *testing.T, extraOpts []Option)) {
	t.Helper()
	runSubtest := func(name string, opts []Option) {
		t.Run(name, func(t *testing.T) {
			parallelByDefault(t)
			fn(t, opts)
		})
	}
	runSubtest("direct", nil)
	runSubtest("outbox", []Option{WithOutbox(outboxmem.New())})
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

// requireExecuteReturnsBy fails the test (with a goroutine dump) if Execute
// hasn't returned via the done channel within the given budget. Use this
// after asserting the observable end-state of a flow (e.g. waitForPhase
// reaching PHASE_SUCCEEDED) so a regression that makes Execute hang can't
// silently fall through to testContext's 10s deadline -- handlers swallow
// ctx.Canceled and surface nil, which would mask the bug.
func requireExecuteReturnsBy(t *testing.T, done <-chan error, budget time.Duration) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(budget):
		buf := make([]byte, 1<<16)
		n := runtime.Stack(buf, true)
		t.Logf("Execute hung past %v. Goroutine dump:\n%s", budget, buf[:n])
		t.Fatalf("Execute did not return within %v -- possible regression in cleanup/exit path", budget)
		return nil
	}
}

// assertNoOutputDuring fails the test if any NODE_OUTPUT event arrives on ch
// within the given window. NODE_UPDATE state events (phase transitions) are
// ignored. Use after a suspend assertion to verify the handler is actually
// behaviorally paused -- not just publishing a PHASE_SUSPENDED state event
// while continuing to emit values in the background.
//
// Sizing the window:
//   - For input-driven handlers (var/action/stream/output/interaction with
//     mocked RPCs): 100ms is comfortable -- handler iterations are sub-ms.
//   - For generator-driven flows: pick a window > 2x the generator
//     interval. Several test ticker fixtures were bumped from 10ms to 50ms
//     interval (see docs/flaky-tests.md); use ~150ms for those, more for
//     longer cadences.
func assertNoOutputDuring(t *testing.T, ch <-chan *pubsub.Message, window time.Duration) {
	t.Helper()
	deadline := time.After(window)
	for {
		select {
		case <-deadline:
			return
		case msg := <-ch:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			if evt.GetEventType() == flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
				node := runtimeNodeFromEvent(evt)
				// Skip terminal markers (Closed/EOF value) since those are
				// expected when the flow tears down.
				if isEOFValue(node.GetValue()) {
					continue
				}
				t.Fatalf("expected no NODE_OUTPUT during suspend window, got value: %v", node.GetValue())
			}
		}
	}
}

// expectOutputWithin reads from ch waiting for a NODE_OUTPUT event (skipping
// NODE_UPDATE state events and EOF terminals). Fails the test if no
// NODE_OUTPUT arrives within the budget. Use after ResumeNode in tests
// where multiple inputs are fed (so iter N+1 produces fresh output after
// resume) -- a "lying" resume that leaves the handler parked would
// silently fall through to ctx expiry without this cap.
//
// NOT for single-input resume tests: in v1beta2 the order is publish then
// checkLifecycle, so iter 1's value emits BEFORE NC.suspend fires; after
// resume the handler reads EOF and exits with PHASE_SUCCEEDED, never
// emitting a fresh NODE_OUTPUT. Use `requirePhaseWithin(SUCCEEDED, ...)`
// for those.
func expectOutputWithin(t *testing.T, ch <-chan *pubsub.Message, budget time.Duration) {
	t.Helper()
	deadline := time.After(budget)
	for {
		select {
		case <-deadline:
			t.Fatalf("expected a NODE_OUTPUT within %v, none arrived (resume may not have unparked the handler)", budget)
		case msg := <-ch:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			if evt.GetEventType() != flowv1beta2.RunSnapshot_FlowEvent_EVENT_TYPE_NODE_OUTPUT {
				continue
			}
			node := runtimeNodeFromEvent(evt)
			if isEOFValue(node.GetValue()) {
				continue
			}
			return
		}
	}
}

// requirePhaseWithin reads FlowEvents from ch until the target phase is
// observed, failing the test (with a goroutine dump) if budget elapses
// first. Companion to `requireExecuteReturnsBy`: gives a tight,
// behavior-driven cap on the post-resume wait so a regression that leaves
// a handler parked surfaces as a quick failure rather than silently
// falling through to testContext's 10s deadline.
func requirePhaseWithin(t *testing.T, ch <-chan *pubsub.Message, target flowv1beta2.RunSnapshot_Phase, budget time.Duration) {
	t.Helper()
	deadline := time.After(budget)
	for {
		select {
		case <-deadline:
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, true)
			t.Logf("phase %v not observed within %v. Goroutine dump:\n%s",
				phaseNames([]flowv1beta2.RunSnapshot_Phase{target})[0], budget, buf[:n])
			t.Fatalf("expected phase %v within %v -- possible regression in resume/cleanup path",
				phaseNames([]flowv1beta2.RunSnapshot_Phase{target})[0], budget)
		case msg := <-ch:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			node := runtimeNodeFromEvent(evt)
			if phaseOf(node) == target {
				return
			}
		}
	}
}

// drainPhasesUntil reads events from ch, recording the phase of each, until
// the target phase is observed (or ctx cancels). Returns the ordered phase
// slice including the terminating phase.
func drainPhasesUntil(ctx context.Context, ch <-chan *pubsub.Message, target flowv1beta2.RunSnapshot_Phase) []flowv1beta2.RunSnapshot_Phase {
	var phases []flowv1beta2.RunSnapshot_Phase
	for {
		select {
		case <-ctx.Done():
			return phases
		case msg := <-ch:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			node := runtimeNodeFromEvent(evt)
			p := phaseOf(node)
			phases = append(phases, p)
			if p == target {
				return phases
			}
		}
	}
}

// assertPhaseOrder asserts `before` is observed before `after` in the phase
// stream. Both must appear; `before` must precede the first `after`.
func assertPhaseOrder(t *testing.T, phases []flowv1beta2.RunSnapshot_Phase, before, after flowv1beta2.RunSnapshot_Phase) {
	t.Helper()
	var sawBefore bool
	for _, p := range phases {
		if p == before {
			sawBefore = true
			continue
		}
		if p == after {
			if !sawBefore {
				t.Errorf("expected phase %s before %s; %s appeared first in %v",
					phaseNames([]flowv1beta2.RunSnapshot_Phase{before})[0],
					phaseNames([]flowv1beta2.RunSnapshot_Phase{after})[0],
					phaseNames([]flowv1beta2.RunSnapshot_Phase{after})[0],
					phaseNames(phases))
			}
			return
		}
	}
	if !sawBefore {
		t.Errorf("expected phase %s in stream, got %v",
			phaseNames([]flowv1beta2.RunSnapshot_Phase{before})[0], phaseNames(phases))
	} else {
		t.Errorf("expected phase %s after %s in stream, got %v",
			phaseNames([]flowv1beta2.RunSnapshot_Phase{after})[0],
			phaseNames([]flowv1beta2.RunSnapshot_Phase{before})[0],
			phaseNames(phases))
	}
}

// nativeToExpr converts a Go native value to a cel.expr.Value using the
// standard CEL conversion pipeline: NativeToValue → cel.ValueAsProto.
//
// Non-WKT proto.Message inputs are wrapped directly via Any. cel-go's
// DefaultTypeAdapter only knows protos pre-registered in its global
// pb.DefaultDb, and falls back to reflection-based mapping for anything
// else - producing a generic MapValue keyed by Go's PascalCase field
// names. Wrapping in Any preserves the type URL so the receive-side
// adapter (which has the proper resolver) can deserialize it as a typed
// proto and CEL can access fields by their proto/JSON names.
func nativeToExpr(v any) (*expr.Value, error) {
	if v == nil {
		return &expr.Value{Kind: &expr.Value_NullValue{}}, nil
	}
	if ev, ok := v.(*expr.Value); ok {
		return ev, nil
	}
	if msg, ok := v.(proto.Message); ok {
		fullName := string(msg.ProtoReflect().Descriptor().FullName())
		if !strings.HasPrefix(fullName, "google.protobuf.") {
			return protoToExpr(msg)
		}
	}
	refVal := types.DefaultTypeAdapter.NativeToValue(v)
	if types.IsError(refVal) {
		return nil, fmt.Errorf("NativeToValue: %v", refVal)
	}
	return cel.ValueAsProto(refVal)
}
