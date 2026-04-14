package runtime

import (
	"context"
	"fmt"

	expr "cel.dev/expr"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"golang.org/x/sync/errgroup"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// pipelineSink is called for each value exiting the transform pipeline.
// eof is true when the value is the EOF sentinel.
type pipelineSink func(ctx context.Context, val *expr.Value, eof bool) error

// stateCallback is called after each silent reduce accumulation (no output produced).
// stepIdx is the 0-based index of the step; acc is the updated accumulator value.
type stateCallback func(ctx context.Context, stepIdx int, acc *expr.Value) error

// transformPipeline holds compiled transform steps that can be started as pubsub-connected goroutines.
type transformPipeline struct {
	steps []transformStep
}

type transformStep struct {
	mapProgram    cel.Program      // non-nil for map transforms
	filterProgram cel.Program      // non-nil for filter transforms
	flatten       bool             // true for flatten transforms
	reduce        *accumulateState // non-nil for reduce transforms (emit on EOF only)
	scan          *accumulateState // non-nil for scan transforms (emit every intermediate)
	adapter       types.Adapter
}

// accumulateState is shared between reduce and scan.
type accumulateState struct {
	accProgram  cel.Program
	accumulator *expr.Value
	lastAcc     *expr.Value // most recently updated accumulator (nil until first accumulation)
	seen        bool        // true once at least one value has been accumulated
	adapter     types.Adapter

	// GroupBy fields (nil when ungrouped).
	keyProgram cel.Program
	initExpr   string                      // original initial expression, used to seed new groups
	celEnv     shared.Env                  // retained for compiling new group init expressions
	groups     map[string]*accumulateState // per-key accumulators
}

// Start launches goroutines for each transform step plus a sink goroutine.
// Subscriptions are created synchronously before goroutines start (no race with Publish).
// onState is called after each silent reduce accumulation; pass nil if not needed.
// Returns the input topic name where the caller should publish *expr.Value messages.
func (p *transformPipeline) Start(ctx context.Context, g *errgroup.Group, ps executor.PubSub, baseTopic string, sink pipelineSink, onState stateCallback) (string, error) {
	inputTopic := baseTopic + ":transforms[input]"

	// Subscribe to all topics before launching goroutines.
	current := inputTopic
	subs := make([]<-chan *pubsub.Message, len(p.steps))
	for i := range p.steps {
		ch, err := ps.Subscribe(ctx, current)
		if err != nil {
			return "", fmt.Errorf("subscribing to transform step %d: %w", i, err)
		}
		subs[i] = ch
		current = fmt.Sprintf("%s:transforms[%d]", baseTopic, i)
	}
	sinkCh, err := ps.Subscribe(ctx, current)
	if err != nil {
		return "", fmt.Errorf("subscribing to transform sink: %w", err)
	}

	// Launch step goroutines.
	for i := range p.steps {
		step := &p.steps[i]
		ch := subs[i]
		outTopic := fmt.Sprintf("%s:transforms[%d]", baseTopic, i)
		g.Go(func() error {
			return runStep(ctx, i, step, onState, ch, ps, outTopic)
		})
	}

	// Launch sink goroutine.
	g.Go(func() error {
		return runSink(ctx, sinkCh, sink)
	})

	return inputTopic, nil
}

// runStep processes messages through a single transform step.
func runStep(ctx context.Context, stepIdx int, step *transformStep, onState stateCallback, in <-chan *pubsub.Message, pub pubsub.Publisher, outTopic string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-in:
			if !ok {
				return nil
			}
			val := msg.Payload.(*expr.Value)

			if isEOFValue(val) {
				// Emit any accumulated values (reduce) before passing EOF through.
				if err := emitOnEOF(step, pub, outTopic); err != nil {
					msg.Nack()
					return err
				}
				if err := pub.Publish(outTopic, pubsub.NewMessage(val)); err != nil {
					msg.Nack()
					return err
				}
				msg.Ack()
				return nil
			}

			outputs, err := applyStep(step, val)
			if err != nil {
				msg.Nack()
				return err
			}
			// Reduce accumulated silently (no output): notify state callback.
			if step.reduce != nil && len(outputs) == 0 && onState != nil && step.reduce.lastAcc != nil {
				if err := onState(ctx, stepIdx, step.reduce.lastAcc); err != nil {
					msg.Nack()
					return err
				}
			}
			for _, out := range outputs {
				if err := pub.Publish(outTopic, pubsub.NewMessage(out)); err != nil {
					msg.Nack()
					return err
				}
			}
			msg.Ack()
		}
	}
}

