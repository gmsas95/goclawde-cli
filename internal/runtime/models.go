// Package runtime implements a durable, resumable agent runtime
// with state persistence, approval flow, and crash recovery
package runtime

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TaskState represents the lifecycle state of a task
type TaskState string

const (
	TaskStatePending   TaskState = "pending"
	TaskStateRunning   TaskState = "running"
	TaskStateBlocked   TaskState = "blocked"
	TaskStateFailed    TaskState = "failed"
	TaskStateCompleted TaskState = "completed"
	TaskStateCancelled TaskState = "cancelled"
)

// Task represents a durable agent task with full lifecycle management
type Task struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description" gorm:"type:text"`
	Goal        string    `json:"goal" gorm:"type:text"`
	State       TaskState `json:"state" gorm:"index:idx_task_state"`

	// Configuration
	MaxIterations   int             `json:"max_iterations"`
	TimeoutSeconds  int             `json:"timeout_seconds"`
	RequireApproval bool            `json:"require_approval"`
	Metadata        json.RawMessage `json:"metadata,omitempty" gorm:"type:text"`

	// Execution tracking
	CurrentIteration int    `json:"current_iteration"`
	CurrentPlan      string `json:"current_plan,omitempty" gorm:"type:text"`
	LastToolResult   string `json:"last_tool_result,omitempty" gorm:"type:text"`
	StopReason       string `json:"stop_reason,omitempty"`

	// Results
	FinalResult string          `json:"final_result,omitempty" gorm:"type:text"`
	Error       string          `json:"error,omitempty" gorm:"type:text"`
	ResultData  json.RawMessage `json:"result_data,omitempty" gorm:"type:text"`

	// Timestamps
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	BlockedAt   *time.Time `json:"blocked_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Relationships
	Steps       []TaskStep       `json:"steps,omitempty" gorm:"foreignKey:TaskID"`
	Events      []TaskEvent      `json:"events,omitempty" gorm:"foreignKey:TaskID"`
	Checkpoints []TaskCheckpoint `json:"checkpoints,omitempty" gorm:"foreignKey:TaskID"`
	Approvals   []TaskApproval   `json:"approvals,omitempty" gorm:"foreignKey:TaskID"`
	Effects     []TaskEffect     `json:"effects,omitempty" gorm:"foreignKey:TaskID"`
}

// TableName returns the table name for Task
func (Task) TableName() string {
	return "runtime_tasks"
}

// BeforeCreate hook for Task
func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = generateID("task")
	}
	if t.State == "" {
		t.State = TaskStatePending
	}
	if t.MaxIterations == 0 {
		t.MaxIterations = 10
	}
	if t.TimeoutSeconds == 0 {
		t.TimeoutSeconds = 300 // 5 minutes
	}
	return nil
}

// CanTransition checks if the task can transition to the given state
func (t *Task) CanTransition(newState TaskState) bool {
	validTransitions := map[TaskState][]TaskState{
		TaskStatePending:   {TaskStateRunning, TaskStateCancelled},
		TaskStateRunning:   {TaskStateBlocked, TaskStateFailed, TaskStateCompleted, TaskStateCancelled},
		TaskStateBlocked:   {TaskStateRunning, TaskStateFailed, TaskStateCancelled},
		TaskStateFailed:    {},
		TaskStateCompleted: {},
		TaskStateCancelled: {},
	}

	allowed, ok := validTransitions[t.State]
	if !ok {
		return false
	}

	for _, s := range allowed {
		if s == newState {
			return true
		}
	}
	return false
}

// StepType represents the type of a task step
type StepType string

const (
	StepTypeThink    StepType = "think"
	StepTypePlan     StepType = "plan"
	StepTypeTool     StepType = "tool"
	StepTypeReflect  StepType = "reflect"
	StepTypeRespond  StepType = "respond"
	StepTypeError    StepType = "error"
	StepTypeApproval StepType = "approval"
)

// StepExecutionState tracks execution status for a persisted step.
type StepExecutionState string

const (
	StepExecutionPending          StepExecutionState = "pending"
	StepExecutionInProgress       StepExecutionState = "in_progress"
	StepExecutionAwaitingApproval StepExecutionState = "awaiting_approval"
	StepExecutionCompleted        StepExecutionState = "completed"
	StepExecutionFailed           StepExecutionState = "failed"
	StepExecutionDenied           StepExecutionState = "denied"
	StepExecutionReplayed         StepExecutionState = "replayed"
	StepExecutionInterrupted      StepExecutionState = "interrupted"
)

// TaskStep represents a single step in task execution
type TaskStep struct {
	ID     string `gorm:"primaryKey" json:"id"`
	TaskID string `gorm:"index:idx_step_task" json:"task_id"`

	// Step info
	Iteration      int                `json:"iteration"`
	Type           StepType           `json:"type"`
	ExecutionState StepExecutionState `json:"execution_state" gorm:"index:idx_step_execution_state"`
	Content        string             `json:"content" gorm:"type:text"`

	// Tool execution (for tool steps)
	ToolName        string          `json:"tool_name,omitempty"`
	ToolInput       json.RawMessage `json:"tool_input,omitempty" gorm:"type:text"`
	ToolOutput      json.RawMessage `json:"tool_output,omitempty" gorm:"type:text"`
	ToolError       string          `json:"tool_error,omitempty" gorm:"type:text"`
	ToolRisk        string          `json:"tool_risk,omitempty"` // safe, write, exec, network
	ToolFingerprint string          `json:"tool_fingerprint,omitempty" gorm:"index:idx_step_tool_fingerprint"`
	ReplayOfStepID  string          `json:"replay_of_step_id,omitempty" gorm:"index"`

	// Timing
	StartedAt  *time.Time `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at"`
	DurationMs int        `json:"duration_ms"`

	// Ordering
	SequenceNumber int       `json:"sequence_number"`
	CreatedAt      time.Time `json:"created_at"`
}

