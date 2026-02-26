package alert

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/prometheus/prometheus/model/rulefmt"

	"github.com/steved/alertreplay/internal/evaluator"
	"github.com/steved/alertreplay/internal/prometheus"
)

// Run executes an alert rule against historical data and returns the resulting alerts.
func Run(
	ctx context.Context,
	exec *prometheus.Executor,
	rule *rulefmt.Rule,
	from time.Time,
	to time.Time,
	interval time.Duration,
) ([]Row, error) {
	vectors, timestamps, err := exec.Execute(ctx, rule.Expr, from, to, interval)
	if err != nil {
		return nil, fmt.Errorf("executing queries: %w", err)
	}

	forDuration := time.Duration(rule.For)
	eval, err := evaluator.New(rule.Alert, rule.Expr, forDuration)
	if err != nil {
		return nil, fmt.Errorf("creating rule evaluator: %w", err)
	}

	queryFn := evaluator.BuildQueryFunc(vectors)
	events, err := eval.Evaluate(ctx, queryFn, timestamps)
	if err != nil {
		return nil, fmt.Errorf("evaluating rule: %w", err)
	}

	return CombineEvents(events, rule.Expr), nil
}

// AddClusterLabel adds a cluster label to alerts that don't have one.
func AddLabel(alerts []Row, label metricsql.LabelFilter) []Row {
	if label.Label == "" {
		return alerts
	}

	for i := range alerts {
		if alerts[i].Labels == nil {
			alerts[i].Labels = map[string]string{label.Label: label.Value}
			continue
		}

		if _, ok := alerts[i].Labels[label.Label]; !ok {
			alerts[i].Labels[label.Label] = label.Value
		}
	}

	return alerts
}

// SortRows sorts alert rows by their opened time.
func SortRows(alerts []Row) {
	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].OpenedAt.Before(alerts[j].OpenedAt)
	})
}
