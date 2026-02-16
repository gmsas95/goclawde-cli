package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// googleEvent represents Google Calendar event format
type googleEvent struct {
	ID          string `json:"id,omitempty"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	Start       struct {
		DateTime string `json:"dateTime,omitempty"`
		Date     string `json:"date,omitempty"`
		TimeZone string `json:"timeZone,omitempty"`
	} `json:"start"`
	End struct {
		DateTime string `json:"dateTime,omitempty"`
		Date     string `json:"date,omitempty"`
		TimeZone string `json:"timeZone,omitempty"`
	} `json:"end"`
	Attendees []struct {
		Email          string `json:"email"`
		DisplayName    string `json:"displayName,omitempty"`
		ResponseStatus string `json:"responseStatus,omitempty"`
		Optional       bool   `json:"optional,omitempty"`
		Organizer      bool   `json:"organizer,omitempty"`
	} `json:"attendees,omitempty"`
	Organizer struct {
		Email string `json:"email"`
	} `json:"organizer,omitempty"`
	Status           string   `json:"status,omitempty"`
	Visibility       string   `json:"visibility,omitempty"`
	RecurringEventID string   `json:"recurringEventId,omitempty"`
	Recurrence       []string `json:"recurrence,omitempty"`
	ConferenceData   struct {
		ConferenceID string `json:"conferenceId,omitempty"`
		EntryPoints  []struct {
			EntryPointType string `json:"entryPointType"`
			URI            string `json:"uri"`
		} `json:"entryPoints,omitempty"`
	} `json:"conferenceData,omitempty"`
	ColorID string `json:"colorId,omitempty"`
	Created string `json:"created,omitempty"`
	Updated string `json:"updated,omitempty"`
}

// GoogleCalendarProvider implements Google Calendar API integration
type GoogleCalendarProvider struct {
	config     *oauth2.Config
	httpClient *http.Client
	logger     *zap.Logger
}

// GoogleCalendarConfig contains Google Calendar OAuth configuration
type GoogleCalendarConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// DefaultGoogleCalendarConfig returns default configuration
func DefaultGoogleCalendarConfig() GoogleCalendarConfig {
	return GoogleCalendarConfig{
		RedirectURL: "http://localhost:8080/auth/calendar/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/calendar",
			"https://www.googleapis.com/auth/calendar.events",
		},
	}
}

// NewGoogleCalendarProvider creates a new Google Calendar provider
func NewGoogleCalendarProvider(config GoogleCalendarConfig, logger *zap.Logger) *GoogleCalendarProvider {
	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       config.Scopes,
		Endpoint:     google.Endpoint,
	}
	
	return &GoogleCalendarProvider{
		config: oauthConfig,
		logger: logger,
	}
}

// GetAuthURL returns the OAuth authorization URL
func (g *GoogleCalendarProvider) GetAuthURL(state string) string {
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeCode exchanges an authorization code for tokens
func (g *GoogleCalendarProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	return token, nil
}

// CreateClient creates an HTTP client with the given token
func (g *GoogleCalendarProvider) CreateClient(ctx context.Context, token *oauth2.Token) *http.Client {
	return g.config.Client(ctx, token)
}

// RefreshToken refreshes an access token
func (g *GoogleCalendarProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}
	
	tokenSource := g.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	
	return newToken, nil
}

// ListCalendars lists the user's calendars
func (g *GoogleCalendarProvider) ListCalendars(ctx context.Context, client *http.Client) ([]Calendar, error) {
	apiURL := "https://www.googleapis.com/calendar/v3/users/me/calendarList"
	
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("calendar list failed: %s - %s", resp.Status, string(body))
	}
	
	var result struct {
		Items []struct {
			ID          string `json:"id"`
			Summary     string `json:"summary"`
			Description string `json:"description"`
			TimeZone    string `json:"timeZone"`
			Primary     bool   `json:"primary"`
			AccessRole  string `json:"accessRole"`
			BackgroundColor string `json:"backgroundColor"`
		} `json:"items"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	calendars := make([]Calendar, len(result.Items))
	for i, item := range result.Items {
		calendars[i] = Calendar{
			ExternalID:  item.ID,
			Name:        item.Summary,
			Description: item.Description,
			Timezone:    item.TimeZone,
			Color:       item.BackgroundColor,
			Provider:    "google",
			IsPrimary:   item.Primary,
			IsWritable:  item.AccessRole == "owner" || item.AccessRole == "writer",
			IsVisible:   true,
		}
	}
	
	return calendars, nil
}

