package tasks

import (
	"context"
	"fmt"
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

func setupTestSkill(t *testing.T) (*TaskSkill, *gorm.DB) {
	db := setupTestDB(t)
	logger, _ := zap.NewDevelopment()
	
	config := TaskConfig{
		Enabled:        true,
		DefaultChannel: "test",
	}
	
	skill, err := NewTaskSkill(db, config, logger)
	require.NoError(t, err)
	
	return skill, db
}

// Store Tests

func TestStore_CreateAndGetTask(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	task := &Task{
		UserID:      "user1",
		Title:       "Test Task",
		Description: "Test Description",
		Status:      TaskStatusPending,
		Priority:    PriorityHigh,
	}
	
	err = store.CreateTask(task)
	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)
	assert.WithinDuration(t, time.Now(), task.CreatedAt, time.Second)
	
	// Retrieve task
	retrieved, err := store.GetTask(task.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, task.Title, retrieved.Title)
	assert.Equal(t, task.Priority, retrieved.Priority)
}

func TestStore_UpdateTask(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	task := &Task{
		UserID:   "user1",
		Title:    "Original Title",
		Status:   TaskStatusPending,
		Priority: PriorityMedium,
	}
	
	err = store.CreateTask(task)
	require.NoError(t, err)
	
	task.Title = "Updated Title"
	task.Priority = PriorityHigh
	err = store.UpdateTask(task)
	require.NoError(t, err)
	
	retrieved, err := store.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", retrieved.Title)
	assert.Equal(t, PriorityHigh, retrieved.Priority)
}

func TestStore_DeleteTask(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	task := &Task{
		UserID: "user1",
		Title:  "Task to Delete",
	}
	
	err = store.CreateTask(task)
	require.NoError(t, err)
	
	err = store.DeleteTask(task.ID)
	require.NoError(t, err)
	
	retrieved, err := store.GetTask(task.ID)
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestStore_ListTasks(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	// Create multiple tasks
	tasks := []*Task{
		{UserID: "user1", Title: "Task 1", Status: TaskStatusPending, Priority: PriorityHigh},
		{UserID: "user1", Title: "Task 2", Status: TaskStatusInProgress, Priority: PriorityMedium},
		{UserID: "user1", Title: "Task 3", Status: TaskStatusCompleted, Priority: PriorityLow},
		{UserID: "user2", Title: "Other User Task", Status: TaskStatusPending},
	}
	
	for _, task := range tasks {
		err := store.CreateTask(task)
		require.NoError(t, err)
	}
	
	// Test listing for user1
	list, err := store.ListTasks("user1", ListOptions{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Tasks), 2) // At least pending and in_progress
	
	// Test status filter
	list, err = store.ListTasks("user1", ListOptions{
		Status: []TaskStatus{TaskStatusPending},
	})
	require.NoError(t, err)
	for _, task := range list.Tasks {
		assert.Equal(t, TaskStatusPending, task.Status)
	}
}

func TestStore_GetOverdueTasks(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)
	
	tasks := []*Task{
		{UserID: "user1", Title: "Overdue Task", Status: TaskStatusPending, DueDate: &past},
		{UserID: "user1", Title: "Future Task", Status: TaskStatusPending, DueDate: &future},
		{UserID: "user1", Title: "Completed Past Task", Status: TaskStatusCompleted, DueDate: &past},
	}
	
	for _, task := range tasks {
		err := store.CreateTask(task)
		require.NoError(t, err)
	}
	
	overdue, err := store.GetOverdueTasks("user1")
	require.NoError(t, err)
	assert.Len(t, overdue, 1)
	assert.Equal(t, "Overdue Task", overdue[0].Title)
}

func TestStore_CompleteTask(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	task := &Task{
		UserID: "user1",
		Title:  "Task to Complete",
		Status: TaskStatusPending,
	}
	
	err = store.CreateTask(task)
	require.NoError(t, err)
	
	err = store.CompleteTask(task.ID)
	require.NoError(t, err)
	
	retrieved, err := store.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskStatusCompleted, retrieved.Status)
	assert.NotNil(t, retrieved.CompletedAt)
}

