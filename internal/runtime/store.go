package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Store provides persistence for runtime tasks
type Store struct {
	db *gorm.DB
}

// NewStore creates a new runtime store
func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// Migrate runs database migrations for runtime tables
func (s *Store) Migrate() error {
	return s.db.AutoMigrate(
		&Task{},
		&TaskStep{},
		&TaskEffect{},
		&TaskEvent{},
		&TaskCheckpoint{},
		&TaskApproval{},
	)
}

// Task Operations

// CreateTask creates a new task
func (s *Store) CreateTask(ctx context.Context, task *Task) error {
	return s.db.WithContext(ctx).Create(task).Error
}

// GetTask retrieves a task by ID with all relationships
func (s *Store) GetTask(ctx context.Context, id string) (*Task, error) {
	var task Task
	err := s.db.WithContext(ctx).
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("sequence_number ASC")
		}).
		Preload("Effects", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("Checkpoints", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		Preload("Approvals").
		First(&task, "id = ?", id).Error

	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetTaskByID retrieves a task by ID without relationships (lighter query)
func (s *Store) GetTaskByID(ctx context.Context, id string) (*Task, error) {
	var task Task
	err := s.db.WithContext(ctx).First(&task, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTasks lists tasks with optional state filter
func (s *Store) ListTasks(ctx context.Context, state *TaskState, limit int) ([]Task, error) {
	var tasks []Task

	query := s.db.WithContext(ctx).Order("created_at DESC")

	if state != nil {
		query = query.Where("state = ?", *state)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&tasks).Error
	return tasks, err
}

// ListRunningTasks returns all tasks that are in running or blocked state
func (s *Store) ListRunningTasks(ctx context.Context) ([]Task, error) {
	var tasks []Task
	err := s.db.WithContext(ctx).
		Where("state IN ?", []TaskState{TaskStateRunning, TaskStateBlocked}).
		Find(&tasks).Error
	return tasks, err
}

// UpdateTaskState updates the state of a task with validation
func (s *Store) UpdateTaskState(ctx context.Context, taskID string, newState TaskState, stopReason string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task Task
		if err := tx.First(&task, "id = ?", taskID).Error; err != nil {
			return err
		}

		if !task.CanTransition(newState) {
			return fmt.Errorf("invalid state transition from %s to %s", task.State, newState)
		}

		updates := map[string]interface{}{
			"state":       newState,
			"stop_reason": stopReason,
		}

		// Update timestamps based on state
		switch newState {
		case TaskStateRunning:
			if task.StartedAt == nil {
				updates["started_at"] = time.Now()
			}
		case TaskStateCompleted, TaskStateFailed, TaskStateCancelled:
			updates["completed_at"] = time.Now()
		case TaskStateBlocked:
			updates["blocked_at"] = time.Now()
		}

		if err := tx.Model(&task).Updates(updates).Error; err != nil {
			return err
		}

		// Log state change event
		event := &TaskEvent{
			TaskID:  taskID,
			Type:    EventTypeStateChange,
			Message: fmt.Sprintf("State changed from %s to %s", task.State, newState),
			Data: ToJSON(map[string]string{
				"from_state": string(task.State),
				"to_state":   string(newState),
				"reason":     stopReason,
			}),
		}

		return tx.Create(event).Error
	})
}

// UpdateTaskProgress updates task execution progress
func (s *Store) UpdateTaskProgress(ctx context.Context, taskID string, iteration int, plan string, lastResult string) error {
	return s.db.WithContext(ctx).
		Model(&Task{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"current_iteration": iteration,
			"current_plan":      plan,
			"last_tool_result":  lastResult,
		}).Error
}

// UpdateTaskResult updates the final result of a task
func (s *Store) UpdateTaskResult(ctx context.Context, taskID string, result string, resultData interface{}, err string) error {
	var resultJSON json.RawMessage
	if resultData != nil {
		data, _ := json.Marshal(resultData)
		resultJSON = data
	}

	return s.db.WithContext(ctx).
		Model(&Task{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"final_result": result,
			"result_data":  resultJSON,
			"error":        err,
		}).Error
}

// DeleteTask deletes a task and all its related data
func (s *Store) DeleteTask(ctx context.Context, taskID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete related records first
		if err := tx.Where("task_id = ?", taskID).Delete(&TaskStep{}).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", taskID).Delete(&TaskEffect{}).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", taskID).Delete(&TaskEvent{}).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", taskID).Delete(&TaskCheckpoint{}).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", taskID).Delete(&TaskApproval{}).Error; err != nil {
			return err
		}

		// Delete the task
		return tx.Delete(&Task{}, "id = ?", taskID).Error
	})
}

