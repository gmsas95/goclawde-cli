package health

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParsedMedication represents a parsed medication input
type ParsedMedication struct {
	Name        string
	Dosage      string
	Form        string
	Frequency   string
	Times       []string
	DaysOfWeek  []int
	WithFood    bool
	BeforeBed   bool
	Notes       string
}

// ParsedMetric represents a parsed health metric input
type ParsedMetric struct {
	Type       string
	Value      float64
	Unit       string
	SubType    string // systolic/diastolic for BP
	Context    string // morning, evening, before_meal, etc.
	Notes      string
}

// ParsedAppointment represents a parsed appointment input
type ParsedAppointment struct {
	Title       string
	Provider    string
	Specialty   string
	DateTime    time.Time
	Duration    int
	Location    string
	Type        string
	Notes       string
}

// Parser handles natural language parsing for health data
type Parser struct {
	// Frequency keywords
	frequencyKeywords map[string]string
	// Time patterns
	timePatterns []string
	// Metric type patterns
	metricPatterns map[string][]string
}

// NewParser creates a new health parser
func NewParser() *Parser {
	return &Parser{
		frequencyKeywords: map[string]string{
			"daily":         "daily",
			"every day":     "daily",
			"once a day":    "daily",
			"twice a day":   "twice_daily",
			"three times":   "three_times",
			"four times":    "four_times",
			"weekly":        "weekly",
			"every week":    "weekly",
			"monthly":       "monthly",
			"as needed":     "as_needed",
			"when needed":   "as_needed",
			"prn":           "as_needed",
			"before bed":    "before_bed",
			"at bedtime":    "before_bed",
		},
		timePatterns: []string{
			`(\d{1,2}):(\d{2})\s*(am|pm)?`,
			`(\d{1,2})\s*(am|pm)`,
			`morning`,
			`evening`,
			`noon`,
			`midnight`,
			`breakfast`,
			`lunch`,
			`dinner`,
		},
		metricPatterns: map[string][]string{
			"weight":        {"weight", "weigh", "pounds", "lbs", "kg", "kilograms"},
			"blood_pressure": {"blood pressure", "bp", "systolic", "diastolic"},
			"heart_rate":    {"heart rate", "pulse", "bpm", "heartbeat"},
			"temperature":   {"temperature", "fever", "temp"},
			"blood_sugar":   {"blood sugar", "glucose", "sugar level"},
			"sleep":         {"sleep", "slept", "hours of sleep"},
			"steps":         {"steps", "walked", "pedometer"},
			"water":         {"water", "glasses", "liters of water", "oz of water"},
		},
	}
}

// ParseMedication parses natural language medication input
func (p *Parser) ParseMedication(text string) *ParsedMedication {
	result := &ParsedMedication{
		Times:      []string{},
		DaysOfWeek: []int{},
	}
	
	text = strings.ToLower(strings.TrimSpace(text))
	
	// Extract name and dosage
	result.Name, result.Dosage = p.extractNameAndDosage(text)
	
	// Extract form
	result.Form = p.extractForm(text)
	
	// Extract frequency
	result.Frequency = p.extractFrequency(text)
	
	// Extract times
	result.Times = p.extractTimes(text)
	
	// Extract days
	result.DaysOfWeek = p.extractDaysOfWeek(text)
	
	// Extract instructions
	result.WithFood = strings.Contains(text, "with food") || strings.Contains(text, "after meal") || strings.Contains(text, "with meals")
	result.BeforeBed = strings.Contains(text, "before bed") || strings.Contains(text, "at bedtime") || strings.Contains(text, "night")
	
	return result
}

func (p *Parser) extractNameAndDosage(text string) (name, dosage string) {
	// Match patterns like "Lisinopril 10mg" or "Metformin 500 mg tablet"
	re := regexp.MustCompile(`(?i)([a-z\s]+)\s+(\d+\s*(?:mg|mcg|g|ml|units?|tablets?|capsules?|pills?))`)
	matches := re.FindStringSubmatch(text)
	
	if len(matches) >= 3 {
		return strings.TrimSpace(matches[1]), matches[2]
	}
	
	// Try just name
	re = regexp.MustCompile(`(?i)^([a-z]+)`)
	matches = re.FindStringSubmatch(text)
	if len(matches) >= 2 {
		return matches[1], ""
	}
	
	return text, ""
}

func (p *Parser) extractForm(text string) string {
	forms := []string{"tablet", "capsule", "pill", "liquid", "injection", "inhaler", "cream", "ointment", "drops", "syrup"}
	
	for _, form := range forms {
		if strings.Contains(text, form) {
			return form
		}
	}
	return ""
}

func (p *Parser) extractFrequency(text string) string {
	for keyword, freq := range p.frequencyKeywords {
		if strings.Contains(text, keyword) {
			return freq
		}
	}
	return "daily" // default
}