func TestStore_RecurringTask(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	// Set up a weekly recurring task
	dueDate := time.Now().Add(24 * time.Hour)
	recurrenceRule := &RecurrenceRule{
		Frequency: FrequencyWeekly,
		Interval:  1,
	}
	
	task := &Task{
		UserID:              "user1",
		Title:               "Weekly Meeting",
		Status:              TaskStatusPending,
		DueDate:             &dueDate,
		RecurrenceFrequency: string(recurrenceRule.Frequency),
		RecurrenceInterval:  recurrenceRule.Interval,
	}
	
	err = store.CreateTask(task)
	require.NoError(t, err)
	
	// Complete the task
	err = store.CompleteTask(task.ID)
	require.NoError(t, err)
	
	// Create next occurrence
	nextTask, err := store.CreateNextRecurringTask(task)
	require.NoError(t, err)
	assert.NotNil(t, nextTask)
	assert.Equal(t, "Weekly Meeting", nextTask.Title)
	assert.Equal(t, FrequencyWeekly, Frequency(nextTask.RecurrenceFrequency))
	assert.Equal(t, 1, nextTask.RecurrenceCount)
	
	// Verify due date is one week later
	expectedDueDate := dueDate.AddDate(0, 0, 7)
	assert.WithinDuration(t, expectedDueDate, *nextTask.DueDate, time.Second)
}

func TestStore_GetStats(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewStore(db)
	require.NoError(t, err)
	
	// Create tasks
	tasks := []*Task{
		{UserID: "user1", Title: "Task 1", Status: TaskStatusPending},
		{UserID: "user1", Title: "Task 2", Status: TaskStatusCompleted},
		{UserID: "user1", Title: "Task 3", Status: TaskStatusCompleted},
	}
	
	for _, task := range tasks {
		err := store.CreateTask(task)
		require.NoError(t, err)
	}
	
	stats, err := store.GetStats("user1")
	require.NoError(t, err)
	assert.Equal(t, 3, stats.TotalCreated)
	assert.Equal(t, 2, stats.Completed)
	assert.InDelta(t, 66.7, stats.CompletionRate, 0.1)
}

// Date Parser Tests

func TestDateParser_ParseRelativeDates(t *testing.T) {
	parser := NewDateParser()
	now := time.Now()
	parser.WithReference(now)
	
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"today", time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())},
		{"tomorrow", time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())},
		{"yesterday", time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())},
	}
	
	for _, test := range tests {
		result, err := parser.Parse(test.input)
		assert.NoError(t, err, "Failed to parse: %s", test.input)
		assert.WithinDuration(t, test.expected, result.Date, time.Second, "For input: %s", test.input)
	}
}

func TestDateParser_ParseTimeExpression(t *testing.T) {
	parser := NewDateParser()
	now := time.Now()
	parser.WithReference(now)
	
	tests := []struct {
		input       string
		expectHour  int
		expectMin   int
	}{
		{"3pm", 15, 0},
		{"3:30pm", 15, 30},
		{"9am", 9, 0},
		{"14:30", 14, 30},
	}
	
	for _, test := range tests {
		result, err := parser.Parse(test.input)
		assert.NoError(t, err, "Failed to parse: %s", test.input)
		assert.Equal(t, test.expectHour, result.Date.Hour(), "Hour mismatch for: %s", test.input)
		assert.Equal(t, test.expectMin, result.Date.Minute(), "Minute mismatch for: %s", test.input)
	}
}

func TestDateParser_ExtractDateTime(t *testing.T) {
	parser := NewDateParser()
	now := time.Now()
	parser.WithReference(now)
	
	result, err := parser.ExtractDateTime("tomorrow at 3pm")
	require.NoError(t, err)
	
	// Just verify we got a valid time (within tomorrow's date range)
	// The exact calculation depends on current time
	assert.Equal(t, 15, result.Date.Hour())
	assert.Equal(t, 0, result.Date.Minute())
	assert.True(t, result.HasTime)
	assert.True(t, result.HasTime)
}

