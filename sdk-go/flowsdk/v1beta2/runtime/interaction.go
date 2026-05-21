package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// interactionHandler presents a prompt and waits for a response from an
// external source (CLI, gRPC bidi stream, etc.). The prompt is sent via the
// prompt channel; the response arrives via TryDeliver from the demux goroutine.
//
// Per-input CEL fields (title, description) are compiled at construction
// time and evaluated at prompt time against the activation vars, then
// shipped as resolved Interaction.Input messages on the
// InteractionRequestEvent. This mirrors v1beta1's wire shape (entire
// resolved UserAction in the event) and is necessary because the
// runtime is the only process holding live flow vars; responders cannot
// re-evaluate CEL on the wire side.
type interactionHandler struct {
	lifecycleMixin
	suspendableMixin
	stoppableMixin
	id          string
	inputs      map[string]<-chan *pubsub.Message // upstream channels feeding the `when` guard
	pubsub      executor.PubSub
	topic       string
	prompt      chan<- *flowv1beta2.InteractionRequestEvent
	deliver     chan *expr.Value // buffered(1), written by TryDeliver, read by promptAndWait
	whenProg    cel.Program
	transforms  *transformPipeline
	transformPS executor.PubSub
	env         shared.Env
	cache       *cacheBackend

	// formInputs is the spec's Interaction.Input list paired with any
	// compiled CEL programs for per-prompt fields. Each element is
	// cloned and CEL-resolved per prompt; the resolved copies are
	// shipped in InteractionRequestEvent.inputs.
	formInputs []interactionFormInput

	tokenMu      sync.Mutex
	pendingToken string
}

// interactionFormInput pairs an Interaction.Input spec with the
// compiled CEL programs for its per-prompt evaluatable fields. Nil
// programs mean the spec value was literal (no `=` prefix) and is
// shipped unchanged.
type interactionFormInput struct {
	spec      *flowv1beta2.Interaction_Input
	titleProg cel.Program
	descProg  cel.Program
}

// compileExprField returns a compiled CEL program for `raw` if it is a
// CEL expression (has the `=` prefix), or (nil, nil) if it's empty or
// a literal string. The caller skips evaluation for nil programs.
func compileExprField(env shared.Env, raw string) (cel.Program, error) {
	if raw == "" {
		return nil, nil
	}
	if _, isCEL := shared.IsValidExpr(raw); !isCEL {
		return nil, nil
	}
	return compileCEL(env, raw)
}

// evalStringField runs a compiled CEL program and returns its string
// result. Returns ("", false, nil) when the program is nil (literal/
// absent field; caller passes through unchanged). Errors are wrapped
// with the interaction/input/field identifier for diagnostic clarity.
func evalStringField(prog cel.Program, vars map[string]any, intrID, inputID, field string) (string, bool, error) {
	if prog == nil {
		return "", false, nil
	}
	val, err := evalCEL(prog, vars)
	if err != nil {
		return "", false, fmt.Errorf("interaction %s input %s %s eval: %w", intrID, inputID, field, err)
	}
	s, ok := val.Value().(string)
	if !ok {
		return "", false, fmt.Errorf("interaction %s input %s %s CEL must yield string, got %T", intrID, inputID, field, val.Value())
	}
	return s, true, nil
}

// compileInteractionInputs pairs each spec input with compiled CEL
// programs for its title and description (nil for literals).
func compileInteractionInputs(env shared.Env, inputs []*flowv1beta2.Interaction_Input) ([]interactionFormInput, error) {
	out := make([]interactionFormInput, len(inputs))
	for i, input := range inputs {
		titleProg, err := compileExprField(env, input.GetTitle())
		if err != nil {
			return nil, fmt.Errorf("compiling input %s title CEL: %w", input.GetId(), err)
		}
		var descProg cel.Program
		if input.HasDescription() {
			descProg, err = compileExprField(env, input.GetDescription())
			if err != nil {
				return nil, fmt.Errorf("compiling input %s description CEL: %w", input.GetId(), err)
			}
		}
		out[i] = interactionFormInput{spec: input, titleProg: titleProg, descProg: descProg}
	}
	return out, nil
}

