package relativetime

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const timeFormat = "2006-01-02 15:04:05"

var (
	relativeTimeRegex = regexp.MustCompile(`(?i)^(\d+)\s*(second|minute|hour|day|week|month|year)s?\s+ago$`)
	// Since we're not doing anything interactive, init the time once during parsing and then freeze it.
	now = sync.OnceValue(time.Now)
)

func Parse(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	if strings.EqualFold(s, "now") {
		return now(), nil
	}

	if matches := relativeTimeRegex.FindStringSubmatch(s); matches != nil {
		n, _ := strconv.Atoi(matches[1])
		unit := strings.ToLower(matches[2])

		var d time.Duration
		switch unit {
		case "second":
			d = time.Duration(n) * time.Second
		case "minute":
			d = time.Duration(n) * time.Minute
		case "hour":
			d = time.Duration(n) * time.Hour
		case "day":
			d = time.Duration(n) * 24 * time.Hour
		case "week":
			d = time.Duration(n) * 7 * 24 * time.Hour
		case "month":
			return now().AddDate(0, -n, 0), nil
		case "year":
			return now().AddDate(-n, 0, 0), nil
		}
		return now().Add(-d), nil
	}

	return time.ParseInLocation(timeFormat, s, time.UTC)
}