// TableName returns the table name for TaskStep
func (TaskStep) TableName() string {
	return "runtime_task_steps"
}

// BeforeCreate hook for TaskStep
func (s *TaskStep) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = generateID("step")
	}
	if s.ExecutionState == "" {
		s.ExecutionState = StepExecutionPending
	}
	return nil
}

// EventType represents the type of a task event
type EventType string

const (
	EventTypeStateChange  EventType = "state_change"
	EventTypeStepCreated  EventType = "step_created"
	EventTypeToolExecuted EventType = "tool_executed"
	EventTypeApprovalReq  EventType = "approval_requested"
	EventTypeApprovalRes  EventType = "approval_resolved"
	EventTypeToolReplayed EventType = "tool_replayed"
	EventTypeError        EventType = "error"
	EventTypeCheckpoint   EventType = "checkpoint"
	EventTypeResume       EventType = "resume"
	EventTypeCancel       EventType = "cancel"
)

// TaskEvent represents an event in the task lifecycle
type TaskEvent struct {
	ID     string `gorm:"primaryKey" json:"id"`
	TaskID string `gorm:"index:idx_event_task" json:"task_id"`

	Type    EventType       `json:"type"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at"`
}

// TableName returns the table name for TaskEvent
func (TaskEvent) TableName() string {
	return "runtime_task_events"
}

// BeforeCreate hook for TaskEvent
func (e *TaskEvent) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = generateID("evt")
	}
	return nil
}

