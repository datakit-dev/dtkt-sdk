package runtime

import (
	"context"
	"fmt"
	"maps"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"golang.org/x/sync/errgroup"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

type switchCase struct {
	condition cel.Program
	result    cel.Program
}

type switchHandler struct {
	flowControlMixin
	id          string
	inputs      map[string]<-chan *pubsub.Message
	pubsub      executor.PubSub
	topic       string
	valueProg   cel.Program
	cases       []switchCase
	defaultProg cel.Program
	transforms  *transformPipeline
	transformPS executor.PubSub
	adapter     types.Adapter
}

// evalSwitch evaluates the switch expression and returns the result value.
func (h *switchHandler) evalSwitch(vars map[string]any) (*expr.Value, error) {
	switchVal, err := evalCEL(h.valueProg, vars)
	if err != nil {
		return nil, fmt.Errorf("switch %s value eval: %w", h.id, err)
	}
	switchExpr, err := refValToExpr(switchVal)
	if err != nil {
		return nil, fmt.Errorf("switch %s value convert: %w", h.id, err)
	}

	thisMap := map[string]any{"value": exprToRefVal(h.adapter, switchExpr)}
	caseVars := maps.Clone(vars)
	caseVars["this"] = thisMap

	var resultExpr = switchExpr
	matched := false
	for i, c := range h.cases {
		condResult, err := evalCEL(c.condition, caseVars)
		if err != nil {
			return nil, fmt.Errorf("switch %s case[%d] condition eval: %w", h.id, i, err)
		}
		if condResult.Value() == true {
			retResult, err := evalCEL(c.result, caseVars)
			if err != nil {
				return nil, fmt.Errorf("switch %s case[%d] return eval: %w", h.id, i, err)
			}
			resultExpr, err = refValToExpr(retResult)
			if err != nil {
				return nil, fmt.Errorf("switch %s case[%d] return convert: %w", h.id, i, err)
			}
			matched = true
			break
		}
	}

	if !matched {
		defResult, err := evalCEL(h.defaultProg, caseVars)
		if err != nil {
			return nil, fmt.Errorf("switch %s default eval: %w", h.id, err)
		}
		resultExpr, err = refValToExpr(defResult)
		if err != nil {
			return nil, fmt.Errorf("switch %s default convert: %w", h.id, err)
		}
	}
	return resultExpr, nil
}

func (h *switchHandler) Run(ctx context.Context) error {
	if h.transforms != nil {
		return h.runWithTransforms(ctx)
	}

	var evalCount uint64
	for {
		act := newActivationFromChannels(ctx, h.inputs, h.adapter)
		vars, err := act.Resolve()
		if err != nil {
			return fmt.Errorf("switch %s resolve: %w", h.id, err)
		}
		if act.AnyEOF() {
			break
		}

		resultExpr, err := h.evalSwitch(vars)
		if err != nil {
			return err
		}

		evalCount++
		node := flowv1beta2.RunSnapshot_VarNode_builder{
			Id:        h.id,
			Value:     resultExpr,
			EvalCount: evalCount,
			Phase:     flowv1beta2.RunSnapshot_PHASE_RUNNING,
		}.Build()
		if err := publishNode(h.pubsub, h.topic, node); err != nil {
			return err
		}
		h.checkFC(vars)
	}
	return publishNode(h.pubsub, h.topic, flowv1beta2.RunSnapshot_VarNode_builder{
		Id:        h.id,
		Value:     newEOFValue(),
		EvalCount: evalCount,
		Phase:     flowv1beta2.RunSnapshot_PHASE_SUCCEEDED,
	}.Build())
}

func (h *switchHandler) runWithTransforms(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	var sinkCount uint64
	sink := func(_ context.Context, val *expr.Value, eof bool) error {
		sinkCount++
		phase := flowv1beta2.RunSnapshot_PHASE_RUNNING
		if eof {
			phase = flowv1beta2.RunSnapshot_PHASE_SUCCEEDED
		}
		return publishNode(h.pubsub, h.topic, flowv1beta2.RunSnapshot_VarNode_builder{
			Id:        h.id,
			Value:     val,
			EvalCount: sinkCount,
			Phase:     phase,
		}.Build())
	}
	onState := newStateCallback(h.pubsub, h.topic, len(h.transforms.steps),
		func(t []*flowv1beta2.RunSnapshot_Transform) executor.StateNode {
			return flowv1beta2.RunSnapshot_VarNode_builder{
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
			act := newActivationFromChannels(ctx, h.inputs, h.adapter)
			vars, err := act.Resolve()
			if err != nil {
				return fmt.Errorf("switch %s resolve: %w", h.id, err)
			}
			if act.AnyEOF() {
				break
			}
			resultExpr, err := h.evalSwitch(vars)
			if err != nil {
				return err
			}
			if err := h.transformPS.Publish(inputTopic, pubsub.NewMessage(resultExpr)); err != nil {
				return err
			}
		}
		return h.transformPS.Publish(inputTopic, pubsub.NewMessage(newEOFValue()))
	})

	return g.Wait()
}
