package dashboard

import (
	"fmt"
	"net/url"
	"time"
)

type prometheus struct {
	base *url.URL
}

func newPrometheus(baseURL string) (*prometheus, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing prometheus base URL: %w", err)
	}

	return &prometheus{base: u}, nil
}

func (p *prometheus) BuildURL(expr string, from time.Time, to *time.Time) string {
	paddedFrom, paddedTo := padRange(from, to)

	u := *p.base
	q := u.Query()
	q.Set("g0.expr", expr)
	q.Set("g0.tab", "0")
	q.Set("g0.range_input", formatDuration(paddedTo.Sub(paddedFrom)))
	q.Set("g0.end_input", paddedTo.Format("2006-01-02T15:04:05"))
	q.Set("g0.moment_input", paddedTo.Format("2006-01-02T15:04:05"))
	u.RawQuery = q.Encode()

	return u.String()
}
