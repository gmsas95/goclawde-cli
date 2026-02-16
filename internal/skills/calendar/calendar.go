package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/skills"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CalendarSkill provides calendar management capabilities
type CalendarSkill struct {
	*skills.BaseSkill
	store    *Store
	parser   *EventParser
	google   *GoogleCalendarProvider
	logger   *zap.Logger
	config   CalendarSkillConfig
}

// CalendarSkillConfig contains calendar skill configuration
type CalendarSkillConfig struct {
	Enabled         bool
	GoogleClientID  string
	GoogleSecret    string
	DefaultTimezone string
	EnableSync      bool
}

// NewCalendarSkill creates a new calendar skill
func NewCalendarSkill(db *gorm.DB, config CalendarSkillConfig, logger *zap.Logger) (*CalendarSkill, error) {
	store, err := NewStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar store: %w", err)
	}
	
	googleConfig := DefaultGoogleCalendarConfig()
	googleConfig.ClientID = config.GoogleClientID
	googleConfig.ClientSecret = config.GoogleSecret
	
	skill := &CalendarSkill{
		BaseSkill: skills.NewBaseSkill("calendar", "Calendar Management", "1.0.0"),
		store:     store,
		parser:    NewEventParser(),
		google:    NewGoogleCalendarProvider(googleConfig, logger),
		logger:    logger,
		config:    config,
	}
	
	skill.registerTools()
	
	return skill, nil
}

// registerTools registers all calendar tools
func (c *CalendarSkill) registerTools() {
	tools := []skills.Tool{
		{
			Name:        "add_event",
			Description: "Add a new event to your calendar using natural language. Examples: 'Meeting with John tomorrow at 3pm', 'Lunch with Sarah on Friday at noon', 'Doctor appointment next Tuesday at 10am for 30 minutes'",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Natural language description of the event (what, when, where, who)",
					},
					"calendar_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional calendar ID (uses primary if not specified)",
					},
				},
				"required": []string{"description"},
			},
		},
		{
			Name:        "list_events",
			Description: "List your upcoming calendar events",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"when": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"today", "tomorrow", "week", "month", "all"},
						"default":     "week",
						"description": "Time period to list events for",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"default":     20,
						"description": "Maximum number of events to return",
					},
					"search": map[string]interface{}{
						"type":        "string",
						"description": "Optional search term to filter events",
					},
				},
			},
		},
		{
			Name:        "get_event",
			Description: "Get details of a specific event",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"event_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the event to get",
					},
				},
				"required": []string{"event_id"},
			},
		},
		{
			Name:        "update_event",
			Description: "Update an existing event",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"event_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the event to update",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "New title",
					},
					"start_time": map[string]interface{}{
						"type":        "string",
						"description": "New start time (natural language or ISO format)",
					},
					"end_time": map[string]interface{}{
						"type":        "string",
						"description": "New end time",
					},
					"location": map[string]interface{}{
						"type":        "string",
						"description": "New location",
					},
				},
				"required": []string{"event_id"},
			},
		},
		{
			Name:        "delete_event",
			Description: "Delete an event from your calendar",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"event_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the event to delete",
					},
					"confirm": map[string]interface{}{
						"type":        "boolean",
						"description": "Confirm deletion",
					},
				},
				"required": []string{"event_id"},
			},
		},
		{
			Name:        "check_availability",
			Description: "Check when you're free or busy",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"date": map[string]interface{}{
						"type":        "string",
						"description": "Date to check (today, tomorrow, Monday, etc.)",
						"default":     "today",
					},
					"duration": map[string]interface{}{
						"type":        "string",
						"description": "Duration needed (e.g., '1 hour', '30 minutes')",
						"default":     "1 hour",
					},
				},
			},
		},
		{
			Name:        "find_free_time",
			Description: "Find available time slots for a meeting",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"duration": map[string]interface{}{
						"type":        "string",
						"description": "Meeting duration (e.g., '30 minutes', '1 hour')",
						"default":     "1 hour",
					},
					"days": map[string]interface{}{
						"type":        "integer",
						"description": "Number of days to look ahead",
						"default":     3,
					},
					"time_range": map[string]interface{}{
						"type":        "string",
						"description": "Preferred time range (e.g., '9am-5pm')",
						"default":     "9am-5pm",
					},
				},
			},
		},
		{
			Name:        "get_schedule",
			Description: "Get your schedule for a specific day",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"date": map[string]interface{}{
						"type":        "string",
						"description": "Date to get schedule for (today, tomorrow, Monday, etc.)",
						"default":     "today",
					},
				},
			},
		},
		{
			Name:        "get_calendar_stats",
			Description: "Get statistics about your calendar",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "connect_google_calendar",
			Description: "Connect your Google Calendar account",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"auth_code": map[string]interface{}{
						"type":        "string",
						"description": "Authorization code from Google OAuth",
					},
				},
			},
		},
	}
	
	for _, tool := range tools {
		tool.Handler = c.handleTool(tool.Name)
		c.AddTool(tool)
	}
}

