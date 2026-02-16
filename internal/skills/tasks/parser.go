package tasks

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DateParser parses natural language dates
type DateParser struct {
	referenceTime time.Time
}

// NewDateParser creates a new date parser
func NewDateParser() *DateParser {
	return &DateParser{
		referenceTime: time.Now(),
	}
}

// WithReference sets the reference time for relative parsing
func (p *DateParser) WithReference(t time.Time) *DateParser {
	p.referenceTime = t
	return p
}

// ParseResult contains the parsed date information
type ParseResult struct {
	Date        time.Time
	HasTime     bool
	Confidence  float64
	Description string
}

// Parse parses a natural language date string
func (p *DateParser) Parse(input string) (*ParseResult, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	
	// Try exact date first
	if result, ok := p.parseExactDate(input); ok {
		return result, nil
	}
	
	// Try relative dates
	if result, ok := p.parseRelativeDate(input); ok {
		return result, nil
	}
	
	// Try day of week
	if result, ok := p.parseDayOfWeek(input); ok {
		return result, nil
	}
	
	// Try time expressions
	if result, ok := p.parseTimeExpression(input); ok {
		return result, nil
	}
	
	return nil, fmt.Errorf("could not parse date: %s", input)
}

// parseExactDate parses exact dates like "2024-01-15" or "Jan 15, 2024"
func (p *DateParser) parseExactDate(input string) (*ParseResult, bool) {
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"01/02/2006",
		"01/02/2006 15:04",
		"Jan 2, 2006",
		"Jan 2, 2006 3:04pm",
		"January 2, 2006",
		"2 Jan 2006",
		"2 January 2006",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, input); err == nil {
			hasTime := strings.Contains(format, "15") || strings.Contains(format, "3")
			return &ParseResult{
				Date:       t,
				HasTime:    hasTime,
				Confidence: 1.0,
			}, true
		}
	}
	
	return nil, false
}

// parseRelativeDate parses relative dates like "tomorrow", "next week"
func (p *DateParser) parseRelativeDate(input string) (*ParseResult, bool) {
	now := p.referenceTime
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	switch input {
	case "today":
		return &ParseResult{
			Date:       today,
			HasTime:    false,
			Confidence: 1.0,
		}, true
		
	case "tomorrow", "tmrw", "tmr":
		return &ParseResult{
			Date:       today.AddDate(0, 0, 1),
			HasTime:    false,
			Confidence: 1.0,
		}, true
		
	case "yesterday":
		return &ParseResult{
			Date:       today.AddDate(0, 0, -1),
			HasTime:    false,
			Confidence: 1.0,
		}, true
		
	case "day after tomorrow":
		return &ParseResult{
			Date:       today.AddDate(0, 0, 2),
			HasTime:    false,
			Confidence: 1.0,
		}, true
	}
	
	// "in X days/hours/minutes"
	if result, ok := p.parseInDuration(input); ok {
		return result, true
	}
	
	// "next week/month/year"
	if result, ok := p.parseNextPeriod(input); ok {
		return result, true
	}
	
	// "this week/month/year"
	if result, ok := p.parseThisPeriod(input); ok {
		return result, true
	}
	
	return nil, false
}

// parseInDuration parses "in X days/hours/minutes"
func (p *DateParser) parseInDuration(input string) (*ParseResult, bool) {
	patterns := []struct {
		regex    *regexp.Regexp
		duration func(int) time.Duration
	}{
		{regexp.MustCompile(`in\s+(\d+)\s*minute(?:s)?`), func(n int) time.Duration { return time.Duration(n) * time.Minute }},
		{regexp.MustCompile(`in\s+(\d+)\s*hour(?:s)?`), func(n int) time.Duration { return time.Duration(n) * time.Hour }},
		{regexp.MustCompile(`in\s+(\d+)\s*day(?:s)?`), func(n int) time.Duration { return time.Duration(n) * 24 * time.Hour }},
		{regexp.MustCompile(`in\s+(\d+)\s*week(?:s)?`), func(n int) time.Duration { return time.Duration(n) * 7 * 24 * time.Hour }},
		{regexp.MustCompile(`in\s+(\d+)\s*month(?:s)?`), func(n int) time.Duration { return time.Duration(n) * 30 * 24 * time.Hour }},
	}
	
	for _, ptn := range patterns {
		matches := ptn.regex.FindStringSubmatch(input)
		if len(matches) > 1 {
			n, _ := strconv.Atoi(matches[1])
			return &ParseResult{
				Date:       p.referenceTime.Add(ptn.duration(n)),
				HasTime:    ptn.duration(1) < 24*time.Hour,
				Confidence: 0.9,
			}, true
		}
	}
	
	return nil, false
}