func (p *Parser) extractTimes(text string) []string {
	var times []string
	
	// Extract specific times like "8am", "2:30 pm", "20:00"
	re := regexp.MustCompile(`(?i)(\d{1,2}):(\d{2})\s*(am|pm)?`)
	matches := re.FindAllStringSubmatch(text, -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			hour, _ := strconv.Atoi(match[1])
			minute := match[2]
			ampm := ""
			if len(match) >= 4 {
				ampm = strings.ToLower(match[3])
			}
			
			// Convert to 24-hour format
			if ampm == "pm" && hour != 12 {
				hour += 12
			}
			if ampm == "am" && hour == 12 {
				hour = 0
			}
			
			times = append(times, fmt.Sprintf("%02s:%s", strconv.Itoa(hour), minute))
		}
	}
	
	// Check for keywords
	if strings.Contains(text, "morning") && !contains(times, "08:00") {
		times = append(times, "08:00")
	}
	if strings.Contains(text, "noon") && !contains(times, "12:00") {
		times = append(times, "12:00")
	}
	if strings.Contains(text, "evening") && !contains(times, "18:00") {
		times = append(times, "18:00")
	}
	if strings.Contains(text, "bedtime") && !contains(times, "22:00") {
		times = append(times, "22:00")
	}
	
	return times
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (p *Parser) extractDaysOfWeek(text string) []int {
	days := []int{}
	
	dayMap := map[string]int{
		"sunday":    0,
		"monday":    1,
		"tuesday":   2,
		"wednesday": 3,
		"thursday":  4,
		"friday":    5,
		"saturday":  6,
	}
	
	for day, num := range dayMap {
		if strings.Contains(text, day) {
			days = append(days, num)
		}
	}
	
	return days
}

// ParseMetric parses natural language metric input
func (p *Parser) ParseMetric(text string) *ParsedMetric {
	result := &ParsedMetric{}
	
	text = strings.ToLower(strings.TrimSpace(text))
	
	// Extract metric type
	result.Type = p.extractMetricType(text)
	
	// Extract value and unit
	result.Value, result.Unit = p.extractValueAndUnit(text)
	
	// Extract context
	result.Context = p.extractContext(text)
	
	// Handle special cases
	if result.Type == "blood_pressure" {
		result.SubType = p.extractBPType(text)
	}
	
	return result
}

func (p *Parser) extractMetricType(text string) string {
	for metricType, keywords := range p.metricPatterns {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				return metricType
			}
		}
	}
	return ""
}

func (p *Parser) extractValueAndUnit(text string) (float64, string) {
	// Pattern: number followed by unit
	re := regexp.MustCompile(`(?i)(\d+\.?\d*)\s*(lbs?|pounds?|kg|kilograms?|mg|milligrams?|mcg|micrograms?|mmhg|bpm|degrees?|°[cf]?|glasses?|oz|ounces?|liters?|litres?|l)\b`)
	matches := re.FindStringSubmatch(text)
	
	if len(matches) >= 3 {
		val, _ := strconv.ParseFloat(matches[1], 64)
		unit := matches[2]
		
		// Normalize units
		if strings.HasPrefix(unit, "lb") || strings.HasPrefix(unit, "pound") {
			return val, "lbs"
		}
		if strings.HasPrefix(unit, "kg") || strings.HasPrefix(unit, "kilogram") {
			return val, "kg"
		}
		if strings.HasPrefix(unit, "glass") {
			return val, "glasses"
		}
		if strings.HasPrefix(unit, "oz") || strings.HasPrefix(unit, "ounce") {
			return val, "oz"
		}
		if unit == "l" || strings.HasPrefix(unit, "liter") || strings.HasPrefix(unit, "litre") {
			return val, "L"
		}
		
		return val, unit
	}
	
	// Just number
	re = regexp.MustCompile(`(\d+\.?\d*)`)
	matches = re.FindStringSubmatch(text)
	if len(matches) >= 2 {
		val, _ := strconv.ParseFloat(matches[1], 64)
		return val, ""
	}
	
	return 0, ""
}

func (p *Parser) extractContext(text string) string {
	contexts := map[string]string{
		"morning":     "morning",
		"evening":     "evening",
		"afternoon":   "afternoon",
		"night":       "night",
		"before meal": "before_meal",
		"after meal":  "after_meal",
		"fasting":     "fasting",
		"resting":     "resting",
		"after exercise": "after_exercise",
	}
	
	for keyword, context := range contexts {
		if strings.Contains(text, keyword) {
			return context
		}
	}
	return ""
}

func (p *Parser) extractBPType(text string) string {
	if strings.Contains(text, "systolic") {
		return "systolic"
	}
	if strings.Contains(text, "diastolic") {
		return "diastolic"
	}
	return ""
}

