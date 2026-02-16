package calendar

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store handles calendar persistence
type Store struct {
	db *gorm.DB
}

// NewStore creates a new calendar store
func NewStore(db *gorm.DB) (*Store, error) {
	store := &Store{db: db}
	
	// Auto-migrate schemas
	if err := db.AutoMigrate(&CalendarEvent{}, &Calendar{}, &CalendarCredentials{}); err != nil {
		return nil, fmt.Errorf("failed to migrate calendar schemas: %w", err)
	}
	
	// Create indexes
	store.createIndexes()
	
	return store, nil
}

// createIndexes creates database indexes
func (s *Store) createIndexes() {
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_events_user_start ON calendar_events(user_id, start_time)")
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_events_user_status ON calendar_events(user_id, status)")
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_calendars_user ON calendars(user_id)")
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_credentials_user ON calendar_credentials(user_id)")
}

// generateID generates a unique ID
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "cal_" + hex.EncodeToString(bytes)
}

// Calendar operations

// CreateCalendar creates a new calendar
func (s *Store) CreateCalendar(calendar *Calendar) error {
	if calendar.ID == "" {
		calendar.ID = generateID()
	}
	calendar.CreatedAt = time.Now()
	calendar.UpdatedAt = time.Now()
	
	return s.db.Create(calendar).Error
}

// GetCalendar retrieves a calendar by ID
func (s *Store) GetCalendar(calendarID string) (*Calendar, error) {
	var calendar Calendar
	err := s.db.Where("id = ?", calendarID).First(&calendar).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &calendar, err
}

// GetUserCalendars gets all calendars for a user
func (s *Store) GetUserCalendars(userID string) ([]Calendar, error) {
	var calendars []Calendar
	err := s.db.Where("user_id = ?", userID).Order("is_primary DESC, name ASC").Find(&calendars).Error
	return calendars, err
}

// GetPrimaryCalendar gets the primary calendar for a user
func (s *Store) GetPrimaryCalendar(userID string) (*Calendar, error) {
	var calendar Calendar
	err := s.db.Where("user_id = ? AND is_primary = ?", userID, true).First(&calendar).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &calendar, err
}

// UpdateCalendar updates a calendar
func (s *Store) UpdateCalendar(calendar *Calendar) error {
	calendar.UpdatedAt = time.Now()
	return s.db.Save(calendar).Error
}

// DeleteCalendar deletes a calendar
func (s *Store) DeleteCalendar(calendarID string) error {
	return s.db.Where("id = ?", calendarID).Delete(&Calendar{}).Error
}

// Event operations

// CreateEvent creates a new event
func (s *Store) CreateEvent(event *CalendarEvent) error {
	if event.ID == "" {
		event.ID = generateID()
	}
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()
	event.NeedsSync = true
	
	return s.db.Create(event).Error
}

// GetEvent retrieves an event by ID
func (s *Store) GetEvent(eventID string) (*CalendarEvent, error) {
	var event CalendarEvent
	err := s.db.Where("id = ?", eventID).First(&event).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &event, err
}

// UpdateEvent updates an event
func (s *Store) UpdateEvent(event *CalendarEvent) error {
	event.UpdatedAt = time.Now()
	event.NeedsSync = true
	return s.db.Save(event).Error
}

// DeleteEvent deletes an event (soft delete by marking cancelled)
func (s *Store) DeleteEvent(eventID string) error {
	return s.db.Model(&CalendarEvent{}).Where("id = ?", eventID).Updates(map[string]interface{}{
		"status":     EventStatusCancelled,
		"needs_sync": true,
		"updated_at": time.Now(),
	}).Error
}

// HardDeleteEvent permanently deletes an event
func (s *Store) HardDeleteEvent(eventID string) error {
	return s.db.Where("id = ?", eventID).Delete(&CalendarEvent{}).Error
}

