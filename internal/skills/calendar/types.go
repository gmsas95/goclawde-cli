package calendar

import (
	"fmt"
	"time"
)

// CalendarEvent represents a calendar event
type CalendarEvent struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	CalendarID  string    `json:"calendar_id"` // Google Calendar ID
	
	// Event details
	Title       string    `json:"title"`
	Description string    `json:"description" gorm:"type:text"`
	Location    string    `json:"location"`
	
	// Timing
	StartTime   time.Time `json:"start_time" gorm:"index"`
	EndTime     time.Time `json:"end_time"`
	AllDay      bool      `json:"all_day"`
	Timezone    string    `json:"timezone"`
	
	// Recurrence
	IsRecurring    bool   `json:"is_recurring"`
	RecurrenceRule string `json:"recurrence_rule,omitempty"` // RRULE format
	RecurringEventID string `json:"recurring_event_id,omitempty"`
	
	// Attendees
	Attendees   string    `json:"attendees"` // JSON array of attendees
	Organizer   string    `json:"organizer,omitempty"`
	
	// Status
	Status      EventStatus `json:"status"` // confirmed, tentative, cancelled
	Visibility  string      `json:"visibility"` // default, public, private, confidential
	
	// Conference data (Google Meet, Zoom, etc.)
	ConferenceURL   string `json:"conference_url,omitempty"`
	ConferenceType  string `json:"conference_type,omitempty"`
	
	// Source tracking
	Source      string    `json:"source"` // google, manual, extracted
	SourceID    string    `json:"source_id,omitempty"` // Original ID from source
	
	// Metadata
	ColorID     string    `json:"color_id,omitempty"`
	Reminders   string    `json:"reminders"` // JSON reminder settings
	
	// Sync tracking
	LastSynced  *time.Time `json:"last_synced,omitempty"`
	NeedsSync   bool       `json:"needs_sync"`
	
	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// EventStatus represents event status
type EventStatus string

const (
	EventStatusConfirmed  EventStatus = "confirmed"
	EventStatusTentative  EventStatus = "tentative"
	EventStatusCancelled  EventStatus = "cancelled"
)

// Attendee represents an event attendee
type Attendee struct {
	Email          string `json:"email"`
	Name           string `json:"name,omitempty"`
	ResponseStatus string `json:"response_status"` // needsAction, declined, tentative, accepted
	Optional       bool   `json:"optional"`
	Organizer      bool   `json:"organizer"`
}

// Reminder represents an event reminder
type Reminder struct {
	Method  string `json:"method"` // email, popup
	Minutes int    `json:"minutes"`
}

// Calendar represents a user's calendar
type Calendar struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	
	// Calendar details
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Timezone    string    `json:"timezone"`
	Color       string    `json:"color,omitempty"`
	
	// Source
	Provider    string    `json:"provider"` // google, apple, outlook
	ExternalID  string    `json:"external_id"`
	
	// Settings
	IsPrimary   bool      `json:"is_primary"`
	IsVisible   bool      `json:"is_visible"`
	IsWritable  bool      `json:"is_writable"`
	
	// Sync tracking
	LastSynced  *time.Time `json:"last_synced,omitempty"`
	SyncToken   string     `json:"sync_token,omitempty"` // For incremental sync
	
	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CalendarCredentials stores OAuth credentials
type CalendarCredentials struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	UserID       string    `json:"user_id" gorm:"index"`
	
	// Provider info
	Provider     string    `json:"provider"` // google, apple, outlook
	
	// OAuth tokens
	AccessToken  string    `json:"access_token" gorm:"type:text"`
	RefreshToken string    `json:"refresh_token" gorm:"type:text"`
	TokenExpiry  time.Time `json:"token_expiry"`
	
	// Scopes granted
	Scopes       string    `json:"scopes"` // Comma-separated
	
	// Status
	IsActive     bool      `json:"is_active"`
	LastError    string    `json:"last_error,omitempty"`
	
	// Timestamps
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EventList represents a list of events
type EventList struct {
	Events      []CalendarEvent `json:"events"`
	Total       int             `json:"total"`
	Today       int             `json:"today"`
	ThisWeek    int             `json:"this_week"`
	Upcoming    int             `json:"upcoming"`
}

// FreeBusySlot represents a free/busy time slot
type FreeBusySlot struct {
	Start   time.Time `json:"start"`
	End     time.Time `json:"end"`
	Busy    bool      `json:"busy"`
	Title   string    `json:"title,omitempty"`
}