// ParseAppointment parses natural language appointment input
func (p *Parser) ParseAppointment(text string) *ParsedAppointment {
	result := &ParsedAppointment{
		Duration: 30, // default 30 minutes
	}
	
	text = strings.ToLower(strings.TrimSpace(text))
	
	// Extract title/type
	result.Title, result.Type = p.extractAppointmentTitle(text)
	
	// Extract provider and specialty
	result.Provider, result.Specialty = p.extractProvider(text)
	
	// Extract date/time
	result.DateTime = p.extractDateTime(text)
	
	// Extract location
	result.Location = p.extractLocation(text)
	
	// Extract duration
	result.Duration = p.extractDuration(text)
	
	return result
}

func (p *Parser) extractAppointmentTitle(text string) (title, apptType string) {
	// Look for common appointment types
	types := map[string]string{
		"checkup":        "checkup",
		"physical":       "checkup",
		"annual":         "checkup",
		"dentist":        "dentist",
		"dental":         "dentist",
		"eye exam":       "eye_exam",
		"vision":         "eye_exam",
		"vaccination":    "vaccination",
		"shot":           "vaccination",
		"vaccine":        "vaccination",
		"blood test":     "test",
		"lab work":       "test",
		"x-ray":          "test",
		"mri":            "test",
		"specialist":     "specialist",
		"follow up":      "follow_up",
		"follow-up":      "follow_up",
	}
	
	for keyword, t := range types {
		if strings.Contains(text, keyword) {
			return strings.Title(keyword), t
		}
	}
	
	return "Doctor Appointment", "checkup"
}

func (p *Parser) extractProvider(text string) (provider, specialty string) {
	// Look for "with Dr. Name" or "see Dr. Name"
	re := regexp.MustCompile(`(?i)(?:with|see|at)\s+(dr\.?\s+)?([a-z\s]+)`)
	matches := re.FindStringSubmatch(text)
	
	if len(matches) >= 3 {
		provider = strings.TrimSpace(matches[2])
	}
	
	// Extract specialty
	specialties := []string{
		"cardiologist", "dermatologist", "neurologist", "orthopedist",
		"pediatrician", "psychiatrist", "therapist", "dentist", "optometrist",
	}
	
	for _, spec := range specialties {
		if strings.Contains(text, spec) {
			specialty = spec
			break
		}
	}
	
	return provider, specialty
}

func (p *Parser) extractDateTime(text string) time.Time {
	// Look for "on Monday at 2pm", "tomorrow at 10am", "next week"
	now := time.Now()
	
	// Tomorrow
	if strings.Contains(text, "tomorrow") {
		return now.AddDate(0, 0, 1)
	}
	
	// Next week
	if strings.Contains(text, "next week") {
		return now.AddDate(0, 0, 7)
	}
	
	// Day of week
	days := map[string]int{
		"monday": 1, "tuesday": 2, "wednesday": 3, "thursday": 4,
		"friday": 5, "saturday": 6, "sunday": 0,
	}
	
	for day, dayNum := range days {
		if strings.Contains(text, day) {
			daysUntil := (dayNum - int(now.Weekday()) + 7) % 7
			if daysUntil == 0 {
				daysUntil = 7 // Next week if today
			}
			return now.AddDate(0, 0, daysUntil)
		}
	}
	
	// Default to tomorrow
	return now.AddDate(0, 0, 1)
}

func (p *Parser) extractLocation(text string) string {
	// Look for "at Location" or "Location Hospital"
	re := regexp.MustCompile(`(?i)(?:at|location)\s+([a-z\s]+(?:hospital|clinic|center|office|medical))`)
	matches := re.FindStringSubmatch(text)
	
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	
	return ""
}

func (p *Parser) extractDuration(text string) int {
	// Look for "for 30 minutes", "1 hour"
	re := regexp.MustCompile(`(?i)(\d+)\s*(minute|hour|hr)`)
	matches := re.FindStringSubmatch(text)
	
	if len(matches) >= 3 {
		val, _ := strconv.Atoi(matches[1])
		if strings.Contains(matches[2], "hour") {
			return val * 60
		}
		return val
	}
	
	return 30 // default
}

// ParseTimeRange parses time range inputs like "last 7 days", "this month", etc.
func (p *Parser) ParseTimeRange(text string) (start, end time.Time) {
	now := time.Now()
	end = now
	
	text = strings.ToLower(text)
	
	switch {
	case strings.Contains(text, "today"):
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case strings.Contains(text, "yesterday"):
		start = time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case strings.Contains(text, "last 7 days") || strings.Contains(text, "past week"):
		start = now.AddDate(0, 0, -7)
	case strings.Contains(text, "last 30 days") || strings.Contains(text, "past month"):
		start = now.AddDate(0, 0, -30)
	case strings.Contains(text, "this month"):
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case strings.Contains(text, "last month"):
		start = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		start = now.AddDate(0, 0, -7) // Default to last 7 days
	}
	
	return start, end
}
