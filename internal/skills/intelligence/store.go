package intelligence

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Store handles intelligence data persistence
type Store struct {
	db *gorm.DB
}

// NewStore creates a new intelligence store
func NewStore(db *gorm.DB) (*Store, error) {
	store := &Store{db: db}
	
	if err := db.AutoMigrate(&UserPattern{}, &Suggestion{}, &AutomatedWorkflow{}, &WorkflowRun{}, &BehaviorEvent{}, &UserProfile{}); err != nil {
		return nil, fmt.Errorf("failed to migrate intelligence schemas: %w", err)
	}
	
	return store, nil
}

func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "int_" + hex.EncodeToString(bytes)
}

// Pattern operations

func (s *Store) CreatePattern(pattern *UserPattern) error {
	if pattern.ID == "" {
		pattern.ID = generateID()
	}
	pattern.CreatedAt = time.Now()
	pattern.UpdatedAt = time.Now()
	return s.db.Create(pattern).Error
}

func (s *Store) GetPattern(id string) (*UserPattern, error) {
	var pattern UserPattern
	err := s.db.Where("id = ?", id).First(&pattern).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &pattern, err
}

func (s *Store) UpdatePattern(pattern *UserPattern) error {
	pattern.UpdatedAt = time.Now()
	return s.db.Save(pattern).Error
}

func (s *Store) ListPatterns(userID string, category string) ([]UserPattern, error) {
	query := s.db.Where("user_id = ? AND status = ?", userID, "active")
	
	if category != "" && category != "all" {
		query = query.Where("category = ?", category)
	}
	
	var patterns []UserPattern
	err := query.Order("confidence DESC, occurrences DESC").Find(&patterns).Error
	return patterns, err
}

func (s *Store) FindSimilarPattern(userID string, patternType string, category string) (*UserPattern, error) {
	var pattern UserPattern
	err := s.db.Where("user_id = ? AND type = ? AND category = ? AND status = ?",
		userID, patternType, category, "active").
		Order("last_seen DESC").
		First(&pattern).Error
	
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &pattern, err
}

// Suggestion operations

func (s *Store) CreateSuggestion(suggestion *Suggestion) error {
	if suggestion.ID == "" {
		suggestion.ID = generateID()
	}
	suggestion.CreatedAt = time.Now()
	suggestion.SuggestedAt = time.Now()
	return s.db.Create(suggestion).Error
}

func (s *Store) GetSuggestion(id string) (*Suggestion, error) {
	var suggestion Suggestion
	err := s.db.Where("id = ?", id).First(&suggestion).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &suggestion, err
}

func (s *Store) UpdateSuggestion(suggestion *Suggestion) error {
	return s.db.Save(suggestion).Error
}

func (s *Store) GetPendingSuggestions(userID string, limit int) ([]Suggestion, error) {
	query := s.db.Where("user_id = ? AND status = ?", userID, "pending").
		Where("valid_until IS NULL OR valid_until > ?", time.Now())
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	var suggestions []Suggestion
	err := query.Order("priority DESC, created_at DESC").Find(&suggestions).Error
	return suggestions, err
}

func (s *Store) DismissSuggestion(id string) error {
	now := time.Now()
	return s.db.Model(&Suggestion{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      "dismissed",
		"dismissed_at": &now,
	}).Error
}

func (s *Store) MarkSuggestionActed(id string) error {
	now := time.Now()
	return s.db.Model(&Suggestion{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":   "acted_upon",
		"acted_at": &now,
	}).Error
}

func (s *Store) GetSuggestionStats(userID string) (map[string]int, error) {
	var stats []struct {
		Status string
		Count  int
	}
	
	err := s.db.Model(&Suggestion{}).
		Select("status, COUNT(*) as count").
		Where("user_id = ?", userID).
		Group("status").
		Scan(&stats).Error
	
	if err != nil {
		return nil, err
	}
	
	result := make(map[string]int)
	for _, stat := range stats {
		result[stat.Status] = stat.Count
	}
	
	return result, nil
}

// Workflow operations

func (s *Store) CreateWorkflow(workflow *AutomatedWorkflow) error {
	if workflow.ID == "" {
		workflow.ID = generateID()
	}
	workflow.CreatedAt = time.Now()
	workflow.UpdatedAt = time.Now()
	return s.db.Create(workflow).Error
}

func (s *Store) GetWorkflow(id string) (*AutomatedWorkflow, error) {
	var workflow AutomatedWorkflow
	err := s.db.Where("id = ?", id).First(&workflow).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &workflow, err
}

func (s *Store) UpdateWorkflow(workflow *AutomatedWorkflow) error {
	workflow.UpdatedAt = time.Now()
	return s.db.Save(workflow).Error
}

func (s *Store) DeleteWorkflow(id string) error {
	return s.db.Where("id = ?", id).Delete(&AutomatedWorkflow{}).Error
}

func (s *Store) ListWorkflows(userID string, enabledOnly bool) ([]AutomatedWorkflow, error) {
	query := s.db.Where("user_id = ?", userID)
	
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	
	var workflows []AutomatedWorkflow
	err := query.Order("created_at DESC").Find(&workflows).Error
	return workflows, err
}