// resolveInteractionInputs evaluates the per-input CEL programs against
// vars and returns clones of the spec inputs with the resolved
// title/description substituted in. Literal-string fields pass through.
func (h *interactionHandler) resolveInteractionInputs(vars map[string]any) ([]*flowv1beta2.Interaction_Input, error) {
	resolved := make([]*flowv1beta2.Interaction_Input, len(h.formInputs))
	for i, fi := range h.formInputs {
		// proto.Clone preserves the element oneof and any non-CEL fields;
		// we override only the title/description below.
		clone := proto.Clone(fi.spec).(*flowv1beta2.Interaction_Input)
		if s, ok, err := evalStringField(fi.titleProg, vars, h.id, fi.spec.GetId(), "title"); err != nil {
			return nil, err
		} else if ok {
			clone.SetTitle(s)
		}
		if s, ok, err := evalStringField(fi.descProg, vars, h.id, fi.spec.GetId(), "description"); err != nil {
			return nil, err
		} else if ok {
			clone.SetDescription(s)
		}
		resolved[i] = clone
	}
	return resolved, nil
}

// responseValue lifts the interaction response (Interaction.Response: one
// typed binding per form input, keyed by input id) into the CEL value the
// node exposes: a map of input id -> binding, so `interactions.<id>.value
// .<input_id>.<field>` resolves. Because each binding is a typed proto
// message, every declared field is always present (`value` at its proto
// zero, plus any per-binding metadata) - so the access path is robust to how
// a responder serialized implicit-presence fields, with no reconstruction.
//
// Inputs the responder omitted entirely are filled with their zero typed
// binding so `.value` still resolves. A non-Response payload is exposed
// verbatim (defensive).
func (h *interactionHandler) responseValue(a *anypb.Any) *expr.Value {
	raw := &expr.Value{Kind: &expr.Value_ObjectValue{ObjectValue: a}}
	if a == nil {
		return raw
	}
	msg, err := a.UnmarshalNew()
	if err != nil {
		return raw
	}
	resp, ok := msg.(*flowv1beta2.Interaction_Response)
	if !ok {
		return raw
	}
	sent := resp.GetBindings()
	entries := make([]*expr.MapValue_Entry, 0, len(h.formInputs))
	for _, fi := range h.formInputs {
		id := fi.spec.GetId()
		bindingAny := sent[id]
		if bindingAny == nil {
			// Responder omitted this input: synthesize its zero typed binding
			// so `.value` resolves at the typed default.
			if zero := bindingForInput(fi.spec); zero != nil {
				if z, err := anypb.New(zero); err == nil {
					bindingAny = z
				}
			}
		}
		if bindingAny == nil {
			continue
		}
		entries = append(entries, &expr.MapValue_Entry{
			Key:   &expr.Value{Kind: &expr.Value_StringValue{StringValue: id}},
			Value: &expr.Value{Kind: &expr.Value_ObjectValue{ObjectValue: bindingAny}},
		})
	}
	return &expr.Value{Kind: &expr.Value_MapValue{MapValue: &expr.MapValue{Entries: entries}}}
}

// bindingForInput returns an empty typed binding proto for the input's
// element kind, or nil if the kind is unrecognised. Mirrors the binding
// selection in flowsdk/v1beta2.GetInteractionBinding (kept inline to avoid a
// runtime -> flowsdk/v1beta2 import cycle).
func bindingForInput(input *flowv1beta2.Interaction_Input) proto.Message {
	switch input.WhichElement() {
	case flowv1beta2.Interaction_Input_Confirm_case:
		return &flowv1beta2.Interaction_ConfirmBinding{}
	case flowv1beta2.Interaction_Input_Input_case:
		return &flowv1beta2.Interaction_InputBinding{}
	case flowv1beta2.Interaction_Input_File_case:
		return &flowv1beta2.Interaction_FileBinding{}
	case flowv1beta2.Interaction_Input_Select_case:
		return &flowv1beta2.Interaction_SelectBinding{}
	case flowv1beta2.Interaction_Input_MultiSelect_case:
		return &flowv1beta2.Interaction_MultiSelectBinding{}
	}
	return nil
}

// TryDeliver validates the token and delivers the value atomically.
// Returns false if the token doesn't match (stale/invalid response).
func (h *interactionHandler) TryDeliver(token string, val *expr.Value) bool {
	h.tokenMu.Lock()
	defer h.tokenMu.Unlock()
	if h.pendingToken == "" || token != h.pendingToken {
		return false
	}
	h.pendingToken = ""
	h.deliver <- val
	return true
}