// ListEvents lists events from a calendar
func (g *GoogleCalendarProvider) ListEvents(ctx context.Context, client *http.Client, calendarID string, timeMin, timeMax time.Time) ([]CalendarEvent, error) {
	baseURL := "https://www.googleapis.com/calendar/v3/calendars/"
	if calendarID == "primary" {
		calendarID = "primary"
	}
	
	// URL encode calendar ID if needed
	if strings.Contains(calendarID, "@") {
		calendarID = neturl.PathEscape(calendarID)
	}
	
	apiURL := fmt.Sprintf("%s%s/events", baseURL, calendarID)
	
	// Build query params
	params := neturl.Values{}
	params.Set("timeMin", timeMin.Format(time.RFC3339))
	params.Set("timeMax", timeMax.Format(time.RFC3339))
	params.Set("singleEvents", "true") // Expand recurring events
	params.Set("orderBy", "startTime")
	params.Set("maxResults", "100")
	
	fullURL := apiURL + "?" + params.Encode()
	
	resp, err := client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("events list failed: %s - %s", resp.Status, string(body))
	}
	
	var result struct {
		Items []googleEvent `json:"items"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	events := make([]CalendarEvent, 0, len(result.Items))
	for _, item := range result.Items {
		event := g.convertGoogleEvent(item)
		events = append(events, event)
	}
	
	return events, nil
}

// CreateEvent creates a new event in Google Calendar
func (g *GoogleCalendarProvider) CreateEvent(ctx context.Context, client *http.Client, calendarID string, event *CalendarEvent) (*CalendarEvent, error) {
	baseURL := "https://www.googleapis.com/calendar/v3/calendars/"
	if strings.Contains(calendarID, "@") {
		calendarID = neturl.PathEscape(calendarID)
	}
	
	apiURL := fmt.Sprintf("%s%s/events", baseURL, calendarID)
	
	// Convert to Google event format
	ge := g.convertToGoogleEvent(event)
	
	body, err := json.Marshal(ge)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}
	
	resp, err := client.Post(apiURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create event failed: %s - %s", resp.Status, string(respBody))
	}
	
	var createdEvt googleEvent
	if err := json.NewDecoder(resp.Body).Decode(&createdEvt); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	result := g.convertGoogleEvent(createdEvt)
	return &result, nil
}

// UpdateEvent updates an existing event
func (g *GoogleCalendarProvider) UpdateEvent(ctx context.Context, client *http.Client, calendarID, eventID string, event *CalendarEvent) (*CalendarEvent, error) {
	baseURL := "https://www.googleapis.com/calendar/v3/calendars/"
	if strings.Contains(calendarID, "@") {
		calendarID = neturl.PathEscape(calendarID)
	}
	
	apiURL := fmt.Sprintf("%s%s/events/%s", baseURL, calendarID, eventID)
	
	ge := g.convertToGoogleEvent(event)
	
	body, err := json.Marshal(ge)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "PUT", apiURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update event failed: %s - %s", resp.Status, string(respBody))
	}
	
	var updated googleEvent
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	result := g.convertGoogleEvent(updated)
	return &result, nil
}

// DeleteEvent deletes an event
func (g *GoogleCalendarProvider) DeleteEvent(ctx context.Context, client *http.Client, calendarID, eventID string) error {
	baseURL := "https://www.googleapis.com/calendar/v3/calendars/"
	if strings.Contains(calendarID, "@") {
		calendarID = neturl.PathEscape(calendarID)
	}
	
	apiURL := fmt.Sprintf("%s%s/events/%s", baseURL, calendarID, eventID)
	
	req, err := http.NewRequestWithContext(ctx, "DELETE", apiURL, nil)
	if err != nil {
		return err
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete event failed: %s", resp.Status)
	}
	
	return nil
}

// GetFreeBusy gets free/busy information
func (g *GoogleCalendarProvider) GetFreeBusy(ctx context.Context, client *http.Client, calendarID string, start, end time.Time) ([]FreeBusySlot, error) {
	apiURL := "https://www.googleapis.com/calendar/v3/freeBusy"
	
	requestBody := map[string]interface{}{
		"timeMin": start.Format(time.RFC3339),
		"timeMax": end.Format(time.RFC3339),
		"items": []map[string]string{
			{"id": calendarID},
		},
	}
	
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	
	resp, err := client.Post(apiURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to get freebusy: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("freebusy failed: %s - %s", resp.Status, string(respBody))
	}
	
	var result struct {
		Calendars map[string]struct {
			Busy []struct {
				Start string `json:"start"`
				End   string `json:"end"`
			} `json:"busy"`
		} `json:"calendars"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	// Convert to slots
	slots := []FreeBusySlot{}
	for _, cal := range result.Calendars {
		for _, busy := range cal.Busy {
			startTime, _ := time.Parse(time.RFC3339, busy.Start)
			endTime, _ := time.Parse(time.RFC3339, busy.End)
			slots = append(slots, FreeBusySlot{
				Start: startTime,
				End:   endTime,
				Busy:  true,
			})
		}
	}
	
	return slots, nil
}


