package dashboard

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

type grafana struct {
	base *url.URL
}

func newGrafana(baseURL string) (*grafana, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing grafana base URL: %w", err)
	}

	return &grafana{base: u}, nil
}

type grafanaLeft struct {
	Queries []grafanaQuery `json:"queries"`
	Range   grafanaRange   `json:"range"`
}

type grafanaQuery struct {
	RefID string `json:"refId"`
	Expr  string `json:"expr"`
}

type grafanaRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (g *grafana) BuildURL(expr string, from time.Time, to *time.Time) string {
	paddedFrom, paddedTo := padRange(from, to)

	left := grafanaLeft{
		Queries: []grafanaQuery{{RefID: "A", Expr: expr}},
		Range: grafanaRange{
			From: fmt.Sprintf("%d", paddedFrom.UnixMilli()),
			To:   fmt.Sprintf("%d", paddedTo.UnixMilli()),
		},
	}

	leftJSON, _ := json.Marshal(left)

	u := *g.base
	q := u.Query()
	q.Set("left", string(leftJSON))
	u.RawQuery = q.Encode()

	return u.String()
}
