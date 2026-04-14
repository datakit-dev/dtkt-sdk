package runtime

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Interaction: basic prompt → response flow

func TestGraph_Interaction_Basic(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_basic.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

		// Auto-respond to interaction prompts.
		go func() {
			for p := range prompt {
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
		defer ps.Close() //nolint:errcheck

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
		defer ps.Close() //nolint:errcheck

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
		defer ps.Close() //nolint:errcheck

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

// Interaction: wrong token is dropped, correct token is delivered

func TestGraph_Interaction_WrongTokenDropped(t *testing.T) {
	withAndWithoutOutbox(t, func(t *testing.T, extraOpts []Option) {
		graph := loadFlow(t, "interaction_basic.yaml")

		prompt := make(chan *flowv1beta2.InteractionRequestEvent, 4)
		response := make(chan *flowv1beta2.InteractionResponseEvent, 4)
		ps := newPubSub()
		defer ps.Close() //nolint:errcheck

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
