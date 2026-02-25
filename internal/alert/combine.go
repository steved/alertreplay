package alert

import (
	"sort"

	"github.com/steved/alertreplay/internal/evaluator"
)

// CombineEvents converts a list of alert events into consolidated alert rows.
func CombineEvents(events []evaluator.Event, expr string) []Row {
	open := make(map[string]*Row)
	var result []Row

	for _, event := range events {
		key := FormatLabels(event.Labels)

		switch event.Type {
		case evaluator.EventOpened:
			ar := &Row{
				OpenedAt:   event.Time,
				Labels:     event.Labels,
				GrafanaURL: BuildGrafanaURL(expr, event.Time, nil),
			}
			open[key] = ar

		case evaluator.EventResolved:
			if ar, ok := open[key]; ok {
				resolvedAt := event.Time
				ar.ResolvedAt = &resolvedAt
				ar.GrafanaURL = BuildGrafanaURL(expr, ar.OpenedAt, &resolvedAt)
				result = append(result, *ar)
				delete(open, key)
			}
		}
	}

	for _, ar := range open {
		result = append(result, *ar)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].OpenedAt.Before(result[j].OpenedAt)
	})

	return result
}
