package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Generators (ticker, cron, range) used to lack stoppableMixin -- StopNode
// fell through to ctx-cancel, conflating stop with terminate at the
// handler level. Their ctx.Done branch unconditionally published
// PHASE_SUCCEEDED, which meant TerminateNode(generator) produced
// wire-order PHASE_CANCELLED -> PHASE_SUCCEEDED (terminate's CANCELLED
// got overridden by the generator's own SUCCEEDED publish).
//
// After the fix:
//   - Generators implement selfStoppable.
//   - Stop signals stopCh -> handler exits with PHASE_SUCCEEDED.
//   - Terminate cancels ctx -> handler returns ctx.Err quietly; the
//     TerminateNode-published PHASE_CANCELLED is the only terminal
//     phase on the wire.
//
// These tests pin those two paths down for each generator type.

// drainGeneratorPhases reads from a generator topic until either
// SUCCEEDED or CANCELLED is observed. Returns the ordered phase slice
// including the terminating phase. Fails the test on deadline.
func drainGeneratorPhases(t *testing.T, ctx context.Context, ch <-chan *pubsub.Message, deadline time.Duration) []flowv1beta2.RunSnapshot_Phase {
	t.Helper()
	var phases []flowv1beta2.RunSnapshot_Phase
	timer := time.After(deadline)
	for {
		select {
		case <-timer:
			t.Fatalf("timed out waiting for generator terminal phase; phases so far: %v", phaseNames(phases))
		case <-ctx.Done():
			t.Fatalf("ctx done while waiting for generator terminal phase; phases so far: %v", phaseNames(phases))
		case msg := <-ch:
			evt := msg.Payload.(*flowv1beta2.RunSnapshot_FlowEvent)
			msg.Ack()
			node := runtimeNodeFromEvent(evt)
			p := phaseOf(node)
			phases = append(phases, p)
			if p == flowv1beta2.RunSnapshot_PHASE_SUCCEEDED ||
				p == flowv1beta2.RunSnapshot_PHASE_CANCELLED {
				return phases
			}
		}
	}
}

// generatorStopFixture parameterizes the tests over generator types.
type generatorStopFixture struct {
	name       string
	yaml       string
	generator  string
	earlyDelay time.Duration // wait this long before firing the operator event
}

var generatorStopFixtures = []generatorStopFixture{
	{name: "ticker", yaml: "suspend_resume_ticker.yaml", generator: "generators.tick", earlyDelay: 30 * time.Millisecond},
	{name: "range", yaml: "suspend_resume_var.yaml", generator: "generators.seq", earlyDelay: 10 * time.Millisecond},
}

// TestGenerator_StopNode_ProducesSucceeded: StopNode on a generator must
// produce PHASE_SUCCEEDED as the terminal phase, with no PHASE_CANCELLED.
func TestGenerator_StopNode_ProducesSucceeded(t *testing.T) {
	for _, tc := range generatorStopFixtures {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			parallelByDefault(t)
			g := loadFlow(t, tc.yaml)
			ps := newPubSub()
			defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

			ctx := testContext(t)
			ch, err := ps.Subscribe(ctx, testTopics.For(tc.generator))
			require.NoError(t, err)

			exec := NewExecutor(ps, testTopics)
			done := make(chan error, 1)
			go func() { done <- exec.Execute(ctx, g) }()

			time.Sleep(tc.earlyDelay)
			exec.StopNode(tc.generator)

			phases := drainGeneratorPhases(t, ctx, ch, 1*time.Second)
			require.NotEmpty(t, phases)
			assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, phases[len(phases)-1],
				"StopNode on generator must produce PHASE_SUCCEEDED; phases=%v", phaseNames(phases))
			for _, p := range phases {
				assert.NotEqual(t, flowv1beta2.RunSnapshot_PHASE_CANCELLED, p,
					"PHASE_CANCELLED must not appear on Stop path; phases=%v", phaseNames(phases))
			}

			err = requireExecuteReturnsBy(t, done, 1*time.Second)
			require.NoError(t, err, "Execute should return naturally after stop")
		})
	}
}

// TestGenerator_TerminateNode_ProducesCancelledOnly: TerminateNode on a
// generator must produce PHASE_CANCELLED as the terminal phase, with NO
// PHASE_SUCCEEDED following it. Regression test for the
// CANCELLED-then-SUCCEEDED bug.
func TestGenerator_TerminateNode_ProducesCancelledOnly(t *testing.T) {
	for _, tc := range generatorStopFixtures {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			parallelByDefault(t)
			g := loadFlow(t, tc.yaml)
			ps := newPubSub()
			defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

			ctx := testContext(t)
			ch, err := ps.Subscribe(ctx, testTopics.For(tc.generator))
			require.NoError(t, err)

			exec := NewExecutor(ps, testTopics)
			done := make(chan error, 1)
			go func() { done <- exec.Execute(ctx, g) }()

			time.Sleep(tc.earlyDelay)
			exec.TerminateNode(tc.generator)

			phases := drainGeneratorPhases(t, ctx, ch, 1*time.Second)
			require.NotEmpty(t, phases)
			assert.Equal(t, flowv1beta2.RunSnapshot_PHASE_CANCELLED, phases[len(phases)-1],
				"TerminateNode on generator must produce PHASE_CANCELLED terminal; phases=%v", phaseNames(phases))

			// Specifically verify NO PHASE_SUCCEEDED appears after CANCELLED.
			cancelledIdx := -1
			for i, p := range phases {
				if p == flowv1beta2.RunSnapshot_PHASE_CANCELLED {
					cancelledIdx = i
					break
				}
			}
			require.GreaterOrEqual(t, cancelledIdx, 0, "expected PHASE_CANCELLED in stream")
			for _, p := range phases[cancelledIdx+1:] {
				assert.NotEqual(t, flowv1beta2.RunSnapshot_PHASE_SUCCEEDED, p,
					"PHASE_SUCCEEDED must not follow PHASE_CANCELLED (regression: ctx.Done overriding terminate); phases=%v", phaseNames(phases))
			}

			// Per-node terminate is scoped; flow may continue or exit.
			// Terminate the flow to ensure clean shutdown for the test.
			exec.Terminate()
			<-done
		})
	}
}
