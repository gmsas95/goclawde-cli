package timeutil

import (
	"testing"
	"time"
)

func TestStartOfDay(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 45, 100, time.UTC)
	start := StartOfDay(now)

	if start.Hour() != 0 || start.Minute() != 0 || start.Second() != 0 {
		t.Error("StartOfDay should set time to 00:00:00")
	}

	if start.Year() != 2024 || start.Month() != 3 || start.Day() != 15 {
		t.Error("StartOfDay should preserve date")
	}
}

func TestEndOfDay(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 45, 100, time.UTC)
	end := EndOfDay(now)

	if end.Hour() != 23 || end.Minute() != 59 || end.Second() != 59 {
		t.Error("EndOfDay should set time to 23:59:59")
	}

	if end.Year() != 2024 || end.Month() != 3 || end.Day() != 15 {
		t.Error("EndOfDay should preserve date")
	}
}

func TestStartOfWeek(t *testing.T) {
	// Test Wednesday (March 13, 2024)
	wednesday := time.Date(2024, 3, 13, 12, 0, 0, 0, time.UTC)
	start := StartOfWeek(wednesday)

	// Monday of that week is March 11, 2024
	if start.Weekday() != time.Monday {
		t.Errorf("Expected Monday, got %v", start.Weekday())
	}
	if start.Day() != 11 {
		t.Errorf("Expected day 11, got %d", start.Day())
	}
}

func TestEndOfWeek(t *testing.T) {
	// Test Wednesday (March 13, 2024)
	wednesday := time.Date(2024, 3, 13, 12, 0, 0, 0, time.UTC)
	end := EndOfWeek(wednesday)

	// Sunday of that week is March 17, 2024
	if end.Weekday() != time.Sunday {
		t.Errorf("Expected Sunday, got %v", end.Weekday())
	}
	if end.Day() != 17 {
		t.Errorf("Expected day 17, got %d", end.Day())
	}
}

func TestStartOfMonth(t *testing.T) {
	now := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
	start := StartOfMonth(now)

	if start.Day() != 1 {
		t.Errorf("Expected day 1, got %d", start.Day())
	}
	if start.Month() != 3 {
		t.Error("Expected March")
	}
}

func TestEndOfMonth(t *testing.T) {
	// Test March 2024 (31 days)
	now := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
	end := EndOfMonth(now)

	if end.Day() != 31 {
		t.Errorf("Expected day 31, got %d", end.Day())
	}
	if end.Month() != 3 {
		t.Error("Expected March")
	}

	// Test February 2024 (leap year - 29 days)
	feb := time.Date(2024, 2, 15, 12, 0, 0, 0, time.UTC)
	endFeb := EndOfMonth(feb)
	if endFeb.Day() != 29 {
		t.Errorf("Expected day 29 for leap year Feb, got %d", endFeb.Day())
	}
}

func TestAddDays(t *testing.T) {
	now := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)

	future := AddDays(now, 5)
	if future.Day() != 20 {
		t.Errorf("Expected day 20, got %d", future.Day())
	}

	past := AddDays(now, -5)
	if past.Day() != 10 {
		t.Errorf("Expected day 10, got %d", past.Day())
	}
}

func TestDaysBetween(t *testing.T) {
	start := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
	end := time.Date(2024, 3, 20, 8, 30, 0, 0, time.UTC)

	days := DaysBetween(start, end)
	if days != 5 {
		t.Errorf("Expected 5 days, got %d", days)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "just now"},
		{1 * time.Minute, "1 minute ago"},
		{5 * time.Minute, "5 minutes ago"},
		{1 * time.Hour, "1 hour ago"},
		{3 * time.Hour, "3 hours ago"},
		{25 * time.Hour, "yesterday"},
		{3 * 24 * time.Hour, "3 days ago"},
		{10 * 24 * time.Hour, "1 week ago"},
		{20 * 24 * time.Hour, "2 weeks ago"},
	}

	for _, test := range tests {
		result := FormatDuration(test.duration)
		if result != test.expected {
			t.Errorf("FormatDuration(%v) = %s, expected %s", test.duration, result, test.expected)
		}
	}
}
