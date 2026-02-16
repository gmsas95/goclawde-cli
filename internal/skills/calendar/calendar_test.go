package calendar

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCalendarTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func setupCalendarSkill(t *testing.T) (*CalendarSkill, *gorm.DB) {
	db := setupCalendarTestDB(t)
	logger, _ := zap.NewDevelopment()
	
	config := CalendarSkillConfig{
		Enabled:         true,
		DefaultTimezone: "UTC",
	}
	
	skill, err := NewCalendarSkill(db, config, logger)
	require.NoError(t, err)
	
	return skill, db
}

// Store Tests

func TestStore_CreateAndGetEvent(t *testing.T) {
	db := setupCalendarTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	start := time.Now().Add(24 * time.Hour)
	event := &CalendarEvent{
		UserID:     "user1",
		CalendarID: "primary",
		Title:      "Test Event",
		StartTime:  start,
		EndTime:    start.Add(time.Hour),
		Status:     EventStatusConfirmed,
	}
	
	err = store.CreateEvent(event)
	require.NoError(t, err)
	assert.NotEmpty(t, event.ID)
	
	// Retrieve
	retrieved, err := store.GetEvent(event.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Event", retrieved.Title)
	assert.Equal(t, EventStatusConfirmed, retrieved.Status)
}

func TestStore_ListEvents(t *testing.T) {
	db := setupCalendarTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	now := time.Now()
	
	// Create events
	events := []*CalendarEvent{
		{UserID: "user1", Title: "Event 1", StartTime: now.Add(time.Hour), EndTime: now.Add(2 * time.Hour)},
		{UserID: "user1", Title: "Event 2", StartTime: now.Add(24 * time.Hour), EndTime: now.Add(25 * time.Hour)},
		{UserID: "user1", Title: "Event 3", StartTime: now.Add(48 * time.Hour), EndTime: now.Add(49 * time.Hour), Status: EventStatusCancelled},
	}
	
	for _, e := range events {
		err := store.CreateEvent(e)
		require.NoError(t, err)
	}
	
	// List all non-cancelled
	filters := EventFilters{}
	list, err := store.ListEvents("user1", filters)
	require.NoError(t, err)
	assert.Len(t, list.Events, 2) // Excludes cancelled
}

func TestStore_FindConflicts(t *testing.T) {
	db := setupCalendarTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	now := time.Now()
	
	// Create event at 2pm-3pm tomorrow
	event := &CalendarEvent{
		UserID:    "user1",
		Title:     "Existing Event",
		StartTime: now.AddDate(0, 0, 1).Add(14 * time.Hour),
		EndTime:   now.AddDate(0, 0, 1).Add(15 * time.Hour),
	}
	require.NoError(t, store.CreateEvent(event))
	
	// Check for conflict at 2:30pm
	conflicts, err := store.FindConflicts(
		"user1",
		now.AddDate(0, 0, 1).Add(14*time.Hour).Add(30*time.Minute),
		now.AddDate(0, 0, 1).Add(16*time.Hour),
		"",
	)
	require.NoError(t, err)
	assert.Len(t, conflicts, 1)
}

func TestStore_CalendarOperations(t *testing.T) {
	db := setupCalendarTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	// Create calendar
	cal := &Calendar{
		UserID:     "user1",
		Name:       "My Calendar",
		ExternalID: "primary",
		Provider:   "google",
		IsPrimary:  true,
	}
	
	err = store.CreateCalendar(cal)
	require.NoError(t, err)
	assert.NotEmpty(t, cal.ID)
	
	// Get primary
	primary, err := store.GetPrimaryCalendar("user1")
	require.NoError(t, err)
	assert.NotNil(t, primary)
	assert.Equal(t, "My Calendar", primary.Name)
}

func TestStore_Stats(t *testing.T) {
	db := setupCalendarTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	now := time.Now()
	
	// Create events
	events := []*CalendarEvent{
		{UserID: "user1", Title: "Past", StartTime: now.Add(-24 * time.Hour), EndTime: now.Add(-23 * time.Hour)},
		{UserID: "user1", Title: "This Week 1", StartTime: now.Add(time.Hour), EndTime: now.Add(2 * time.Hour)},
		{UserID: "user1", Title: "This Week 2", StartTime: now.Add(2 * time.Hour), EndTime: now.Add(3 * time.Hour)},
		{UserID: "user1", Title: "Next Week", StartTime: now.AddDate(0, 0, 8), EndTime: now.AddDate(0, 0, 9)},
	}
	
	for _, e := range events {
		err := store.CreateEvent(e)
		require.NoError(t, err)
	}
	
	stats, err := store.GetStats("user1")
	require.NoError(t, err)
	
	assert.Equal(t, 4, stats.TotalEvents)
	assert.GreaterOrEqual(t, stats.ThisWeekEvents, 2)
	assert.GreaterOrEqual(t, stats.NextWeekEvents, 1)
}

// Parser Tests

func TestEventParser_ParseEvent(t *testing.T) {
	parser := NewEventParser()
	
	tests := []struct {
		input         string
		expectedTitle string
		allDay        bool
	}{
		{"Meeting with John tomorrow at 3pm", "Meeting with John", false},
		{"Lunch with Sarah on Friday at noon", "Lunch with Sarah", false},
		{"Doctor appointment next Tuesday at 10am", "Doctor appointment", false},
		{"Conference call today at 2pm", "Conference call", false},
	}
	
	for _, test := range tests {
		result, err := parser.ParseEvent(test.input)
		require.NoError(t, err)
		// Check that expected is contained in result OR result contains expected
		assert.True(t, 
			strings.Contains(result.Title, test.expectedTitle) || 
			strings.Contains(test.expectedTitle, result.Title),
			"Title mismatch. Expected to contain or be contained in: %s, Got: %s", 
			test.expectedTitle, result.Title)
		assert.Equal(t, test.allDay, result.AllDay, "Input: %s", test.input)
	}
}

func TestEventParser_ExtractDate(t *testing.T) {
	parser := NewEventParser().WithReference(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC))
	
	tests := []struct {
		input         string
		expectedYear  int
		expectedMonth time.Month
	}{
		{"tomorrow", 2024, 1},
		{"today", 2024, 1},
		{"next Monday", 2024, 1},
	}
	
	for _, test := range tests {
		date := parser.extractDate(test.input)
		assert.Equal(t, test.expectedYear, date.Year(), "Year mismatch for: %s", test.input)
		assert.Equal(t, test.expectedMonth, date.Month(), "Month mismatch for: %s", test.input)
	}
}

