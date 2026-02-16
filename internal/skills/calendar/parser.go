package calendar

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// EventParser parses natural language into calendar events
type EventParser struct {
	referenceTime time.Time
}

// NewEventParser creates a new event parser
func NewEventParser() *EventParser {
	return &EventParser{
		referenceTime: time.Now(),
	}
}

// WithReference sets the reference time
func (p *EventParser) WithReference(t time.Time) *EventParser {
	p.referenceTime = t
	return p
}

// ParseEvent parses natural language event description
func (p *EventParser) ParseEvent(text string) (*ParseResult, error) {
	text = strings.TrimSpace(text)
	
	result := &ParseResult{
		RawText:    text,
		Confidence: 0.5,
	}
	
	// Extract title
	title := p.extractTitle(text)
	if title != "" {
		result.Title = title
		result.Confidence += 0.1
	}
	
	// Extract date and time
	start, end, allDay := p.extractDateTime(text)
	if !start.IsZero() {
		result.StartTime = start
		result.EndTime = end
		result.AllDay = allDay
		result.Confidence += 0.3
	}
	
	// Extract location
	location := p.extractLocation(text)
	if location != "" {
		result.Location = location
		result.Confidence += 0.1
	}
	
	// Extract attendees
	attendees := p.extractAttendees(text)
	if len(attendees) > 0 {
		result.Attendees = attendees
		result.Confidence += 0.1
	}
	
	// Check for recurrence
	recurrence := p.extractRecurrence(text)
	if recurrence != "" {
		result.IsRecurring = true
		result.Recurrence = recurrence
		result.Confidence += 0.1
	}
	
	// Calculate duration if end time not set
	if result.EndTime.IsZero() && !result.StartTime.IsZero() {
		// Default to 1 hour
		result.EndTime = result.StartTime.Add(time.Hour)
	}
	
	return result, nil
}

// extractTitle extracts the event title from text
func (p *EventParser) extractTitle(text string) string {
	// Common patterns for event titles
	patterns := []string{
		`(?i)(?:schedule|add|create|book)\s+(?:a\s+)?(?:meeting|call|appointment|event)?\s*(?:with\s+[^\s]+\s+)?(?:about|for|to\s+discuss)?\s+(.+?)(?:\s+(?:on|at|tomorrow|today|next|\d{1,2}[:\d]{2}))`,
		`(?i)(?:have\s+a|have\s+an)\s+(.+?)(?:\s+(?:on|at|tomorrow|today|next))`,
		`(?i)^(.+?)(?:\s+(?:with|at|on|tomorrow|today))`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			title := strings.TrimSpace(matches[1])
			// Clean up common words
			title = regexp.MustCompile(`(?i)^(a|an|the)\s+`).ReplaceAllString(title, "")
			return title
		}
	}
	
	// Fallback: use first few words
	words := strings.Fields(text)
	if len(words) > 0 {
		end := 5
		if len(words) < end {
			end = len(words)
		}
		return strings.Join(words[:end], " ")
	}
	
	return "New Event"
}

// extractDateTime extracts date and time from text
func (p *EventParser) extractDateTime(text string) (start, end time.Time, allDay bool) {
	now := p.referenceTime
	
	// Check for all-day events
	allDayPatterns := []string{
		`(?i)\ball\s+day\b`,
		`(?i)\bfull\s+day\b`,
	}
	
	for _, pattern := range allDayPatterns {
		if regexp.MustCompile(pattern).MatchString(text) {
			allDay = true
			break
		}
	}
	
	// Extract date
	date := p.extractDate(text)
	if date.IsZero() {
		// Default to today
		date = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}
	
	// Extract time
	timeVal, duration := p.extractTime(text)
	
	if allDay {
		start = date
		end = date.AddDate(0, 0, 1)
		return
	}
	
	// Combine date and time
	if !timeVal.IsZero() {
		start = time.Date(
			date.Year(), date.Month(), date.Day(),
			timeVal.Hour(), timeVal.Minute(), 0, 0,
			now.Location(),
		)
	} else {
		// Default time based on context
		start = p.inferTimeFromContext(text, date)
	}
	
	// Calculate end time
	if duration > 0 {
		end = start.Add(duration)
	} else {
		// Default duration based on event type
		end = start.Add(p.inferDuration(text))
	}
	
	return
}

