package prometheus

import (
	"context"
	"time"

	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/rules"
)

func CachedQueryFunc(cache map[int64]promql.Vector) rules.QueryFunc {
	return func(_ context.Context, _ string, t time.Time) (promql.Vector, error) {
		return cache[t.UnixMilli()], nil
	}
}
