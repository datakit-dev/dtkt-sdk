package impl

import (
	"context"
	"fmt"
	"time"

	"cel.dev/cel-go/cel"
	celast "cel.dev/cel-go/common/ast"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flow/shared"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"golang.org/x/sync/errgroup"
)

type (
	Metrics struct {
		groups MetricGroups
		values util.SyncMap[string, any]
	}
	Metric struct {
		id       string
		interval time.Duration
		program  shared.Program
		sources  []string
	}
	MetricGroups [][]*Metric
)

func NewMetrics(env shared.Env, nodes map[string]*flowv1beta1.Metric) (*Metrics, error) {
	var metrics []*Metric
	for id, node := range nodes {
		metric, err := NewMetric(env, id, node)
		if err != nil {
			return nil, err
		}

		metrics = append(metrics, metric)
	}

	return &Metrics{
		groups: util.SliceGroupsFunc(metrics, func(a, b *Metric) bool {
			return a.interval == b.interval
		}),
	}, nil
}

func NewMetric(env shared.Env, id string, node *flowv1beta1.Metric) (*Metric, error) {
	interval := node.GetInterval().AsDuration()
	if interval <= 0 {
		return nil, fmt.Errorf("metrics.%s invalid interval: %s", id, interval)
	}

	var sources []string
	ast, err := env.ParseExpr("value", node.GetValue(), func(expr *cel.Ast) {
		celast.PreOrderVisit(
			celast.NavigateAST(expr.NativeRep()),
			celast.NewExprVisitor(func(expr celast.Expr) {
				switch expr.Kind() {
				case celast.SelectKind:
					if expr.AsSelect().Operand().Kind() == celast.IdentKind {
						source := expr.AsSelect().Operand().AsIdent()
						if shared.IsNodeIdent(source) {
							// if expr.AsSelect().FieldName() != "" {
							// source += "." + expr.AsSelect().FieldName()
							// }
							sources = append(sources, source)
						}
					}
				}
			}),
		)
	})
	if err != nil {
		return nil, err
	}

	program, err := env.CompileExpr("value", ast)
	if err != nil {
		return nil, fmt.Errorf("metrics.%s: %w", id, err)
	}

	return &Metric{
		id:       id,
		interval: interval,
		program:  program,
		sources:  util.SliceSet(sources),
	}, nil
}

func (m *Metrics) Values() map[string]any {
	return m.values.ToNativeMap()
}

func (m *Metrics) Run(ctx context.Context, env shared.Env, pipe shared.PipeOf[map[string]any, any]) error {
	if len(m.groups) == 0 {
		return nil
	}

	errGroup, ctx := errgroup.WithContext(ctx)
	for idx := range m.groups {
		if interval, ok := m.groups.Interval(idx); ok {
			errGroup.Go(func() error {
				for {
					select {
					case <-ctx.Done():
						return nil
					case <-time.After(interval):
					}

					values, err := m.groups.Eval(ctx, env, idx)
					if err != nil {
						return err
					}

					m.values.StoreNative(values)
				}
			})
		}
	}

	errGroup.Go(func() error {
		return pipe.Run(ctx, func(ctx context.Context, value any) (map[string]any, error) {
			return m.Values(), nil
		})
	})

	return errGroup.Wait()
}

func (g MetricGroups) Interval(idx int) (i time.Duration, ok bool) {
	if idx < 0 || len(g) == 0 || idx > len(g)-1 {
		return
	} else if len(g[idx]) == 0 {
		return
	}
	return g[idx][0].interval, true
}

func (g MetricGroups) Eval(ctx context.Context, env shared.Env, idx int) (map[string]any, error) {
	if idx < 0 || len(g) == 0 || idx > len(g)-1 {
		return nil, fmt.Errorf("metrics group index out of range: %d > %d", idx, len(g)-1)
	} else if len(g[idx]) == 0 {
		return map[string]any{}, nil
	}

	group := g[idx]
	values, err := util.ContextMapParallelErr(ctx, group, func(ctx context.Context, metric *Metric) (any, error) {
		return metric.Eval(ctx, env)
	})
	if err != nil {
		return nil, err
	}

	return util.SliceReduceToMap(values, func(i int, value any) (string, any, bool) {
		return group[i].id, value, true
	}), nil
}

func (m *Metric) Eval(ctx context.Context, env shared.Env) (any, error) {
	// for _, source := range m.sources {
	// 	value, ok := vars.ResolveName(source)
	// 	if ok {
	// 		if refVal, ok := value.(ref.Val); ok {
	// 			value, err := env.Types().ValueToNative(refVal)
	// 			if err != nil {
	// 				return nil, err
	// 			}

	// 			fmt.Printf("%s %v\n", source, value)
	// 		}
	// 	}
	// }

	// value, _, err := m.program.Eval(ctx, vars)
	// if err != nil {
	// 	return nil, err
	// }

	// return env.Types().ValueToNative(value)

	return m.program.Eval(ctx, env), nil
}