// extractDate extracts date from text
func (p *EventParser) extractDate(text string) time.Time {
	now := p.referenceTime
	
	// Tomorrow
	if regexp.MustCompile(`(?i)\btomorrow\b`).MatchString(text) {
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	}
	
	// Today
	if regexp.MustCompile(`(?i)\btoday\b`).MatchString(text) {
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}
	
	// Day of week
	days := map[string]int{
		"sunday": 0, "monday": 1, "tuesday": 2, "wednesday": 3,
		"thursday": 4, "friday": 5, "saturday": 6,
	}
	
	for day, num := range days {
		pattern := regexp.MustCompile(`(?i)\b` + day + `\b`)
		if pattern.MatchString(text) {
			targetDay := num
			currentDay := int(now.Weekday())
			daysUntil := (targetDay - currentDay + 7) % 7
			if daysUntil == 0 {
				daysUntil = 7 // Next week
			}
			
			// Check for "next"
			if regexp.MustCompile(`(?i)\bnext\s+` + day + `\b`).MatchString(text) {
				daysUntil += 7
			}
			
			return now.AddDate(0, 0, daysUntil)
		}
	}
	
	// Date patterns (Jan 15, 1/15, 15th)
	datePatterns := []string{
		`(?i)(?:on\s+)?(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+(\d{1,2})(?:st|nd|rd|th)?`,
		`(?i)(\d{1,2})/(\d{1,2})(?:/(\d{2,4}))?`,
		`(?i)(\d{1,2})(?:st|nd|rd|th)?\s+(?:of\s+)?(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)`,
	}
	
	months := map[string]time.Month{
		"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
		"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
	}
	
	// Try first pattern (Month Day)
	re := regexp.MustCompile(datePatterns[0])
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 3 {
		month := months[strings.ToLower(matches[1][:3])]
		day := 0
		if d, err := parseInt(matches[2]); err == nil {
			day = d
		}
		if day > 0 {
			return time.Date(now.Year(), month, day, 0, 0, 0, 0, now.Location())
		}
	}
	
	// Try second pattern (M/D or M/D/YYYY)
	re = regexp.MustCompile(datePatterns[1])
	matches = re.FindStringSubmatch(text)
	if len(matches) >= 3 {
		month := 0
		day := 0
		if m, err := parseInt(matches[1]); err == nil {
			month = m
		}
		if d, err := parseInt(matches[2]); err == nil {
			day = d
		}
		if month > 0 && day > 0 {
			year := now.Year()
			if len(matches) >= 4 && matches[3] != "" {
				if y, err := parseInt(matches[3]); err == nil {
					if y < 100 {
						y += 2000
					}
					year = y
				}
			}
			return time.Date(year, time.Month(month), day, 0, 0, 0, 0, now.Location())
		}
	}
	
	return time.Time{}
}