// ListEvents lists events with filters
func (s *Store) ListEvents(userID string, filters EventFilters) (*EventList, error) {
	query := s.db.Where("user_id = ? AND status != ?", userID, EventStatusCancelled)
	
	// Calendar filter
	if filters.CalendarID != "" {
		query = query.Where("calendar_id = ?", filters.CalendarID)
	}
	
	// Time range filters
	if filters.StartAfter != nil {
		query = query.Where("start_time >= ?", filters.StartAfter)
	}
	if filters.StartBefore != nil {
		query = query.Where("start_time <= ?", filters.StartBefore)
	}
	
	// Search
	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		query = query.Where(
			"title LIKE ? OR description LIKE ? OR location LIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}
	
	// Order by start time
	query = query.Order("start_time ASC")
	
	// Pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	
	var events []CalendarEvent
	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}
	
	// Calculate stats
	now := time.Now()
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	weekEnd := todayEnd.AddDate(0, 0, 7)
	
	var today, thisWeek, upcoming int
	for _, e := range events {
		if e.StartTime.Before(todayEnd) && e.StartTime.After(now.Add(-24*time.Hour)) {
			today++
		}
		if e.StartTime.Before(weekEnd) && e.StartTime.After(now) {
			thisWeek++
		}
		if e.StartTime.After(now) {
			upcoming++
		}
	}
	
	return &EventList{
		Events:    events,
		Total:     len(events),
		Today:     today,
		ThisWeek:  thisWeek,
		Upcoming:  upcoming,
	}, nil
}

// EventFilters contains filters for listing events
type EventFilters struct {
	CalendarID   string
	StartAfter   *time.Time
	StartBefore  *time.Time
	EndAfter     *time.Time
	EndBefore    *time.Time
	Search       string
	Limit        int
}

// GetEventsForDay gets all events for a specific day
func (s *Store) GetEventsForDay(userID string, day time.Time) ([]CalendarEvent, error) {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := start.AddDate(0, 0, 1)
	
	var events []CalendarEvent
	err := s.db.Where(
		"user_id = ? AND status != ? AND start_time < ? AND end_time > ?",
		userID, EventStatusCancelled, end, start,
	).Order("start_time ASC").Find(&events).Error
	
	return events, err
}

// GetEventsForWeek gets events for a week
func (s *Store) GetEventsForWeek(userID string, weekStart time.Time) ([]CalendarEvent, error) {
	start := time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
	end := start.AddDate(0, 0, 7)
	
	var events []CalendarEvent
	err := s.db.Where(
		"user_id = ? AND status != ? AND start_time >= ? AND start_time < ?",
		userID, EventStatusCancelled, start, end,
	).Order("start_time ASC").Find(&events).Error
	
	return events, err
}

// GetUpcomingEvents gets upcoming events
func (s *Store) GetUpcomingEvents(userID string, limit int) ([]CalendarEvent, error) {
	now := time.Now()
	
	var events []CalendarEvent
	err := s.db.Where(
		"user_id = ? AND status != ? AND start_time >= ?",
		userID, EventStatusCancelled, now,
	).Order("start_time ASC").Limit(limit).Find(&events).Error
	
	return events, err
}

// GetEventsNeedingSync gets events that need to be synced
func (s *Store) GetEventsNeedingSync(userID string, limit int) ([]CalendarEvent, error) {
	var events []CalendarEvent
	err := s.db.Where("user_id = ? AND needs_sync = ?", userID, true).
		Limit(limit).Find(&events).Error
	return events, err
}

// FindConflicts finds events that conflict with a time range
func (s *Store) FindConflicts(userID string, start, end time.Time, excludeID string) ([]CalendarEvent, error) {
	query := s.db.Where(
		"user_id = ? AND status != ? AND start_time < ? AND end_time > ?",
		userID, EventStatusCancelled, end, start,
	)
	
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	
	var events []CalendarEvent
	err := query.Order("start_time ASC").Find(&events).Error
	return events, err
}

// Credential operations

// SaveCredentials saves calendar credentials
func (s *Store) SaveCredentials(creds *CalendarCredentials) error {
	if creds.ID == "" {
		creds.ID = generateID()
		creds.CreatedAt = time.Now()
	}
	creds.UpdatedAt = time.Now()
	
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "provider"}},
		UpdateAll: true,
	}).Create(creds).Error
}

