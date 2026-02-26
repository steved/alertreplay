package alert

import (
	"github.com/steved/alertreplay/internal/evaluator"
)

func CombineEvents(events []evaluator.Event, expr string) []Alert {
	var (
		open   = make(map[string]*Alert)
		result []Alert
	)

	for _, event := range events {
		key := FormatLabels(event.Labels)

		switch event.Type {
		case evaluator.EventOpened:
			open[key] = &Alert{
				OpenedAt: event.Time,
				Labels:   event.Labels,
				// GrafanaURL: BuildGrafanaURL(expr, event.Time, nil),
			}

		case evaluator.EventResolved:
			if ar, ok := open[key]; ok {
				ar.ResolvedAt = new(event.Time)
				// ar.GrafanaURL = BuildGrafanaURL(expr, ar.OpenedAt, &resolvedAt)
				result = append(result, *ar)
				delete(open, key)
			}
		}
	}

	for _, ar := range open {
		result = append(result, *ar)
	}

	Sort(result)

	return result
}
