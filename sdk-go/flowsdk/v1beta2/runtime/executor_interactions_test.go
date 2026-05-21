package runtime

import (
	"regexp"
	"testing"

	expr "cel.dev/expr"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// bareIDPattern matches the buf-validate regex on every per-event /
// per-node id field whose category is implicit (InteractionRequestEvent.id,
// InteractionResponseEvent.id, etc.). Asserting against this in interaction
// test responders pins the format the runtime must emit.
var bareIDPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// Interaction: basic prompt → response flow

func TestGraph_Interaction_Basic(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_basic.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Auto-respond to interaction prompts.
		// Each prompt's id must be bare (Format A) per the protobuf
		// validator on InteractionRequestEvent.id; if the runtime
		// regresses to emitting fully-qualified ids the wire round-trip
		// breaks even though in-process tests don't normally validate.
		go func() {
			for p := range prompt {
				assert.Regexp(t, bareIDPattern, p.GetId(),
					"InteractionRequestEvent.id must be bare per its protobuf validator pattern")
				anyVal, _ := common.WrapProtoAny(int64(100))
				response <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(response)
		}()

		feedInput(ps, "inputs.x", int64(1))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		close(prompt) // Unblock auto-respond goroutine.
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(100), results[0].GetValue().GetInt64Value())
	})
}

// Interaction: multiple prompts (multi-input graph)

func TestGraph_Interaction_MultiplePrompts(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_multiple.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Auto-respond with incrementing values.
		var count int64
		go func() {
			for p := range prompt {
				count++
				anyVal, _ := common.WrapProtoAny(count * 10)
				response <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(response)
		}()

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		close(prompt) // Unblock auto-respond goroutine.
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 3)
		assert.Equal(t, int64(10), results[0].GetValue().GetInt64Value())
		assert.Equal(t, int64(20), results[1].GetValue().GetInt64Value())
		assert.Equal(t, int64(30), results[2].GetValue().GetInt64Value())
	})
}

// Interaction: missing WithInteractions option returns error

func TestGraph_Interaction_MissingOption(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_missing_option.yaml")

		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, extraOpts...).Execute(ctx, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires WithInteractions option")
	})
}

// Interaction: response channel close → EOF

func TestGraph_Interaction_ResponseClose(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_response_close.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// Respond once, then close the response channel.
		go func() {
			p := <-prompt
			anyVal, _ := common.WrapProtoAny(int64(42))
			response <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			close(response)
		}()

		// Feed 2 input values; only the first interaction gets a real response,
		// the second sees the response channel close → EOF propagates.
		feedInput(ps, "inputs.x", int64(1), int64(2))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		// First interaction responded with 42; second sees channel close → EOF.
		require.Len(t, results, 1)
		assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
	})
}

// Interaction: response value passes through a transform pipeline.

func TestGraph_Interaction_Transforms(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_transforms.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		go func() {
			for p := range prompt {
				anyVal, _ := common.WrapProtoAny(int64(5))
				response <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(response)
		}()

		feedInput(ps, "inputs.x", int64(1))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		close(prompt)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		// Transform: response 5 mapped *10 -> 50.
		assert.Equal(t, int64(50), results[0].GetValue().GetInt64Value())
	})
}

// Interaction: form with select / multi_select / file elements. Verifies the
// runtime accepts the full form-element oneof set without a crash and routes
// the response through.

// TestGraph_Interaction_FormElements_All verifies the runtime accepts an
// interaction with multiple form-element variants (select, multi_select,
// file). The form definition's variants are validated at graph.Build()
// time -- a malformed variant would fail loadFlow. Runtime behavior:
// prompt is emitted on inputs.x>0, response with value=11 flows back to
// the output. We assert (a) at least one prompt was raised with a non-empty
// id+token (proves the form-bearing interaction reached the prompt path),
// (b) the response value flows through to the output.
func TestGraph_Interaction_FormElements_All(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_form_elements.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		var promptCount int
		go func() {
			for p := range prompt {
				promptCount++
				assert.NotEmpty(t, p.GetId(), "prompt must carry interaction id")
				assert.NotEmpty(t, p.GetToken(), "prompt must carry token")
				anyVal, _ := common.WrapProtoAny(int64(11))
				response <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(response)
		}()

		feedInput(ps, "inputs.x", int64(1))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		close(prompt)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, promptCount, 1, "form-elements interaction must raise a prompt")

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(11), results[0].GetValue().GetInt64Value())
	})
}

