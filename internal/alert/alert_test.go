package alert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	for _, tt := range []struct {
		name string
		a, b Alert
		want bool
	}{
		{
			name: "identical alerts match",
			a:    Alert{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			b:    Alert{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			want: true,
		},
		{
			name: "within threshold matches",
			a:    Alert{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			b:    Alert{OpenedAt: base.Add(1 * time.Minute), Labels: map[string]string{"job": "api"}},
			want: true,
		},
		{
			name: "at threshold boundary matches",
			a:    Alert{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			b:    Alert{OpenedAt: base.Add(2 * time.Minute), Labels: map[string]string{"job": "api"}},
			want: true,
		},
		{
			name: "beyond threshold does not match",
			a:    Alert{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			b:    Alert{OpenedAt: base.Add(2*time.Minute + time.Second), Labels: map[string]string{"job": "api"}},
			want: false,
		},
		{
			name: "different labels do not match",
			a:    Alert{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			b:    Alert{OpenedAt: base, Labels: map[string]string{"job": "server"}},
			want: false,
		},
		{
			name: "nil vs empty labels do not match",
			a:    Alert{OpenedAt: base, Labels: nil},
			b:    Alert{OpenedAt: base, Labels: map[string]string{}},
			want: false,
		},
		{
			name: "both nil labels match",
			a:    Alert{OpenedAt: base, Labels: nil},
			b:    Alert{OpenedAt: base, Labels: nil},
			want: true,
		},
		{
			name: "reversed time difference within threshold",
			a:    Alert{OpenedAt: base.Add(90 * time.Second), Labels: map[string]string{"job": "api"}},
			b:    Alert{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			want: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.a.Match(tt.b))
		})
	}
}

func TestSort(t *testing.T) {
	var alerts []Alert
	Sort(alerts)
	assert.Empty(t, alerts)

	t1 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC)

	alerts = []Alert{
		{OpenedAt: t3},
		{OpenedAt: t1},
		{OpenedAt: t2},
	}

	Sort(alerts)

	assert.Equal(t, t1, alerts[0].OpenedAt)
	assert.Equal(t, t2, alerts[1].OpenedAt)
	assert.Equal(t, t3, alerts[2].OpenedAt)
}

func TestFormatLabels(t *testing.T) {
	for _, tt := range []struct {
		name   string
		labels map[string]string
		want   string
	}{
		{
			name:   "empty labels",
			labels: map[string]string{},
			want:   "{}",
		},
		{
			name:   "single label",
			labels: map[string]string{"job": "api"},
			want:   `{job="api"}`,
		},
		{
			name:   "multiple labels sorted",
			labels: map[string]string{"zone": "us-east", "app": "web"},
			want:   `{app="web", zone="us-east"}`,
		},
		{
			name:   "excludes __name__",
			labels: map[string]string{"__name__": "metric", "job": "api"},
			want:   `{job="api"}`,
		},
		{
			name:   "excludes alertname",
			labels: map[string]string{"alertname": "HighLatency", "job": "api"},
			want:   `{job="api"}`,
		},
		{
			name:   "excludes both __name__ and alertname",
			labels: map[string]string{"__name__": "metric", "alertname": "HighLatency"},
			want:   "{}",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatLabels(tt.labels))
		})
	}
}
