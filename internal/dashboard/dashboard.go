package dashboard

import (
	"fmt"
	"time"
)

type Type string

const (
	VMUI       Type = "vmui"
	Prometheus Type = "prometheus"
	Grafana    Type = "grafana"
)

type URLBuilder interface {
	BuildURL(expr string, from time.Time, to *time.Time) string
}

func New(dashboardType Type, baseURL string) (URLBuilder, error) {
	switch dashboardType {
	case Prometheus:
		return newPrometheus(baseURL)
	case VMUI:
		return newVMUI(baseURL)
	case Grafana:
		return newGrafana(baseURL)
	default:
		return nil, fmt.Errorf("unknown type: %q (must be prometheus, vmui, or grafana)", dashboardType)
	}
}