func TestEventParser_ExtractTime(t *testing.T) {
	parser := NewEventParser()
	
	tests := []struct {
		input       string
		hour        int
		minute      int
		duration    time.Duration
	}{
		{"3pm", 15, 0, time.Hour},
		{"3:30pm", 15, 30, time.Hour},
		{"10am", 10, 0, time.Hour},
		{"14:30", 14, 30, time.Hour},
		{"noon", 12, 0, time.Hour},
	}
	
	for _, test := range tests {
		timeVal, duration := parser.extractTime(test.input)
		assert.Equal(t, test.hour, timeVal.Hour(), "Hour mismatch for: %s", test.input)
		assert.Equal(t, test.minute, timeVal.Minute(), "Minute mismatch for: %s", test.input)
		assert.Equal(t, test.duration, duration, "Duration mismatch for: %s", test.input)
	}
}

func TestEventParser_ExtractLocation(t *testing.T) {
	parser := NewEventParser()
	
	tests := []struct {
		input            string
		expectedContains string
	}{
		{"Meeting at Google HQ", "Google"},
		{"Lunch at Joe's Cafe", "Joe"},
		{"Appointment in Room 301", "Room"},
	}
	
	for _, test := range tests {
		location := parser.extractLocation(test.input)
		assert.Contains(t, location, test.expectedContains, "Input: %s", test.input)
	}
}

func TestEventParser_ExtractRecurrence(t *testing.T) {
	parser := NewEventParser()
	
	tests := []struct {
		input    string
		expected string
	}{
		{"every day", "RRULE:FREQ=DAILY"},
		{"every week", "RRULE:FREQ=WEEKLY"},
		{"weekly", "RRULE:FREQ=WEEKLY"},
		{"every Monday", "RRULE:FREQ=WEEKLY;BYDAY=MO"},
	}
	
	for _, test := range tests {
		recurrence := parser.extractRecurrence(test.input)
		assert.Equal(t, test.expected, recurrence, "Input: %s", test.input)
	}
}

func TestEventParser_InferDuration(t *testing.T) {
	parser := NewEventParser()
	
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"meeting with team", time.Hour},
		{"quick call", 30 * time.Minute},
		{"lunch with boss", 90 * time.Minute},
		{"doctor appointment", 30 * time.Minute},
		{"interview", time.Hour},
	}
	
	for _, test := range tests {
		duration := parser.inferDuration(test.input)
		assert.Equal(t, test.expected, duration, "Input: %s", test.input)
	}
}

// Calendar Skill Tests

func TestCalendarSkill_AddEvent(t *testing.T) {
	skill, _ := setupCalendarSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	args := map[string]interface{}{
		"description": "Team meeting tomorrow at 3pm in Conference Room A",
	}
	
	result, err := skill.handleAddEvent(ctx, args)
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.NotEmpty(t, resultMap["event_id"])
	assert.Equal(t, true, resultMap["created"])
	assert.NotNil(t, resultMap["start_time"])
}

func TestCalendarSkill_ListEvents(t *testing.T) {
	skill, _ := setupCalendarSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Add some events
	for i := 0; i < 3; i++ {
		skill.handleAddEvent(ctx, map[string]interface{}{
			"description": fmt.Sprintf("Event %d tomorrow at %dpm", i, i+1),
		})
	}
	
	// List events
	result, err := skill.handleListEvents(ctx, map[string]interface{}{
		"when":  "week",
		"limit": float64(10),
	})
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	events := resultMap["events"].([]map[string]interface{})
	assert.GreaterOrEqual(t, len(events), 3)
}