// handleTool handles tool calls
func (c *CalendarSkill) handleTool(name string) skills.ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		switch name {
		case "add_event":
			return c.handleAddEvent(ctx, args)
		case "list_events":
			return c.handleListEvents(ctx, args)
		case "get_event":
			return c.handleGetEvent(ctx, args)
		case "update_event":
			return c.handleUpdateEvent(ctx, args)
		case "delete_event":
			return c.handleDeleteEvent(ctx, args)
		case "check_availability":
			return c.handleCheckAvailability(ctx, args)
		case "find_free_time":
			return c.handleFindFreeTime(ctx, args)
		case "get_schedule":
			return c.handleGetSchedule(ctx, args)
		case "get_calendar_stats":
			return c.handleGetStats(ctx, args)
		case "connect_google_calendar":
			return c.handleConnectGoogle(ctx, args)
		default:
			return nil, fmt.Errorf("unknown tool: %s", name)
		}
	}
}

// handleAddEvent adds a new event
func (c *CalendarSkill) handleAddEvent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	description, _ := args["description"].(string)
	calendarID, _ := args["calendar_id"].(string)
	
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	
	// Parse the natural language description
	parseResult, err := c.parser.ParseEvent(description)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}
	
	userID := c.getUserID(ctx)
	
	// Get primary calendar if not specified
	if calendarID == "" {
		primary, err := c.store.GetPrimaryCalendar(userID)
		if err == nil && primary != nil {
			calendarID = primary.ExternalID
		}
		if calendarID == "" {
			calendarID = "primary"
		}
	}
	
	// Create event
	event := &CalendarEvent{
		UserID:      userID,
		CalendarID:  calendarID,
		Title:       parseResult.Title,
		Description: description,
		Location:    parseResult.Location,
		StartTime:   parseResult.StartTime,
		EndTime:     parseResult.EndTime,
		AllDay:      parseResult.AllDay,
		Status:      EventStatusConfirmed,
		Source:      "manual",
	}
	
	if parseResult.IsRecurring {
		event.IsRecurring = true
		event.RecurrenceRule = parseResult.Recurrence
	}
	
	// Handle attendees
	if len(parseResult.Attendees) > 0 {
		attendeesJSON, _ := json.Marshal(parseResult.Attendees)
		event.Attendees = string(attendeesJSON)
	}
	
	if err := c.store.CreateEvent(event); err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}
	
	c.logger.Info("Event created",
		zap.String("title", event.Title),
		zap.Time("start", event.StartTime),
	)
	
	return map[string]interface{}{
		"event_id":    event.ID,
		"title":       event.Title,
		"start_time":  event.FormatTimeRange(),
		"location":    event.Location,
		"created":     true,
		"confidence":  parseResult.Confidence,
	}, nil
}

