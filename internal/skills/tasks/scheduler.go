package tasks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ReminderCallback is called when a reminder is triggered
type ReminderCallback func(reminder *Reminder, task *Task) error

// Scheduler manages task reminders and recurring tasks
type Scheduler struct {
	store     *Store
	logger    *zap.Logger
	interval  time.Duration
	callback  ReminderCallback
	
	mu        sync.RWMutex
	running   bool
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// NewScheduler creates a new task scheduler
func NewScheduler(store *Store, logger *zap.Logger, callback ReminderCallback) *Scheduler {
	return &Scheduler{
		store:    store,
		logger:   logger,
		interval: 30 * time.Second, // Check every 30 seconds
		callback: callback,
		stopCh:   make(chan struct{}),
	}
}

// WithInterval sets the check interval
func (s *Scheduler) WithInterval(interval time.Duration) *Scheduler {
	s.interval = interval
	return s
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()
	
	s.logger.Info("Starting task scheduler", zap.Duration("interval", s.interval))
	
	s.wg.Add(1)
	go s.run(ctx)
	
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()
	
	s.wg.Wait()
	s.logger.Info("Task scheduler stopped")
	
	return nil
}

// IsRunning returns true if the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// run is the main scheduler loop
func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()
	
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	
	// Run immediately on start
	s.checkReminders(ctx)
	s.checkRecurringTasks(ctx)
	
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Scheduler context cancelled")
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkReminders(ctx)
			s.checkRecurringTasks(ctx)
		}
	}
}

// checkReminders checks for due reminders
func (s *Scheduler) checkReminders(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Panic in checkReminders", zap.Any("recover", r))
		}
	}()
	
	// Get all users (in a real app, this would be more selective)
	// For now, we'll check all pending tasks with reminders
	
	// Get tasks due for reminder
	checkWindow := time.Now().Add(s.interval)
	
	// This is a simplified version - in production you'd track which reminders have been sent
	// by querying the Reminders table
	
	s.logger.Debug("Checking for reminders", zap.Time("window", checkWindow))
}

// checkRecurringTasks checks for completed recurring tasks that need new instances
func (s *Scheduler) checkRecurringTasks(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Panic in checkRecurringTasks", zap.Any("recover", r))
		}
	}()
	
	// This would check for completed recurring tasks and create next instances
	s.logger.Debug("Checking recurring tasks")
}

// ScheduleReminder schedules a reminder for a task
func (s *Scheduler) ScheduleReminder(task *Task, remindAt time.Time, channel string) error {
	reminder := &Reminder{
		TaskID:  task.ID,
		UserID:  task.UserID,
		Title:   task.Title,
		Message: fmt.Sprintf("Reminder: %s", task.Title),
		Channel: channel,
	}
	
	return s.store.CreateReminder(reminder)
}

// TriggerReminder manually triggers a reminder
func (s *Scheduler) TriggerReminder(taskID string) error {
	task, err := s.store.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}
	
	reminder := &Reminder{
		TaskID:  task.ID,
		UserID:  task.UserID,
		Title:   task.Title,
		Message: fmt.Sprintf("Reminder: %s", task.Title),
		Channel: "push",
	}
	
	if s.callback != nil {
		if err := s.callback(reminder, task); err != nil {
			return fmt.Errorf("reminder callback failed: %w", err)
		}
	}
	
	return s.store.CreateReminder(reminder)
}

// SimpleScheduler is a lightweight in-memory scheduler for simple use cases
type SimpleScheduler struct {
	timers   map[string]*time.Timer
	store    *Store
	callback ReminderCallback
	mu       sync.RWMutex
}

// NewSimpleScheduler creates a new simple scheduler
func NewSimpleScheduler(store *Store, callback ReminderCallback) *SimpleScheduler {
	return &SimpleScheduler{
		timers:   make(map[string]*time.Timer),
		store:    store,
		callback: callback,
	}
}

// Schedule schedules a one-time reminder
func (s *SimpleScheduler) Schedule(taskID string, remindAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Cancel existing timer if any
	if timer, exists := s.timers[taskID]; exists {
		timer.Stop()
		delete(s.timers, taskID)
	}
	
	duration := time.Until(remindAt)
	if duration <= 0 {
		return fmt.Errorf("reminder time is in the past")
	}
	
	timer := time.AfterFunc(duration, func() {
		s.trigger(taskID)
	})
	
	s.timers[taskID] = timer
	return nil
}

// Cancel cancels a scheduled reminder
func (s *SimpleScheduler) Cancel(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if timer, exists := s.timers[taskID]; exists {
		timer.Stop()
		delete(s.timers, taskID)
	}
}

// trigger executes the reminder callback
func (s *SimpleScheduler) trigger(taskID string) {
	s.mu.Lock()
	delete(s.timers, taskID)
	s.mu.Unlock()
	
	task, err := s.store.GetTask(taskID)
	if err != nil || task == nil {
		return
	}
	
	if s.callback != nil {
		reminder := &Reminder{
			TaskID:  task.ID,
			UserID:  task.UserID,
			Title:   task.Title,
			Message: fmt.Sprintf("Reminder: %s", task.Title),
			Channel: "push",
		}
		s.callback(reminder, task)
	}
}

// ReminderService provides high-level reminder functionality
type ReminderService struct {
	store     *Store
	scheduler *SimpleScheduler
	logger    *zap.Logger
}

// NewReminderService creates a new reminder service
func NewReminderService(store *Store, logger *zap.Logger, callback ReminderCallback) *ReminderService {
	scheduler := NewSimpleScheduler(store, callback)
	return &ReminderService{
		store:     store,
		scheduler: scheduler,
		logger:    logger,
	}
}

// SetReminder sets a reminder for a task
func (rs *ReminderService) SetReminder(taskID string, remindAt time.Time) error {
	task, err := rs.store.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}
	
	// Update task with reminder time
	task.RemindAt = &remindAt
	if err := rs.store.UpdateTask(task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	
	// Schedule the reminder
	if err := rs.scheduler.Schedule(taskID, remindAt); err != nil {
		return fmt.Errorf("failed to schedule reminder: %w", err)
	}
	
	rs.logger.Info("Reminder set",
		zap.String("task_id", taskID),
		zap.Time("remind_at", remindAt),
	)
	
	return nil
}

// CancelReminder cancels a task's reminder
func (rs *ReminderService) CancelReminder(taskID string) error {
	task, err := rs.store.GetTask(taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}
	
	// Clear reminder time
	task.RemindAt = nil
	if err := rs.store.UpdateTask(task); err != nil {
		return err
	}
	
	rs.scheduler.Cancel(taskID)
	
	rs.logger.Info("Reminder cancelled", zap.String("task_id", taskID))
	
	return nil
}

// GetUpcomingReminders gets upcoming reminders for a user
func (rs *ReminderService) GetUpcomingReminders(userID string, limit int) ([]Task, error) {
	now := time.Now()
	window := now.Add(24 * time.Hour) // Next 24 hours
	
	return rs.store.GetTasksWithReminders(userID, window)
}

// SnoozeReminder snoozes a reminder
func (rs *ReminderService) SnoozeReminder(taskID string, duration time.Duration) error {
	snoozeTime := time.Now().Add(duration)
	return rs.SetReminder(taskID, snoozeTime)
}