// Interaction: form with inputs[] (confirm + input elements). The runtime
// accepts the form definition, prompts, and forwards the response value.

// TestGraph_Interaction_FormInputs verifies an interaction with form
// inputs (confirm + input elements) round-trips. The form definition is
// validated at graph.Build(); runtime emits the prompt and routes the
// response back. Same shape as FormElements_All but with the
// interaction_form.yaml fixture variant.
func TestGraph_Interaction_FormInputs(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_form.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		var promptCount int
		// Capture form input details from the first prompt so we can
		// assert on form-input ids and the dynamic title outside the
		// auto-responder goroutine (Go test asserts must run on the test
		// goroutine).
		var capturedFormInputIds []string
		var capturedNotesTitle string
		go func() {
			for p := range prompt {
				promptCount++
				assert.NotEmpty(t, p.GetId(), "prompt must carry interaction id")
				assert.NotEmpty(t, p.GetToken(), "prompt must carry token")
				if capturedFormInputIds == nil {
					for _, in := range p.GetInputs() {
						capturedFormInputIds = append(capturedFormInputIds, in.GetId())
						if in.GetId() == "notes" {
							capturedNotesTitle = in.GetTitle()
						}
					}
				}
				anyVal, _ := common.WrapProtoAny(int64(7))
				response <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(response)
		}()

		feedInput(ps, "inputs.x", int64(1))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		close(prompt)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, promptCount, 1, "form-input interaction must raise a prompt")

		// Spec contract: form-input ids must appear in the prompt's Inputs[]
		// list, and dynamic CEL titles must be resolved against upstream
		// state (here, inputs.x.value=1 -> "Notes for value 1").
		assert.ElementsMatch(t, []string{"agree", "notes"}, capturedFormInputIds,
			"prompt must carry form-input ids from the YAML (agree, notes)")
		assert.Equal(t, "Notes for value 1", capturedNotesTitle,
			"dynamic CEL title must be resolved against inputs.x.value=1")

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(7), results[0].GetValue().GetInt64Value())
	})
}

// Interaction: wrong token is dropped, correct token is delivered

func TestGraph_Interaction_WrongTokenDropped(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_basic.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		// First send a response with the wrong token (dropped), then send correct.
		go func() {
			for p := range prompt {
				// Wrong token -- should be silently dropped.
				wrongVal, _ := common.WrapProtoAny(int64(999))
				response <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: "wrong-token", Value: wrongVal}

				// Correct token -- should be delivered.
				correctVal, _ := common.WrapProtoAny(int64(42))
				response <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: correctVal}
			}
			close(response)
		}()

		feedInput(ps, "inputs.x", int64(1))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		close(prompt) // Unblock auto-respond goroutine.
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1)
		assert.Equal(t, int64(42), results[0].GetValue().GetInt64Value())
	})
}

// buildInteractionResponse builds the canonical production interaction-response
// wire shape: an Interaction.Response carrying one TYPED binding per input id,
// each packed as its own Any, wrapped in an outer Any. Mirrors the CLI's
// interactionResponseValue and the UI responder. The runtime lifts this into
// the CEL map `interactions.<id>.value.<input_id>.<binding_field>`.
//
// Because bindings travel typed (not flattened into a generic
// google.protobuf.Struct), every declared field is always present -- so this
// helper is the only one needed: there is no "sparse" variant, a zero-valued
// binding (e.g. an empty InputBinding) still carries its `value` field.
//
// Tests that send a primitive directly via common.WrapProtoAny bypass this
// binding-preserving wrap and can't catch breakage in the access path; use
// this helper for fixtures whose CEL path crosses the input boundary.
func buildInteractionResponse(t *testing.T, bindings map[string]proto.Message) *anypb.Any {
	t.Helper()
	out := make(map[string]*anypb.Any, len(bindings))
	for id, binding := range bindings {
		b, err := anypb.New(binding)
		require.NoError(t, err)
		out[id] = b
	}
	resp := flowv1beta2.Interaction_Response_builder{Bindings: out}.Build()
	any, err := anypb.New(resp)
	require.NoError(t, err)
	return any
}

