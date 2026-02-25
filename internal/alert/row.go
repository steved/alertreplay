package alert

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

const grafanaBaseURL = "https://grafana.groq.io/explore"

// Row represents a single alert instance with its firing period.
type Row struct {
	OpenedAt   time.Time
	ResolvedAt *time.Time
	Labels     map[string]string
	GrafanaURL string
	Source     string
}

// FormatLabels converts a label map to a string representation.
func FormatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "{}"
	}

	keys := make([]string, 0, len(labels))
	for k := range labels {
		if k == "__name__" || k == "alertname" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", k, labels[k]))
	}

	return "{" + strings.Join(parts, ", ") + "}"
}

// BuildGrafanaURL constructs a Grafana explore URL for investigating an alert.
func BuildGrafanaURL(expr string, from time.Time, to *time.Time) string {
	fromMs := from.Add(-5 * time.Minute).UnixMilli()
	var toMs int64
	if to != nil {
		toMs = to.Add(5 * time.Minute).UnixMilli()
	} else {
		toMs = from.Add(1 * time.Hour).UnixMilli()
	}

	leftPane := fmt.Sprintf(
		`{"datasource":"production","queries":[{"refId":"A","expr":%q}],"range":{"from":"%d","to":"%d"}}`,
		expr,
		fromMs,
		toMs,
	)

	return fmt.Sprintf(
		"%s?orgId=1&left=%s",
		grafanaBaseURL,
		url.QueryEscape(leftPane),
	)
}
