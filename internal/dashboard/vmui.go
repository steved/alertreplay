package dashboard

import (
	"fmt"
	"net/url"
	"time"
)

type vmui struct {
	base *url.URL
}

func newVMUI(baseURL string) (*vmui, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing vmui base URL: %w", err)
	}

	return &vmui{base: u}, nil
}

func (v *vmui) BuildURL(expr string, from time.Time, to *time.Time) string {
	paddedFrom, paddedTo := padRange(from, to)

	u := *v.base
	q := u.Query()
	q.Set("g0.expr", expr)
	q.Set("g0.range_input", formatDuration(paddedTo.Sub(paddedFrom)))
	q.Set("g0.end_input", paddedTo.Format("2006-01-02T15:04:05"))
	u.RawQuery = q.Encode()

	return u.String()
}