// emitOnEOF publishes any final values from stateful steps (reduce accumulators).
func emitOnEOF(step *transformStep, pub pubsub.Publisher, outTopic string) error {
	if step.reduce == nil || !step.reduce.seen {
		return nil
	}
	if step.reduce.groups != nil {
		for _, group := range step.reduce.groups {
			if err := pub.Publish(outTopic, pubsub.NewMessage(group.accumulator)); err != nil {
				return err
			}
		}
	} else {
		if err := pub.Publish(outTopic, pubsub.NewMessage(step.reduce.accumulator)); err != nil {
			return err
		}
	}
	return nil
}

// applyStep applies a single transform step to a value, returning zero or more outputs.
func applyStep(step *transformStep, val *expr.Value) ([]*expr.Value, error) {
	if step.mapProgram != nil {
		rv := exprToRefVal(step.adapter, val)
		result, err := evalCEL(step.mapProgram, map[string]any{
			"this": map[string]any{"value": rv},
		})
		if err != nil {
			return nil, fmt.Errorf("map eval: %w", err)
		}
		out, err := refValToExpr(result)
		if err != nil {
			return nil, fmt.Errorf("map result convert: %w", err)
		}
		return []*expr.Value{out}, nil
	}

	if step.filterProgram != nil {
		rv := exprToRefVal(step.adapter, val)
		result, err := evalCEL(step.filterProgram, map[string]any{
			"this": map[string]any{"value": rv},
		})
		if err != nil {
			return nil, fmt.Errorf("filter eval: %w", err)
		}
		if pass, ok := result.Value().(bool); ok && pass {
			return []*expr.Value{val}, nil
		}
		return nil, nil
	}

	if step.flatten {
		if list := val.GetListValue(); list != nil {
			return list.GetValues(), nil
		}
		return []*expr.Value{val}, nil
	}

	if step.reduce != nil {
		if err := step.reduce.accumulate(val); err != nil {
			return nil, fmt.Errorf("reduce: %w", err)
		}
		// Reduce accumulates silently; output on EOF.
		return nil, nil
	}

	if step.scan != nil {
		if err := step.scan.accumulate(val); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		return []*expr.Value{step.scan.accumulator}, nil
	}

	return []*expr.Value{val}, nil
}

// runSink reads processed values from the pipeline and calls sink for each.
func runSink(ctx context.Context, in <-chan *pubsub.Message, sink pipelineSink) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-in:
			if !ok {
				return nil
			}
			val := msg.Payload.(*expr.Value)
			eof := isEOFValue(val)
			if err := sink(ctx, val, eof); err != nil {
				msg.Nack()
				return err
			}
			msg.Ack()
			if eof {
				return nil
			}
		}
	}
}