// Step Operations

// CreateStep creates a new task step
func (s *Store) CreateStep(ctx context.Context, step *TaskStep) error {
	return s.db.WithContext(ctx).Create(step).Error
}

// GetStep retrieves a step by ID
func (s *Store) GetStep(ctx context.Context, stepID string) (*TaskStep, error) {
	var step TaskStep
	err := s.db.WithContext(ctx).First(&step, "id = ?", stepID).Error
	if err != nil {
		return nil, err
	}
	return &step, nil
}

// Effect Operations

// CreateEffect creates a new task effect journal entry.
func (s *Store) CreateEffect(ctx context.Context, effect *TaskEffect) error {
	return s.db.WithContext(ctx).Create(effect).Error
}

// GetEffectByStep retrieves an effect journal entry for a step.
func (s *Store) GetEffectByStep(ctx context.Context, stepID string) (*TaskEffect, error) {
	var effect TaskEffect
	err := s.db.WithContext(ctx).Where("step_id = ?", stepID).First(&effect).Error
	if err != nil {
		return nil, err
	}
	return &effect, nil
}

// FindCompletedEffect finds a previously completed effect by fingerprint.
func (s *Store) FindCompletedEffect(ctx context.Context, taskID string, fingerprint string, excludeStepID string) (*TaskEffect, error) {
	var effect TaskEffect
	query := s.db.WithContext(ctx).
		Where("task_id = ? AND tool_fingerprint = ? AND status = ?", taskID, fingerprint, EffectStatusCompleted)

	if excludeStepID != "" {
		query = query.Where("step_id <> ?", excludeStepID)
	}

	err := query.Order("created_at DESC").First(&effect).Error
	if err != nil {
		return nil, err
	}
	return &effect, nil
}

// UpdateEffectResult finalizes an effect journal entry.
func (s *Store) UpdateEffectResult(ctx context.Context, effectID string, status EffectStatus, output json.RawMessage, errMsg string) error {
	updates := map[string]interface{}{
		"status":      status,
		"output":      output,
		"error":       errMsg,
		"finished_at": time.Now(),
	}
	return s.db.WithContext(ctx).Model(&TaskEffect{}).Where("id = ?", effectID).Updates(updates).Error
}

// MarkEffectReplayed marks an effect as replayed from another effect.
func (s *Store) MarkEffectReplayed(ctx context.Context, effectID string, source *TaskEffect) error {
	return s.db.WithContext(ctx).
		Model(&TaskEffect{}).
		Where("id = ?", effectID).
		Updates(map[string]interface{}{
			"status":              EffectStatusReplayed,
			"output":              source.Output,
			"error":               source.Error,
			"replay_of_effect_id": source.ID,
			"finished_at":         time.Now(),
		}).Error
}

// MarkTaskEffectsInterrupted marks in-flight effects as interrupted during recovery.
func (s *Store) MarkTaskEffectsInterrupted(ctx context.Context, taskID string) error {
	return s.db.WithContext(ctx).
		Model(&TaskEffect{}).
		Where("task_id = ? AND status = ?", taskID, EffectStatusStarted).
		Updates(map[string]interface{}{
			"status":      EffectStatusInterrupted,
			"finished_at": time.Now(),
		}).Error
}

