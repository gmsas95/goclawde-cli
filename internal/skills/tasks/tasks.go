package tasks

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/skills"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TaskSkill provides task and reminder management
type TaskSkill struct {
	*skills.BaseSkill
	store           *Store
	scheduler       *ReminderService
	dateParser      *DateParser
	logger          *zap.Logger
	reminderCallback ReminderCallback
}

// TaskConfig contains task skill configuration
type TaskConfig struct {
	Enabled          bool
	DefaultChannel   string // telegram, web, push
	DefaultPriority  Priority
	ReminderCallback ReminderCallback
}

// NewTaskSkill creates a new task skill
func NewTaskSkill(db *gorm.DB, config TaskConfig, logger *zap.Logger) (*TaskSkill, error) {
	store, err := NewStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create task store: %w", err)
	}
	
	reminderCallback := config.ReminderCallback
	if reminderCallback == nil {
		reminderCallback = func(r *Reminder, t *Task) error {
			logger.Info("Reminder triggered",
				zap.String("task", t.Title),
				zap.String("user", t.UserID),
			)
			return nil
		}
	}
	
	scheduler := NewReminderService(store, logger, reminderCallback)
	
	skill := &TaskSkill{
		BaseSkill:        skills.NewBaseSkill("tasks", "Task & Reminder Management", "1.0.0"),
		store:            store,
		scheduler:        scheduler,
		dateParser:       NewDateParser(),
		logger:           logger,
		reminderCallback: reminderCallback,
	}
	
	// Register tools
	skill.registerTools()
	
	return skill, nil
}

// registerTools registers all task management tools
func (t *TaskSkill) registerTools() {
	tools := []skills.Tool{
		{
			Name:        "create_task",
			Description: "Create a new task or reminder with natural language support. Examples: 'Buy groceries tomorrow', 'Call mom at 3pm', 'Pay rent every month'",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Task title (what needs to be done)",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Optional detailed description",
					},
					"due_date": map[string]interface{}{
						"type":        "string",
						"description": "Due date in natural language (e.g., 'tomorrow', 'next Tuesday', 'Jan 15') or ISO format",
					},
					"due_time": map[string]interface{}{
						"type":        "string",
						"description": "Due time (e.g., '3pm', '15:30')",
					},
					"priority": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"low", "medium", "high", "critical"},
						"description": "Task priority",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Task category (e.g., 'work', 'personal', 'shopping')",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Tags for the task",
					},
					"location": map[string]interface{}{
						"type":        "string",
						"description": "Location where task should be done",
					},
					"remind_at": map[string]interface{}{
						"type":        "string",
						"description": "When to send reminder (natural language or time before due date like '30 minutes before')",
					},
					"recurrence": map[string]interface{}{
						"type":        "string",
						"description": "Recurrence pattern (e.g., 'daily', 'weekly', 'monthly', 'every Monday')",
					},
				},
				"required": []string{"title"},
			},
		},
		{
			Name:        "update_task",
			Description: "Update an existing task",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the task to update",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "New task title",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "New description",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"pending", "in_progress", "completed", "cancelled"},
						"description": "New status",
					},
					"due_date": map[string]interface{}{
						"type":        "string",
						"description": "New due date",
					},
					"priority": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"low", "medium", "high", "critical"},
						"description": "New priority",
					},
					"add_tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Tags to add",
					},
					"remove_tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Tags to remove",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "delete_task",
			Description: "Delete a task",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the task to delete",
					},
					"confirm": map[string]interface{}{
						"type":        "boolean",
						"description": "Confirm deletion",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "list_tasks",
			Description: "List tasks with optional filters",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"pending", "in_progress", "completed", "all"},
						"description": "Filter by status",
					},
					"priority": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"low", "medium", "high", "critical"},
						"description": "Filter by priority",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Filter by category",
					},
					"tag": map[string]interface{}{
						"type":        "string",
						"description": "Filter by tag",
					},
					"due": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"today", "tomorrow", "week", "overdue", "any"},
						"description": "Filter by due date",
					},
					"search": map[string]interface{}{
						"type":        "string",
						"description": "Search in title and description",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"default":     20,
						"description": "Maximum number of tasks to return",
					},
				},
			},
		},
		{
			Name:        "complete_task",
			Description: "Mark a task as completed",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the task to complete",
					},
					"create_recurring": map[string]interface{}{
						"type":        "boolean",
						"default":     true,
						"description": "Create next occurrence for recurring tasks",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "snooze_task",
			Description: "Snooze a task reminder",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the task to snooze",
					},
					"duration": map[string]interface{}{
						"type":        "string",
						"description": "Snooze duration (e.g., '30 minutes', '1 hour', 'tomorrow')",
						"default":     "1 hour",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "get_task_stats",
			Description: "Get task statistics and overview",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"period": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"day", "week", "month", "all"},
						"default":     "all",
						"description": "Time period for statistics",
					},
				},
			},
		},
	}
	
	for _, tool := range tools {
		tool.Handler = t.handleTool(tool.Name)
		t.AddTool(tool)
	}
}

