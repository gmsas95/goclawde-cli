package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// EventLogger provides structured logging for task events
type EventLogger struct {
	store *Store
}

// NewEventLogger creates a new event logger
func NewEventLogger(store *Store) *EventLogger {
	return &EventLogger{store: store}
}

// LogStateChange logs a state transition
func (el *EventLogger) LogStateChange(ctx context.Context, taskID string, fromState, toState TaskState, reason string) error {
	event := &TaskEvent{
		TaskID:  taskID,
		Type:    EventTypeStateChange,
		Message: fmt.Sprintf("Task state changed from %s to %s", fromState, toState),
		Data: ToJSON(map[string]interface{}{
			"from_state": fromState,
			"to_state":   toState,
			"reason":     reason,
			"timestamp":  time.Now().Unix(),
		}),
	}
	return el.store.CreateEvent(ctx, event)
}

// LogStepCreated logs when a new step is created
func (el *EventLogger) LogStepCreated(ctx context.Context, taskID, stepID string, stepType StepType, iteration int) error {
	event := &TaskEvent{
		TaskID:  taskID,
		Type:    EventTypeStepCreated,
		Message: fmt.Sprintf("Created %s step (iteration %d)", stepType, iteration),
		Data: ToJSON(map[string]interface{}{
			"step_id":   stepID,
			"step_type": stepType,
			"iteration": iteration,
		}),
	}
	return el.store.CreateEvent(ctx, event)
}

// LogToolExecuted logs a tool execution
func (el *EventLogger) LogToolExecuted(ctx context.Context, taskID, stepID, toolName string, success bool, durationMs int) error {
	status := "success"
	if !success {
		status = "failure"
	}

	event := &TaskEvent{
		TaskID:  taskID,
		Type:    EventTypeToolExecuted,
		Message: fmt.Sprintf("Tool %s executed: %s (%dms)", toolName, status, durationMs),
		Data: ToJSON(map[string]interface{}{
			"step_id":     stepID,
			"tool_name":   toolName,
			"success":     success,
			"duration_ms": durationMs,
		}),
	}
	return el.store.CreateEvent(ctx, event)
}

// LogToolReplayed logs when a persisted tool result is replayed instead of re-executed.
func (el *EventLogger) LogToolReplayed(ctx context.Context, taskID, stepID, sourceStepID, toolName string) error {
	event := &TaskEvent{
		TaskID:  taskID,
		Type:    EventTypeToolReplayed,
		Message: fmt.Sprintf("Tool %s replayed from persisted result", toolName),
		Data: ToJSON(map[string]interface{}{
			"step_id":        stepID,
			"source_step_id": sourceStepID,
			"tool_name":      toolName,
			"replayed_at":    time.Now().Unix(),
		}),
	}
	return el.store.CreateEvent(ctx, event)
}

// LogApprovalRequested logs when approval is requested
func (el *EventLogger) LogApprovalRequested(ctx context.Context, taskID, approvalID, stepID, toolName, riskLevel string) error {
	event := &TaskEvent{
		TaskID:  taskID,
		Type:    EventTypeApprovalReq,
		Message: fmt.Sprintf("Approval requested for %s tool (risk: %s)", toolName, riskLevel),
		Data: ToJSON(map[string]interface{}{
			"approval_id": approvalID,
			"step_id":     stepID,
			"tool_name":   toolName,
			"risk_level":  riskLevel,
		}),
	}
	return el.store.CreateEvent(ctx, event)
}

// LogApprovalResolved logs when approval is resolved
func (el *EventLogger) LogApprovalResolved(ctx context.Context, taskID, approvalID string, status ApprovalStatus, approvedBy string) error {
	event := &TaskEvent{
		TaskID:  taskID,
		Type:    EventTypeApprovalRes,
		Message: fmt.Sprintf("Approval %s by %s", status, approvedBy),
		Data: ToJSON(map[string]interface{}{
			"approval_id": approvalID,
			"status":      status,
			"approved_by": approvedBy,
		}),
	}
	return el.store.CreateEvent(ctx, event)
}

// LogError logs an error event
func (el *EventLogger) LogError(ctx context.Context, taskID string, err error, context string) error {
	event := &TaskEvent{
		TaskID:  taskID,
		Type:    EventTypeError,
		Message: fmt.Sprintf("Error in %s: %v", context, err),
		Data: ToJSON(map[string]interface{}{
			"error":   err.Error(),
			"context": context,
			"time":    time.Now().Unix(),
		}),
	}
	return el.store.CreateEvent(ctx, event)
}

// LogCheckpoint logs a checkpoint creation
func (el *EventLogger) LogCheckpoint(ctx context.Context, taskID, checkpointID string, iteration int, canResume bool) error {
	event := &TaskEvent{
		TaskID:  taskID,
		Type:    EventTypeCheckpoint,
		Message: fmt.Sprintf("Checkpoint created at iteration %d (resumable: %v)", iteration, canResume),
		Data: ToJSON(map[string]interface{}{
			"checkpoint_id": checkpointID,
			"iteration":     iteration,
			"can_resume":    canResume,
		}),
	}
	return el.store.CreateEvent(ctx, event)
}

// LogResume logs a task resume
func (el *EventLogger) LogResume(ctx context.Context, taskID string, fromIteration int, reason string) error {
	event := &TaskEvent{
		TaskID:  taskID,
		Type:    EventTypeResume,
		Message: fmt.Sprintf("Task resumed from iteration %d", fromIteration),
		Data: ToJSON(map[string]interface{}{
			"from_iteration": fromIteration,
			"reason":         reason,
			"resumed_at":     time.Now().Unix(),
		}),
	}
	return el.store.CreateEvent(ctx, event)
}

// LogCancel logs a task cancellation
func (el *EventLogger) LogCancel(ctx context.Context, taskID string, reason string) error {
	event := &TaskEvent{
		TaskID:  taskID,
		Type:    EventTypeCancel,
		Message: fmt.Sprintf("Task cancelled: %s", reason),
		Data: ToJSON(map[string]interface{}{
			"reason":       reason,
			"cancelled_at": time.Now().Unix(),
		}),
	}
	return el.store.CreateEvent(ctx, event)
}

// EventFilter provides filtering options for events
type EventFilter struct {
	Types  []EventType
	After  *time.Time
	Before *time.Time
	Limit  int
	Offset int
}

// GetTaskTimeline returns a timeline of task events with optional filtering
func (el *EventLogger) GetTaskTimeline(ctx context.Context, taskID string, filter *EventFilter) ([]TaskEvent, error) {
	// For now, return all events. In the future, implement filtering in the query
	events, err := el.store.ListEvents(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Apply filters in memory for now
	var filtered []TaskEvent
	for _, event := range events {
		// Filter by type
		if len(filter.Types) > 0 {
			found := false
			for _, t := range filter.Types {
				if event.Type == t {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by time range
		if filter.After != nil && event.CreatedAt.Before(*filter.After) {
			continue
		}
		if filter.Before != nil && event.CreatedAt.After(*filter.Before) {
			continue
		}

		filtered = append(filtered, event)
	}

	// Apply limit and offset
	if filter.Offset >= len(filtered) {
		return []TaskEvent{}, nil
	}

	end := filter.Offset + filter.Limit
	if filter.Limit == 0 || end > len(filtered) {
		end = len(filtered)
	}

	return filtered[filter.Offset:end], nil
}

// GetEventData unmarshals event data into a value
func GetEventData(event TaskEvent, v interface{}) error {
	if len(event.Data) == 0 {
		return nil
	}
	return json.Unmarshal(event.Data, v)
}
