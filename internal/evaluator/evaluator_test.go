package evaluator

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	for _, tt := range []struct {
		name        string
		alertName   string
		expr        string
		forDuration time.Duration
		wantErr     bool
	}{
		{
			name:        "valid expression",
			alertName:   "HighLatency",
			expr:        `sum(rate(http_requests_total[5m])) > 100`,
			forDuration: 5 * time.Minute,
		},
		{
			name:      "invalid expression",
			alertName: "Bad",
			expr:      `sum(rate(http_requests_total[5m]))>>>>>`,
			wantErr:   true,
		},
		{
			name:        "zero for duration",
			alertName:   "Instant",
			expr:        `up == 0`,
			forDuration: 0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			eval, err := New(tt.alertName, tt.expr, tt.forDuration)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, eval)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, eval)
			}
		})
	}
}

func cachedQueryFunc(cache map[int64]promql.Vector) rules.QueryFunc {
	return func(_ context.Context, _ string, t time.Time) (promql.Vector, error) {
		return cache[t.UnixMilli()], nil
	}
}

func TestEvaluate_noData(t *testing.T) {
	eval, err := New("TestAlert", `up == 0`, 0)
	require.NoError(t, err)

	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	timestamps := []time.Time{base, base.Add(30 * time.Second), base.Add(60 * time.Second)}

	events, err := eval.Evaluate(context.Background(), cachedQueryFunc(nil), timestamps)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestEvaluate_firingThenResolved(t *testing.T) {
	eval, err := New("TestAlert", `up == 0`, 0)
	require.NoError(t, err)

	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	step := 30 * time.Second

	metric := labels.FromStrings("__name__", "up", "alertname", "TestAlert", "job", "node")

	// The query function returns the pre-evaluated result of the expression.
	// A non-empty vector means the alert condition is met; empty/nil means it's not.
	cache := map[int64]promql.Vector{
		base.UnixMilli():           {{T: base.UnixMilli(), F: 0, Metric: metric}},
		base.Add(step).UnixMilli(): {{T: base.Add(step).UnixMilli(), F: 0, Metric: metric}},
		// timestamps 2*step and 3*step have no entries → nil vector → alert resolves
	}

	timestamps := []time.Time{
		base,
		base.Add(step),
		base.Add(2 * step),
		base.Add(3 * step),
	}

	events, err := eval.Evaluate(context.Background(), cachedQueryFunc(cache), timestamps)
	require.NoError(t, err)

	// Should have an open and a resolved event
	require.Len(t, events, 2)
	assert.Equal(t, EventOpened, events[0].Type)
	assert.Equal(t, EventResolved, events[1].Type)
	assert.Equal(t, base.Add(2*step), events[1].Time) // resolved when data disappeared
}

func TestEvaluate_unresolvedAtEnd(t *testing.T) {
	eval, err := New("TestAlert", `up == 0`, 0)
	require.NoError(t, err)

	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	step := 30 * time.Second

	metric := labels.FromStrings("__name__", "up", "alertname", "TestAlert", "job", "node")

	cache := map[int64]promql.Vector{
		base.UnixMilli():           {{T: base.UnixMilli(), F: 0, Metric: metric}},
		base.Add(step).UnixMilli(): {{T: base.Add(step).UnixMilli(), F: 0, Metric: metric}},
	}

	timestamps := []time.Time{base, base.Add(step)}

	events, err := eval.Evaluate(context.Background(), cachedQueryFunc(cache), timestamps)
	require.NoError(t, err)

	// Should have only an open event (still firing at end)
	require.Len(t, events, 1)
	assert.Equal(t, EventOpened, events[0].Type)
}

func TestEvaluate_emptyTimestamps(t *testing.T) {
	eval, err := New("TestAlert", `up == 0`, 0)
	require.NoError(t, err)

	events, err := eval.Evaluate(context.Background(), cachedQueryFunc(nil), nil)
	require.NoError(t, err)
	assert.Empty(t, events)
}