// ScheduleSuggestion represents a suggested meeting time
type ScheduleSuggestion struct {
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Score       float64   `json:"score"` // 0-1 preference score
	Conflicts   []string  `json:"conflicts,omitempty"`
	Reason      string    `json:"reason"`
}

// ParseResult represents the result of parsing a natural language event
type ParseResult struct {
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Location    string    `json:"location,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	AllDay      bool      `json:"all_day"`
	Attendees   []string  `json:"attendees,omitempty"`
	IsRecurring bool      `json:"is_recurring"`
	Recurrence  string    `json:"recurrence,omitempty"`
	Confidence  float64   `json:"confidence"`
	RawText     string    `json:"raw_text"`
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	Added       int       `json:"added"`
	Updated     int       `json:"updated"`
	Deleted     int       `json:"deleted"`
	Errors      []string  `json:"errors,omitempty"`
	NextSyncToken string  `json:"next_sync_token,omitempty"`
}

// CalendarStats represents calendar statistics
type CalendarStats struct {
	TotalEvents      int            `json:"total_events"`
	ThisWeekEvents   int            `json:"this_week_events"`
	NextWeekEvents   int            `json:"next_week_events"`
	HoursThisWeek    float64        `json:"hours_this_week"`
	TopCategories    map[string]int `json:"top_categories"`
	BusiestDay       string         `json:"busiest_day"`
}

// Helper methods

// IsPast returns true if the event is in the past
func (e *CalendarEvent) IsPast() bool {
	return e.EndTime.Before(time.Now())
}

// IsUpcoming returns true if the event is upcoming (within next 24 hours)
func (e *CalendarEvent) IsUpcoming() bool {
	now := time.Now()
	return e.StartTime.After(now) && e.StartTime.Before(now.Add(24*time.Hour))
}

// IsToday returns true if the event is today
func (e *CalendarEvent) IsToday() bool {
	now := time.Now()
	return e.StartTime.Year() == now.Year() &&
		e.StartTime.Month() == now.Month() &&
		e.StartTime.Day() == now.Day()
}

// Duration returns the event duration
func (e *CalendarEvent) Duration() time.Duration {
	return e.EndTime.Sub(e.StartTime)
}

// ConflictsWith checks if this event conflicts with another
func (e *CalendarEvent) ConflictsWith(other *CalendarEvent) bool {
	return e.StartTime.Before(other.EndTime) && e.EndTime.After(other.StartTime)
}

// NeedsReminder returns true if a reminder should be sent
func (e *CalendarEvent) NeedsReminder(now time.Time) bool {
	if e.IsPast() {
		return false
	}
	
	// Parse reminders
	// For now, check if within 15 minutes of start
	return now.After(e.StartTime.Add(-15*time.Minute)) && now.Before(e.StartTime)
}

// IsRecurringInstance returns true if this is an instance of a recurring event
func (e *CalendarEvent) IsRecurringInstance() bool {
	return e.RecurringEventID != ""
}

// GetAttendeeList parses the attendees JSON
func (e *CalendarEvent) GetAttendeeList() []Attendee {
	// Would parse JSON in real implementation
	return []Attendee{}
}

// FormatDuration formats the event duration for display
func (e *CalendarEvent) FormatDuration() string {
	d := e.Duration()
	if d < time.Hour {
		return fmt.Sprintf("%d min", int(d.Minutes()))
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins == 0 {
		return fmt.Sprintf("%d hr", hours)
	}
	return fmt.Sprintf("%d hr %d min", hours, mins)
}

// FormatTimeRange formats the event time range
func (e *CalendarEvent) FormatTimeRange() string {
	if e.AllDay {
		return e.StartTime.Format("Mon, Jan 2") + " (All day)"
	}
	
	startStr := e.StartTime.Format("Mon, Jan 2 3:04 PM")
	endStr := e.EndTime.Format("3:04 PM")
	
	if e.StartTime.Day() != e.EndTime.Day() {
		endStr = e.EndTime.Format("Mon, Jan 2 3:04 PM")
	}
	
	return startStr + " - " + endStr
}

// HasConference returns true if the event has a conference link
func (e *CalendarEvent) HasConference() bool {
	return e.ConferenceURL != ""
}

// GetConferenceProvider returns the conference provider name
func (e *CalendarEvent) GetConferenceProvider() string {
	switch e.ConferenceType {
	case "google_meet":
		return "Google Meet"
	case "zoom":
		return "Zoom"
	case "teams":
		return "Microsoft Teams"
	default:
		if e.ConferenceURL != "" {
			return "Video call"
		}
		return ""
	}
}