// extractTime extracts time from text
func (p *EventParser) extractTime(text string) (time.Time, time.Duration) {
	var timeVal time.Time
	var duration time.Duration
	
	// Time patterns
	patterns := []struct {
		regex    *regexp.Regexp
		handler  func([]string) (int, int)
		duration time.Duration
	}{
		// 3pm, 3:30pm
		{regexp.MustCompile(`(?i)(\d{1,2}):(\d{2})\s*(am|pm)`), 
			func(m []string) (int, int) {
				h, _ := parseInt(m[1])
				min, _ := parseInt(m[2])
				if strings.ToLower(m[3]) == "pm" && h != 12 {
					h += 12
				}
				if strings.ToLower(m[3]) == "am" && h == 12 {
					h = 0
				}
				return h, min
			}, time.Hour},
		// 3 pm (space)
		{regexp.MustCompile(`(?i)(\d{1,2})\s*(am|pm)`), 
			func(m []string) (int, int) {
				h, _ := parseInt(m[1])
				if strings.ToLower(m[2]) == "pm" && h != 12 {
					h += 12
				}
				if strings.ToLower(m[2]) == "am" && h == 12 {
					h = 0
				}
				return h, 0
			}, time.Hour},
		// 14:30 (24h)
		{regexp.MustCompile(`\b(\d{1,2}):(\d{2})\b`), 
			func(m []string) (int, int) {
				h, _ := parseInt(m[1])
				min, _ := parseInt(m[2])
				return h, min
			}, time.Hour},
		// "noon", "midnight"
		{regexp.MustCompile(`(?i)\b(noon)\b`), 
			func(m []string) (int, int) { return 12, 0 }, time.Hour},
		{regexp.MustCompile(`(?i)\b(midnight)\b`), 
			func(m []string) (int, int) { return 0, 0 }, time.Hour},
	}
	
	for _, p := range patterns {
		matches := p.regex.FindStringSubmatch(text)
		if len(matches) > 0 {
			h, min := p.handler(matches)
			timeVal = time.Date(0, 0, 0, h, min, 0, 0, time.UTC)
			duration = p.duration
			break
		}
	}
	
	// Look for duration mentions
	durationPatterns := []struct {
		regex    *regexp.Regexp
		duration time.Duration
	}{
		{regexp.MustCompile(`(?i)\b(30\s*min|half\s*hour)\b`), 30 * time.Minute},
		{regexp.MustCompile(`(?i)\b(1\s*hour|one\s*hour)\b`), time.Hour},
		{regexp.MustCompile(`(?i)\b(2\s*hours|two\s*hours)\b`), 2 * time.Hour},
		{regexp.MustCompile(`(?i)\b(15\s*min|quarter\s*hour)\b`), 15 * time.Minute},
		{regexp.MustCompile(`(?i)\b(45\s*min)\b`), 45 * time.Minute},
	}
	
	for _, p := range durationPatterns {
		if p.regex.MatchString(text) {
			duration = p.duration
			break
		}
	}
	
	return timeVal, duration
}

