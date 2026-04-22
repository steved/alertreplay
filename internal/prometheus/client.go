package prometheus

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql"
	zlog "github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

const (
	defaultQueryTimeout = 2 * time.Minute
)

type Client interface {
	LabelValues(context.Context, string, time.Time, time.Time) ([]metricsql.LabelFilter, error)
	QueryExpr(context.Context, string, time.Time, time.Time, time.Duration) (map[int64]promql.Vector, []time.Time, error)
}

type APIClient struct {
	api          v1.API
	parallelism  int
	queryTimeout time.Duration
}

func NewAPIClient(prometheusURL string, parallelism int) (*APIClient, error) {
	client, err := api.NewClient(api.Config{Address: prometheusURL})
	if err != nil {
		return nil, fmt.Errorf("creating Prometheus client: %w", err)
	}

	return &APIClient{
		api:          v1.NewAPI(client),
		parallelism:  parallelism,
		queryTimeout: defaultQueryTimeout,
	}, nil
}

func (a *APIClient) LabelValues(ctx context.Context, label string, from, to time.Time) ([]metricsql.LabelFilter, error) {
	labelValues, warnings, err := a.api.LabelValues(ctx, label, nil, from, to)
	if err != nil {
		return nil, err
	}

	for _, w := range warnings {
		zlog.Warn().Str("warning", w).Msg("label values discovery warning")
	}

	if len(labelValues) == 0 {
		return nil, fmt.Errorf("no values found for label %q", label)
	}

	slices.Sort(labelValues)

	values := make([]metricsql.LabelFilter, 0, len(labelValues))
	for _, labelValue := range labelValues {
		values = append(values, metricsql.LabelFilter{Label: label, Value: string(labelValue)})
	}

	return values, nil
}

func (a *APIClient) QueryExpr(
	ctx context.Context,
	expr string,
	from time.Time,
	to time.Time,
	interval time.Duration,
) (map[int64]promql.Vector, []time.Time, error) {
	from = alignToStep(from, interval)
	to = alignToStep(to, interval)

	var (
		timestamps = generateTimestamps(from, to, interval)
		vectors    = make(map[int64]promql.Vector, len(timestamps))
		vectorsMu  sync.Mutex
	)

	var (
		eg           errgroup.Group
		totalWindows = len(timestamps)
	)
	eg.SetLimit(a.parallelism)

	zlog.Debug().Int("windows", totalWindows).Msg("split time range")

	for i, start := range timestamps {
		var (
			windowNumber = i + 1
			windowEnd    = start.Add(interval)
		)

		if windowEnd.After(to) {
			windowEnd = to
		}

		eg.Go(func() error {
			zlog.Debug().
				Str("query", expr).
				Int("window", windowNumber).
				Int("total", totalWindows).
				Time("from", start).
				Time("to", windowEnd).
				Msg("executing query")

			matrix, err := a.queryRange(ctx, expr, start, windowEnd, interval)
			if err != nil {
				return fmt.Errorf("querying window %d/%d: %w", windowNumber, totalWindows, err)
			}

			samples := a.processMatrix(matrix, start)

			vectorsMu.Lock()
			defer vectorsMu.Unlock()

			for _, sample := range samples {
				vectors[sample.T] = append(vectors[sample.T], sample)
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, nil, err
	}

	return vectors, timestamps, nil
}

func (a *APIClient) queryRange(
	ctx context.Context,
	expr string,
	from, to time.Time,
	interval time.Duration,
) (model.Matrix, error) {
	ctx, cancel := context.WithTimeout(ctx, a.queryTimeout)
	defer cancel()

	result, warnings, err := a.api.QueryRange(ctx, expr, v1.Range{
		Start: from,
		End:   to,
		Step:  interval,
	})
	if err != nil {
		return nil, err
	}

	for _, w := range warnings {
		zlog.Warn().Str("warning", w).Msg("query warning")
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	return matrix, nil
}

func (a *APIClient) processMatrix(matrix model.Matrix, expectedTimestamp time.Time) []promql.Sample {
	var samples []promql.Sample

	for _, stream := range matrix {
		lb := labels.NewBuilder(labels.EmptyLabels())
		for k, v := range stream.Metric {
			lb.Set(string(k), string(v))
		}
		metricLabels := lb.Labels()

		for _, sample := range stream.Values {
			ts := sample.Timestamp.Time().UnixMilli()
			if expectedTimestamp.UnixMilli() != ts {
				continue
			}

			sample := promql.Sample{
				T:      ts,
				F:      float64(sample.Value),
				Metric: metricLabels,
			}

			samples = append(samples, sample)
		}
	}

	return samples
}

func generateTimestamps(from time.Time, to time.Time, interval time.Duration) []time.Time {
	n := int((to.Sub(from) / interval) + 1)
	timestamps := make([]time.Time, 0, n)
	for t := from; !t.After(to); t = t.Add(interval) {
		timestamps = append(timestamps, t)
	}

	return timestamps
}

func alignToStep(t time.Time, step time.Duration) time.Time {
	return time.UnixMilli((t.UnixMilli() / int64(step/time.Millisecond)) * int64(step/time.Millisecond)).UTC()
}