// TaskCheckpoint represents a recoverable checkpoint in task execution
type TaskCheckpoint struct {
	ID     string `gorm:"primaryKey" json:"id"`
	TaskID string `gorm:"index:idx_checkpoint_task" json:"task_id"`

	// Checkpoint data
	Iteration     int             `json:"iteration"`
	LoopState     json.RawMessage `json:"loop_state" gorm:"type:text"`
	ContextWindow json.RawMessage `json:"context_window,omitempty" gorm:"type:text"`

	// Recovery info
	CanResumeFrom bool   `json:"can_resume_from"`
	ResumeHint    string `json:"resume_hint,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// TableName returns the table name for TaskCheckpoint
func (TaskCheckpoint) TableName() string {
	return "runtime_task_checkpoints"
}

// BeforeCreate hook for TaskCheckpoint
func (c *TaskCheckpoint) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = generateID("chk")
	}
	return nil
}

// ApprovalStatus represents the status of an approval request
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusDenied   ApprovalStatus = "denied"
	ApprovalStatusExpired  ApprovalStatus = "expired"
)

// TaskApproval represents a pending approval request
type TaskApproval struct {
	ID     string `gorm:"primaryKey" json:"id"`
	TaskID string `gorm:"index:idx_approval_task" json:"task_id"`

	// What needs approval
	StepID      string          `json:"step_id"`
	ToolName    string          `json:"tool_name"`
	ToolInput   json.RawMessage `json:"tool_input" gorm:"type:text"`
	RiskLevel   string          `json:"risk_level"` // safe, write, exec, network, admin
	Description string          `json:"description" gorm:"type:text"`

	// Approval status
	Status       ApprovalStatus `json:"status"`
	ApprovedBy   string         `json:"approved_by,omitempty"`
	ApprovedAt   *time.Time     `json:"approved_at"`
	DenialReason string         `json:"denial_reason,omitempty"`

	// Timing
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// TableName returns the table name for TaskApproval
func (TaskApproval) TableName() string {
	return "runtime_task_approvals"
}

// BeforeCreate hook for TaskApproval
func (a *TaskApproval) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = generateID("aprv")
	}
	if a.Status == "" {
		a.Status = ApprovalStatusPending
	}
	return nil
}

// IsExpired checks if the approval request has expired
func (a *TaskApproval) IsExpired() bool {
	if a.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*a.ExpiresAt)
}

// EffectStatus tracks the lifecycle of a persisted side-effect attempt.
type EffectStatus string

const (
	EffectStatusStarted     EffectStatus = "started"
	EffectStatusCompleted   EffectStatus = "completed"
	EffectStatusFailed      EffectStatus = "failed"
	EffectStatusReplayed    EffectStatus = "replayed"
	EffectStatusInterrupted EffectStatus = "interrupted"
)

// TaskEffect journals tool-side effects for crash recovery and replay safety.
type TaskEffect struct {
	ID     string `gorm:"primaryKey" json:"id"`
	TaskID string `gorm:"index:idx_effect_task" json:"task_id"`
	StepID string `gorm:"index:idx_effect_step" json:"step_id"`

	ToolName         string          `json:"tool_name"`
	ToolFingerprint  string          `json:"tool_fingerprint" gorm:"index:idx_effect_fingerprint"`
	Status           EffectStatus    `json:"status" gorm:"index:idx_effect_status"`
	Input            json.RawMessage `json:"input,omitempty" gorm:"type:text"`
	Output           json.RawMessage `json:"output,omitempty" gorm:"type:text"`
	Error            string          `json:"error,omitempty" gorm:"type:text"`
	ReplayOfEffectID string          `json:"replay_of_effect_id,omitempty" gorm:"index"`

	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// TableName returns the table name for TaskEffect.
func (TaskEffect) TableName() string {
	return "runtime_task_effects"
}

// BeforeCreate hook for TaskEffect.
func (e *TaskEffect) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = generateID("eff")
	}
	if e.Status == "" {
		e.Status = EffectStatusStarted
	}
	if e.StartedAt.IsZero() {
		e.StartedAt = time.Now()
	}
	return nil
}

// generateID creates a unique ID with timestamp and random component
func generateID(prefix string) string {
	return prefix + "_" + time.Now().Format("20060102150405") + "_" + randomString(8) + "_" + randomString(4)
}

// randomString generates a cryptographically secure random string
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}

func toolCallFingerprint(name string, rawArgs string) string {
	normalized := rawArgs
	if rawArgs != "" {
		var payload interface{}
		if err := json.Unmarshal([]byte(rawArgs), &payload); err == nil {
			if b, err := json.Marshal(payload); err == nil {
				normalized = string(b)
			}
		}
	}

	sum := sha256.Sum256([]byte(name + "\n" + normalized))
	return fmt.Sprintf("%x", sum[:])
}
