package alert

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/prometheus/model/rulefmt"

	"github.com/steved/alertreplay/internal/dashboard"
	"github.com/steved/alertreplay/internal/evaluator"
	"github.com/steved/alertreplay/internal/prometheus"
)

func Evaluate(
	ctx context.Context,
	client prometheus.Client,
	rule rulefmt.Rule,
	from time.Time,
	to time.Time,
	interval time.Duration,
	urlBuilder dashboard.URLBuilder,
) ([]Alert, error) {
	vectors, timestamps, err := client.QueryExpr(ctx, rule.Expr, from, to, interval)
	if err != nil {
		return nil, fmt.Errorf("executing queries: %w", err)
	}

	forDuration := time.Duration(rule.For)
	eval, err := evaluator.New(rule.Alert, rule.Expr, forDuration)
	if err != nil {
		return nil, fmt.Errorf("creating rule evaluator: %w", err)
	}

	events, err := eval.Evaluate(ctx, prometheus.CachedQueryFunc(vectors), timestamps)
	if err != nil {
		return nil, fmt.Errorf("evaluating rule: %w", err)
	}

	return CombineEvents(events, rule.Expr, urlBuilder), nil
}
