package hcore

import (
	"fmt"
	"time"
)

// HumanDuration formats a duration into a human-readable string.
func HumanDuration(d time.Duration) string {
	switch {
	case d < time.Second: return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute: return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour: return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	case d < 24*time.Hour: return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	default: days := int(d.Hours()) / 24; return fmt.Sprintf("%dd%dh", days, int(d.Hours())%24)
	}
}

// RelativeTime returns a human-readable relative time string.
func RelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute: return "just now"
	case d < time.Hour: return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	case d < 24*time.Hour: return fmt.Sprintf("%d hours ago", int(d.Hours()))
	default: return fmt.Sprintf("%d days ago", int(d.Hours())/24)
	}
}

// ParseDuration extends time.ParseDuration with support for "d" (days) and "w" (weeks).
func ParseDuration(s string) (time.Duration, error) {
	if len(s) > 1 {
		suffix := s[len(s)-1]
		if suffix == 'd' {
			var n int
			if _, err := fmt.Sscanf(s, "%dd", &n); err == nil { return time.Duration(n) * 24 * time.Hour, nil }
		}
		if suffix == 'w' {
			var n int
			if _, err := fmt.Sscanf(s, "%dw", &n); err == nil { return time.Duration(n) * 7 * 24 * time.Hour, nil }
		}
	}
	return time.ParseDuration(s)
}