// UpdateStepExecutionState updates a step execution state and optionally closes it.
func (s *Store) UpdateStepExecutionState(ctx context.Context, stepID string, state StepExecutionState, markEnded bool) error {
	updates := map[string]interface{}{
		"execution_state": state,
	}
	if markEnded {
		updates["ended_at"] = time.Now()
	}

	return s.db.WithContext(ctx).
		Model(&TaskStep{}).
		Where("id = ?", stepID).
		Updates(updates).Error
}

// MarkTaskStepsInterrupted marks unfinished active steps as interrupted after crash recovery.
func (s *Store) MarkTaskStepsInterrupted(ctx context.Context, taskID string) error {
	return s.db.WithContext(ctx).
		Model(&TaskStep{}).
		Where("task_id = ? AND execution_state IN ?", taskID, []StepExecutionState{StepExecutionPending, StepExecutionInProgress}).
		Updates(map[string]interface{}{
			"execution_state": StepExecutionInterrupted,
			"ended_at":        time.Now(),
		}).Error
}

// FindReplayableStep finds a previously completed tool step with the same fingerprint.
func (s *Store) FindReplayableStep(ctx context.Context, taskID string, fingerprint string, excludeStepID string) (*TaskStep, error) {
	var step TaskStep
	query := s.db.WithContext(ctx).
		Where("task_id = ? AND type = ? AND tool_fingerprint = ? AND tool_error = ''", taskID, StepTypeTool, fingerprint).
		Where("ended_at IS NOT NULL")

	if excludeStepID != "" {
		query = query.Where("id <> ?", excludeStepID)
	}

	err := query.Order("created_at DESC").First(&step).Error
	if err != nil {
		return nil, err
	}

	return &step, nil
}

// UpdateStepResult updates the result of a step
func (s *Store) UpdateStepResult(ctx context.Context, stepID string, output json.RawMessage, toolErr string, durationMs int) error {
	state := StepExecutionCompleted
	if toolErr != "" {
		state = StepExecutionFailed
	}

	return s.db.WithContext(ctx).
		Model(&TaskStep{}).
		Where("id = ?", stepID).
		Updates(map[string]interface{}{
			"tool_output":     output,
			"tool_error":      toolErr,
			"duration_ms":     durationMs,
			"ended_at":        time.Now(),
			"execution_state": state,
		}).Error
}

// MarkStepReplayed copies a previous step result onto a new step.
func (s *Store) MarkStepReplayed(ctx context.Context, stepID string, sourceStep *TaskStep) error {
	return s.db.WithContext(ctx).
		Model(&TaskStep{}).
		Where("id = ?", stepID).
		Updates(map[string]interface{}{
			"tool_output":       sourceStep.ToolOutput,
			"tool_error":        sourceStep.ToolError,
			"duration_ms":       0,
			"ended_at":          time.Now(),
			"execution_state":   StepExecutionReplayed,
			"replay_of_step_id": sourceStep.ID,
		}).Error
}

// ListSteps lists all steps for a task in order
func (s *Store) ListSteps(ctx context.Context, taskID string) ([]TaskStep, error) {
	var steps []TaskStep
	err := s.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("sequence_number ASC").
		Find(&steps).Error
	return steps, err
}

// GetNextSequenceNumber returns the next sequence number for a task
func (s *Store) GetNextSequenceNumber(ctx context.Context, taskID string) (int, error) {
	var maxSeq int
	err := s.db.WithContext(ctx).
		Model(&TaskStep{}).
		Where("task_id = ?", taskID).
		Select("COALESCE(MAX(sequence_number), 0)").
		Scan(&maxSeq).Error

	return maxSeq + 1, err
}

// Event Operations

// CreateEvent creates a new task event
func (s *Store) CreateEvent(ctx context.Context, event *TaskEvent) error {
	return s.db.WithContext(ctx).Create(event).Error
}

