package evaluator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/prometheus/prometheus/rules"
	zlog "github.com/rs/zerolog/log"
)

// EventType represents the type of alert event.
type EventType int

const (
	// EventOpened indicates an alert has started firing.
	EventOpened EventType = iota
	// EventResolved indicates an alert has stopped firing.
	EventResolved
)

// Event represents a single alert state change.
type Event struct {
	Time   time.Time
	Labels map[string]string
	Type   EventType
}

// Evaluator evaluates alert rules against pre-fetched data.
type Evaluator struct {
	rule *rules.AlertingRule
}

// New creates a new rule evaluator.
func New(name string, expr string, forDuration time.Duration) (*Evaluator, error) {
	parsedExpr, err := parser.ParseExpr(expr)
	if err != nil {
		return nil, fmt.Errorf("parsing expression: %w", err)
	}

	rule := rules.NewAlertingRule(
		name,
		parsedExpr,
		forDuration,
		0, // keepFiringFor
		labels.EmptyLabels(),
		labels.EmptyLabels(),
		labels.EmptyLabels(),
		"",
		true, // restored - treat as if state was restored to avoid initial pending
		slog.Default(),
	)

	return &Evaluator{
		rule: rule,
	}, nil
}

type firingAlert struct {
	labels  map[string]string
	firedAt time.Time
}

// Evaluate runs the alert rule against the provided timestamps using queryFn.
func (e *Evaluator) Evaluate(
	ctx context.Context,
	queryFn rules.QueryFunc,
	timestamps []time.Time,
) ([]Event, error) {
	var events []Event
	firing := make(map[string]*firingAlert)

	for _, ts := range timestamps {
		_, err := e.rule.Eval(ctx, 0, ts, queryFn, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("evaluating at %s: %w", ts.Format(time.RFC3339), err)
		}

		currentlyFiring := make(map[string]struct{})

		e.rule.ForEachActiveAlert(func(alert *rules.Alert) {
			if alert.State != rules.StateFiring {
				return
			}

			key := alert.Labels.String()
			currentlyFiring[key] = struct{}{}

			if _, ok := firing[key]; ok {
				return
			}

			lbls := make(map[string]string)
			alert.Labels.Range(func(l labels.Label) {
				lbls[l.Name] = l.Value
			})

			firing[key] = &firingAlert{
				labels:  lbls,
				firedAt: alert.FiredAt,
			}

			events = append(events, Event{
				Time:   alert.FiredAt,
				Labels: lbls,
				Type:   EventOpened,
			})

			zlog.Debug().
				Time("firedAt", alert.FiredAt).
				Interface("labels", lbls).
				Msg("alert opened")
		})

		for key, alert := range firing {
			if _, stillFiring := currentlyFiring[key]; !stillFiring {
				events = append(events, Event{
					Time:   ts,
					Labels: alert.labels,
					Type:   EventResolved,
				})

				zlog.Debug().
					Time("resolvedAt", ts).
					Interface("labels", alert.labels).
					Msg("alert resolved")

				delete(firing, key)
			}
		}
	}

	return events, nil
}

// BuildQueryFunc creates a rules.QueryFunc that retrieves data from a pre-populated cache.
func BuildQueryFunc(cache map[int64]promql.Vector) rules.QueryFunc {
	return func(_ context.Context, _ string, t time.Time) (promql.Vector, error) {
		vec, ok := cache[t.UnixMilli()]
		if !ok {
			return nil, nil
		}

		return vec, nil
	}
}
