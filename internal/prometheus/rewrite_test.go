package prometheus

import (
	"testing"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteExpr(t *testing.T) {
	for _, tt := range []struct {
		name     string
		expr     string
		filters  []metricsql.LabelFilter
		expected string
	}{
		{
			name:     "basic rewrite",
			expr:     `sum(rate(http_requests_total{job="api"}[5m]))`,
			filters:  []metricsql.LabelFilter{{Label: "cluster", Value: "region1"}},
			expected: `sum(rate(http_requests_total{job="api",cluster="region1"}[5m]))`,
		},
		{
			name:     "complex rewrite",
			expr:     `sum(rate(http_requests_total{job="api"}[5m])) and sum(rate(http_requests_total{job="server"}[5m])) or sum(rate(http_requests_total{job="db"}[5m]))`,
			filters:  []metricsql.LabelFilter{{Label: "cluster", Value: "region1"}},
			expected: `(sum(rate(http_requests_total{job="api",cluster="region1"}[5m])) and sum(rate(http_requests_total{job="server",cluster="region1"}[5m]))) or sum(rate(http_requests_total{job="db",cluster="region1"}[5m]))`,
		},
		{
			name:     "bare metric",
			expr:     "http_requests_total",
			filters:  []metricsql.LabelFilter{{Label: "cluster", Value: "region1"}},
			expected: `http_requests_total{cluster="region1"}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rewritten, err := RewriteExpr(tt.expr, tt.filters...)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, rewritten)
		})
	}
}