// GetCredentials retrieves credentials for a user and provider
func (s *Store) GetCredentials(userID, provider string) (*CalendarCredentials, error) {
	var creds CalendarCredentials
	err := s.db.Where("user_id = ? AND provider = ?", userID, provider).First(&creds).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &creds, err
}

// GetActiveCredentials gets all active credentials for a user
func (s *Store) GetActiveCredentials(userID string) ([]CalendarCredentials, error) {
	var creds []CalendarCredentials
	err := s.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&creds).Error
	return creds, err
}

// DeleteCredentials deletes credentials
func (s *Store) DeleteCredentials(credsID string) error {
	return s.db.Where("id = ?", credsID).Delete(&CalendarCredentials{}).Error
}

// MarkEventSynced marks an event as synced
func (s *Store) MarkEventSynced(eventID string, externalID string) error {
	now := time.Now()
	return s.db.Model(&CalendarEvent{}).Where("id = ?", eventID).Updates(map[string]interface{}{
		"needs_sync":   false,
		"last_synced":  &now,
		"source_id":    externalID,
		"updated_at":   now,
	}).Error
}

// UpdateSyncToken updates the sync token for a calendar
func (s *Store) UpdateSyncToken(calendarID string, syncToken string) error {
	return s.db.Model(&Calendar{}).Where("id = ?", calendarID).Updates(map[string]interface{}{
		"sync_token":  syncToken,
		"last_synced": time.Now(),
	}).Error
}

// GetStats gets calendar statistics
func (s *Store) GetStats(userID string) (*CalendarStats, error) {
	stats := &CalendarStats{
		TopCategories: make(map[string]int),
	}
	
	now := time.Now()
	weekStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart = weekStart.AddDate(0, 0, -int(weekStart.Weekday())) // Start of week (Sunday)
	weekEnd := weekStart.AddDate(0, 0, 7)
	nextWeekStart := weekEnd
	nextWeekEnd := nextWeekStart.AddDate(0, 0, 7)
	
	// Total events
	var total int64
	s.db.Model(&CalendarEvent{}).Where("user_id = ? AND status != ?", userID, EventStatusCancelled).Count(&total)
	stats.TotalEvents = int(total)
	
	// This week
	var thisWeek int64
	s.db.Model(&CalendarEvent{}).Where(
		"user_id = ? AND status != ? AND start_time >= ? AND start_time < ?",
		userID, EventStatusCancelled, weekStart, weekEnd,
	).Count(&thisWeek)
	stats.ThisWeekEvents = int(thisWeek)
	
	// Next week
	var nextWeek int64
	s.db.Model(&CalendarEvent{}).Where(
		"user_id = ? AND status != ? AND start_time >= ? AND start_time < ?",
		userID, EventStatusCancelled, nextWeekStart, nextWeekEnd,
	).Count(&nextWeek)
	stats.NextWeekEvents = int(nextWeek)
	
	// Hours this week
	var weekEvents []CalendarEvent
	s.db.Where(
		"user_id = ? AND status != ? AND start_time >= ? AND start_time < ?",
		userID, EventStatusCancelled, weekStart, weekEnd,
	).Find(&weekEvents)
	
	var totalHours float64
	dayCounts := make(map[string]int)
	for _, e := range weekEvents {
		hours := e.Duration().Hours()
		totalHours += hours
		day := e.StartTime.Weekday().String()
		dayCounts[day]++
	}
	stats.HoursThisWeek = totalHours
	
	// Busiest day
	busiestDay := ""
	maxCount := 0
	for day, count := range dayCounts {
		if count > maxCount {
			maxCount = count
			busiestDay = day
		}
	}
	stats.BusiestDay = busiestDay
	
	return stats, nil
}

// UpsertEvent creates or updates an event
func (s *Store) UpsertEvent(event *CalendarEvent) error {
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(event).Error
}