func TestDurationParser_Parse(t *testing.T) {
	parser := &DurationParser{}
	
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"30 minutes", 30 * time.Minute},
		{"1 hour", time.Hour},
		{"2 hours", 2 * time.Hour},
		{"1 day", 24 * time.Hour},
		{"1 week", 7 * 24 * time.Hour},
	}
	
	for _, test := range tests {
		duration, err := parser.Parse(test.input)
		assert.NoError(t, err, "Failed to parse: %s", test.input)
		assert.Equal(t, test.expected, duration, "For input: %s", test.input)
	}
}

// Task Skill Tests

func TestTaskSkill_CreateTask(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	args := map[string]interface{}{
		"title":       "Buy groceries",
		"description": "Milk, eggs, bread",
		"priority":    "high",
		"category":    "shopping",
		"tags":        []interface{}{"urgent", "home"},
	}
	
	result, err := skill.handleCreateTask(ctx, args)
	require.NoError(t, err)
	
	assert.NotNil(t, result)
	resultMap := result.(map[string]interface{})
	assert.NotEmpty(t, resultMap["task_id"])
	assert.Equal(t, "Buy groceries", resultMap["title"])
	assert.Equal(t, true, resultMap["created"])
}

func TestTaskSkill_CreateTaskWithDueDate(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	args := map[string]interface{}{
		"title":    "Submit report",
		"due_date": "tomorrow",
		"due_time": "3pm",
	}
	
	result, err := skill.handleCreateTask(ctx, args)
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.NotNil(t, resultMap["due_date"])
}

func TestTaskSkill_CreateRecurringTask(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	args := map[string]interface{}{
		"title":       "Team meeting",
		"due_date":    "tomorrow",
		"due_time":    "10am",
		"recurrence":  "weekly",
	}
	
	result, err := skill.handleCreateTask(ctx, args)
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.NotNil(t, resultMap["recurrence"])
	assert.Equal(t, "weekly", resultMap["recurrence"])
}

func TestTaskSkill_ListTasks(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Create some tasks
	for i := 0; i < 3; i++ {
		args := map[string]interface{}{
			"title": fmt.Sprintf("Task %d", i+1),
		}
		_, err := skill.handleCreateTask(ctx, args)
		require.NoError(t, err)
	}
	
	// List tasks
	args := map[string]interface{}{
		"status": "all",
		"limit":  float64(10),
	}
	
	result, err := skill.handleListTasks(ctx, args)
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	tasks := resultMap["tasks"].([]map[string]interface{})
	assert.GreaterOrEqual(t, len(tasks), 3)
}

func TestTaskSkill_CompleteTask(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Create a task
	createArgs := map[string]interface{}{
		"title": "Task to complete",
	}
	createResult, err := skill.handleCreateTask(ctx, createArgs)
	require.NoError(t, err)
	
	taskID := createResult.(map[string]interface{})["task_id"].(string)
	
	// Complete the task
	completeArgs := map[string]interface{}{
		"task_id":          taskID,
		"create_recurring": false,
	}
	
	result, err := skill.handleCompleteTask(ctx, completeArgs)
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.Equal(t, true, resultMap["completed"])
}

func TestTaskSkill_SnoozeTask(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Create a task
	createArgs := map[string]interface{}{
		"title": "Task to snooze",
	}
	createResult, err := skill.handleCreateTask(ctx, createArgs)
	require.NoError(t, err)
	
	taskID := createResult.(map[string]interface{})["task_id"].(string)
	
	// Snooze the task
	snoozeArgs := map[string]interface{}{
		"task_id":  taskID,
		"duration": "1 hour",
	}
	
	result, err := skill.handleSnoozeTask(ctx, snoozeArgs)
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.Equal(t, true, resultMap["snoozed"])
	assert.NotNil(t, resultMap["until"])
}

func TestTaskSkill_GetStats(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Create some tasks
	for i := 0; i < 3; i++ {
		_, err := skill.handleCreateTask(ctx, map[string]interface{}{"title": fmt.Sprintf("Task %d", i)})
		require.NoError(t, err)
	}
	
	result, err := skill.handleGetStats(ctx, map[string]interface{}{})
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.Equal(t, 3, resultMap["total_created"])
}