// convertGoogleEvent converts a Google event to our format
func (g *GoogleCalendarProvider) convertGoogleEvent(ge googleEvent) CalendarEvent {
	event := CalendarEvent{
		SourceID:   ge.ID,
		Title:      ge.Summary,
		Description: ge.Description,
		Location:   ge.Location,
		Status:     EventStatusConfirmed,
		Source:     "google",
	}
	
	// Parse start time
	if ge.Start.DateTime != "" {
		event.StartTime, _ = time.Parse(time.RFC3339, ge.Start.DateTime)
		event.Timezone = ge.Start.TimeZone
	} else if ge.Start.Date != "" {
		event.StartTime, _ = time.Parse("2006-01-02", ge.Start.Date)
		event.AllDay = true
	}
	
	// Parse end time
	if ge.End.DateTime != "" {
		event.EndTime, _ = time.Parse(time.RFC3339, ge.End.DateTime)
	} else if ge.End.Date != "" {
		event.EndTime, _ = time.Parse("2006-01-02", ge.End.Date)
	}
	
	// Status
	switch ge.Status {
	case "confirmed":
		event.Status = EventStatusConfirmed
	case "tentative":
		event.Status = EventStatusTentative
	case "cancelled":
		event.Status = EventStatusCancelled
	}
	
	// Visibility
	if ge.Visibility != "" {
		event.Visibility = ge.Visibility
	}
	
	// Recurring
	if ge.RecurringEventID != "" {
		event.RecurringEventID = ge.RecurringEventID
		event.IsRecurring = true
	}
	if len(ge.Recurrence) > 0 {
		event.RecurrenceRule = ge.Recurrence[0]
		event.IsRecurring = true
	}
	
	// Conference data
	if len(ge.ConferenceData.EntryPoints) > 0 {
		for _, ep := range ge.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" {
				event.ConferenceURL = ep.URI
				if strings.Contains(ep.URI, "meet.google.com") {
					event.ConferenceType = "google_meet"
				}
			}
		}
	}
	
	// Attendees
	if len(ge.Attendees) > 0 {
		attendees := make([]Attendee, len(ge.Attendees))
		for i, a := range ge.Attendees {
			attendees[i] = Attendee{
				Email:          a.Email,
				Name:           a.DisplayName,
				ResponseStatus: a.ResponseStatus,
				Optional:       a.Optional,
				Organizer:      a.Organizer,
			}
		}
		attendeesJSON, _ := json.Marshal(attendees)
		event.Attendees = string(attendeesJSON)
	}
	
	// Organizer
	if ge.Organizer.Email != "" {
		event.Organizer = ge.Organizer.Email
	}
	
	// Color
	if ge.ColorID != "" {
		event.ColorID = ge.ColorID
	}
	
	return event
}

