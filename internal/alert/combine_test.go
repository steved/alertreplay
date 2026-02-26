package alert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/steved/alertreplay/internal/evaluator"
)

func TestCombineEvents(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	for _, tt := range []struct {
		name   string
		events []evaluator.Event
		want   []Alert
	}{
		{
			name:   "no events returns nil",
			events: nil,
			want:   nil,
		},
		{
			name: "open then resolve produces one alert",
			events: []evaluator.Event{
				{Time: base, Labels: map[string]string{"job": "api"}, Type: evaluator.EventOpened},
				{Time: base.Add(5 * time.Minute), Labels: map[string]string{"job": "api"}, Type: evaluator.EventResolved},
			},
			want: []Alert{
				{
					OpenedAt:   base,
					ResolvedAt: new(base.Add(5 * time.Minute)),
					Labels:     map[string]string{"job": "api"},
				},
			},
		},
		{
			name: "unresolved alert returned without ResolvedAt",
			events: []evaluator.Event{
				{Time: base, Labels: map[string]string{"job": "api"}, Type: evaluator.EventOpened},
			},
			want: []Alert{
				{
					OpenedAt: base,
					Labels:   map[string]string{"job": "api"},
				},
			},
		},
		{
			name: "resolve without open is ignored",
			events: []evaluator.Event{
				{Time: base, Labels: map[string]string{"job": "api"}, Type: evaluator.EventResolved},
			},
			want: nil,
		},
		{
			name: "multiple label sets tracked independently",
			events: []evaluator.Event{
				{Time: base, Labels: map[string]string{"job": "api"}, Type: evaluator.EventOpened},
				{Time: base, Labels: map[string]string{"job": "server"}, Type: evaluator.EventOpened},
				{Time: base.Add(5 * time.Minute), Labels: map[string]string{"job": "api"}, Type: evaluator.EventResolved},
			},
			want: []Alert{
				{
					OpenedAt:   base,
					ResolvedAt: new(base.Add(5 * time.Minute)),
					Labels:     map[string]string{"job": "api"},
				},
				{
					OpenedAt: base,
					Labels:   map[string]string{"job": "server"},
				},
			},
		},
		{
			name: "results are sorted by OpenedAt",
			events: []evaluator.Event{
				{Time: base.Add(10 * time.Minute), Labels: map[string]string{"job": "late"}, Type: evaluator.EventOpened},
				{Time: base, Labels: map[string]string{"job": "early"}, Type: evaluator.EventOpened},
				{Time: base.Add(15 * time.Minute), Labels: map[string]string{"job": "late"}, Type: evaluator.EventResolved},
				{Time: base.Add(5 * time.Minute), Labels: map[string]string{"job": "early"}, Type: evaluator.EventResolved},
			},
			want: []Alert{
				{
					OpenedAt:   base,
					ResolvedAt: new(base.Add(5 * time.Minute)),
					Labels:     map[string]string{"job": "early"},
				},
				{
					OpenedAt:   base.Add(10 * time.Minute),
					ResolvedAt: new(base.Add(15 * time.Minute)),
					Labels:     map[string]string{"job": "late"},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := CombineEvents(tt.events, "test_expr", nil)
			assert.Equal(t, tt.want, got)
		})
	}
}
