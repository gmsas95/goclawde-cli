package persona

import (
	"fmt"
	"time"
)

// TimeAwareness provides time-based context for the AI
type TimeAwareness struct {
	location *time.Location
}

// NewTimeAwareness creates a new time awareness system
func NewTimeAwareness() *TimeAwareness {
	// Use local timezone
	return &TimeAwareness{
		location: time.Local,
	}
}

// GetContext returns time-aware context for the system prompt
func (ta *TimeAwareness) GetContext() string {
	now := time.Now().In(ta.location)

	var parts []string
	parts = append(parts, "## Current Context")
	parts = append(parts, fmt.Sprintf("Current time: %s", now.Format("Monday, January 2, 2006 3:04 PM")))
	parts = append(parts, fmt.Sprintf("Time of day: %s", ta.getTimeOfDay(now)))

	// Add time-appropriate guidance
	guidance := ta.getTimeGuidance(now)
	if guidance != "" {
		parts = append(parts, fmt.Sprintf("Context: %s", guidance))
	}

	return fmt.Sprintf("%s\n", joinLines(parts))
}

// GetGreeting returns a time-appropriate greeting
func (ta *TimeAwareness) GetGreeting() string {
	hour := time.Now().In(ta.location).Hour()

	switch {
	case hour >= 5 && hour < 12:
		return "Good morning"
	case hour >= 12 && hour < 17:
		return "Good afternoon"
	case hour >= 17 && hour < 22:
		return "Good evening"
	default:
		return "Hello"
	}
}

// getTimeOfDay returns the current time period
func (ta *TimeAwareness) getTimeOfDay(t time.Time) string {
	hour := t.Hour()

	switch {
	case hour >= 5 && hour < 8:
		return "early morning"
	case hour >= 8 && hour < 12:
		return "morning"
	case hour >= 12 && hour < 14:
		return "midday"
	case hour >= 14 && hour < 17:
		return "afternoon"
	case hour >= 17 && hour < 20:
		return "evening"
	case hour >= 20 && hour < 22:
		return "night"
	default:
		return "late night"
	}
}

// getTimeGuidance returns guidance based on time of day
func (ta *TimeAwareness) getTimeGuidance(t time.Time) string {
	hour := t.Hour()
	weekday := t.Weekday()

	// Weekend guidance
	if weekday == time.Saturday || weekday == time.Sunday {
		switch {
		case hour >= 9 && hour < 12:
			return "It's a weekend morning. The user might be working on personal projects or relaxing."
		case hour >= 14 && hour < 18:
			return "It's a weekend afternoon. The user has free time for hobbies or deep work."
		}
	}

	// Weekday guidance
	switch {
	case hour >= 6 && hour < 9:
		return "It's early morning. The user might be starting their day, reviewing priorities, or commuting."
	case hour >= 9 && hour < 12:
		return "It's morning work hours. The user is likely focused on productive work."
	case hour >= 12 && hour < 14:
		return "It's midday. The user might be taking a lunch break or doing lighter tasks."
	case hour >= 14 && hour < 17:
		return "It's afternoon. The user is likely in deep work mode."
	case hour >= 17 && hour < 19:
		return "It's late afternoon/early evening. The user might be wrapping up work or transitioning to personal time."
	case hour >= 19 && hour < 22:
		return "It's evening. The user is likely focused on personal projects, learning, or relaxation."
	default:
		return "It's late night. The user might be working on passion projects or need quick assistance."
	}
}

// IsWorkHours returns true if current time is typical work hours
func (ta *TimeAwareness) IsWorkHours() bool {
	now := time.Now().In(ta.location)
	hour := now.Hour()
	weekday := now.Weekday()

	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	return hour >= 9 && hour < 18
}

// FormatDuration formats a duration in a human-friendly way
func (ta *TimeAwareness) FormatDuration(d time.Duration) string {
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
	return fmt.Sprintf("%d days ago", days)
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}
