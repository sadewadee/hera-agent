package cron

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParseNLCron converts a natural language schedule description into a cron expression.
// Examples:
//
//	"every 5 minutes"        → "*/5 * * * *"
//	"every hour"             → "0 * * * *"
//	"every day at 9am"       → "0 9 * * *"
//	"every monday at 10:30"  → "30 10 * * 1"
//	"every weekday at 8am"   → "0 8 * * 1-5"
//	"every 1st of month"     → "0 0 1 * *"
//	"twice a day"            → "0 0,12 * * *"
//	"every 15 minutes"       → "*/15 * * * *"
//	"daily at midnight"      → "0 0 * * *"
//	"weekly on friday at 5pm"→ "0 17 * * 5"
func ParseNLCron(nl string) (string, error) {
	nl = strings.ToLower(strings.TrimSpace(nl))
	if nl == "" {
		return "", fmt.Errorf("empty schedule description")
	}

	// Direct cron expression passthrough (5 fields).
	if isCronExpr(nl) {
		return nl, nil
	}

	// Shorthand passthrough.
	shorthands := map[string]string{
		"@yearly": "@yearly", "@annually": "@annually",
		"@monthly": "@monthly", "@weekly": "@weekly",
		"@daily": "@daily", "@midnight": "@midnight",
		"@hourly": "@hourly",
	}
	if expr, ok := shorthands[nl]; ok {
		return expr, nil
	}

	// Pattern matching.

	// "every N minutes"
	if m := reEveryNMinutes.FindStringSubmatch(nl); m != nil {
		n, _ := strconv.Atoi(m[1])
		if n < 1 || n > 59 {
			return "", fmt.Errorf("invalid minute interval: %d", n)
		}
		return fmt.Sprintf("*/%d * * * *", n), nil
	}

	// "every N hours"
	if m := reEveryNHours.FindStringSubmatch(nl); m != nil {
		n, _ := strconv.Atoi(m[1])
		if n < 1 || n > 23 {
			return "", fmt.Errorf("invalid hour interval: %d", n)
		}
		return fmt.Sprintf("0 */%d * * *", n), nil
	}

	// "every minute"
	if reEveryMinute.MatchString(nl) {
		return "* * * * *", nil
	}

	// "every hour"
	if reEveryHour.MatchString(nl) {
		return "0 * * * *", nil
	}

	// "every day at HH:MM" or "daily at HH:MM"
	if m := reDailyAt.FindStringSubmatch(nl); m != nil {
		hour, min := parseTime(m[1])
		return fmt.Sprintf("%d %d * * *", min, hour), nil
	}

	// "every day" / "daily" (without time)
	if reDaily.MatchString(nl) {
		return "0 0 * * *", nil
	}

	// "every <weekday> at HH:MM"
	if m := reWeekdayAt.FindStringSubmatch(nl); m != nil {
		dow := dayOfWeek(m[1])
		if dow < 0 {
			return "", fmt.Errorf("unknown day: %s", m[1])
		}
		hour, min := parseTime(m[2])
		return fmt.Sprintf("%d %d * * %d", min, hour, dow), nil
	}

	// "every <weekday>" (without time)
	if m := reEveryWeekday.FindStringSubmatch(nl); m != nil {
		dow := dayOfWeek(m[1])
		if dow < 0 {
			return "", fmt.Errorf("unknown day: %s", m[1])
		}
		return fmt.Sprintf("0 0 * * %d", dow), nil
	}

	// "every weekday at HH:MM" (mon-fri)
	if m := reWeekdaysAt.FindStringSubmatch(nl); m != nil {
		hour, min := parseTime(m[1])
		return fmt.Sprintf("%d %d * * 1-5", min, hour), nil
	}

	// "every weekday" (mon-fri, no time)
	if reWeekdays.MatchString(nl) {
		return "0 9 * * 1-5", nil
	}

	// "weekly" / "every week"
	if reWeekly.MatchString(nl) {
		return "0 0 * * 0", nil
	}

	// "monthly" / "every month"
	if reMonthly.MatchString(nl) {
		return "0 0 1 * *", nil
	}

	// "every Nth of month at HH:MM"
	if m := reMonthDayAt.FindStringSubmatch(nl); m != nil {
		day, _ := strconv.Atoi(m[1])
		if day < 1 || day > 31 {
			return "", fmt.Errorf("invalid day of month: %d", day)
		}
		hour, min := parseTime(m[2])
		return fmt.Sprintf("%d %d %d * *", min, hour, day), nil
	}

	// "twice a day"
	if reTwiceDaily.MatchString(nl) {
		return "0 0,12 * * *", nil
	}

	// "three times a day"
	if reThreeTimesDaily.MatchString(nl) {
		return "0 0,8,16 * * *", nil
	}

	return "", fmt.Errorf("could not parse schedule: %q (try: 'every 5 minutes', 'daily at 9am', 'every monday at 10:30')", nl)
}

