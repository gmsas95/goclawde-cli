package tasks

import (
	"time"
)

// Task represents a user task or reminder
type Task struct {
	ID          string     `json:"id" gorm:"primaryKey"`
	UserID      string     `json:"user_id" gorm:"index"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status" gorm:"index"`
	Priority    Priority   `json:"priority"`
	
	// Timing
	DueDate     *time.Time `json:"due_date,omitempty" gorm:"index"`
	RemindAt    *time.Time `json:"remind_at,omitempty" gorm:"index"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	
	// Recurrence (stored as JSON)
	RecurrenceFrequency  string `json:"recurrence_frequency,omitempty"`
	RecurrenceInterval   int    `json:"recurrence_interval,omitempty"`
	RecurrenceEndDate    *time.Time `json:"recurrence_end_date,omitempty"`
	RecurrenceOccurrences int    `json:"recurrence_occurrences,omitempty"`
	RecurrenceCount      int    `json:"recurrence_count,omitempty"`
	
	// Location
	Location    string  `json:"location,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
	GeofenceRadius int  `json:"geofence_radius,omitempty"` // meters
	
	// Tags and categories
	Tags       string   `json:"tags"` // Comma-separated tags
	Category   string   `json:"category,omitempty"`
	Source     string   `json:"source,omitempty"` // natural, email, document, etc.
	
	// Metadata
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusCancelled  TaskStatus = "cancelled"
	TaskStatusSnoozed    TaskStatus = "snoozed"
)

// Priority represents task priority
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// RecurrenceRule defines how a task repeats
type RecurrenceRule struct {
	Frequency  Frequency `json:"frequency"` // daily, weekly, monthly, yearly
	Interval   int       `json:"interval"`  // every N days/weeks/months
	ByWeekday  []int     `json:"by_weekday,omitempty"` // 0=Sunday, 1=Monday, etc.
	ByMonthDay []int     `json:"by_month_day,omitempty"`
	EndDate    *time.Time `json:"end_date,omitempty"`
	Occurrences int      `json:"occurrences,omitempty"`
	Count       int       `json:"count"` // how many times created so far
}

// Frequency represents recurrence frequency
type Frequency string

const (
	FrequencyDaily   Frequency = "daily"
	FrequencyWeekly  Frequency = "weekly"
	FrequencyMonthly Frequency = "monthly"
	FrequencyYearly  Frequency = "yearly"
)

// IsRecurring returns true if the task is recurring
func (t *Task) IsRecurring() bool {
	return t.RecurrenceFrequency != ""
}

// GetRecurrenceRule returns the recurrence rule
func (t *Task) GetRecurrenceRule() *RecurrenceRule {
	if !t.IsRecurring() {
		return nil
	}
	return &RecurrenceRule{
		Frequency:   Frequency(t.RecurrenceFrequency),
		Interval:    t.RecurrenceInterval,
		EndDate:     t.RecurrenceEndDate,
		Occurrences: t.RecurrenceOccurrences,
		Count:       t.RecurrenceCount,
	}
}

// IsOverdue returns true if the task is overdue
func (t *Task) IsOverdue() bool {
	if t.Status == TaskStatusCompleted || t.Status == TaskStatusCancelled {
		return false
	}
	if t.DueDate == nil {
		return false
	}
	return time.Now().After(*t.DueDate)
}

// ShouldRemind returns true if the task should trigger a reminder now
func (t *Task) ShouldRemind() bool {
	if t.Status != TaskStatusPending {
		return false
	}
	if t.RemindAt == nil {
		return false
	}
	return time.Now().After(*t.RemindAt)
}

// SetRecurrenceRule sets the recurrence rule
func (t *Task) SetRecurrenceRule(rule *RecurrenceRule) {
	if rule == nil {
		t.RecurrenceFrequency = ""
		t.RecurrenceInterval = 0
		t.RecurrenceEndDate = nil
		t.RecurrenceOccurrences = 0
		t.RecurrenceCount = 0
		return
	}
	t.RecurrenceFrequency = string(rule.Frequency)
	t.RecurrenceInterval = rule.Interval
	t.RecurrenceEndDate = rule.EndDate
	t.RecurrenceOccurrences = rule.Occurrences
	t.RecurrenceCount = rule.Count
}

// Reminder represents a reminder instance
type Reminder struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	TaskID    string    `json:"task_id" gorm:"index"`
	UserID    string    `json:"user_id" gorm:"index"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Channel   string    `json:"channel"` // telegram, web, push
	SentAt    *time.Time `json:"sent_at,omitempty"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// TaskList represents a list of tasks
type TaskList struct {
	Tasks      []Task `json:"tasks"`
	Total      int    `json:"total"`
	Pending    int    `json:"pending"`
	Overdue    int    `json:"overdue"`
	DueToday   int    `json:"due_today"`
	DueWeek    int    `json:"due_week"`
}

// TaskStats represents task statistics
type TaskStats struct {
	TotalCreated   int `json:"total_created"`
	Completed      int `json:"completed"`
	CompletionRate float64 `json:"completion_rate"`
	OverdueCount   int `json:"overdue_count"`
	AvgCompletionTime time.Duration `json:"avg_completion_time"`
}

// ParseNaturalDateResult represents the result of parsing natural language date
type ParseNaturalDateResult struct {
	Date        time.Time `json:"date"`
	IsRelative  bool      `json:"is_relative"`
	Description string    `json:"description"`
}