// TestGraph_Interaction_ZeroValuedBinding proves a zero-valued binding still
// resolves: a blank text input is an Interaction_InputBinding{} whose `value`
// is the proto3 zero "". With the typed-binding wire shape, that field is
// always present, so the output's CEL `interactions.ask.value.note.value`
// resolves to "" instead of erroring "no such key: value" -- the structural
// fix for the ticker-v2interaction-showcase bug, which a generic-Struct
// encoding (now removed) could drop.
func TestGraph_Interaction_ZeroValuedBinding(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_zero_binding.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 16)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 16)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		const maxIters uint64 = 3
		var promptCount int
		go func() {
			for p := range prompt {
				promptCount++
				// Zero-valued InputBinding: value is "" (proto3 zero). Typed
				// transport keeps the field present regardless.
				anyVal := buildInteractionResponse(t, map[string]proto.Message{
					"note": &flowv1beta2.Interaction_InputBinding{},
				})
				response <- &flowv1beta2.InteractionResponseEvent{
					Id: p.GetId(), Token: p.GetToken(), Value: anyVal,
				}
			}
			close(response)
		}()

		feedInput(ps, "inputs.maxIters", maxIters)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		close(prompt)
		require.NoError(t, err,
			"a zero-valued InputBinding must resolve `.value` to \"\", not error 'no such key: value'")

		results := collectOutputs(ctx, ps, "outputs.echo")
		t.Logf("prompts=%d outputs=%d", promptCount, len(results))
		require.Len(t, results, promptCount, "every prompt must yield an output")
		for i, r := range results {
			_, isStr := r.GetValue().GetKind().(*expr.Value_StringValue)
			assert.True(t, isStr, "output[%d] must be a string kind (InputBinding.value zero), got %T", i, r.GetValue().GetKind())
			assert.Equal(t, "", r.GetValue().GetStringValue(), "output[%d] is the blank input's default", i)
		}
	})
}

// TestGraph_Interaction_TickerVarRepro reproduces an interactive-ticker
// topology that the simpler interaction_output_filter_idiomatic fixture
// failed to reproduce. Production logs showed a "no such key: discard"
// error from the output's CEL eval despite the matching idiomatic test
// passing. Difference: this fixture introduces a generator + var
// upstream of the interaction (title CEL references both), which
// affects graph wiring and node-iteration cadence.
//
// Drives maxIters=5 so the multi-iteration cadence is exercised:
// every iteration must (a) prompt, (b) flow the response through to
// the output, and (c) emit a post-transform StringValue. A stuck
// handler or a wire-shape regression would surface as wrong counts
// here, not a missing key like the simpler fixture caught.
//
// Always responds with discard=false so the output's filter passes
// and the count of outputs matches the count of prompts.
func TestGraph_Interaction_TickerVarRepro(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_ticker_var_repro.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 16)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 16)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		const maxIters uint64 = 5
		var promptCount int
		go func() {
			for p := range prompt {
				promptCount++
				binding := &flowv1beta2.Interaction_ConfirmBinding{}
				binding.SetValue(false) // always pass the filter
				anyVal := buildInteractionResponse(t, map[string]proto.Message{
					"discard": binding,
				})
				response <- &flowv1beta2.InteractionResponseEvent{
					Id: p.GetId(), Token: p.GetToken(), Value: anyVal,
				}
			}
			close(response)
		}()

		feedInput(ps, "inputs.maxIters", maxIters)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		close(prompt)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.evenOrOdd")
		t.Logf("prompts=%d outputs=%d", promptCount, len(results))
		for i, r := range results {
			t.Logf("  output[%d] kind=%T value=%v", i, r.GetValue().GetKind(), r.GetValue())
		}
		assert.Equal(t, int(maxIters), promptCount,
			"expected exactly maxIters prompts; the var.flow_control.stop_when fires after maxIters ticks and the interaction must drain a prompt for each (tick, var) iteration")
		require.Len(t, results, int(maxIters),
			"with discard=false on every prompt the output filter must pass exactly once per prompt")
		for i, r := range results {
			assert.NotNil(t, r.GetValue().GetStringValue(),
				"output[%d] post-transform value must be a StringValue, got kind=%T", i, r.GetValue().GetKind())
		}
	})
}

