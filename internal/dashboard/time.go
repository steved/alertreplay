package dashboard

import (
	"fmt"
	"time"
)

func padRange(from time.Time, to *time.Time) (time.Time, time.Time) {
	var (
		paddedFrom = from.Add(-5 * time.Minute)
		paddedTo   = from
	)

	if to != nil {
		paddedTo = *to
	}

	paddedTo = paddedTo.Add(5 * time.Minute)

	return paddedFrom, paddedTo
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}

	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60

	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}

	return fmt.Sprintf("%dh%dm", hours, mins)
}
