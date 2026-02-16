package tasks

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store handles task persistence
type Store struct {
	db *gorm.DB
}

// NewStore creates a new task store
func NewStore(db *gorm.DB) (*Store, error) {
	store := &Store{db: db}
	
	// Auto-migrate schemas
	if err := db.AutoMigrate(&Task{}, &Reminder{}); err != nil {
		return nil, fmt.Errorf("failed to migrate task schemas: %w", err)
	}
	
	return store, nil
}

// generateID generates a unique ID
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "task_" + hex.EncodeToString(bytes)
}

// CreateTask creates a new task
func (s *Store) CreateTask(task *Task) error {
	if task.ID == "" {
		task.ID = generateID()
	}
	if task.Status == "" {
		task.Status = TaskStatusPending
	}
	if task.Priority == "" {
		task.Priority = PriorityMedium
	}
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	
	return s.db.Create(task).Error
}

// GetTask retrieves a task by ID
func (s *Store) GetTask(taskID string) (*Task, error) {
	var task Task
	err := s.db.Where("id = ?", taskID).First(&task).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &task, err
}

// UpdateTask updates an existing task
func (s *Store) UpdateTask(task *Task) error {
	task.UpdatedAt = time.Now()
	return s.db.Save(task).Error
}

// DeleteTask deletes a task
func (s *Store) DeleteTask(taskID string) error {
	return s.db.Where("id = ?", taskID).Delete(&Task{}).Error
}

// ListTasks lists tasks with optional filters
func (s *Store) ListTasks(userID string, opts ListOptions) (*TaskList, error) {
	query := s.db.Where("user_id = ?", userID)
	
	// Apply status filter
	if len(opts.Status) > 0 {
		query = query.Where("status IN ?", opts.Status)
	}
	
	// Apply priority filter
	if opts.Priority != "" {
		query = query.Where("priority = ?", opts.Priority)
	}
	
	// Apply category filter
	if opts.Category != "" {
		query = query.Where("category = ?", opts.Category)
	}
	
	// Apply tag filter
	if opts.Tags != "" {
		query = query.Where("tags LIKE ?", fmt.Sprintf("%%%s%%", opts.Tags))
	}
	
	// Apply date filters
	if opts.DueBefore != nil {
		query = query.Where("due_date <= ?", *opts.DueBefore)
	}
	if opts.DueAfter != nil {
		query = query.Where("due_date >= ?", *opts.DueAfter)
	}
	
	// Apply search
	if opts.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", opts.Search)
		query = query.Where(
			"title LIKE ? OR description LIKE ? OR location LIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}
	
	// Order by
	orderBy := "due_date ASC, created_at DESC"
	if opts.OrderBy != "" {
		orderBy = opts.OrderBy
	}
	query = query.Order(orderBy)
	
	// Pagination
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}
	
	var tasks []Task
	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}
	
	// Get counts
	var total int64
	s.db.Model(&Task{}).Where("user_id = ?", userID).Count(&total)
	
	var pending int64
	s.db.Model(&Task{}).Where("user_id = ? AND status = ?", userID, TaskStatusPending).Count(&pending)
	
	now := time.Now()
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	weekEnd := todayEnd.AddDate(0, 0, 7)
	
	var overdue int64
	s.db.Model(&Task{}).
		Where("user_id = ? AND status = ? AND due_date < ?", userID, TaskStatusPending, now).
		Count(&overdue)
	
	var dueToday int64
	s.db.Model(&Task{}).
		Where("user_id = ? AND status = ? AND due_date <= ? AND due_date >= ?", 
			userID, TaskStatusPending, todayEnd, now).
		Count(&dueToday)
	
	var dueWeek int64
	s.db.Model(&Task{}).
		Where("user_id = ? AND status = ? AND due_date <= ? AND due_date >= ?", 
			userID, TaskStatusPending, weekEnd, now).
		Count(&dueWeek)
	
	return &TaskList{
		Tasks:    tasks,
		Total:    int(total),
		Pending:  int(pending),
		Overdue:  int(overdue),
		DueToday: int(dueToday),
		DueWeek:  int(dueWeek),
	}, nil
}

