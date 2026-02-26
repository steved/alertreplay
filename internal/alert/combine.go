package alert

import (
	"github.com/steved/alertreplay/internal/dashboard"
	"github.com/steved/alertreplay/internal/evaluator"
)

func CombineEvents(events []evaluator.Event, expr string, urlBuilder dashboard.URLBuilder) (result []Alert) {
	open := make(map[string]*Alert)

	for _, event := range events {
		key := FormatLabels(event.Labels)

		switch event.Type {
		case evaluator.EventOpened:
			open[key] = &Alert{
				OpenedAt: event.Time,
				Labels:   event.Labels,
			}
		case evaluator.EventResolved:
			if alert, ok := open[key]; ok {
				alert.ResolvedAt = new(event.Time)
				result = append(result, *alert)
				delete(open, key)
			}
		}
	}

	for _, alert := range open {
		result = append(result, *alert)
	}

	Sort(result)

	if urlBuilder != nil {
		for i := range result {
			alert := &result[i]
			alert.URL = urlBuilder.BuildURL(expr, alert.OpenedAt, alert.ResolvedAt)
		}
	}

	return result
}