// TestGraph_Interaction_TickerVarRepro_DiscardAll mirrors the
// production "approve all" path: every response sets discard=true,
// so the output's filter rejects every value. We expect
// `maxIters` prompts but ZERO outputs, matching what the user sees
// when approving every prompt in tickerv2_interactive.
func TestGraph_Interaction_TickerVarRepro_DiscardAll(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_ticker_var_repro.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 16)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 16)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		const maxIters uint64 = 5
		var promptCount int
		go func() {
			for p := range prompt {
				promptCount++
				binding := &flowv1beta2.Interaction_ConfirmBinding{}
				binding.SetValue(true) // filter rejects
				anyVal := buildInteractionResponse(t, map[string]proto.Message{
					"discard": binding,
				})
				response <- &flowv1beta2.InteractionResponseEvent{
					Id: p.GetId(), Token: p.GetToken(), Value: anyVal,
				}
			}
			close(response)
		}()

		feedInput(ps, "inputs.maxIters", maxIters)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		close(prompt)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.evenOrOdd")
		t.Logf("prompts=%d outputs=%d", promptCount, len(results))
		assert.Equal(t, int(maxIters), promptCount,
			"every iteration must prompt; the filter rejecting downstream is unrelated to whether the prompt fires")
		assert.Len(t, results, 0, "with discard=true on every prompt the filter must reject every value")
	})
}

// TestGraph_Interaction_TickerOnlyRepro is a bisection between
// VarRepro (passes) and TickerVarRepro (hangs): ticker upstream
// (no var) on the interaction. If this hangs, the bug is "ticker
// generator + interaction" multi-iter; if it passes, the bug is
// specific to the (ticker, var) pair on the interaction's deps.
func TestGraph_Interaction_TickerOnlyRepro(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_ticker_only_repro.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 16)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 16)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		const maxIters uint64 = 5
		var promptCount int
		go func() {
			for p := range prompt {
				promptCount++
				binding := &flowv1beta2.Interaction_ConfirmBinding{}
				binding.SetValue(false)
				anyVal := buildInteractionResponse(t, map[string]proto.Message{
					"discard": binding,
				})
				response <- &flowv1beta2.InteractionResponseEvent{
					Id: p.GetId(), Token: p.GetToken(), Value: anyVal,
				}
			}
			close(response)
		}()

		feedInput(ps, "inputs.maxIters", maxIters)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		close(prompt)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.tick")
		t.Logf("prompts=%d outputs=%d", promptCount, len(results))
		assert.Equal(t, int(maxIters), promptCount, "ticker-only multi-iter should fire maxIters prompts")
	})
}

// TestGraph_Interaction_VarRepro is a bisection point between
// the existing TestGraph_Interaction_MultiplePrompts (which works)
// and TestGraph_Interaction_TickerVarRepro (which hangs). It uses
// the same var+interaction+output shape as TickerVar but feeds the
// upstream from inputs.x instead of a ticker generator. If this
// passes while TickerVar fails, the bug is specific to ticker as
// the upstream driver.
func TestGraph_Interaction_VarRepro(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_var_repro.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 16)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 16)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		var promptCount int
		go func() {
			for p := range prompt {
				promptCount++
				binding := &flowv1beta2.Interaction_ConfirmBinding{}
				binding.SetValue(false)
				anyVal := buildInteractionResponse(t, map[string]proto.Message{
					"discard": binding,
				})
				response <- &flowv1beta2.InteractionResponseEvent{
					Id: p.GetId(), Token: p.GetToken(), Value: anyVal,
				}
			}
			close(response)
		}()

		feedInput(ps, "inputs.x", int64(1), int64(2), int64(3), int64(4), int64(5))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		close(prompt)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.tag")
		t.Logf("prompts=%d outputs=%d", promptCount, len(results))
		assert.Equal(t, 5, promptCount, "var-driven multi-iter should fire 5 prompts")
		assert.Len(t, results, 5, "var-driven multi-iter should produce 5 outputs (all discard=false)")
	})
}

