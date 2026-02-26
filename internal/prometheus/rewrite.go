package prometheus

import (
	"github.com/VictoriaMetrics/metricsql"
)

func RewriteExpr(expr string, filters ...metricsql.LabelFilter) (string, error) {
	if len(filters) == 0 {
		return expr, nil
	}

	parsed, err := metricsql.Parse(expr)
	if err != nil {
		return "", err
	}

	metricsql.VisitAll(parsed, func(e metricsql.Expr) {
		metric, ok := e.(*metricsql.MetricExpr)
		if !ok {
			return
		}

		if len(metric.LabelFilterss) == 0 {
			metric.LabelFilterss = [][]metricsql.LabelFilter{{}}
		}
		for i := range metric.LabelFilterss {
			metric.LabelFilterss[i] = append(metric.LabelFilterss[i], filters...)
		}
	})

	buf := parsed.AppendString(nil)
	return string(buf), nil
}
