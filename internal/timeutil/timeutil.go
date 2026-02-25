// Package timeutil provides utility functions for common time calculations.
package timeutil

import (
	"fmt"
	"time"
)

// Now returns the current time.
// This is a convenience wrapper that can be mocked in tests.
var Now = time.Now

// StartOfDay returns the start of the day for the given time.
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay returns the end of the day (23:59:59) for the given time.
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
}

// StartOfWeek returns the start of the week (Monday) for the given time.
func StartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday is 0, make it 7
	}
	daysToMonday := weekday - 1
	return StartOfDay(t.AddDate(0, 0, -daysToMonday))
}

// EndOfWeek returns the end of the week (Sunday 23:59:59) for the given time.
func EndOfWeek(t time.Time) time.Time {
	startOfWeek := StartOfWeek(t)
	return EndOfDay(startOfWeek.AddDate(0, 0, 6))
}

// StartOfMonth returns the first day of the month for the given time.
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth returns the last day of the month for the given time.
func EndOfMonth(t time.Time) time.Time {
	nextMonth := t.AddDate(0, 1, 0)
	firstOfNextMonth := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, t.Location())
	return EndOfDay(firstOfNextMonth.AddDate(0, 0, -1))
}

// AddDays adds the specified number of days to the given time.
func AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

// AddWeeks adds the specified number of weeks to the given time.
func AddWeeks(t time.Time, weeks int) time.Time {
	return t.AddDate(0, 0, weeks*7)
}

// IsToday returns true if the given time is today.
func IsToday(t time.Time) bool {
	now := Now()
	return t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day()
}

// IsThisWeek returns true if the given time is in the current week.
func IsThisWeek(t time.Time) bool {
	now := Now()
	startOfWeek := StartOfWeek(now)
	endOfWeek := EndOfWeek(now)
	return !t.Before(startOfWeek) && !t.After(endOfWeek)
}

// IsThisMonth returns true if the given time is in the current month.
func IsThisMonth(t time.Time) bool {
	now := Now()
	return t.Year() == now.Year() && t.Month() == now.Month()
}

// DaysBetween returns the number of days between two times.
func DaysBetween(start, end time.Time) int {
	start = StartOfDay(start)
	end = StartOfDay(end)
	return int(end.Sub(start).Hours() / 24)
}

// FormatDuration formats a duration as a human-readable string.
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "yesterday"
	}
	if days < 7 {
		return fmt.Sprintf("%d days ago", days)
	}
	if days < 30 {
		weeks := days / 7
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	}
	months := days / 30
	if months == 1 {
		return "1 month ago"
	}
	return fmt.Sprintf("%d months ago", months)
}