// extractLocation extracts location from text
func (p *EventParser) extractLocation(text string) string {
	patterns := []string{
		`(?i)(?:at|in)\s+(?:the\s+)?([A-Z][a-zA-Z\s]+(?:Building|Office|Room|Suite|Center|Centre|Plaza|Mall|Hotel|Restaurant|Cafe|Park|Hospital|School|University|Airport|Station))`,
		`(?i)(?:at|in)\s+(?:the\s+)?([A-Z][a-z]+(?:\s+[A-Z][a-z]+){0,2})`,
		`(?i)location[:\s]+([^,]+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			location := strings.TrimSpace(matches[1])
			// Filter out common non-locations
			if !p.isCommonNonLocation(location) {
				return location
			}
		}
	}
	
	return ""
}

// extractAttendees extracts attendee emails/names from text
func (p *EventParser) extractAttendees(text string) []string {
	attendees := []string{}
	
	// Email pattern
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emails := emailPattern.FindAllString(text, -1)
	attendees = append(attendees, emails...)
	
	// "with" pattern
	withPattern := regexp.MustCompile(`(?i)(?:with|invite|including)\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?)`)
	matches := withPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			name := strings.TrimSpace(match[1])
			if !p.isCommonNonPerson(name) {
				attendees = append(attendees, name)
			}
		}
	}
	
	return attendees
}

// extractRecurrence extracts recurrence pattern
func (p *EventParser) extractRecurrence(text string) string {
	patterns := map[string]string{
		`(?i)\bevery\s+day\b`:                "RRULE:FREQ=DAILY",
		`(?i)\bevery\s+week\b`:               "RRULE:FREQ=WEEKLY",
		`(?i)\bevery\s+month\b`:              "RRULE:FREQ=MONTHLY",
		`(?i)\bdaily\b`:                      "RRULE:FREQ=DAILY",
		`(?i)\bweekly\b`:                     "RRULE:FREQ=WEEKLY",
		`(?i)\bmonthly\b`:                    "RRULE:FREQ=MONTHLY",
		`(?i)\bevery\s+(mon|tues|wednes|thurs|fri|satur|sun)day\b`: "RRULE:FREQ=WEEKLY",
	}
	
	for pattern, rrule := range patterns {
		if regexp.MustCompile(pattern).MatchString(text) {
			// For specific days, add BYDAY
			if strings.Contains(pattern, "mon") {
				return "RRULE:FREQ=WEEKLY;BYDAY=MO"
			}
			if strings.Contains(pattern, "tues") {
				return "RRULE:FREQ=WEEKLY;BYDAY=TU"
			}
			if strings.Contains(pattern, "wednes") {
				return "RRULE:FREQ=WEEKLY;BYDAY=WE"
			}
			if strings.Contains(pattern, "thurs") {
				return "RRULE:FREQ=WEEKLY;BYDAY=TH"
			}
			if strings.Contains(pattern, "fri") {
				return "RRULE:FREQ=WEEKLY;BYDAY=FR"
			}
			return rrule
		}
	}
	
	return ""
}

// inferTimeFromContext infers time from context keywords
func (p *EventParser) inferTimeFromContext(text string, date time.Time) time.Time {
	now := p.referenceTime
	
	// Morning -> 9am
	if regexp.MustCompile(`(?i)\bmorning\b`).MatchString(text) {
		return time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, now.Location())
	}
	
	// Afternoon -> 2pm
	if regexp.MustCompile(`(?i)\bafternoon\b`).MatchString(text) {
		return time.Date(date.Year(), date.Month(), date.Day(), 14, 0, 0, 0, now.Location())
	}
	
	// Evening -> 6pm
	if regexp.MustCompile(`(?i)\bevening\b`).MatchString(text) {
		return time.Date(date.Year(), date.Month(), date.Day(), 18, 0, 0, 0, now.Location())
	}
	
	// Lunch -> 12pm
	if regexp.MustCompile(`(?i)\blunch\b`).MatchString(text) {
		return time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, now.Location())
	}
	
	// Breakfast -> 8am
	if regexp.MustCompile(`(?i)\bbreakfast\b`).MatchString(text) {
		return time.Date(date.Year(), date.Month(), date.Day(), 8, 0, 0, 0, now.Location())
	}
	
	// Dinner -> 7pm
	if regexp.MustCompile(`(?i)\bdinner\b`).MatchString(text) {
		return time.Date(date.Year(), date.Month(), date.Day(), 19, 0, 0, 0, now.Location())
	}
	
	// Default to next hour
	nextHour := now.Add(time.Hour).Truncate(time.Hour)
	return time.Date(date.Year(), date.Month(), date.Day(), nextHour.Hour(), 0, 0, 0, now.Location())
}

// inferDuration infers duration from context
func (p *EventParser) inferDuration(text string) time.Duration {
	// Meeting -> 1 hour
	if regexp.MustCompile(`(?i)\bmeeting\b`).MatchString(text) {
		return time.Hour
	}
	
	// Call -> 30 min
	if regexp.MustCompile(`(?i)\bcall\b`).MatchString(text) {
		return 30 * time.Minute
	}
	
	// Lunch/Dinner -> 1.5 hours
	if regexp.MustCompile(`(?i)\b(lunch|dinner)\b`).MatchString(text) {
		return 90 * time.Minute
	}
	
	// Appointment -> 30 min
	if regexp.MustCompile(`(?i)\bappointment\b`).MatchString(text) {
		return 30 * time.Minute
	}
	
	// Interview -> 1 hour
	if regexp.MustCompile(`(?i)\binterview\b`).MatchString(text) {
		return time.Hour
	}
	
	// Default: 1 hour
	return time.Hour
}

// isCommonNonLocation filters out common non-location words
func (p *EventParser) isCommonNonLocation(word string) bool {
	common := map[string]bool{
		"the": true, "a": true, "an": true, "my": true, "your": true,
		"this": true, "that": true, "these": true, "those": true,
		"i": true, "you": true, "he": true, "she": true, "we": true, "they": true,
		"it": true, "me": true, "him": true, "her": true, "us": true, "them": true,
		"what": true, "when": true, "where": true, "why": true, "how": true,
		"and": true, "or": true, "but": true, "so": true, "if": true,
		"meeting": true, "call": true, "appointment": true, "event": true,
	}
	return common[strings.ToLower(word)]
}

// isCommonNonPerson filters out common non-person words
func (p *EventParser) isCommonNonPerson(word string) bool {
	common := map[string]bool{
		"the": true, "me": true, "you": true, "him": true, "her": true,
		"us": true, "them": true, "it": true,
		"everyone": true, "everybody": true, "someone": true, "somebody": true,
		"anyone": true, "anybody": true, "no one": true, "nobody": true,
	}
	return common[strings.ToLower(word)]
}

// parseInt parses an integer from string
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