// Compiled regex patterns.
var (
	reEveryNMinutes   = regexp.MustCompile(`every\s+(\d+)\s+min(?:ute)?s?`)
	reEveryNHours     = regexp.MustCompile(`every\s+(\d+)\s+hours?`)
	reEveryMinute     = regexp.MustCompile(`^every\s+minute$`)
	reEveryHour       = regexp.MustCompile(`^every\s+hour$`)
	reDailyAt         = regexp.MustCompile(`(?:every\s+day|daily|each\s+day)\s+at\s+(\S+)`)
	reDaily           = regexp.MustCompile(`^(?:every\s+day|daily)$`)
	reWeekdayAt       = regexp.MustCompile(`every\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday|mon|tue|wed|thu|fri|sat|sun)\s+at\s+(\S+)`)
	reEveryWeekday    = regexp.MustCompile(`^every\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday|mon|tue|wed|thu|fri|sat|sun)$`)
	reWeekdaysAt      = regexp.MustCompile(`every\s+weekday\s+at\s+(\S+)`)
	reWeekdays        = regexp.MustCompile(`^every\s+weekday$`)
	reWeekly          = regexp.MustCompile(`^(?:weekly|every\s+week)$`)
	reMonthly         = regexp.MustCompile(`^(?:monthly|every\s+month)$`)
	reMonthDayAt      = regexp.MustCompile(`every\s+(\d+)(?:st|nd|rd|th)?\s+(?:of\s+(?:the\s+)?month\s+)?at\s+(\S+)`)
	reTwiceDaily      = regexp.MustCompile(`twice\s+(?:a|per)\s+day`)
	reThreeTimesDaily = regexp.MustCompile(`(?:three|3)\s+times?\s+(?:a|per)\s+day`)
)

func isCronExpr(s string) bool {
	fields := strings.Fields(s)
	return len(fields) == 5
}

// parseTime parses time strings like "9am", "10:30", "17:00", "5pm", "midnight", "noon".
func parseTime(s string) (hour, min int) {
	s = strings.ToLower(strings.TrimSpace(s))

	switch s {
	case "midnight":
		return 0, 0
	case "noon":
		return 12, 0
	}

	isPM := strings.HasSuffix(s, "pm")
	isAM := strings.HasSuffix(s, "am")
	s = strings.TrimSuffix(s, "pm")
	s = strings.TrimSuffix(s, "am")

	parts := strings.SplitN(s, ":", 2)
	hour, _ = strconv.Atoi(parts[0])
	if len(parts) > 1 {
		min, _ = strconv.Atoi(parts[1])
	}

	if isPM && hour < 12 {
		hour += 12
	}
	if isAM && hour == 12 {
		hour = 0
	}

	return hour, min
}

func dayOfWeek(s string) int {
	days := map[string]int{
		"sunday": 0, "sun": 0,
		"monday": 1, "mon": 1,
		"tuesday": 2, "tue": 2,
		"wednesday": 3, "wed": 3,
		"thursday": 4, "thu": 4,
		"friday": 5, "fri": 5,
		"saturday": 6, "sat": 6,
	}
	if d, ok := days[strings.ToLower(s)]; ok {
		return d
	}
	return -1
}