// lintTransforms validates all CEL expressions in a transform pipeline without
// producing executable programs. Returns structured diagnostics for each issue.
func lintTransforms(transforms []*flowv1beta2.Transform) []LintDiagnostic {
	var diags []LintDiagnostic
	for i, t := range transforms {
		switch t.WhichType() {
		case flowv1beta2.Transform_Map_case:
			if _, err := parseCEL(t.GetMap()); err != nil {
				diags = append(diags, LintDiagnostic{
					Severity: SeverityError,
					Path:     fmt.Sprintf("transforms[%d].map", i),
					Message:  err.Error(),
					Code:     CodeInvalidCEL,
				})
			}
		case flowv1beta2.Transform_Filter_case:
			if _, err := parseCEL(t.GetFilter()); err != nil {
				diags = append(diags, LintDiagnostic{
					Severity: SeverityError,
					Path:     fmt.Sprintf("transforms[%d].filter", i),
					Message:  err.Error(),
					Code:     CodeInvalidCEL,
				})
			}
		case flowv1beta2.Transform_Flatten_case:
			// no CEL to validate
		case flowv1beta2.Transform_Reduce_case:
			r := t.GetReduce()
			if _, err := parseCEL(r.GetInitial()); err != nil {
				diags = append(diags, LintDiagnostic{
					Severity: SeverityError,
					Path:     fmt.Sprintf("transforms[%d].reduce.initial", i),
					Message:  err.Error(),
					Code:     CodeInvalidCEL,
				})
			}
			if r.GetAccumulator() != "" {
				if _, err := parseCEL(r.GetAccumulator()); err != nil {
					diags = append(diags, LintDiagnostic{
						Severity: SeverityError,
						Path:     fmt.Sprintf("transforms[%d].reduce.accumulator", i),
						Message:  err.Error(),
						Code:     CodeInvalidCEL,
					})
				}
			}
			if gb := r.GetGroupBy(); gb != nil && gb.GetKey() != "" {
				if _, err := parseCEL(gb.GetKey()); err != nil {
					diags = append(diags, LintDiagnostic{
						Severity: SeverityError,
						Path:     fmt.Sprintf("transforms[%d].reduce.group_by.key", i),
						Message:  err.Error(),
						Code:     CodeInvalidCEL,
					})
				}
			}
		case flowv1beta2.Transform_Scan_case:
			s := t.GetScan()
			if _, err := parseCEL(s.GetInitial()); err != nil {
				diags = append(diags, LintDiagnostic{
					Severity: SeverityError,
					Path:     fmt.Sprintf("transforms[%d].scan.initial", i),
					Message:  err.Error(),
					Code:     CodeInvalidCEL,
				})
			}
			if s.GetAccumulator() != "" {
				if _, err := parseCEL(s.GetAccumulator()); err != nil {
					diags = append(diags, LintDiagnostic{
						Severity: SeverityError,
						Path:     fmt.Sprintf("transforms[%d].scan.accumulator", i),
						Message:  err.Error(),
						Code:     CodeInvalidCEL,
					})
				}
			}
		default:
			diags = append(diags, LintDiagnostic{
				Severity: SeverityError,
				Path:     fmt.Sprintf("transforms[%d]", i),
				Message:  "unsupported transform type",
				Code:     CodeSchemaError,
			})
		}
	}
	return diags
}