// handleTool handles tool calls
func (t *TaskSkill) handleTool(name string) skills.ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		switch name {
		case "create_task":
			return t.handleCreateTask(ctx, args)
		case "update_task":
			return t.handleUpdateTask(ctx, args)
		case "delete_task":
			return t.handleDeleteTask(ctx, args)
		case "list_tasks":
			return t.handleListTasks(ctx, args)
		case "complete_task":
			return t.handleCompleteTask(ctx, args)
		case "snooze_task":
			return t.handleSnoozeTask(ctx, args)
		case "get_task_stats":
			return t.handleGetStats(ctx, args)
		default:
			return nil, fmt.Errorf("unknown tool: %s", name)
		}
	}
}

// handleCreateTask creates a new task
func (t *TaskSkill) handleCreateTask(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	title, _ := args["title"].(string)
	description, _ := args["description"].(string)
	dueDateStr, _ := args["due_date"].(string)
	dueTimeStr, _ := args["due_time"].(string)
	priorityStr, _ := args["priority"].(string)
	category, _ := args["category"].(string)
	location, _ := args["location"].(string)
	remindAtStr, _ := args["remind_at"].(string)
	recurrence, _ := args["recurrence"].(string)
	
	tags := ""
	if tagsArr, ok := args["tags"].([]interface{}); ok {
		tagList := []string{}
		for _, tag := range tagsArr {
			if s, ok := tag.(string); ok {
				tagList = append(tagList, s)
			}
		}
		tags = strings.Join(tagList, ",")
	}
	
	// Parse priority
	priority := PriorityMedium
	if priorityStr != "" {
		priority = Priority(priorityStr)
	}
	
	// Parse due date
	var dueDate *time.Time
	if dueDateStr != "" {
		dateInput := dueDateStr
		if dueTimeStr != "" {
			dateInput = fmt.Sprintf("%s at %s", dueDateStr, dueTimeStr)
		}
		
		result, err := t.dateParser.ExtractDateTime(dateInput)
		if err != nil {
			return nil, fmt.Errorf("could not parse due date: %w", err)
		}
		dueDate = &result.Date
	}
	
	// Parse reminder
	var remindAt *time.Time
	if remindAtStr != "" {
		remindAt = t.parseReminderTime(remindAtStr, dueDate)
	}
	
	// Get user ID from context
	userID := t.getUserID(ctx)
	
	task := &Task{
		UserID:      userID,
		Title:       title,
		Description: description,
		Status:      TaskStatusPending,
		Priority:    priority,
		DueDate:     dueDate,
		RemindAt:    remindAt,
		Tags:        tags,
		Category:    category,
		Location:    location,
		Source:      "natural",
	}
	
	// Parse recurrence
	if recurrence != "" {
		recurrenceRule := t.parseRecurrence(recurrence)
		task.SetRecurrenceRule(recurrenceRule)
	}
	
	if err := t.store.CreateTask(task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	
	// Schedule reminder if set
	if remindAt != nil {
		t.scheduler.SetReminder(task.ID, *remindAt)
	}
	
	// Format response
	response := map[string]interface{}{
		"task_id":     task.ID,
		"title":       task.Title,
		"status":      task.Status,
		"priority":    task.Priority,
		"created":     true,
		"description": t.formatTaskDescription(task),
	}
	
	if dueDate != nil {
		response["due_date"] = dueDate.Format("Jan 2, 2006 3:04 PM")
	}
	if remindAt != nil {
		response["reminder"] = FormatRelativeTime(*remindAt)
	}
	if task.IsRecurring() {
		response["recurrence"] = task.RecurrenceFrequency
	}
	
	t.logger.Info("Task created",
		zap.String("task_id", task.ID),
		zap.String("title", task.Title),
		zap.String("user_id", userID),
	)
	
	return response, nil
}

// handleUpdateTask updates an existing task
func (t *TaskSkill) handleUpdateTask(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	
	task, err := t.store.GetTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	
	// Update fields if provided
	if title, ok := args["title"].(string); ok && title != "" {
		task.Title = title
	}
	if description, ok := args["description"].(string); ok {
		task.Description = description
	}
	if status, ok := args["status"].(string); ok && status != "" {
		task.Status = TaskStatus(status)
	}
	if priority, ok := args["priority"].(string); ok && priority != "" {
		task.Priority = Priority(priority)
	}
	if dueDateStr, ok := args["due_date"].(string); ok && dueDateStr != "" {
		result, err := t.dateParser.Parse(dueDateStr)
		if err == nil {
			task.DueDate = &result.Date
		}
	}
	
	// Handle tags
	if addTags, ok := args["add_tags"].([]interface{}); ok {
		tagList := []string{}
		if task.Tags != "" {
			tagList = strings.Split(task.Tags, ",")
		}
		for _, tag := range addTags {
			if s, ok := tag.(string); ok {
				tagList = append(tagList, s)
			}
		}
		task.Tags = strings.Join(tagList, ",")
	}
	
	if err := t.store.UpdateTask(task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}
	
	return map[string]interface{}{
		"task_id": task.ID,
		"updated": true,
		"task":    task,
	}, nil
}

// handleDeleteTask deletes a task
func (t *TaskSkill) handleDeleteTask(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	confirm, _ := args["confirm"].(bool)
	
	if !confirm {
		return map[string]interface{}{
			"task_id": taskID,
			"confirm_required": true,
			"message": "Please confirm deletion by setting confirm=true",
		}, nil
	}
	
	if err := t.store.DeleteTask(taskID); err != nil {
		return nil, fmt.Errorf("failed to delete task: %w", err)
	}
	
	return map[string]interface{}{
		"task_id": taskID,
		"deleted": true,
	}, nil
}

// handleListTasks lists tasks with filters
func (t *TaskSkill) handleListTasks(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := t.getUserID(ctx)
	
	opts := ListOptions{
		Status: []TaskStatus{TaskStatusPending, TaskStatusInProgress},
	}
	
	// Parse status filter
	if status, ok := args["status"].(string); ok && status != "" {
		switch status {
		case "all":
			opts.Status = []TaskStatus{}
		case "pending":
			opts.Status = []TaskStatus{TaskStatusPending}
		case "in_progress":
			opts.Status = []TaskStatus{TaskStatusInProgress}
		case "completed":
			opts.Status = []TaskStatus{TaskStatusCompleted}
		}
	}
	
	// Parse priority filter
	if priority, ok := args["priority"].(string); ok && priority != "" {
		opts.Priority = Priority(priority)
	}
	
	// Parse category filter
	if category, ok := args["category"].(string); ok && category != "" {
		opts.Category = category
	}
	
	// Parse due date filter
	if due, ok := args["due"].(string); ok && due != "" {
		now := time.Now()
		todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		
		switch due {
		case "today":
			opts.DueAfter = &now
			opts.DueBefore = &todayEnd
		case "tomorrow":
			tomorrowStart := todayEnd.Add(time.Second)
			tomorrowEnd := todayEnd.Add(24 * time.Hour)
			opts.DueAfter = &tomorrowStart
			opts.DueBefore = &tomorrowEnd
		case "week":
			weekEnd := todayEnd.AddDate(0, 0, 7)
			opts.DueAfter = &now
			opts.DueBefore = &weekEnd
		case "overdue":
			opts.DueBefore = &now
		}
	}
	
	// Parse search
	if search, ok := args["search"].(string); ok && search != "" {
		opts.Search = search
	}
	
	// Parse limit
	if limit, ok := args["limit"].(float64); ok {
		opts.Limit = int(limit)
	} else {
		opts.Limit = 20
	}
	
	list, err := t.store.ListTasks(userID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	
	// Format tasks for display
	formattedTasks := make([]map[string]interface{}, len(list.Tasks))
	for i, task := range list.Tasks {
		formattedTasks[i] = t.formatTaskForDisplay(&task)
	}
	
	return map[string]interface{}{
		"tasks":      formattedTasks,
		"total":      list.Total,
		"pending":    list.Pending,
		"overdue":    list.Overdue,
		"due_today":  list.DueToday,
		"due_week":   list.DueWeek,
	}, nil
}

// handleCompleteTask marks a task as completed
func (t *TaskSkill) handleCompleteTask(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	createRecurring := true
	if cr, ok := args["create_recurring"].(bool); ok {
		createRecurring = cr
	}
	
	task, err := t.store.GetTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	
	// Cancel any pending reminder
	t.scheduler.CancelReminder(taskID)
	
	// Complete the task
	if err := t.store.CompleteTask(taskID); err != nil {
		return nil, fmt.Errorf("failed to complete task: %w", err)
	}
	
	result := map[string]interface{}{
		"task_id":   taskID,
		"completed": true,
		"task":      task.Title,
	}
	
	// Create next recurring task if applicable
	if createRecurring && task.IsRecurring() {
		nextTask, err := t.store.CreateNextRecurringTask(task)
		if err != nil {
			t.logger.Error("Failed to create recurring task", zap.Error(err))
		} else if nextTask != nil {
			result["next_occurrence"] = map[string]interface{}{
				"task_id":  nextTask.ID,
				"due_date": nextTask.DueDate.Format("Jan 2, 2006"),
			}
		}
	}
	
	return result, nil
}

// handleSnoozeTask snoozes a task reminder
func (t *TaskSkill) handleSnoozeTask(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	durationStr, _ := args["duration"].(string)
	if durationStr == "" {
		durationStr = "1 hour"
	}
	
	durationParser := &DurationParser{}
	duration, err := durationParser.Parse(durationStr)
	if err != nil {
		// Try to parse as date
		result, parseErr := t.dateParser.Parse(durationStr)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid duration: %s", durationStr)
		}
		duration = time.Until(result.Date)
	}
	
	if err := t.scheduler.SnoozeReminder(taskID, duration); err != nil {
		return nil, fmt.Errorf("failed to snooze task: %w", err)
	}
	
	return map[string]interface{}{
		"task_id":  taskID,
		"snoozed":  true,
		"duration": durationStr,
		"until":    time.Now().Add(duration).Format("3:04 PM"),
	}, nil
}

// handleGetStats gets task statistics
func (t *TaskSkill) handleGetStats(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := t.getUserID(ctx)
	
	stats, err := t.store.GetStats(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	
	// Get overdue tasks
	overdue, _ := t.store.GetOverdueTasks(userID)
	
	// Get tasks due today
	now := time.Now()
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	todayTasks, _ := t.store.ListTasks(userID, ListOptions{
		Status:    []TaskStatus{TaskStatusPending},
		DueAfter:  &now,
		DueBefore: &todayEnd,
	})
	
	return map[string]interface{}{
		"total_created":    stats.TotalCreated,
		"completed":        stats.Completed,
		"completion_rate":  fmt.Sprintf("%.1f%%", stats.CompletionRate),
		"overdue_count":    stats.OverdueCount,
		"overdue_tasks":    len(overdue),
		"due_today":        todayTasks.DueToday,
		"pending":          stats.TotalCreated - stats.Completed,
	}, nil
}

// Helper methods

func (t *TaskSkill) getUserID(ctx context.Context) string {
	// Try to get from context
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return "default_user"
}

func (t *TaskSkill) parseReminderTime(input string, dueDate *time.Time) *time.Time {
	// Handle relative reminders like "30 minutes before"
	patterns := []string{
		`(\d+)\s*minutes?\s*before`,
		`(\d+)\s*hours?\s*before`,
		`(\d+)\s*days?\s*before`,
	}
	
	if dueDate != nil {
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(input)
			if len(matches) > 1 {
				n, _ := strconv.Atoi(matches[1])
				
				switch {
				case strings.Contains(pattern, "minute"):
					t := dueDate.Add(-time.Duration(n) * time.Minute)
					return &t
				case strings.Contains(pattern, "hour"):
					t := dueDate.Add(-time.Duration(n) * time.Hour)
					return &t
				case strings.Contains(pattern, "day"):
					t := dueDate.AddDate(0, 0, -n)
					return &t
				}
			}
		}
	}
	
	// Try to parse as absolute time
	result, err := t.dateParser.Parse(input)
	if err == nil {
		return &result.Date
	}
	
	return nil
}

func (t *TaskSkill) parseRecurrence(input string) *RecurrenceRule {
	input = strings.ToLower(input)
	
	rule := &RecurrenceRule{Interval: 1}
	
	switch input {
	case "daily", "every day", "each day":
		rule.Frequency = FrequencyDaily
	case "weekly", "every week", "each week":
		rule.Frequency = FrequencyWeekly
	case "monthly", "every month", "each month":
		rule.Frequency = FrequencyMonthly
	case "yearly", "every year", "each year", "annually":
		rule.Frequency = FrequencyYearly
	default:
		// Check for patterns like "every N days"
		if strings.HasPrefix(input, "every ") {
			parts := strings.Fields(input[6:])
			if len(parts) >= 2 {
				if n, err := strconv.Atoi(parts[0]); err == nil {
					rule.Interval = n
					switch parts[1] {
					case "day", "days":
						rule.Frequency = FrequencyDaily
					case "week", "weeks":
						rule.Frequency = FrequencyWeekly
					case "month", "months":
						rule.Frequency = FrequencyMonthly
					case "year", "years":
						rule.Frequency = FrequencyYearly
					}
				}
			}
		}
		
		// Check for weekday patterns
		days := map[string]int{
			"sunday": 0, "monday": 1, "tuesday": 2, "wednesday": 3,
			"thursday": 4, "friday": 5, "saturday": 6,
		}
		for day, num := range days {
			if strings.Contains(input, day) {
				rule.Frequency = FrequencyWeekly
				rule.ByWeekday = []int{num}
				break
			}
		}
	}
	
	if rule.Frequency == "" {
		return nil
	}
	
	return rule
}

func (t *TaskSkill) formatTaskDescription(task *Task) string {
	parts := []string{task.Title}
	
	if task.IsOverdue() {
		parts = append(parts, "[OVERDUE]")
	} else if task.DueDate != nil {
		parts = append(parts, fmt.Sprintf("(Due: %s)", FormatRelativeTime(*task.DueDate)))
	}
	
	return strings.Join(parts, " ")
}

func (t *TaskSkill) formatTaskForDisplay(task *Task) map[string]interface{} {
	result := map[string]interface{}{
		"id":       task.ID,
		"title":    task.Title,
		"status":   task.Status,
		"priority": task.Priority,
	}
	
	if task.Description != "" {
		result["description"] = task.Description
	}
	
	if task.DueDate != nil {
		result["due_date"] = task.DueDate.Format("Jan 2, 3:04 PM")
		result["due_relative"] = FormatRelativeTime(*task.DueDate)
		result["is_overdue"] = task.IsOverdue()
	}
	
	if task.RemindAt != nil {
		result["reminder"] = FormatRelativeTime(*task.RemindAt)
	}
	
	if task.Tags != "" {
		result["tags"] = strings.Split(task.Tags, ",")
	}
	
	if task.Category != "" {
		result["category"] = task.Category
	}
	
	if task.Location != "" {
		result["location"] = task.Location
	}
	
	if task.IsRecurring() {
		result["recurring"] = task.RecurrenceFrequency
	}
	
	return result
}

// Additional tool handlers for natural language understanding

// ProcessNaturalLanguage processes natural language task requests
func (t *TaskSkill) ProcessNaturalLanguage(ctx context.Context, text string) (interface{}, error) {
	// This would use LLM to extract task information from natural language
	// For now, return a simple response
	return map[string]interface{}{
		"message": "I've noted your request. Use create_task tool to add it formally.",
		"text":    text,
	}, nil
}
