package runtime

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Recovery handles crash recovery and task resumption on startup
type Recovery struct {
	store  *Store
	runner *Runner
	logger *zap.Logger
}

// NewRecovery creates a new recovery manager
func NewRecovery(store *Store, runner *Runner, logger *zap.Logger) *Recovery {
	return &Recovery{
		store:  store,
		runner: runner,
		logger: logger,
	}
}

// RecoverOnStartup should be called during application startup to resume interrupted tasks
func (r *Recovery) RecoverOnStartup(ctx context.Context) error {
	r.logger.Info("Checking for tasks to recover after startup")

	// Find all tasks that were running or blocked when the process stopped
	tasks, err := r.store.ListRunningTasks(ctx)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		r.logger.Info("No tasks to recover")
		return nil
	}

	r.logger.Info("Found tasks to recover",
		zap.Int("count", len(tasks)))

	for _, task := range tasks {
		r.logger.Info("Recovering task",
			zap.String("task_id", task.ID),
			zap.String("state", string(task.State)),
			zap.Int("iteration", task.CurrentIteration))

		// Blocked tasks are waiting on user input and should remain blocked after
		// restart.
		if task.State != TaskStateRunning {
			continue
		}

		if err := r.store.MarkTaskStepsInterrupted(ctx, task.ID); err != nil {
			r.logger.Warn("Failed to mark interrupted steps",
				zap.String("task_id", task.ID),
				zap.Error(err))
		}
		if err := r.store.MarkTaskEffectsInterrupted(ctx, task.ID); err != nil {
			r.logger.Warn("Failed to mark interrupted effects",
				zap.String("task_id", task.ID),
				zap.Error(err))
		}

		// Log recovery event
		if err := r.runner.eventLogger.LogResume(ctx, task.ID, task.CurrentIteration, "crash_recovery"); err != nil {
			r.logger.Warn("Failed to log recovery event", zap.Error(err))
		}

		// Resume the task
		_, err := r.runner.ResumeTask(ctx, task.ID)
		if err != nil {
			r.logger.Error("Failed to recover task",
				zap.String("task_id", task.ID),
				zap.Error(err))

			// Mark as failed if we can't resume
			if err := r.store.UpdateTaskState(ctx, task.ID, TaskStateFailed, "recovery_failed"); err != nil {
				r.logger.Error("Failed to mark task as failed",
					zap.String("task_id", task.ID),
					zap.Error(err))
			}
		}
	}

	// Clean up expired approvals
	if err := r.store.CleanupExpiredApprovals(ctx); err != nil {
		r.logger.Warn("Failed to cleanup expired approvals", zap.Error(err))
	}

	return nil
}

// RecoveryStatus represents the status of recovery operations
type RecoveryStatus struct {
	TasksFound   int       `json:"tasks_found"`
	TasksResumed int       `json:"tasks_resumed"`
	TasksFailed  int       `json:"tasks_failed"`
	RecoveryTime time.Time `json:"recovery_time"`
	Errors       []string  `json:"errors,omitempty"`
}

// GetRecoveryStatus returns the status of the last recovery operation
func (r *Recovery) GetRecoveryStatus(ctx context.Context) (*RecoveryStatus, error) {
	tasks, err := r.store.ListRunningTasks(ctx)
	if err != nil {
		return nil, err
	}

	return &RecoveryStatus{
		TasksFound:   len(tasks),
		RecoveryTime: time.Now(),
	}, nil
}

// MonitorRunningTasks periodically checks running tasks for timeout/stuck detection
func (r *Recovery) MonitorRunningTasks(ctx context.Context, checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.checkStuckTasks(ctx)
		}
	}
}

// checkStuckTasks checks for tasks that appear stuck and logs warnings
func (r *Recovery) checkStuckTasks(ctx context.Context) {
	tasks, err := r.store.ListRunningTasks(ctx)
	if err != nil {
		r.logger.Error("Failed to list running tasks during monitoring", zap.Error(err))
		return
	}

	now := time.Now()
	for _, task := range tasks {
		// Check if task has been running too long
		if task.StartedAt != nil {
			runningTime := now.Sub(*task.StartedAt)
			timeout := time.Duration(task.TimeoutSeconds) * time.Second

			if runningTime > timeout {
				r.logger.Warn("Task appears to be stuck/timed out",
					zap.String("task_id", task.ID),
					zap.Duration("running_time", runningTime),
					zap.Duration("timeout", timeout))

				// Optionally auto-cancel stuck tasks
				// r.runner.CancelTask(ctx, task.ID, "timeout: task exceeded maximum running time")
			}
		}

		// Check blocked tasks
		if task.State == TaskStateBlocked && task.BlockedAt != nil {
			blockedTime := now.Sub(*task.BlockedAt)
			if blockedTime > 24*time.Hour {
				r.logger.Warn("Task has been blocked for over 24 hours",
					zap.String("task_id", task.ID),
					zap.Duration("blocked_time", blockedTime))
			}
		}
	}
}

// CleanupOldTasks removes old completed/failed tasks based on retention policy
func (r *Recovery) CleanupOldTasks(ctx context.Context, retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	r.logger.Info("Cleaning up old tasks",
		zap.Time("cutoff", cutoff),
		zap.Int("retention_days", retentionDays))

	// This would require a new method in Store to delete old tasks
	// For now, just log that this would happen
	r.logger.Info("Task cleanup not yet implemented - would delete tasks older than cutoff")

	return nil
}