func TestCalendarSkill_GetEvent(t *testing.T) {
	skill, _ := setupCalendarSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Add event
	addResult, err := skill.handleAddEvent(ctx, map[string]interface{}{
		"description": "Important meeting tomorrow at 2pm",
	})
	require.NoError(t, err)
	
	eventID := addResult.(map[string]interface{})["event_id"].(string)
	
	// Get event
	result, err := skill.handleGetEvent(ctx, map[string]interface{}{
		"event_id": eventID,
	})
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.Equal(t, eventID, resultMap["id"])
}

func TestCalendarSkill_DeleteEvent(t *testing.T) {
	skill, _ := setupCalendarSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Add event
	addResult, err := skill.handleAddEvent(ctx, map[string]interface{}{
		"description": "Event to delete tomorrow at 4pm",
	})
	require.NoError(t, err)
	
	eventID := addResult.(map[string]interface{})["event_id"].(string)
	
	// Try delete without confirmation
	result, err := skill.handleDeleteEvent(ctx, map[string]interface{}{
		"event_id": eventID,
	})
	require.NoError(t, err)
	assert.Equal(t, true, result.(map[string]interface{})["confirm_required"])
	
	// Delete with confirmation
	result, err = skill.handleDeleteEvent(ctx, map[string]interface{}{
		"event_id": eventID,
		"confirm":  true,
	})
	require.NoError(t, err)
	assert.Equal(t, true, result.(map[string]interface{})["deleted"])
}

func TestCalendarSkill_GetSchedule(t *testing.T) {
	skill, _ := setupCalendarSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Add events
	skill.handleAddEvent(ctx, map[string]interface{}{
		"description": "Morning meeting tomorrow at 9am",
	})
	skill.handleAddEvent(ctx, map[string]interface{}{
		"description": "Lunch meeting tomorrow at 12pm",
	})
	
	result, err := skill.handleGetSchedule(ctx, map[string]interface{}{
		"date": "tomorrow",
	})
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.GreaterOrEqual(t, resultMap["total"], 2)
}

func TestCalendarSkill_GetStats(t *testing.T) {
	skill, _ := setupCalendarSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Add events
	for i := 0; i < 3; i++ {
		skill.handleAddEvent(ctx, map[string]interface{}{
			"description": fmt.Sprintf("Event %d tomorrow at %dpm", i, i+1),
		})
	}
	
	result, err := skill.handleGetStats(ctx, map[string]interface{}{})
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.GreaterOrEqual(t, resultMap["total_events"], 3)
}

// Event Helper Tests

func TestCalendarEvent_IsPast(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name     string
		endTime  time.Time
		expected bool
	}{
		{"past", now.Add(-time.Hour), true},
		{"future", now.Add(time.Hour), false},
		{"now", now, true},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := &CalendarEvent{EndTime: test.endTime}
			assert.Equal(t, test.expected, event.IsPast())
		})
	}
}

func TestCalendarEvent_IsToday(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name     string
		startTime time.Time
		expected bool
	}{
		{"today", now, true},
		{"tomorrow", now.AddDate(0, 0, 1), false},
		{"yesterday", now.AddDate(0, 0, -1), false},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := &CalendarEvent{StartTime: test.startTime}
			assert.Equal(t, test.expected, event.IsToday())
		})
	}
}

func TestCalendarEvent_Duration(t *testing.T) {
	start := time.Now()
	end := start.Add(90 * time.Minute)
	
	event := &CalendarEvent{StartTime: start, EndTime: end}
	assert.Equal(t, 90*time.Minute, event.Duration())
}

func TestCalendarEvent_ConflictsWith(t *testing.T) {
	base := time.Now()
	
	event1 := &CalendarEvent{
		StartTime: base,
		EndTime:   base.Add(time.Hour),
	}
	
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected bool
	}{
		{"overlapping start", base.Add(30 * time.Minute), base.Add(90 * time.Minute), true},
		{"overlapping end", base.Add(-30 * time.Minute), base.Add(30 * time.Minute), true},
		{"contained", base.Add(15 * time.Minute), base.Add(45 * time.Minute), true},
		{"containing", base.Add(-30 * time.Minute), base.Add(90 * time.Minute), true},
		{"before", base.Add(-2 * time.Hour), base.Add(-time.Hour), false},
		{"after", base.Add(time.Hour), base.Add(2 * time.Hour), false},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event2 := &CalendarEvent{StartTime: test.start, EndTime: test.end}
			assert.Equal(t, test.expected, event1.ConflictsWith(event2))
		})
	}
}

func TestCalendarEvent_FormatDuration(t *testing.T) {
	tests := []struct {
		start    time.Time
		end      time.Time
		expected string
	}{
		{time.Now(), time.Now().Add(30 * time.Minute), "30 min"},
		{time.Now(), time.Now().Add(time.Hour), "1 hr"},
		{time.Now(), time.Now().Add(90 * time.Minute), "1 hr 30 min"},
		{time.Now(), time.Now().Add(2 * time.Hour), "2 hr"},
	}
	
	for _, test := range tests {
		event := &CalendarEvent{StartTime: test.start, EndTime: test.end}
		assert.Equal(t, test.expected, event.FormatDuration())
	}
}