// TestGraph_Interaction_TickerVarRepro_Mixed alternates discard
// true/false so we verify the exact set of outputs (not just count):
// output K should be present iff the K-th response had discard=false.
// Catches a class of bug where the runtime would mismatch
// var-iter vs interaction-iter (e.g. interaction reading a stale
// response cached from a prior iteration).
func TestGraph_Interaction_TickerVarRepro_Mixed(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_ticker_var_repro.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 16)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 16)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		const maxIters uint64 = 6
		// Pattern: alternate F,T,F,T,F,T -- so iters 1/3/5 produce
		// outputs ("odd","odd","odd" since odd at counts 1,3,5),
		// iters 2/4/6 are filtered out.
		discardPattern := []bool{false, true, false, true, false, true}
		var promptCount int
		go func() {
			for p := range prompt {
				idx := promptCount
				promptCount++
				if idx >= len(discardPattern) {
					t.Errorf("more prompts (%d) than discardPattern length (%d)", promptCount, len(discardPattern))
					close(response)
					return
				}
				binding := &flowv1beta2.Interaction_ConfirmBinding{}
				binding.SetValue(discardPattern[idx])
				anyVal := buildInteractionResponse(t, map[string]proto.Message{
					"discard": binding,
				})
				response <- &flowv1beta2.InteractionResponseEvent{
					Id: p.GetId(), Token: p.GetToken(), Value: anyVal,
				}
			}
			close(response)
		}()

		feedInput(ps, "inputs.maxIters", maxIters)
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		close(prompt)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.evenOrOdd")
		t.Logf("prompts=%d outputs=%d", promptCount, len(results))
		for i, r := range results {
			t.Logf("  output[%d] = %q", i, r.GetValue().GetStringValue())
		}
		assert.Equal(t, int(maxIters), promptCount, "every iteration must prompt")
		// Iter K (1-indexed) eval_count=K -> "odd" if K%2==1, "even"
		// if K%2==0. Pattern F,T,F,T,F,T keeps K=1,3,5 -> all "odd".
		require.Len(t, results, 3, "discard pattern F,T,F,T,F,T must keep exactly the 3 false slots")
		for i, r := range results {
			assert.Equal(t, "odd", r.GetValue().GetStringValue(),
				"results[%d]: with maxIters=6 and the F/T/F/T/F/T pattern, all kept values are odd", i)
		}
	})
}

// Idiomatic pattern: graph-aware logic in the producer's main
// expression composes a struct with the value + interaction gate;
// transforms operate on `this` alone (filter on gate, map to value).
// Replaces the older naive shape that put `interactions.X.value`
// directly in a transform filter -- now rejected by lint.
//
// This test mirrors the CLI's production response shape (binding
// preserved, struct keyed by input id) so the CEL path
// `interactions.confirmDiscard.value.discard.value` exercises the
// same chain of conversions that broke in production:
//   - executor wraps the response Any as expr.Value{ObjectValue: Any}
//   - CEL ProtoAsValue unmarshals the Any into a *structpb.Struct
//   - field access `.discard` returns a *structpb.Value{StructValue:...}
//   - field access `.value` reaches the bool inside the binding
//
// Two inputs (1, 2). Auto-respond Confirm{value=true} for the first
// prompt (filter rejects → no output) and Confirm{value=false} for
// the second (filter passes → map extracts value 2).
func TestGraph_Interaction_OutputFilterIdiomatic(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_output_filter_idiomatic.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck // deferred test teardown; runs after assertions, no recovery path

		var promptCount int
		go func() {
			for p := range prompt {
				promptCount++
				discard := promptCount%2 == 1 // first true, second false
				binding := &flowv1beta2.Interaction_ConfirmBinding{}
				binding.SetValue(discard)
				anyVal := buildInteractionResponse(t, map[string]proto.Message{
					"discard": binding,
				})
				response <- &flowv1beta2.InteractionResponseEvent{Id: p.GetId(), Token: p.GetToken(), Value: anyVal}
			}
			close(response)
		}()

		feedInput(ps, "inputs.x", int64(1), int64(2))
		ctx := testContext(t)
		err := NewExecutor(ps, testTopics, append([]Option{WithInteractions(prompt, response)}, extraOpts...)...).Execute(ctx, graph)
		close(prompt)
		require.NoError(t, err)

		results := collectOutputs(ctx, ps, "outputs.result")
		require.Len(t, results, 1, "filter passes exactly once (second input, discard=false)")
		assert.Equal(t, int64(2), results[0].GetValue().GetInt64Value())
	})
}
