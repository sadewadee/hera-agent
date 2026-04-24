package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// nextCronTime parses a simplified cron expression and returns the next occurrence after t.
// Supported format: "minute hour day_of_month month day_of_week" (5 fields).
// Supports: numbers, *, */N (step), and comma-separated values.
// This is a lightweight cron parser that avoids external dependencies.
func nextCronTime(expr string, after time.Time) (time.Time, error) {
	shorthands := map[string]string{
		"@yearly":   "0 0 1 1 *",
		"@annually": "0 0 1 1 *",
		"@monthly":  "0 0 1 * *",
		"@weekly":   "0 0 * * 0",
		"@daily":    "0 0 * * *",
		"@midnight": "0 0 * * *",
		"@hourly":   "0 * * * *",
	}
	if replacement, ok := shorthands[strings.TrimSpace(expr)]; ok {
		expr = replacement
	}

	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return time.Time{}, fmt.Errorf("expected 5 fields, got %d", len(fields))
	}

	minuteSet, err := parseField(fields[0], 0, 59)
	if err != nil {
		return time.Time{}, fmt.Errorf("minute: %w", err)
	}
	hourSet, err := parseField(fields[1], 0, 23)
	if err != nil {
		return time.Time{}, fmt.Errorf("hour: %w", err)
	}
	domSet, err := parseField(fields[2], 1, 31)
	if err != nil {
		return time.Time{}, fmt.Errorf("day of month: %w", err)
	}
	monthSet, err := parseField(fields[3], 1, 12)
	if err != nil {
		return time.Time{}, fmt.Errorf("month: %w", err)
	}
	dowSet, err := parseField(fields[4], 0, 6)
	if err != nil {
		return time.Time{}, fmt.Errorf("day of week: %w", err)
	}

	// Start searching from the next minute after `after`.
	candidate := after.Truncate(time.Minute).Add(time.Minute)

	// Search up to 1 year ahead.
	limit := after.Add(366 * 24 * time.Hour)
	for candidate.Before(limit) {
		if !monthSet[int(candidate.Month())] {
			// Advance to next month.
			candidate = time.Date(candidate.Year(), candidate.Month()+1, 1, 0, 0, 0, 0, candidate.Location())
			continue
		}
		if !domSet[candidate.Day()] {
			candidate = candidate.AddDate(0, 0, 1)
			candidate = time.Date(candidate.Year(), candidate.Month(), candidate.Day(), 0, 0, 0, 0, candidate.Location())
			continue
		}
		if !dowSet[int(candidate.Weekday())] {
			candidate = candidate.AddDate(0, 0, 1)
			candidate = time.Date(candidate.Year(), candidate.Month(), candidate.Day(), 0, 0, 0, 0, candidate.Location())
			continue
		}
		if !hourSet[candidate.Hour()] {
			candidate = candidate.Add(time.Hour)
			candidate = time.Date(candidate.Year(), candidate.Month(), candidate.Day(), candidate.Hour(), 0, 0, 0, candidate.Location())
			continue
		}
		if !minuteSet[candidate.Minute()] {
			candidate = candidate.Add(time.Minute)
			continue
		}
		return candidate, nil
	}

	return time.Time{}, fmt.Errorf("no matching time found within 1 year")
}

// parseField parses a single cron field into a set of valid values.
func parseField(field string, min, max int) (map[int]bool, error) {
	result := make(map[int]bool)

	parts := strings.Split(field, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Handle step: */N or N/N
		if strings.Contains(part, "/") {
			stepParts := strings.SplitN(part, "/", 2)
			step, err := strconv.Atoi(stepParts[1])
			if err != nil || step <= 0 {
				return nil, fmt.Errorf("invalid step: %s", part)
			}
			start := min
			if stepParts[0] != "*" {
				start, err = strconv.Atoi(stepParts[0])
				if err != nil {
					return nil, fmt.Errorf("invalid range start: %s", stepParts[0])
				}
			}
			for i := start; i <= max; i += step {
				result[i] = true
			}
			continue
		}

		// Handle wildcard.
		if part == "*" {
			for i := min; i <= max; i++ {
				result[i] = true
			}
			continue
		}

		// Handle range: N-M
		if strings.Contains(part, "-") {
			rangeParts := strings.SplitN(part, "-", 2)
			low, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			high, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			if low < min || high > max || low > high {
				return nil, fmt.Errorf("range out of bounds: %s", part)
			}
			for i := low; i <= high; i++ {
				result[i] = true
			}
			continue
		}

		// Handle single value.
		val, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %s", part)
		}
		if val < min || val > max {
			return nil, fmt.Errorf("value %d out of range [%d, %d]", val, min, max)
		}
		result[val] = true
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("empty field")
	}
	return result, nil
}