// ListOptions contains filter options for listing tasks
type ListOptions struct {
	Status    []TaskStatus
	Priority  Priority
	Category  string
	Tags      string
	DueBefore *time.Time
	DueAfter  *time.Time
	Search    string
	OrderBy   string
	Limit     int
	Offset    int
}

// GetOverdueTasks gets all overdue tasks
func (s *Store) GetOverdueTasks(userID string) ([]Task, error) {
	var tasks []Task
	err := s.db.Where(
		"user_id = ? AND status = ? AND due_date < ?",
		userID, TaskStatusPending, time.Now(),
	).Find(&tasks).Error
	return tasks, err
}

// GetTasksDueSoon gets tasks due within the specified duration
func (s *Store) GetTasksDueSoon(userID string, within time.Duration) ([]Task, error) {
	deadline := time.Now().Add(within)
	var tasks []Task
	err := s.db.Where(
		"user_id = ? AND status = ? AND due_date <= ? AND due_date >= ?",
		userID, TaskStatusPending, deadline, time.Now(),
	).Order("due_date ASC").Find(&tasks).Error
	return tasks, err
}

// GetTasksWithReminders gets tasks that have reminders set
func (s *Store) GetTasksWithReminders(userID string, before time.Time) ([]Task, error) {
	var tasks []Task
	err := s.db.Where(
		"user_id = ? AND status = ? AND remind_at <= ? AND remind_at >= ?",
		userID, TaskStatusPending, before, time.Now().Add(-time.Minute),
	).Order("remind_at ASC").Find(&tasks).Error
	return tasks, err
}

// CompleteTask marks a task as completed
func (s *Store) CompleteTask(taskID string) error {
	now := time.Now()
	return s.db.Model(&Task{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":       TaskStatusCompleted,
		"completed_at": &now,
		"updated_at":   now,
	}).Error
}

// CreateNextRecurringTask creates the next occurrence of a recurring task
func (s *Store) CreateNextRecurringTask(task *Task) (*Task, error) {
	if !task.IsRecurring() {
		return nil, nil
	}
	
	rule := task.GetRecurrenceRule()
	if rule == nil {
		return nil, nil
	}
	
	// Check if should continue
	if rule.EndDate != nil && time.Now().After(*rule.EndDate) {
		return nil, nil
	}
	if rule.Occurrences > 0 && rule.Count >= rule.Occurrences {
		return nil, nil
	}
	
	// Calculate next due date
	var nextDueDate time.Time
	if task.DueDate != nil {
		nextDueDate = calculateNextDate(*task.DueDate, rule)
	}
	
	// Calculate next reminder
	var nextRemindAt *time.Time
	if task.RemindAt != nil && task.DueDate != nil {
		diff := task.DueDate.Sub(*task.RemindAt)
		next := nextDueDate.Add(-diff)
		nextRemindAt = &next
	}
	
	// Create new task
	nextTask := &Task{
		UserID:         task.UserID,
		Title:          task.Title,
		Description:    task.Description,
		Status:         TaskStatusPending,
		Priority:       task.Priority,
		DueDate:        &nextDueDate,
		RemindAt:       nextRemindAt,
		Location:       task.Location,
		Latitude:       task.Latitude,
		Longitude:      task.Longitude,
		GeofenceRadius: task.GeofenceRadius,
		Tags:                  task.Tags,
		Category:              task.Category,
		Source:                task.Source,
		RecurrenceFrequency:   string(rule.Frequency),
		RecurrenceInterval:    rule.Interval,
		RecurrenceEndDate:     rule.EndDate,
		RecurrenceOccurrences: rule.Occurrences,
		RecurrenceCount:       rule.Count + 1,
	}
	
	if err := s.CreateTask(nextTask); err != nil {
		return nil, err
	}
	
	return nextTask, nil
}

