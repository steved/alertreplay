package prometheus

import (
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/promql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTimestamps(t *testing.T) {
	for _, tt := range []struct {
		name     string
		from, to time.Time
		interval time.Duration
		want     []time.Time
	}{
		{
			name:     "single timestamp when from equals to",
			from:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			to:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			interval: time.Minute,
			want:     []time.Time{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		{
			name:     "multiple timestamps",
			from:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			to:       time.Date(2026, 1, 1, 0, 2, 0, 0, time.UTC),
			interval: time.Minute,
			want: []time.Time{
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
				time.Date(2026, 1, 1, 0, 2, 0, 0, time.UTC),
			},
		},
		{
			name:     "to not aligned to interval",
			from:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			to:       time.Date(2026, 1, 1, 0, 1, 30, 0, time.UTC),
			interval: time.Minute,
			want: []time.Time{
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := generateTimestamps(tt.from, tt.to, tt.interval)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAlignToStep(t *testing.T) {
	for _, tt := range []struct {
		name string
		t    time.Time
		step time.Duration
		want time.Time
	}{
		{
			name: "already aligned",
			t:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			step: time.Minute,
			want: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "truncates to previous step",
			t:    time.Date(2026, 1, 1, 0, 0, 45, 0, time.UTC),
			step: time.Minute,
			want: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "30s step alignment",
			t:    time.Date(2026, 1, 1, 0, 1, 15, 0, time.UTC),
			step: 30 * time.Second,
			want: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := alignToStep(tt.t, tt.step)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProcessMatrix(t *testing.T) {
	client := &APIClient{}
	ts := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	for _, tt := range []struct {
		name              string
		matrix            model.Matrix
		expectedTimestamp time.Time
		wantLen           int
		wantFirst         *promql.Sample
	}{
		{
			name:              "empty matrix",
			matrix:            model.Matrix{},
			expectedTimestamp: ts,
			wantLen:           0,
		},
		{
			name: "matching timestamp extracted",
			matrix: model.Matrix{
				&model.SampleStream{
					Metric: model.Metric{"job": "api"},
					Values: []model.SamplePair{
						{Timestamp: model.TimeFromUnixNano(ts.UnixNano()), Value: 42},
					},
				},
			},
			expectedTimestamp: ts,
			wantLen:           1,
			wantFirst: &promql.Sample{
				T: ts.UnixMilli(),
				F: 42,
			},
		},
		{
			name: "non-matching timestamp filtered",
			matrix: model.Matrix{
				&model.SampleStream{
					Metric: model.Metric{"job": "api"},
					Values: []model.SamplePair{
						{Timestamp: model.TimeFromUnixNano(ts.Add(time.Hour).UnixNano()), Value: 42},
					},
				},
			},
			expectedTimestamp: ts,
			wantLen:           0,
		},
		{
			name: "multiple streams multiple samples",
			matrix: model.Matrix{
				&model.SampleStream{
					Metric: model.Metric{"job": "api"},
					Values: []model.SamplePair{
						{Timestamp: model.TimeFromUnixNano(ts.UnixNano()), Value: 1},
						{Timestamp: model.TimeFromUnixNano(ts.Add(time.Minute).UnixNano()), Value: 2},
					},
				},
				&model.SampleStream{
					Metric: model.Metric{"job": "server"},
					Values: []model.SamplePair{
						{Timestamp: model.TimeFromUnixNano(ts.UnixNano()), Value: 3},
					},
				},
			},
			expectedTimestamp: ts,
			wantLen:           2,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := client.processMatrix(tt.matrix, tt.expectedTimestamp)
			require.Len(t, got, tt.wantLen)
			if tt.wantFirst != nil {
				assert.Equal(t, tt.wantFirst.T, got[0].T)
				assert.Equal(t, tt.wantFirst.F, got[0].F)
			}
		})
	}
}