// parseNextPeriod parses "next week/month/year"
func (p *DateParser) parseNextPeriod(input string) (*ParseResult, bool) {
	now := p.referenceTime
	
	switch input {
	case "next week":
		// Start of next week (Monday)
		daysUntilMonday := (8 - int(now.Weekday())) % 7
		if daysUntilMonday == 0 {
			daysUntilMonday = 7
		}
		nextMonday := time.Date(now.Year(), now.Month(), now.Day()+daysUntilMonday, 0, 0, 0, 0, now.Location())
		return &ParseResult{
			Date:       nextMonday,
			HasTime:    false,
			Confidence: 0.9,
		}, true
		
	case "next month":
		firstOfNextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
		return &ParseResult{
			Date:       firstOfNextMonth,
			HasTime:    false,
			Confidence: 0.9,
		}, true
		
	case "next year":
		firstOfNextYear := time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location())
		return &ParseResult{
			Date:       firstOfNextYear,
			HasTime:    false,
			Confidence: 0.9,
		}, true
	}
	
	return nil, false
}

// parseThisPeriod parses "this week/month/year"
func (p *DateParser) parseThisPeriod(input string) (*ParseResult, bool) {
	now := p.referenceTime
	
	switch input {
	case "this week":
		// Start of this week (Monday)
		daysSinceMonday := (int(now.Weekday()) + 6) % 7
		thisMonday := time.Date(now.Year(), now.Month(), now.Day()-daysSinceMonday, 0, 0, 0, 0, now.Location())
		return &ParseResult{
			Date:       thisMonday,
			HasTime:    false,
			Confidence: 0.9,
		}, true
		
	case "this month":
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return &ParseResult{
			Date:       firstOfMonth,
			HasTime:    false,
			Confidence: 0.9,
		}, true
		
	case "this year":
		firstOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return &ParseResult{
			Date:       firstOfYear,
			HasTime:    false,
			Confidence: 0.9,
		}, true
	}
	
	return nil, false
}

// parseDayOfWeek parses day names like "monday", "next tuesday"
func (p *DateParser) parseDayOfWeek(input string) (*ParseResult, bool) {
	days := map[string]time.Weekday{
		"sunday":    0,
		"monday":    1,
		"tuesday":   2,
		"wednesday": 3,
		"thursday":  4,
		"friday":    5,
		"saturday":  6,
	}
	
	now := p.referenceTime
	today := now.Weekday()
	
	// Check for "next X" pattern
	nextPattern := regexp.MustCompile(`^next\s+(\w+)$`)
	nextMatches := nextPattern.FindStringSubmatch(input)
	
	if len(nextMatches) > 1 {
		dayName := nextMatches[1]
		if targetDay, ok := days[dayName]; ok {
			daysUntil := int(targetDay) - int(today)
			if daysUntil <= 0 {
				daysUntil += 7
			}
			targetDate := time.Date(now.Year(), now.Month(), now.Day()+daysUntil, 0, 0, 0, 0, now.Location())
			return &ParseResult{
				Date:       targetDate,
				HasTime:    false,
				Confidence: 0.9,
			}, true
		}
	}
	
	// Check for plain day name
	if targetDay, ok := days[input]; ok {
		daysUntil := int(targetDay) - int(today)
		if daysUntil <= 0 {
			daysUntil += 7 // Next occurrence
		}
		targetDate := time.Date(now.Year(), now.Month(), now.Day()+daysUntil, 0, 0, 0, 0, now.Location())
		return &ParseResult{
			Date:       targetDate,
			HasTime:    false,
			Confidence: 0.85,
		}, true
	}
	
	return nil, false
}

// parseTimeExpression parses time expressions like "3pm", "15:30"
func (p *DateParser) parseTimeExpression(input string) (*ParseResult, bool) {
	now := p.referenceTime
	
	// Try 12-hour format with am/pm
	patterns12h := []string{
		`(\d{1,2}):(\d{2})\s*(am|pm)`,
		`(\d{1,2})\s*(am|pm)`,
	}
	
	for _, pattern := range patterns12h {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(input)
		if len(matches) >= 3 {
			hour, _ := strconv.Atoi(matches[1])
			minute := 0
			if len(matches) > 3 && matches[2] != "am" && matches[2] != "pm" {
				minute, _ = strconv.Atoi(matches[2])
			}
			ampm := matches[len(matches)-1]
			
			if ampm == "pm" && hour != 12 {
				hour += 12
			} else if ampm == "am" && hour == 12 {
				hour = 0
			}
			
			result := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
			if result.Before(now) && !p.isToday(result) {
				result = result.AddDate(0, 0, 1)
			}
			
			return &ParseResult{
				Date:       result,
				HasTime:    true,
				Confidence: 0.9,
			}, true
		}
	}
	
	// Try 24-hour format
	re24h := regexp.MustCompile(`(\d{1,2}):(\d{2})(?::(\d{2}))?`)
	matches := re24h.FindStringSubmatch(input)
	if len(matches) >= 3 {
		hour, _ := strconv.Atoi(matches[1])
		minute, _ := strconv.Atoi(matches[2])
		
		if hour >= 0 && hour < 24 && minute >= 0 && minute < 60 {
			result := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
			if result.Before(now) && !p.isToday(result) {
				result = result.AddDate(0, 0, 1)
			}
			
			return &ParseResult{
				Date:       result,
				HasTime:    true,
				Confidence: 0.9,
			}, true
		}
	}
	
	return nil, false
}