// handleListEvents lists events
func (c *CalendarSkill) handleListEvents(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	when, _ := args["when"].(string)
	search, _ := args["search"].(string)
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	
	userID := c.getUserID(ctx)
	
	// Build time filters
	now := time.Now()
	filters := EventFilters{
		Limit: limit,
		Search: search,
	}
	
	switch when {
	case "today":
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		filters.StartAfter = &today
		tomorrow := today.AddDate(0, 0, 1)
		filters.StartBefore = &tomorrow
	case "tomorrow":
		tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		filters.StartAfter = &tomorrow
		afterTomorrow := tomorrow.AddDate(0, 0, 1)
		filters.StartBefore = &afterTomorrow
	case "week":
		filters.StartAfter = &now
		weekEnd := now.AddDate(0, 0, 7)
		filters.StartBefore = &weekEnd
	case "month":
		filters.StartAfter = &now
		monthEnd := now.AddDate(0, 1, 0)
		filters.StartBefore = &monthEnd
	}
	
	list, err := c.store.ListEvents(userID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	
	// Format events
	formatted := make([]map[string]interface{}, len(list.Events))
	for i, e := range list.Events {
		formatted[i] = map[string]interface{}{
			"id":           e.ID,
			"title":        e.Title,
			"time":         e.FormatTimeRange(),
			"duration":     e.FormatDuration(),
			"location":     e.Location,
			"is_today":     e.IsToday(),
			"is_upcoming":  e.IsUpcoming(),
			"has_meeting":  e.HasConference(),
		}
	}
	
	return map[string]interface{}{
		"events":     formatted,
		"total":      list.Total,
		"today":      list.Today,
		"this_week":  list.ThisWeek,
		"upcoming":   list.Upcoming,
		"period":     when,
	}, nil
}

// handleGetEvent gets event details
func (c *CalendarSkill) handleGetEvent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	eventID, _ := args["event_id"].(string)
	
	if eventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}
	
	event, err := c.store.GetEvent(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}
	if event == nil {
		return nil, fmt.Errorf("event not found")
	}
	
	return map[string]interface{}{
		"id":          event.ID,
		"title":       event.Title,
		"description": event.Description,
		"time":        event.FormatTimeRange(),
		"duration":    event.FormatDuration(),
		"location":    event.Location,
		"status":      event.Status,
		"is_recurring": event.IsRecurring,
		"conference":  event.ConferenceURL,
	}, nil
}

// handleUpdateEvent updates an event
func (c *CalendarSkill) handleUpdateEvent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	eventID, _ := args["event_id"].(string)
	
	event, err := c.store.GetEvent(eventID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, fmt.Errorf("event not found")
	}
	
	// Update fields
	if title, ok := args["title"].(string); ok && title != "" {
		event.Title = title
	}
	if location, ok := args["location"].(string); ok {
		event.Location = location
	}
	
	if err := c.store.UpdateEvent(event); err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}
	
	return map[string]interface{}{
		"event_id": event.ID,
		"updated":  true,
	}, nil
}

// handleDeleteEvent deletes an event
func (c *CalendarSkill) handleDeleteEvent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	eventID, _ := args["event_id"].(string)
	confirm, _ := args["confirm"].(bool)
	
	if eventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}
	
	if !confirm {
		return map[string]interface{}{
			"confirm_required": true,
			"message":          "Set confirm=true to delete this event",
		}, nil
	}
	
	if err := c.store.DeleteEvent(eventID); err != nil {
		return nil, fmt.Errorf("failed to delete event: %w", err)
	}
	
	return map[string]interface{}{
		"event_id": eventID,
		"deleted":  true,
	}, nil
}

