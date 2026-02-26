package prometheus

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/prometheus/promql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachedQueryFunc(t *testing.T) {
	ts := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	sample := promql.Sample{T: ts.UnixMilli(), F: 1.0}
	cache := map[int64]promql.Vector{
		ts.UnixMilli(): {sample},
	}

	qf := CachedQueryFunc(cache)

	t.Run("returns cached vector for known timestamp", func(t *testing.T) {
		got, err := qf(context.Background(), "ignored", ts)
		require.NoError(t, err)
		assert.Equal(t, promql.Vector{sample}, got)
	})

	t.Run("returns nil for unknown timestamp", func(t *testing.T) {
		got, err := qf(context.Background(), "ignored", ts.Add(time.Hour))
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}