// Close signals the handler that no more responses will arrive (EOF).
func (h *interactionHandler) Close() {
	close(h.deliver)
}

func (h *interactionHandler) Run(ctx context.Context) error {
	if h.transforms != nil {
		return h.runWithTransforms(ctx)
	}

	for {
		// Pause point: between iterations only. An in-flight prompt
		// completes naturally - we don't cancel a prompt the operator
		// is already responding to.
		act := h.cache.newActivation(ctx, h.inputs, h.env, h.SuspendChan(), h.StopChan())
		vars, err := act.Resolve()
		if errors.Is(err, errOperatorStopped) {
			break
		}
		if errors.Is(err, errOperatorSuspended) {
			res := h.waitForResume(ctx, h.StopChan())
			if res == suspendCancelled {
				return ctx.Err()
			}
			if res == suspendStopped {
				break
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("interaction %s resolve: %w", h.id, err)
		}
		if act.AnyEOF() {
			break
		}

		if h.whenProg != nil {
			result, err := evalCEL(h.whenProg, vars)
			if err != nil {
				return fmt.Errorf("interaction %s when: %w", h.id, err)
			}
			if result.Value() != true {
				continue
			}
		}

		val, err := h.promptAndWait(ctx, vars)
		if errors.Is(err, errOperatorStopped) {
			break
		}
		if errors.Is(err, errOperatorSuspended) {
			res := h.waitForResume(ctx, h.StopChan())
			if res == suspendCancelled {
				return ctx.Err()
			}
			if res == suspendStopped {
				break
			}
			continue
		}
		if err != nil {
			return err
		}
		if isEOFValue(val) {
			break
		}

		node := flowv1beta2.RunSnapshot_InteractionNode_builder{
			Id:        h.id,
			Value:     val,
			Submitted: true,
			Phase:     flowv1beta2.RunSnapshot_PHASE_RUNNING,
		}.Build()
		if err := publishNode(h.pubsub, h.topic, node); err != nil {
			return err
		}
		h.checkLifecycle(vars)
	}
	return publishNode(h.pubsub, h.topic, flowv1beta2.RunSnapshot_InteractionNode_builder{
		Id:    h.id,
		Value: newEOFValue(),
		Phase: flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
	}.Build())
}

func (h *interactionHandler) runWithTransforms(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	var sinkCount uint64
	sink := func(_ context.Context, val *expr.Value, eof bool) error {
		sinkCount++
		phase := flowv1beta2.RunSnapshot_PHASE_RUNNING
		if eof {
			phase = flowv1beta2.RunSnapshot_PHASE_SUCCEEDED
		}
		return publishNode(h.pubsub, h.topic, flowv1beta2.RunSnapshot_InteractionNode_builder{
			Id:        h.id,
			Value:     val,
			Submitted: true,
			Phase:     phase,
		}.Build())
	}
	onState := newStateCallback(h.pubsub, h.topic, len(h.transforms.steps),
		func(t []*flowv1beta2.RunSnapshot_Transform) executor.StateNode {
			return flowv1beta2.RunSnapshot_InteractionNode_builder{
				Id:         h.id,
				Transforms: t,
				Phase:      flowv1beta2.RunSnapshot_PHASE_RUNNING,
			}.Build()
		})

	inputTopic, err := h.transforms.Start(ctx, g, h.transformPS, h.topic, sink, onState)
	if err != nil {
		return err
	}

	g.Go(func() error {
		for {
			act := h.cache.newActivation(ctx, h.inputs, h.env, h.SuspendChan(), h.StopChan())
			vars, err := act.Resolve()
			if errors.Is(err, errOperatorStopped) {
				break
			}
			if errors.Is(err, errOperatorSuspended) {
				res := h.waitForResume(ctx, h.StopChan())
				if res == suspendCancelled {
					return ctx.Err()
				}
				if res == suspendStopped {
					break
				}
				continue
			}
			if err != nil {
				return fmt.Errorf("interaction %s resolve: %w", h.id, err)
			}
			if act.AnyEOF() {
				break
			}

			if h.whenProg != nil {
				result, err := evalCEL(h.whenProg, vars)
				if err != nil {
					return fmt.Errorf("interaction %s when: %w", h.id, err)
				}
				if result.Value() != true {
					continue
				}
			}

			val, err := h.promptAndWait(ctx, vars)
			if errors.Is(err, errOperatorStopped) {
				break
			}
			if errors.Is(err, errOperatorSuspended) {
				res := h.waitForResume(ctx, h.StopChan())
				if res == suspendCancelled {
					return ctx.Err()
				}
				if res == suspendStopped {
					break
				}
				continue
			}
			if err != nil {
				return err
			}
			if isEOFValue(val) {
				break
			}
			if err := h.transformPS.Publish(inputTopic, pubsub.NewMessage(val)); err != nil {
				return err
			}
		}
		return h.transformPS.Publish(inputTopic, pubsub.NewMessage(newEOFValue()))
	})

	return g.Wait()
}

// promptAndWait sends a prompt with a UUIDv7 token to the external source and
// blocks until a matching response arrives or the context is cancelled.
//
// vars carries the activation's resolved variables; per-input title and
// description CEL on the spec is evaluated against this map and the
// resolved Interaction.Input list is shipped on the event so external
// responders (CLI, web UI, etc.) render literal strings without needing
// flow-var access.
func (h *interactionHandler) promptAndWait(ctx context.Context, vars map[string]any) (*expr.Value, error) {
	// Drain any stale value that landed in h.deliver from a previous
	// prompt where TryDeliver completed but the consume was preempted
	// by an operator signal (stop/suspend) in the same select. Safe
	// because at this point pendingToken == "" so TryDeliver won't be
	// writing concurrently.
	select {
	case <-h.deliver:
	default:
	}

	token, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("interaction %s: generating token: %w", h.id, err)
	}
	tokenStr := token.String()

	// Store the pending token so the demux goroutine can validate responses.
	h.tokenMu.Lock()
	h.pendingToken = tokenStr
	h.tokenMu.Unlock()

	// Publish pending state with token on the InteractionNode snapshot.
	// Uses a state event (not output) so downstream nodes don't see it as a value.
	pending := flowv1beta2.RunSnapshot_InteractionNode_builder{
		Id:    h.id,
		Token: tokenStr,
		Phase: flowv1beta2.RunSnapshot_PHASE_RUNNING,
	}.Build()
	if err := publishStateEvent(h.pubsub, h.topic, pending); err != nil {
		return nil, err
	}

	// Resolve per-input CEL once, ship in the prompt event.
	resolvedInputs, err := h.resolveInteractionInputs(vars)
	if err != nil {
		return nil, err
	}

	// Send prompt with node ID, token, and resolved inputs. Observe
	// operator signals here too -- a full prompt channel would
	// otherwise leave us stuck until ctx cancellation. clearToken is
	// reused below for the wait select.
	clearToken := func() {
		h.tokenMu.Lock()
		h.pendingToken = ""
		h.tokenMu.Unlock()
	}
	select {
	case <-ctx.Done():
		clearToken()
		return nil, ctx.Err()
	case <-h.StopChan():
		clearToken()
		return nil, errOperatorStopped
	case <-h.SuspendChan():
		clearToken()
		return nil, errOperatorSuspended
	case h.prompt <- flowv1beta2.InteractionRequestEvent_builder{
		Id:     h.id,
		Token:  tokenStr,
		Inputs: resolvedInputs,
	}.Build():
	}

	// Wait for response. Observe operator signals so an in-flight
	// prompt unblocks when the user/operator stops or suspends the
	// flow -- without these, the handler stays parked on h.deliver
	// until the stop-timeout escalates to a hard terminate.
	//
	// On stop/suspend we clear the pending token so any late-arriving
	// response is dropped (TryDeliver checks pendingToken). A racing
	// TryDeliver that already wrote to h.deliver before we observed
	// the operator signal leaves a stale value behind; the drain at
	// the top of the next promptAndWait clears it.
	select {
	case <-ctx.Done():
		clearToken()
		return nil, ctx.Err()
	case <-h.StopChan():
		clearToken()
		return nil, errOperatorStopped
	case <-h.SuspendChan():
		clearToken()
		return nil, errOperatorSuspended
	case val, ok := <-h.deliver:
		if !ok {
			return newEOFValue(), nil
		}
		return val, nil
	}
}