// ListEvents lists all events for a task
func (s *Store) ListEvents(ctx context.Context, taskID string) ([]TaskEvent, error) {
	var events []TaskEvent
	err := s.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at ASC").
		Find(&events).Error
	return events, err
}

// Checkpoint Operations

// CreateCheckpoint creates a new checkpoint
func (s *Store) CreateCheckpoint(ctx context.Context, checkpoint *TaskCheckpoint) error {
	return s.db.WithContext(ctx).Create(checkpoint).Error
}

// GetLatestCheckpoint retrieves the most recent checkpoint for a task
func (s *Store) GetLatestCheckpoint(ctx context.Context, taskID string) (*TaskCheckpoint, error) {
	var checkpoint TaskCheckpoint
	err := s.db.WithContext(ctx).
		Where("task_id = ? AND can_resume_from = ?", taskID, true).
		Order("created_at DESC").
		First(&checkpoint).Error

	if err != nil {
		return nil, err
	}
	return &checkpoint, nil
}

// ListCheckpoints lists all checkpoints for a task
func (s *Store) ListCheckpoints(ctx context.Context, taskID string) ([]TaskCheckpoint, error) {
	var checkpoints []TaskCheckpoint
	err := s.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at DESC").
		Find(&checkpoints).Error
	return checkpoints, err
}

// Approval Operations

// CreateApproval creates a new approval request
func (s *Store) CreateApproval(ctx context.Context, approval *TaskApproval) error {
	return s.db.WithContext(ctx).Create(approval).Error
}

// GetApproval retrieves an approval by ID
func (s *Store) GetApproval(ctx context.Context, approvalID string) (*TaskApproval, error) {
	var approval TaskApproval
	err := s.db.WithContext(ctx).First(&approval, "id = ?", approvalID).Error
	if err != nil {
		return nil, err
	}
	return &approval, nil
}

// GetPendingApprovalForStep retrieves a pending approval for a specific step
func (s *Store) GetPendingApprovalForStep(ctx context.Context, stepID string) (*TaskApproval, error) {
	var approval TaskApproval
	err := s.db.WithContext(ctx).
		Where("step_id = ? AND status = ?", stepID, ApprovalStatusPending).
		First(&approval).Error

	if err != nil {
		return nil, err
	}
	return &approval, nil
}

// ListPendingApprovals lists all pending approvals for a task
func (s *Store) ListPendingApprovals(ctx context.Context, taskID string) ([]TaskApproval, error) {
	var approvals []TaskApproval
	err := s.db.WithContext(ctx).
		Where("task_id = ? AND status = ?", taskID, ApprovalStatusPending).
		Find(&approvals).Error
	return approvals, err
}

// ResolveApproval updates the approval status
func (s *Store) ResolveApproval(ctx context.Context, approvalID string, status ApprovalStatus, approvedBy string, denialReason string) error {
	now := time.Now()
	return s.db.WithContext(ctx).
		Model(&TaskApproval{}).
		Where("id = ?", approvalID).
		Updates(map[string]interface{}{
			"status":        status,
			"approved_by":   approvedBy,
			"approved_at":   now,
			"denial_reason": denialReason,
		}).Error
}

// CleanupExpiredApprovals marks expired approvals
func (s *Store) CleanupExpiredApprovals(ctx context.Context) error {
	return s.db.WithContext(ctx).
		Model(&TaskApproval{}).
		Where("status = ? AND expires_at < ?", ApprovalStatusPending, time.Now()).
		Updates(map[string]interface{}{
			"status": ApprovalStatusExpired,
		}).Error
}

// Utility

// ToJSON converts any value to JSON bytes
func ToJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return json.RawMessage(b)
}

// FromJSON parses JSON bytes into a value
func FromJSON(data json.RawMessage, v interface{}) error {
	return json.Unmarshal(data, v)
}