// calculateNextDate calculates the next occurrence date
func calculateNextDate(from time.Time, rule *RecurrenceRule) time.Time {
	interval := rule.Interval
	if interval == 0 {
		interval = 1
	}
	
	switch rule.Frequency {
	case FrequencyDaily:
		return from.AddDate(0, 0, interval)
	case FrequencyWeekly:
		return from.AddDate(0, 0, 7*interval)
	case FrequencyMonthly:
		return from.AddDate(0, interval, 0)
	case FrequencyYearly:
		return from.AddDate(interval, 0, 0)
	default:
		return from.AddDate(0, 0, interval)
	}
}

// CreateReminder creates a reminder record
func (s *Store) CreateReminder(reminder *Reminder) error {
	if reminder.ID == "" {
		reminder.ID = generateID()
	}
	reminder.CreatedAt = time.Now()
	return s.db.Create(reminder).Error
}

// MarkReminderSent marks a reminder as sent
func (s *Store) MarkReminderSent(reminderID string) error {
	now := time.Now()
	return s.db.Model(&Reminder{}).Where("id = ?", reminderID).Update("sent_at", &now).Error
}

// MarkReminderRead marks a reminder as read
func (s *Store) MarkReminderRead(reminderID string) error {
	now := time.Now()
	return s.db.Model(&Reminder{}).Where("id = ?", reminderID).Update("read_at", &now).Error
}

// GetStats gets task statistics
func (s *Store) GetStats(userID string) (*TaskStats, error) {
	var total int64
	s.db.Model(&Task{}).Where("user_id = ?", userID).Count(&total)
	
	var completed int64
	s.db.Model(&Task{}).Where("user_id = ? AND status = ?", userID, TaskStatusCompleted).Count(&completed)
	
	var overdue int64
	s.db.Model(&Task{}).
		Where("user_id = ? AND status = ? AND due_date < ?", userID, TaskStatusPending, time.Now()).
		Count(&overdue)
	
	completionRate := 0.0
	if total > 0 {
		completionRate = float64(completed) / float64(total) * 100
	}
	
	return &TaskStats{
		TotalCreated:   int(total),
		Completed:      int(completed),
		CompletionRate: completionRate,
		OverdueCount:   int(overdue),
	}, nil
}

// SearchTasks searches tasks by text
func (s *Store) SearchTasks(userID string, query string) ([]Task, error) {
	var tasks []Task
	searchPattern := fmt.Sprintf("%%%s%%", query)
	err := s.db.Where(
		"user_id = ? AND (title LIKE ? OR description LIKE ? OR location LIKE ?)",
		userID, searchPattern, searchPattern, searchPattern,
	).Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

// GetTasksByCategory gets tasks by category
func (s *Store) GetTasksByCategory(userID string, category string) ([]Task, error) {
	var tasks []Task
	err := s.db.Where("user_id = ? AND category = ?", userID, category).
		Order("due_date ASC").Find(&tasks).Error
	return tasks, err
}

// GetTasksByTag gets tasks by tag
func (s *Store) GetTasksByTag(userID string, tag string) ([]Task, error) {
	var tasks []Task
	err := s.db.Where("user_id = ? AND tags LIKE ?", userID, fmt.Sprintf("%%%s%%", tag)).
		Order("due_date ASC").Find(&tasks).Error
	return tasks, err
}

// BulkUpdateStatus updates status for multiple tasks
func (s *Store) BulkUpdateStatus(taskIDs []string, status TaskStatus) error {
	return s.db.Model(&Task{}).Where("id IN ?", taskIDs).Update("status", status).Error
}

// DeleteCompletedTasks deletes all completed tasks older than specified duration
func (s *Store) DeleteCompletedTasks(userID string, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	return s.db.Where(
		"user_id = ? AND status = ? AND completed_at < ?",
		userID, TaskStatusCompleted, cutoff,
	).Delete(&Task{}).Error
}

// UpsertTask creates or updates a task
func (s *Store) UpsertTask(task *Task) error {
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(task).Error
}
