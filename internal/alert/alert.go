package alert

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"
)

const matchThreshold = 2 * time.Minute

type Alert struct {
	OpenedAt   time.Time
	ResolvedAt *time.Time
	Labels     map[string]string
	URL        string
	Source     string
}

func (a Alert) Match(b Alert) bool {
	if !reflect.DeepEqual(a.Labels, b.Labels) {
		return false
	}
	return a.OpenedAt.Sub(b.OpenedAt).Abs() <= matchThreshold
}

func Sort(alerts []Alert) {
	slices.SortFunc(alerts, func(l, r Alert) int {
		return l.OpenedAt.Compare(r.OpenedAt)
	})
}

func FormatLabels(labels map[string]string) string {
	var (
		keys  = slices.Sorted(maps.Keys(labels))
		parts = make([]string, 0, len(keys))
	)

	for _, k := range keys {
		if k == "__name__" || k == "alertname" {
			continue
		}

		parts = append(parts, fmt.Sprintf("%s=%q", k, labels[k]))
	}

	return "{" + strings.Join(parts, ", ") + "}"
}
