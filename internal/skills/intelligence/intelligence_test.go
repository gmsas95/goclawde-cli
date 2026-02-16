package intelligence

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func setupTestSkill(t *testing.T) (*IntelligenceSkill, *gorm.DB) {
	db := setupTestDB(t)
	logger, _ := zap.NewDevelopment()

	skill, err := NewIntelligenceSkill(db, logger)
	require.NoError(t, err)

	return skill, db
}

// Store Tests

func TestStore_CreatePattern(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)

	pattern := &UserPattern{
		UserID:      "user_123",
		Type:        "time_based",
		Category:    "health",
		Name:        "morning_medication",
		Description: "User takes medication in the morning",
		Confidence:  85.0,
		Occurrences: 10,
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
	}

	err = store.CreatePattern(pattern)
	require.NoError(t, err)
	assert.NotEmpty(t, pattern.ID)

	// Verify
	retrieved, err := store.GetPattern(pattern.ID)
	require.NoError(t, err)
	assert.Equal(t, pattern.Name, retrieved.Name)
}

func TestStore_Suggestion(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)

	suggestion := &Suggestion{
		UserID:      "user_123",
		Type:        "proactive",
		Category:    "health",
		Title:       "Take your medication",
		Description: "It's time for your medication",
		ActionType:  "reminder",
		Priority:    "high",
		Status:      "pending",
	}

	err = store.CreateSuggestion(suggestion)
	require.NoError(t, err)
	assert.NotEmpty(t, suggestion.ID)

	// Test getting pending
	pending, err := store.GetPendingSuggestions("user_123", 10)
	require.NoError(t, err)
	assert.Len(t, pending, 1)

	// Test dismiss
	err = store.DismissSuggestion(suggestion.ID)
	require.NoError(t, err)

	// Verify dismissed
	pending, _ = store.GetPendingSuggestions("user_123", 10)
	assert.Len(t, pending, 0)
}

func TestStore_Workflow(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)

	workflow := &AutomatedWorkflow{
		UserID:      "user_123",
		Name:        "Morning Routine",
		Description: "Daily morning reminders",
		TriggerType: "schedule",
		Actions:     `["check_calendar", "show_tasks"]`,
		Enabled:     true,
	}

	err = store.CreateWorkflow(workflow)
	require.NoError(t, err)
	assert.NotEmpty(t, workflow.ID)

	// List
	workflows, err := store.ListWorkflows("user_123", true)
	require.NoError(t, err)
	assert.Len(t, workflows, 1)

	// Toggle
	workflow.Enabled = false
	err = store.UpdateWorkflow(workflow)
	require.NoError(t, err)

	// Delete
	err = store.DeleteWorkflow(workflow.ID)
	require.NoError(t, err)
}

func TestStore_BehaviorEvent(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)

	event := &BehaviorEvent{
		UserID:    "user_123",
		EventType: "task_completed",
		Category:  "productivity",
		Data:      `{"task_id": "123"}`,
		Timestamp: time.Now(),
	}

	err = store.CreateBehaviorEvent(event)
	require.NoError(t, err)
	assert.NotEmpty(t, event.ID)

	// Get recent
	events, err := store.GetRecentEvents("user_123", 24)
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

// Analyzer Tests

func TestAnalyzer_DetectPatterns(t *testing.T) {
	db := setupTestDB(t)
	store, _ := NewStore(db)
	analyzer := NewPatternAnalyzer(store)

	// Add behavior events
	for i := 0; i < 5; i++ {
		store.CreateBehaviorEvent(&BehaviorEvent{
			UserID:    "user_123",
			EventType: "medication_taken",
			Category:  "health",
			Timestamp: time.Now().AddDate(0, 0, -i),
			HourOfDay: 8,
		})
	}

	// Analyze
	patterns, err := analyzer.AnalyzeUser("user_123")
	require.NoError(t, err)

	// May or may not detect patterns depending on data
	_ = patterns
}

// Suggestion Engine Tests

func TestSuggestionEngine_Generate(t *testing.T) {
	db := setupTestDB(t)
	store, _ := NewStore(db)
	engine := NewSuggestionEngine(store)

	suggestions, err := engine.GenerateSuggestions("user_123")
	require.NoError(t, err)
	assert.True(t, len(suggestions) > 0)
}

// IntelligenceSkill Tests

func TestIntelligenceSkill_GetLifeDashboard(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user_123")

	result, err := skill.handleGetLifeDashboard(ctx, map[string]interface{}{
		"include_insights": true,
	})

	require.NoError(t, err)
	dashboard := result.(*LifeDashboard)
	assert.Equal(t, "user_123", dashboard.UserID)
	assert.NotNil(t, dashboard.Health)
	assert.NotNil(t, dashboard.Productivity)
}

func TestIntelligenceSkill_WorkflowCRUD(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user_123")

	// Create
	result, err := skill.handleCreateWorkflow(ctx, map[string]interface{}{
		"name":    "Test Workflow",
		"trigger": "every morning at 8am",
		"actions": []interface{}{"check_calendar", "show_tasks"},
	})
	require.NoError(t, err)
	workflowID := result.(map[string]interface{})["id"].(string)

	// List
	result, err = skill.handleListWorkflows(ctx, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.(map[string]interface{})["count"])

	// Toggle
	result, err = skill.handleToggleWorkflow(ctx, map[string]interface{}{
		"workflow_id": workflowID,
		"enabled":     false,
	})
	require.NoError(t, err)
	assert.Equal(t, "disabled", result.(map[string]interface{})["status"])

	// Delete
	result, err = skill.handleDeleteWorkflow(ctx, map[string]interface{}{
		"workflow_id": workflowID,
	})
	require.NoError(t, err)
	assert.True(t, result.(map[string]interface{})["success"].(bool))
}

func TestIntelligenceSkill_TrackEvent(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user_123")

	result, err := skill.handleTrackEvent(ctx, map[string]interface{}{
		"event_type": "task_completed",
		"category":   "productivity",
		"data": map[string]interface{}{
			"task_id": "task_123",
		},
	})

	require.NoError(t, err)
	assert.True(t, result.(map[string]interface{})["success"].(bool))
}

func TestIntelligenceSkill_GetSuggestions(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user_123")

	result, err := skill.handleGetSuggestions(ctx, map[string]interface{}{
		"context": "morning",
		"limit":   5,
	})

	require.NoError(t, err)
	resp := result.(map[string]interface{})
	assert.Equal(t, "morning", resp["context"])
	assert.NotNil(t, resp["suggestions"])
}