// isToday checks if a date is today
func (p *DateParser) isToday(t time.Time) bool {
	now := p.referenceTime
	return t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day()
}

// ExtractDateTime extracts both date and time from a phrase like "tomorrow at 3pm"
func (p *DateParser) ExtractDateTime(input string) (*ParseResult, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	
	// Common patterns
	patterns := []struct {
		regex   *regexp.Regexp
		handler func([]string) (*ParseResult, bool)
	}{
		// "tomorrow at 3pm"
		{
			regexp.MustCompile(`^(tomorrow|today|yesterday)\s+at\s+(.+)$`),
			func(matches []string) (*ParseResult, bool) {
				dateResult, ok := p.parseRelativeDate(matches[1])
				if !ok {
					return nil, false
				}
				timeResult, ok := p.parseTimeExpression(matches[2])
				if !ok {
					return nil, false
				}
				
				// Combine date and time
				combined := time.Date(
					dateResult.Date.Year(), dateResult.Date.Month(), dateResult.Date.Day(),
					timeResult.Date.Hour(), timeResult.Date.Minute(), 0, 0,
					dateResult.Date.Location(),
				)
				
				return &ParseResult{
					Date:       combined,
					HasTime:    true,
					Confidence: 0.95,
				}, true
			},
		},
		// "next tuesday at 2pm"
		{
			regexp.MustCompile(`^(next\s+\w+)\s+at\s+(.+)$`),
			func(matches []string) (*ParseResult, bool) {
				dateResult, ok := p.parseDayOfWeek(matches[1])
				if !ok {
					return nil, false
				}
				timeResult, ok := p.parseTimeExpression(matches[2])
				if !ok {
					return nil, false
				}
				
				combined := time.Date(
					dateResult.Date.Year(), dateResult.Date.Month(), dateResult.Date.Day(),
					timeResult.Date.Hour(), timeResult.Date.Minute(), 0, 0,
					dateResult.Date.Location(),
				)
				
				return &ParseResult{
					Date:       combined,
					HasTime:    true,
					Confidence: 0.95,
				}, true
			},
		},
	}
	
	for _, ptn := range patterns {
		matches := ptn.regex.FindStringSubmatch(input)
		if len(matches) > 0 {
			if result, ok := ptn.handler(matches[1:]); ok {
				return result, nil
			}
		}
	}
	
	// Try simple parsing
	return p.Parse(input)
}

// DurationParser parses duration strings
type DurationParser struct{}

// Parse parses a duration string like "30 minutes", "2 hours", "1 day"
func (dp *DurationParser) Parse(input string) (time.Duration, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	
	patterns := []struct {
		regex    *regexp.Regexp
		duration func(int) time.Duration
	}{
		{regexp.MustCompile(`^(\d+)\s*(m|min|minute|minutes)$`), func(n int) time.Duration { return time.Duration(n) * time.Minute }},
		{regexp.MustCompile(`^(\d+)\s*(h|hr|hour|hours)$`), func(n int) time.Duration { return time.Duration(n) * time.Hour }},
		{regexp.MustCompile(`^(\d+)\s*(d|day|days)$`), func(n int) time.Duration { return time.Duration(n) * 24 * time.Hour }},
		{regexp.MustCompile(`^(\d+)\s*(w|week|weeks)$`), func(n int) time.Duration { return time.Duration(n) * 7 * 24 * time.Hour }},
	}
	
	for _, ptn := range patterns {
		matches := ptn.regex.FindStringSubmatch(input)
		if len(matches) > 1 {
			n, _ := strconv.Atoi(matches[1])
			return ptn.duration(n), nil
		}
	}
	
	return 0, fmt.Errorf("could not parse duration: %s", input)
}

// FormatRelativeTime formats a time as a relative string
func FormatRelativeTime(t time.Time) string {
	duration := time.Until(t)
	isPast := duration < 0
	duration = duration.Abs()
	
	if duration < time.Minute {
		if isPast {
			return "just now"
		}
		return "in a few seconds"
	}
	
	if duration < time.Hour {
		mins := int(duration.Minutes())
		if isPast {
			return fmt.Sprintf("%d minutes ago", mins)
		}
		return fmt.Sprintf("in %d minutes", mins)
	}
	
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if isPast {
			return fmt.Sprintf("%d hours ago", hours)
		}
		return fmt.Sprintf("in %d hours", hours)
	}
	
	days := int(duration.Hours() / 24)
	if days == 1 {
		if isPast {
			return "yesterday"
		}
		return "tomorrow"
	}
	
	if days < 7 {
		if isPast {
			return fmt.Sprintf("%d days ago", days)
		}
		return fmt.Sprintf("in %d days", days)
	}
	
	weeks := days / 7
	if weeks < 4 {
		if isPast {
			return fmt.Sprintf("%d weeks ago", weeks)
		}
		return fmt.Sprintf("in %d weeks", weeks)
	}
	
	return t.Format("Jan 2, 2006")
}