// compileTransforms compiles a sequence of proto Transform messages into a reusable pipeline.
// Returns nil if there are no transforms.
func compileTransforms(env shared.Env, transforms []*flowv1beta2.Transform) (*transformPipeline, error) {
	if len(transforms) == 0 {
		return nil, nil
	}

	steps := make([]transformStep, 0, len(transforms))
	adapter := env.TypeAdapter()
	for i, t := range transforms {
		var step transformStep
		step.adapter = adapter
		switch t.WhichType() {
		case flowv1beta2.Transform_Map_case:
			prog, err := compileCEL(env, t.GetMap())
			if err != nil {
				return nil, fmt.Errorf("transform[%d] map: %w", i, err)
			}
			step.mapProgram = prog

		case flowv1beta2.Transform_Filter_case:
			prog, err := compileCEL(env, t.GetFilter())
			if err != nil {
				return nil, fmt.Errorf("transform[%d] filter: %w", i, err)
			}
			step.filterProgram = prog

		case flowv1beta2.Transform_Flatten_case:
			step.flatten = t.GetFlatten()

		case flowv1beta2.Transform_Reduce_case:
			r := t.GetReduce()
			state, err := compileAccumulate(env, r.GetInitial(), r.GetAccumulator())
			if err != nil {
				return nil, fmt.Errorf("transform[%d] reduce: %w", i, err)
			}
			if gb := r.GetGroupBy(); gb != nil && gb.GetKey() != "" {
				keyProg, err := compileCEL(env, gb.GetKey())
				if err != nil {
					return nil, fmt.Errorf("transform[%d] reduce group_by key: %w", i, err)
				}
				state.keyProgram = keyProg
				state.initExpr = r.GetInitial()
				state.celEnv = env
				state.groups = make(map[string]*accumulateState)
			}
			step.reduce = state

		case flowv1beta2.Transform_Scan_case:
			s := t.GetScan()
			state, err := compileAccumulate(env, s.GetInitial(), s.GetAccumulator())
			if err != nil {
				return nil, fmt.Errorf("transform[%d] scan: %w", i, err)
			}
			step.scan = state

		default:
			return nil, fmt.Errorf("transform[%d]: unsupported type", i)
		}
		steps = append(steps, step)
	}

	return &transformPipeline{steps: steps}, nil
}

// compileAccumulate compiles initial + accumulator CEL expressions into shared state.
func compileAccumulate(env shared.Env, initial, accumulator string) (*accumulateState, error) {
	initProg, err := compileCEL(env, initial)
	if err != nil {
		return nil, fmt.Errorf("initial: %w", err)
	}
	initResult, err := evalCEL(initProg, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("init eval: %w", err)
	}
	initVal, err := refValToExpr(initResult)
	if err != nil {
		return nil, fmt.Errorf("init convert: %w", err)
	}
	state := &accumulateState{
		accumulator: initVal,
		adapter:     env.TypeAdapter(),
	}
	if accumulator != "" {
		accProg, err := compileCEL(env, accumulator)
		if err != nil {
			return nil, fmt.Errorf("accumulator: %w", err)
		}
		state.accProgram = accProg
	}
	return state, nil
}

// accumulate runs the accumulator expression with the current value and updates state.
func (a *accumulateState) accumulate(value *expr.Value) error {
	// If grouped, route to the per-key accumulator.
	if a.groups != nil {
		rv := exprToRefVal(a.adapter, value)
		keyResult, err := evalCEL(a.keyProgram, map[string]any{
			"this": map[string]any{"value": rv},
		})
		if err != nil {
			return fmt.Errorf("group_by key eval: %w", err)
		}
		key := fmt.Sprintf("%v", keyResult.Value())
		group, ok := a.groups[key]
		if !ok {
			group, err = compileAccumulate(a.celEnv, a.initExpr, "")
			if err != nil {
				return fmt.Errorf("group_by init: %w", err)
			}
			group.accProgram = a.accProgram
			a.groups[key] = group
		}
		if err := group.accumulateValue(value); err != nil {
			return fmt.Errorf("group_by accumulate: %w", err)
		}
		a.lastAcc = group.accumulator // track for STATE notification
		a.seen = true
		return nil
	}
	return a.accumulateValue(value)
}

// accumulateValue runs the accumulator CEL on a single value (no grouping logic).
func (a *accumulateState) accumulateValue(value *expr.Value) error {
	rv := exprToRefVal(a.adapter, value)
	accRV := exprToRefVal(a.adapter, a.accumulator)
	result, err := evalCEL(a.accProgram, map[string]any{
		"this": map[string]any{
			"value":       rv,
			"accumulator": accRV,
		},
	})
	if err != nil {
		return fmt.Errorf("eval: %w", err)
	}
	newAcc, err := refValToExpr(result)
	if err != nil {
		return fmt.Errorf("result convert: %w", err)
	}
	a.accumulator = newAcc
	a.lastAcc = newAcc
	a.seen = true
	return nil
}
