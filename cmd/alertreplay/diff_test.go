package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/steved/alertreplay/internal/alert"
)

func TestFindMatchingAlert(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	for _, tt := range []struct {
		name   string
		target alert.Alert
		alerts []alert.Alert
		used   map[int]bool
		want   int
	}{
		{
			name:   "finds matching alert",
			target: alert.Alert{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			alerts: []alert.Alert{
				{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			},
			used: map[int]bool{},
			want: 0,
		},
		{
			name:   "skips already used alerts",
			target: alert.Alert{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			alerts: []alert.Alert{
				{OpenedAt: base, Labels: map[string]string{"job": "api"}},
				{OpenedAt: base.Add(time.Minute), Labels: map[string]string{"job": "api"}},
			},
			used: map[int]bool{0: true},
			want: 1,
		},
		{
			name:   "returns -1 when no match",
			target: alert.Alert{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			alerts: []alert.Alert{
				{OpenedAt: base, Labels: map[string]string{"job": "server"}},
			},
			used: map[int]bool{},
			want: -1,
		},
		{
			name:   "returns -1 for empty list",
			target: alert.Alert{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			alerts: nil,
			used:   map[int]bool{},
			want:   -1,
		},
		{
			name:   "returns -1 when all are used",
			target: alert.Alert{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			alerts: []alert.Alert{
				{OpenedAt: base, Labels: map[string]string{"job": "api"}},
			},
			used: map[int]bool{0: true},
			want: -1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := findMatchingAlert(tt.target, tt.alerts, tt.used)
			assert.Equal(t, tt.want, got)
		})
	}
}