// handleCheckAvailability checks availability
func (c *CalendarSkill) handleCheckAvailability(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dateStr, _ := args["date"].(string)
	durationStr, _ := args["duration"].(string)
	
	// Parse date
	date := c.parseDate(dateStr)
	if date.IsZero() {
		date = time.Now()
	}
	
	userID := c.getUserID(ctx)
	
	// Get events for the day
	events, err := c.store.GetEventsForDay(userID, date)
	if err != nil {
		return nil, err
	}
	
	// Calculate busy slots
	busySlots := []map[string]interface{}{}
	for _, e := range events {
		busySlots = append(busySlots, map[string]interface{}{
			"start": e.StartTime.Format("3:04 PM"),
			"end":   e.EndTime.Format("3:04 PM"),
			"title": e.Title,
		})
	}
	
	return map[string]interface{}{
		"date":       date.Format("Monday, Jan 2"),
		"busy_slots": busySlots,
		"events_count": len(events),
		"duration_requested": durationStr,
	}, nil
}

// handleFindFreeTime finds free time slots
func (c *CalendarSkill) handleFindFreeTime(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Simplified implementation
	return map[string]interface{}{
		"message": "Free time finder - suggest checking your calendar for gaps between events",
	}, nil
}

// handleGetSchedule gets schedule for a day
func (c *CalendarSkill) handleGetSchedule(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dateStr, _ := args["date"].(string)
	
	date := c.parseDate(dateStr)
	if date.IsZero() {
		date = time.Now()
	}
	
	userID := c.getUserID(ctx)
	
	events, err := c.store.GetEventsForDay(userID, date)
	if err != nil {
		return nil, err
	}
	
	// Format schedule
	schedule := []map[string]interface{}{}
	for _, e := range events {
		schedule = append(schedule, map[string]interface{}{
			"time":    fmt.Sprintf("%s - %s", e.StartTime.Format("3:04 PM"), e.EndTime.Format("3:04 PM")),
			"title":   e.Title,
			"location": e.Location,
		})
	}
	
	return map[string]interface{}{
		"date":     date.Format("Monday, Jan 2, 2006"),
		"schedule": schedule,
		"total":    len(events),
	}, nil
}

// handleGetStats gets calendar statistics
func (c *CalendarSkill) handleGetStats(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := c.getUserID(ctx)
	
	stats, err := c.store.GetStats(userID)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"total_events":     stats.TotalEvents,
		"this_week":        stats.ThisWeekEvents,
		"next_week":        stats.NextWeekEvents,
		"hours_this_week":  fmt.Sprintf("%.1f", stats.HoursThisWeek),
		"busiest_day":      stats.BusiestDay,
	}, nil
}

// handleConnectGoogle handles Google Calendar OAuth
func (c *CalendarSkill) handleConnectGoogle(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	authCode, _ := args["auth_code"].(string)
	
	if authCode == "" {
		// Return auth URL
		state := "random-state" // Should be generated securely
		authURL := c.google.GetAuthURL(state)
		
		return map[string]interface{}{
			"auth_required": true,
			"auth_url":      authURL,
			"instructions":  "Visit the auth URL, complete authorization, then provide the auth_code",
		}, nil
	}
	
	// Exchange code for token
	token, err := c.google.ExchangeCode(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	
	// Save credentials
	userID := c.getUserID(ctx)
	creds := &CalendarCredentials{
		UserID:       userID,
		Provider:     "google",
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenExpiry:  token.Expiry,
		IsActive:     true,
	}
	
	if err := c.store.SaveCredentials(creds); err != nil {
		return nil, fmt.Errorf("failed to save credentials: %w", err)
	}
	
	return map[string]interface{}{
		"connected": true,
		"provider":  "google",
	}, nil
}

// Helper methods

func (c *CalendarSkill) getUserID(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return "default_user"
}

func (c *CalendarSkill) parseDate(dateStr string) time.Time {
	switch strings.ToLower(dateStr) {
	case "today":
		return time.Now()
	case "tomorrow":
		return time.Now().AddDate(0, 0, 1)
	default:
		return time.Time{}
	}
}