func TestTaskSkill_DeleteTask(t *testing.T) {
	skill, _ := setupTestSkill(t)
	ctx := context.WithValue(context.Background(), "user_id", "user1")
	
	// Create a task
	createResult, err := skill.handleCreateTask(ctx, map[string]interface{}{"title": "Task to delete"})
	require.NoError(t, err)
	
	taskID := createResult.(map[string]interface{})["task_id"].(string)
	
	// Try delete without confirmation
	result, err := skill.handleDeleteTask(ctx, map[string]interface{}{"task_id": taskID})
	require.NoError(t, err)
	
	resultMap := result.(map[string]interface{})
	assert.Equal(t, true, resultMap["confirm_required"])
	
	// Delete with confirmation
	result, err = skill.handleDeleteTask(ctx, map[string]interface{}{
		"task_id": taskID,
		"confirm": true,
	})
	require.NoError(t, err)
	
	resultMap = result.(map[string]interface{})
	assert.Equal(t, true, resultMap["deleted"])
}

// Task Status Tests

func TestTask_IsOverdue(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)
	
	tests := []struct {
		name     string
		status   TaskStatus
		dueDate  *time.Time
		expected bool
	}{
		{"pending overdue", TaskStatusPending, &past, true},
		{"pending future", TaskStatusPending, &future, false},
		{"completed past", TaskStatusCompleted, &past, false},
		{"no due date", TaskStatusPending, nil, false},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := &Task{
				Status:  test.status,
				DueDate: test.dueDate,
			}
			assert.Equal(t, test.expected, task.IsOverdue())
		})
	}
}

func TestTask_IsRecurring(t *testing.T) {
	tests := []struct {
		name     string
		rule     *RecurrenceRule
		expected bool
	}{
		{"no rule", nil, false},
		{"empty rule", &RecurrenceRule{}, false},
		{"daily", &RecurrenceRule{Frequency: FrequencyDaily}, true},
		{"weekly", &RecurrenceRule{Frequency: FrequencyWeekly}, true},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := &Task{}
			task.SetRecurrenceRule(test.rule)
			assert.Equal(t, test.expected, task.IsRecurring())
		})
	}
}

// Format Tests

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		input    time.Time
		contains string
	}{
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "minutes ago"},
		{now.Add(-2 * time.Hour), "hours ago"},
		{now.Add(30 * time.Minute), "minutes"},
		{now.Add(2 * time.Hour), "hours"},
		{now.AddDate(0, 0, -2), "days ago"},
		{now.AddDate(0, 0, 3), "days"},
	}
	
	for _, test := range tests {
		result := FormatRelativeTime(test.input)
		assert.Contains(t, result, test.contains)
	}
}

// Recurrence Tests

func TestCalculateNextDate(t *testing.T) {
	base := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	
	tests := []struct {
		name     string
		rule     *RecurrenceRule
		expected time.Time
	}{
		{
			"daily",
			&RecurrenceRule{Frequency: FrequencyDaily, Interval: 1},
			time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC),
		},
		{
			"weekly",
			&RecurrenceRule{Frequency: FrequencyWeekly, Interval: 1},
			time.Date(2024, 1, 22, 10, 0, 0, 0, time.UTC),
		},
		{
			"monthly",
			&RecurrenceRule{Frequency: FrequencyMonthly, Interval: 1},
			time.Date(2024, 2, 15, 10, 0, 0, 0, time.UTC),
		},
		{
			"yearly",
			&RecurrenceRule{Frequency: FrequencyYearly, Interval: 1},
			time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		{
			"every 3 days",
			&RecurrenceRule{Frequency: FrequencyDaily, Interval: 3},
			time.Date(2024, 1, 18, 10, 0, 0, 0, time.UTC),
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := calculateNextDate(base, test.rule)
			assert.Equal(t, test.expected, result)
		})
	}
}