func (s *Store) GetActiveWorkflowsByTrigger(userID string, triggerType string) ([]AutomatedWorkflow, error) {
	var workflows []AutomatedWorkflow
	err := s.db.Where("user_id = ? AND enabled = ? AND trigger_type = ?",
		userID, true, triggerType).
		Find(&workflows).Error
	return workflows, err
}

// WorkflowRun operations

func (s *Store) CreateWorkflowRun(run *WorkflowRun) error {
	if run.ID == "" {
		run.ID = generateID()
	}
	run.StartedAt = time.Now()
	run.CreatedAt = time.Now()
	return s.db.Create(run).Error
}

func (s *Store) UpdateWorkflowRun(run *WorkflowRun) error {
	return s.db.Save(run).Error
}

func (s *Store) GetWorkflowRuns(workflowID string, limit int) ([]WorkflowRun, error) {
	query := s.db.Where("workflow_id = ?", workflowID).Order("started_at DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	var runs []WorkflowRun
	err := query.Find(&runs).Error
	return runs, err
}

// BehaviorEvent operations

func (s *Store) CreateBehaviorEvent(event *BehaviorEvent) error {
	if event.ID == "" {
		event.ID = generateID()
	}
	
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	event.DayOfWeek = int(event.Timestamp.Weekday())
	event.HourOfDay = event.Timestamp.Hour()
	event.CreatedAt = time.Now()
	
	return s.db.Create(event).Error
}

func (s *Store) GetBehaviorEvents(userID string, eventType string, start, end time.Time, limit int) ([]BehaviorEvent, error) {
	query := s.db.Where("user_id = ?", userID)
	
	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}
	if !start.IsZero() {
		query = query.Where("timestamp >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("timestamp <= ?", end)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	var events []BehaviorEvent
	err := query.Order("timestamp DESC").Find(&events).Error
	return events, err
}

func (s *Store) GetRecentEvents(userID string, hours int) ([]BehaviorEvent, error) {
	start := time.Now().Add(-time.Duration(hours) * time.Hour)
	return s.GetBehaviorEvents(userID, "", start, time.Time{}, 0)
}

func (s *Store) GetEventsByHour(userID string, days int) (map[int]int, error) {
	start := time.Now().AddDate(0, 0, -days)
	
	var results []struct {
		Hour  int
		Count int
	}
	
	err := s.db.Model(&BehaviorEvent{}).
		Select("hour_of_day as hour, COUNT(*) as count").
		Where("user_id = ? AND timestamp >= ?", userID, start).
		Group("hour_of_day").
		Scan(&results).Error
	
	if err != nil {
		return nil, err
	}
	
	hourMap := make(map[int]int)
	for _, r := range results {
		hourMap[r.Hour] = r.Count
	}
	
	return hourMap, nil
}

func (s *Store) GetEventsByDay(userID string, days int) (map[int]int, error) {
	start := time.Now().AddDate(0, 0, -days)
	
	var results []struct {
		Day   int
		Count int
	}
	
	err := s.db.Model(&BehaviorEvent{}).
		Select("day_of_week as day, COUNT(*) as count").
		Where("user_id = ? AND timestamp >= ?", userID, start).
		Group("day_of_week").
		Scan(&results).Error
	
	if err != nil {
		return nil, err
	}
	
	dayMap := make(map[int]int)
	for _, r := range results {
		dayMap[r.Day] = r.Count
	}
	
	return dayMap, nil
}

// UserProfile operations

func (s *Store) GetUserProfile(userID string) (*UserProfile, error) {
	var profile UserProfile
	err := s.db.Where("user_id = ?", userID).First(&profile).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create default profile
		profile = UserProfile{
			UserID:    userID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		return &profile, nil
	}
	
	return &profile, err
}

func (s *Store) SaveUserProfile(profile *UserProfile) error {
	profile.UpdatedAt = time.Now()
	
	var existing UserProfile
	err := s.db.Where("user_id = ?", profile.UserID).First(&existing).Error
	
	if err == gorm.ErrRecordNotFound {
		profile.CreatedAt = time.Now()
		return s.db.Create(profile).Error
	}
	
	return s.db.Save(profile).Error
}

// Analytics queries

func (s *Store) GetEventCountsByCategory(userID string, days int) (map[string]int, error) {
	start := time.Now().AddDate(0, 0, -days)
	
	var results []struct {
		Category string
		Count    int
	}
	
	err := s.db.Model(&BehaviorEvent{}).
		Select("category, COUNT(*) as count").
		Where("user_id = ? AND timestamp >= ?", userID, start).
		Group("category").
		Scan(&results).Error
	
	if err != nil {
		return nil, err
	}
	
	categoryMap := make(map[string]int)
	for _, r := range results {
		categoryMap[r.Category] = r.Count
	}
	
	return categoryMap, nil
}

func (s *Store) GetActivityTimeline(userID string, days int) ([]struct {
	Date  string
	Count int
}, error) {
	start := time.Now().AddDate(0, 0, -days)
	
	var results []struct {
		Date  string
		Count int
	}
	
	err := s.db.Model(&BehaviorEvent{}).
		Select("DATE(timestamp) as date, COUNT(*) as count").
		Where("user_id = ? AND timestamp >= ?", userID, start).
		Group("DATE(timestamp)").
		Order("date").
		Scan(&results).Error
	
	return results, err
}