// convertToGoogleEvent converts our event to Google format
func (g *GoogleCalendarProvider) convertToGoogleEvent(event *CalendarEvent) googleEvent {
	ge := googleEvent{
		Summary:     event.Title,
		Description: event.Description,
		Location:    event.Location,
	}
	
	// Set times
	if event.AllDay {
		ge.Start.Date = event.StartTime.Format("2006-01-02")
		ge.End.Date = event.EndTime.Format("2006-01-02")
	} else {
		ge.Start.DateTime = event.StartTime.Format(time.RFC3339)
		ge.Start.TimeZone = event.Timezone
		ge.End.DateTime = event.EndTime.Format(time.RFC3339)
		ge.End.TimeZone = event.Timezone
	}
	
	// Status
	switch event.Status {
	case EventStatusConfirmed:
		ge.Status = "confirmed"
	case EventStatusTentative:
		ge.Status = "tentative"
	case EventStatusCancelled:
		ge.Status = "cancelled"
	}
	
	// Visibility
	if event.Visibility != "" {
		ge.Visibility = event.Visibility
	}
	
	// Recurrence
	if event.RecurrenceRule != "" {
		ge.Recurrence = []string{event.RecurrenceRule}
	}
	
	// Attendees
	if event.Attendees != "" {
		var attendees []Attendee
		if err := json.Unmarshal([]byte(event.Attendees), &attendees); err == nil {
			ge.Attendees = make([]struct {
				Email          string `json:"email"`
				DisplayName    string `json:"displayName,omitempty"`
				ResponseStatus string `json:"responseStatus,omitempty"`
				Optional       bool   `json:"optional,omitempty"`
				Organizer      bool   `json:"organizer,omitempty"`
			}, len(attendees))
			
			for i, a := range attendees {
				ge.Attendees[i].Email = a.Email
				ge.Attendees[i].DisplayName = a.Name
				ge.Attendees[i].ResponseStatus = a.ResponseStatus
				ge.Attendees[i].Optional = a.Optional
				ge.Attendees[i].Organizer = a.Organizer
			}
		}
	}
	
	return ge
}

// Sync performs a sync with Google Calendar
func (g *GoogleCalendarProvider) Sync(ctx context.Context, client *http.Client, calendarID, syncToken string) (*SyncResult, error) {
	baseURL := "https://www.googleapis.com/calendar/v3/calendars/"
	if strings.Contains(calendarID, "@") {
		calendarID = neturl.PathEscape(calendarID)
	}
	
	apiURL := fmt.Sprintf("%s%s/events", baseURL, calendarID)
	
	params := neturl.Values{}
	params.Set("singleEvents", "true")
	
	if syncToken != "" {
		params.Set("syncToken", syncToken)
	} else {
		// Initial sync - get events from last 30 days
		params.Set("timeMin", time.Now().AddDate(0, 0, -30).Format(time.RFC3339))
	}
	
	fullURL := apiURL + "?" + params.Encode()
	
	resp, err := client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("sync failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Handle sync token expiration
	if resp.StatusCode == http.StatusGone {
		// Token expired, do full sync
		return g.Sync(ctx, client, calendarID, "")
	}
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sync failed: %s - %s", resp.Status, string(body))
	}
	
	var result struct {
		Items      []googleEvent `json:"items"`
		NextSyncToken string `json:"nextSyncToken"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	syncResult := &SyncResult{
		NextSyncToken: result.NextSyncToken,
	}
	
	for _, item := range result.Items {
		switch item.Status {
		case "cancelled":
			syncResult.Deleted++
		default:
			if item.Created == item.Updated {
				syncResult.Added++
			} else {
				syncResult.Updated++
			}
		}
	}
	
	return syncResult, nil
}
