package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/agent"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/pkg/tools"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Runner executes tasks with durability, checkpointing, and resume capability
type Runner struct {
	store          *Store
	logger         *zap.Logger
	toolRegistry   *tools.Registry
	skillsRegistry interface { // Minimal interface for skills execution
		ExecuteTool(ctx context.Context, name string, args json.RawMessage) (interface{}, error)
	}
	llmClient     *llm.Client
	eventLogger   *EventLogger
	workspaceRoot string
	allowedCmds   []string
	policyEngine  *PolicyEngine
}

// RunnerConfig configures the task runner
type RunnerConfig struct {
	Logger         *zap.Logger
	Store          *Store
	ToolRegistry   *tools.Registry
	SkillsRegistry interface {
		ExecuteTool(ctx context.Context, name string, args json.RawMessage) (interface{}, error)
	}
	LLMClient     *llm.Client
	WorkspaceRoot string
	AllowedCmds   []string
}

// NewRunner creates a new task runner
func NewRunner(cfg RunnerConfig) *Runner {
	return &Runner{
		store:          cfg.Store,
		logger:         cfg.Logger,
		toolRegistry:   cfg.ToolRegistry,
		skillsRegistry: cfg.SkillsRegistry,
		llmClient:      cfg.LLMClient,
		eventLogger:    NewEventLogger(cfg.Store),
		workspaceRoot:  cfg.WorkspaceRoot,
		allowedCmds:    cfg.AllowedCmds,
		policyEngine:   NewPolicyEngine(cfg.WorkspaceRoot, cfg.AllowedCmds),
	}
}

// LoopState represents the current state of the agent loop
type LoopState struct {
	Iteration      int            `json:"iteration"`
	CurrentPlan    string         `json:"current_plan"`
	LastToolResult interface{}    `json:"last_tool_result"`
	ActionHistory  []agent.Action `json:"action_history"`
	SystemPrompt   string         `json:"system_prompt"`
	ContextWindow  []llm.Message  `json:"context_window"`
}

// StartTask starts a new task execution
func (r *Runner) StartTask(ctx context.Context, goal, description string, maxIterations int, requireApproval bool) (*Task, error) {
	// Create the task
	task := &Task{
		Title:           truncate(goal, 100),
		Description:     description,
		Goal:            goal,
		State:           TaskStatePending,
		MaxIterations:   maxIterations,
		RequireApproval: requireApproval,
	}

	if maxIterations <= 0 {
		task.MaxIterations = 10
	}

	// Save to database
	if err := r.store.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	r.logger.Info("Task created",
		zap.String("task_id", task.ID),
		zap.String("goal", goal))

	// Transition to running
	if err := r.store.UpdateTaskState(ctx, task.ID, TaskStateRunning, ""); err != nil {
		return nil, fmt.Errorf("failed to start task: %w", err)
	}

	// Start execution in background
	go r.executeTask(context.Background(), task.ID)

	return task, nil
}

// ResumeTask resumes a task from its last checkpoint
func (r *Runner) ResumeTask(ctx context.Context, taskID string) (*Task, error) {
	task, err := r.store.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Can only resume blocked or running tasks
	if task.State != TaskStateBlocked && task.State != TaskStateRunning {
		return nil, fmt.Errorf("cannot resume task in state %s", task.State)
	}

	// Log resume event
	if err := r.eventLogger.LogResume(ctx, taskID, task.CurrentIteration, "user_requested"); err != nil {
		r.logger.Warn("Failed to log resume event", zap.Error(err))
	}

	// Transition blocked tasks back to running. Tasks already marked running may
	// have been interrupted by a process restart and should continue as-is.
	if task.State == TaskStateBlocked {
		if err := r.store.UpdateTaskState(ctx, taskID, TaskStateRunning, ""); err != nil {
			return nil, fmt.Errorf("failed to resume task: %w", err)
		}
	}

	// Resume execution in background
	go r.executeTask(context.Background(), taskID)

	r.logger.Info("Task resumed",
		zap.String("task_id", taskID),
		zap.Int("iteration", task.CurrentIteration))

	return task, nil
}

// CancelTask cancels a running or pending task
func (r *Runner) CancelTask(ctx context.Context, taskID string, reason string) error {
	task, err := r.store.GetTaskByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if !task.CanTransition(TaskStateCancelled) {
		return fmt.Errorf("cannot cancel task in state %s", task.State)
	}

	// Log cancel event
	if err := r.eventLogger.LogCancel(ctx, taskID, reason); err != nil {
		r.logger.Warn("Failed to log cancel event", zap.Error(err))
	}

	return r.store.UpdateTaskState(ctx, taskID, TaskStateCancelled, reason)
}

// ApproveAction approves a pending action
func (r *Runner) ApproveAction(ctx context.Context, approvalID string, approvedBy string) error {
	approval, err := r.store.GetApproval(ctx, approvalID)
	if err != nil {
		return fmt.Errorf("approval not found: %w", err)
	}

	if approval.Status != ApprovalStatusPending {
		return fmt.Errorf("approval already resolved: %s", approval.Status)
	}

	// Resolve the approval
	if err := r.store.ResolveApproval(ctx, approvalID, ApprovalStatusApproved, approvedBy, ""); err != nil {
		return fmt.Errorf("failed to approve: %w", err)
	}

	// Log the approval
	if err := r.eventLogger.LogApprovalResolved(ctx, approval.TaskID, approvalID, ApprovalStatusApproved, approvedBy); err != nil {
		r.logger.Warn("Failed to log approval resolution", zap.Error(err))
	}

	if err := r.executeApprovedStep(ctx, approval); err != nil {
		return fmt.Errorf("failed to execute approved action: %w", err)
	}

	// Resume the task
	_, err = r.ResumeTask(ctx, approval.TaskID)
	return err
}

// DenyAction denies a pending action
func (r *Runner) DenyAction(ctx context.Context, approvalID string, deniedBy string, reason string) error {
	approval, err := r.store.GetApproval(ctx, approvalID)
	if err != nil {
		return fmt.Errorf("approval not found: %w", err)
	}

	if approval.Status != ApprovalStatusPending {
		return fmt.Errorf("approval already resolved: %s", approval.Status)
	}

	// Resolve the approval
	if err := r.store.ResolveApproval(ctx, approvalID, ApprovalStatusDenied, deniedBy, reason); err != nil {
		return fmt.Errorf("failed to deny: %w", err)
	}

	// Log the denial
	if err := r.eventLogger.LogApprovalResolved(ctx, approval.TaskID, approvalID, ApprovalStatusDenied, deniedBy); err != nil {
		r.logger.Warn("Failed to log denial", zap.Error(err))
	}

	// Get the step and mark it as failed
	step, err := r.store.GetStep(ctx, approval.StepID)
	if err != nil {
		r.logger.Warn("Failed to get step for denial", zap.Error(err))
	} else {
		// Update step with denial info
		denialResult := map[string]string{
			"status":    "denied",
			"reason":    reason,
			"denied_by": deniedBy,
		}
		resultJSON := ToJSON(denialResult)
		r.store.UpdateStepResult(ctx, step.ID, resultJSON, "Action denied by user", 0)
		r.store.UpdateStepExecutionState(ctx, step.ID, StepExecutionDenied, true)
		r.store.UpdateTaskProgress(ctx, approval.TaskID, step.Iteration, "", string(resultJSON))
	}

	// Resume the task - it will continue with the denial information
	_, err = r.ResumeTask(ctx, approval.TaskID)
	return err
}

// executeTask is the main execution loop (runs in background)
func (r *Runner) executeTask(ctx context.Context, taskID string) {
	logger := r.logger.With(zap.String("task_id", taskID))
	logger.Info("Starting task execution")

	// Load task
	task, err := r.store.GetTask(ctx, taskID)
	if err != nil {
		logger.Error("Failed to load task", zap.Error(err))
		return
	}

	// Create timeout context
	timeout := time.Duration(task.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Restore loop state from checkpoint if resuming
	loopState := &LoopState{
		Iteration: task.CurrentIteration,
	}

	if task.CurrentIteration > 0 {
		// Try to restore from checkpoint
		checkpoint, err := r.store.GetLatestCheckpoint(ctx, taskID)
		if err == nil && checkpoint != nil {
			if err := FromJSON(checkpoint.LoopState, loopState); err != nil {
				logger.Warn("Failed to restore checkpoint", zap.Error(err))
			} else {
				logger.Info("Restored from checkpoint",
					zap.Int("iteration", checkpoint.Iteration))
			}
		}
	}

	if loopState.LastToolResult == nil && task.LastToolResult != "" {
		loopState.LastToolResult = task.LastToolResult
	}

	// Build system prompt
	loopState.SystemPrompt = r.buildSystemPrompt(task)

	// Execute iterations
	for loopState.Iteration < task.MaxIterations {
		select {
		case <-ctx.Done():
			r.handleTaskError(ctx, task, "Task timed out")
			return
		default:
		}

		// Check if task was cancelled
		task, err = r.store.GetTaskByID(ctx, taskID)
		if err != nil {
			r.handleTaskError(ctx, task, fmt.Sprintf("Failed to check task state: %v", err))
			return
		}

		if task.State == TaskStateCancelled {
			logger.Info("Task was cancelled")
			return
		}

		loopState.Iteration++
		logger.Info("Starting iteration", zap.Int("iteration", loopState.Iteration))

		// Create checkpoint before iteration
		if err := r.createCheckpoint(ctx, taskID, loopState); err != nil {
			logger.Warn("Failed to create checkpoint", zap.Error(err))
		}

		// Get next action from LLM
		action, err := r.getNextAction(ctx, task, loopState)
		if err != nil {
			logger.Error("Failed to get next action", zap.Error(err))
			r.handleTaskError(ctx, task, fmt.Sprintf("LLM error: %v", err))
			return
		}

		// Persist step
		step, err := r.persistStep(ctx, taskID, action, loopState.Iteration)
		if err != nil {
			logger.Error("Failed to persist step", zap.Error(err))
			r.handleTaskError(ctx, task, fmt.Sprintf("Persistence error: %v", err))
			return
		}

		// Update task progress
		r.store.UpdateTaskProgress(ctx, taskID, loopState.Iteration, loopState.CurrentPlan, "")

		// Execute action
		result, shouldContinue, err := r.executeAction(ctx, task, step, action, loopState)
		if err != nil {
			logger.Error("Action execution failed", zap.Error(err))
			r.handleTaskError(ctx, task, fmt.Sprintf("Execution error: %v", err))
			return
		}

		// Update step result
		if result != nil {
			resultJSON := ToJSON(result)
			var errStr string
			if resMap, ok := result.(map[string]interface{}); ok {
				if errVal, ok := resMap["error"]; ok {
					errStr = fmt.Sprintf("%v", errVal)
				}
			}
			if step.ExecutionState != StepExecutionReplayed {
				r.store.UpdateStepResult(ctx, step.ID, resultJSON, errStr, 0)
			}
			loopState.LastToolResult = result
			r.store.UpdateTaskProgress(ctx, taskID, loopState.Iteration, loopState.CurrentPlan, string(resultJSON))
		}

		// Record the executed action so completion/inspection can see the latest step.
		loopState.ActionHistory = append(loopState.ActionHistory, *action)

		if err := r.createCheckpoint(ctx, taskID, loopState); err != nil {
			logger.Warn("Failed to create post-action checkpoint", zap.Error(err))
		}

		// Check if we should stop
		if !shouldContinue {
			if status, ok := getResultStatus(result); ok && status == "blocked" {
				logger.Info("Task blocked awaiting approval")
				return
			}

			// Task completed
			finalResult := ""
			if status, ok := getResultStatus(result); ok && status == "complete" {
				finalResult = getResultResponse(result)
			}

			r.handleTaskCompletion(ctx, task, loopState, finalResult)
			return
		}
	}

	// Max iterations reached
	r.handleTaskError(ctx, task, fmt.Sprintf("Maximum iterations (%d) reached", task.MaxIterations))
}

// getNextAction gets the next action from the LLM
func (r *Runner) getNextAction(ctx context.Context, task *Task, loopState *LoopState) (*agent.Action, error) {
	// Build prompt
	prompt := r.buildActionPrompt(task, loopState)

	// Call LLM
	resp, err := r.llmClient.SimpleChat(ctx, loopState.SystemPrompt, prompt)
	if err != nil {
		return nil, err
	}

	// Parse action
	return r.parseActionResponse(resp)
}

// buildSystemPrompt creates the system prompt for the agent
func (r *Runner) buildSystemPrompt(task *Task) string {
	var sb strings.Builder
	sb.WriteString(`You are an autonomous AI agent. You can think, plan, use tools, reflect, and respond when complete.

Your goal: ` + task.Goal + `

You operate in a loop with these action types:
1. "think" - Think step by step about what to do next
2. "plan" - Create or update your plan
3. "tool" - Use a tool to interact with the environment
4. "reflect" - Reflect on progress and adjust strategy
5. "respond" - Provide final answer when task is complete

Guidelines:
- ALWAYS think before using tools
- Create a plan and update it as you learn
- Use reflection every 2-3 iterations to assess progress
- If stuck, try a different approach
- For file operations, prefer reading before writing
- Always verify your changes worked
- When complete, use "respond" with a summary

Respond with JSON in this format:
{
  "type": "think|plan|tool|reflect|respond",
  "content": "your thinking/plan/reflection/response",
  "tool_call": { // only for "tool" type
    "id": "call_1",
    "type": "function",
    "function": {
      "name": "tool_name",
      "arguments": "{\"arg\": \"value\"}"
    }
  }
}`)

	return sb.String()
}

// buildActionPrompt creates the prompt for the next action
func (r *Runner) buildActionPrompt(task *Task, loopState *LoopState) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Current task: %s\n\n", task.Goal))

	if loopState.CurrentPlan != "" {
		sb.WriteString(fmt.Sprintf("Current plan: %s\n\n", loopState.CurrentPlan))
	}

	if loopState.LastToolResult != nil {
		resultStr := fmt.Sprintf("%v", loopState.LastToolResult)
		if len(resultStr) > 500 {
			resultStr = resultStr[:500] + "..."
		}
		sb.WriteString(fmt.Sprintf("Last tool result: %s\n\n", resultStr))
	}

	if len(loopState.ActionHistory) > 0 {
		sb.WriteString(fmt.Sprintf("Previous actions (%d total):\n", len(loopState.ActionHistory)))
		start := 0
		if len(loopState.ActionHistory) > 5 {
			start = len(loopState.ActionHistory) - 5
		}
		for i := start; i < len(loopState.ActionHistory); i++ {
			action := loopState.ActionHistory[i]
			sb.WriteString(fmt.Sprintf("[%d] %s: %s\n",
				action.Iteration, action.Type, action.Content))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Iteration %d of %d\n\n", loopState.Iteration+1, task.MaxIterations))
	sb.WriteString("What is your next action? Respond with JSON.")

	return sb.String()
}

// parseActionResponse parses the LLM response
func (r *Runner) parseActionResponse(response string) (*agent.Action, error) {
	response = strings.TrimSpace(response)

	// Handle markdown code blocks
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	var rawAction struct {
		Type     string        `json:"type"`
		Content  string        `json:"content"`
		ToolCall *llm.ToolCall `json:"tool_call,omitempty"`
	}

	if err := json.Unmarshal([]byte(response), &rawAction); err != nil {
		// If parsing fails, treat as a think action
		return &agent.Action{
			Type:    "think",
			Content: response,
		}, nil
	}

	return &agent.Action{
		Type:     rawAction.Type,
		Content:  rawAction.Content,
		ToolCall: rawAction.ToolCall,
	}, nil
}

// persistStep persists a step to the database
func (r *Runner) persistStep(ctx context.Context, taskID string, action *agent.Action, iteration int) (*TaskStep, error) {
	seqNum, err := r.store.GetNextSequenceNumber(ctx, taskID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	step := &TaskStep{
		TaskID:         taskID,
		Iteration:      iteration,
		Type:           StepType(action.Type),
		ExecutionState: StepExecutionPending,
		Content:        action.Content,
		SequenceNumber: seqNum,
		StartedAt:      &now,
	}

	if action.ToolCall != nil {
		step.ToolName = action.ToolCall.Function.Name
		step.ToolInput = json.RawMessage(action.ToolCall.Function.Arguments)
		step.ToolRisk = r.assessToolRisk(action.ToolCall)
		step.ToolFingerprint = toolCallFingerprint(action.ToolCall.Function.Name, action.ToolCall.Function.Arguments)
	}

	if err := r.store.CreateStep(ctx, step); err != nil {
		return nil, err
	}

	// Log event
	if err := r.eventLogger.LogStepCreated(ctx, taskID, step.ID, step.Type, iteration); err != nil {
		r.logger.Warn("Failed to log step creation", zap.Error(err))
	}

	return step, nil
}

// executeAction executes a single action
func (r *Runner) executeAction(ctx context.Context, task *Task, step *TaskStep, action *agent.Action, loopState *LoopState) (interface{}, bool, error) {
	switch action.Type {
	case "think":
		// Just log it
		r.logger.Debug("Agent thinking", zap.String("thought", action.Content))
		_ = r.store.UpdateStepExecutionState(ctx, step.ID, StepExecutionCompleted, true)
		return nil, true, nil

	case "plan":
		loopState.CurrentPlan = action.Content
		_ = r.store.UpdateStepExecutionState(ctx, step.ID, StepExecutionCompleted, false)
		return map[string]string{"status": "plan_updated"}, true, nil

	case "tool":
		if action.ToolCall == nil {
			_ = r.store.UpdateStepExecutionState(ctx, step.ID, StepExecutionFailed, true)
			return map[string]string{"error": "tool action without tool call"}, true, nil
		}

		if err := r.store.UpdateStepExecutionState(ctx, step.ID, StepExecutionInProgress, false); err != nil {
			return nil, false, err
		}

		effect, err := r.beginToolEffect(ctx, task.ID, step)
		if err != nil {
			return nil, false, err
		}

		if cachedResult, reused, err := r.tryReuseToolResult(ctx, task.ID, step, effect, action.ToolCall); err != nil {
			return nil, false, err
		} else if reused {
			return cachedResult, true, nil
		}

		// Check if approval is needed
		if task.RequireApproval && step.ToolRisk != "safe" {
			// Create approval request
			approval := &TaskApproval{
				TaskID:      task.ID,
				StepID:      step.ID,
				ToolName:    action.ToolCall.Function.Name,
				ToolInput:   step.ToolInput,
				RiskLevel:   step.ToolRisk,
				Description: fmt.Sprintf("Execute %s tool", action.ToolCall.Function.Name),
				ExpiresAt:   &[]time.Time{time.Now().Add(24 * time.Hour)}[0],
			}

			if err := r.store.CreateApproval(ctx, approval); err != nil {
				return nil, false, fmt.Errorf("failed to create approval: %w", err)
			}

			// Log approval request
			if err := r.eventLogger.LogApprovalRequested(ctx, task.ID, approval.ID, step.ID, action.ToolCall.Function.Name, step.ToolRisk); err != nil {
				r.logger.Warn("Failed to log approval request", zap.Error(err))
			}

			if err := r.store.UpdateStepExecutionState(ctx, step.ID, StepExecutionAwaitingApproval, false); err != nil {
				return nil, false, err
			}

			// Block the task
			if err := r.store.UpdateTaskState(ctx, task.ID, TaskStateBlocked, "waiting_for_approval"); err != nil {
				return nil, false, err
			}

			return map[string]string{
				"status":      "blocked",
				"approval_id": approval.ID,
				"reason":      "waiting_for_approval",
			}, false, nil // Stop execution, will resume after approval
		}

		// Execute the tool
		start := time.Now()
		result, err := r.executeTool(ctx, action.ToolCall)
		duration := int(time.Since(start).Milliseconds())

		// Log tool execution
		success := err == nil
		if logErr := r.eventLogger.LogToolExecuted(ctx, task.ID, step.ID, action.ToolCall.Function.Name, success, duration); logErr != nil {
			r.logger.Warn("Failed to log tool execution", zap.Error(logErr))
		}

		if err != nil {
			_ = r.store.UpdateEffectResult(ctx, effect.ID, EffectStatusFailed, ToJSON(map[string]string{"error": err.Error()}), err.Error())
			return map[string]string{"error": err.Error()}, true, nil
		}

		if err := r.store.UpdateEffectResult(ctx, effect.ID, EffectStatusCompleted, ToJSON(result), ""); err != nil {
			return nil, false, err
		}

		return result, true, nil

	case "reflect":
		// Just log it
		r.logger.Debug("Agent reflecting", zap.String("reflection", action.Content))
		_ = r.store.UpdateStepExecutionState(ctx, step.ID, StepExecutionCompleted, true)
		return nil, true, nil

	case "respond":
		// Task complete
		_ = r.store.UpdateStepExecutionState(ctx, step.ID, StepExecutionCompleted, false)
		return map[string]string{
			"status":   "complete",
			"response": action.Content,
		}, false, nil

	default:
		_ = r.store.UpdateStepExecutionState(ctx, step.ID, StepExecutionFailed, true)
		return map[string]string{"error": fmt.Sprintf("unknown action type: %s", action.Type)}, true, nil
	}
}

// executeTool executes a tool call
func (r *Runner) executeTool(ctx context.Context, toolCall *llm.ToolCall) (interface{}, error) {
	if err := r.policyEngine.ValidateToolCall(toolCall); err != nil {
		return nil, err
	}

	// Try skills registry first
	if r.skillsRegistry != nil {
		result, err := r.skillsRegistry.ExecuteTool(ctx, toolCall.Function.Name, []byte(toolCall.Function.Arguments))
		if err == nil {
			return result, nil
		}
	}

	// Fall back to tools registry
	if r.toolRegistry != nil {
		return r.toolRegistry.ExecuteJSON(ctx, toolCall.Function.Name, toolCall.Function.Arguments)
	}

	return nil, fmt.Errorf("no tool registry available")
}

// assessToolRisk assesses the risk level of a tool call
func (r *Runner) assessToolRisk(toolCall *llm.ToolCall) string {
	name := strings.ToLower(toolCall.Function.Name)
	args := strings.ToLower(toolCall.Function.Arguments)

	// Check for write operations
	writeTools := []string{"write_file", "edit_file", "append_file", "create", "update", "delete"}
	for _, t := range writeTools {
		if strings.Contains(name, t) {
			return "write"
		}
	}

	// Check for exec operations
	execTools := []string{"exec", "shell", "command", "run"}
	for _, t := range execTools {
		if strings.Contains(name, t) {
			// Check if it's a dangerous command
			dangerous := []string{"rm -rf", "format", "drop", "delete", "shutdown"}
			for _, d := range dangerous {
				if strings.Contains(args, d) {
					return "admin"
				}
			}
			return "exec"
		}
	}

	// Check for network operations
	networkTools := []string{"fetch", "web", "http", "curl", "wget", "download"}
	for _, t := range networkTools {
		if strings.Contains(name, t) {
			return "network"
		}
	}

	return "safe"
}

// createCheckpoint creates a checkpoint for recovery
func (r *Runner) createCheckpoint(ctx context.Context, taskID string, loopState *LoopState) error {
	checkpoint := &TaskCheckpoint{
		TaskID:        taskID,
		Iteration:     loopState.Iteration,
		LoopState:     ToJSON(loopState),
		CanResumeFrom: true,
		ResumeHint:    fmt.Sprintf("Iteration %d", loopState.Iteration),
	}

	if err := r.store.CreateCheckpoint(ctx, checkpoint); err != nil {
		return err
	}

	// Log checkpoint creation
	if err := r.eventLogger.LogCheckpoint(ctx, taskID, checkpoint.ID, loopState.Iteration, true); err != nil {
		r.logger.Warn("Failed to log checkpoint", zap.Error(err))
	}

	return nil
}

// handleTaskCompletion handles successful task completion
func (r *Runner) handleTaskCompletion(ctx context.Context, task *Task, loopState *LoopState, finalResult string) {
	r.logger.Info("Task completed successfully",
		zap.String("task_id", task.ID),
		zap.Int("iterations", loopState.Iteration))

	if finalResult == "" && len(loopState.ActionHistory) > 0 {
		lastAction := loopState.ActionHistory[len(loopState.ActionHistory)-1]
		if lastAction.Type == "respond" {
			finalResult = lastAction.Content
		}
	}

	// Update task result
	if err := r.store.UpdateTaskResult(ctx, task.ID, finalResult, map[string]interface{}{
		"iterations": loopState.Iteration,
		"plan":       loopState.CurrentPlan,
	}, ""); err != nil {
		r.logger.Error("Failed to update task result", zap.Error(err))
	}

	// Mark as completed
	if err := r.store.UpdateTaskState(ctx, task.ID, TaskStateCompleted, ""); err != nil {
		r.logger.Error("Failed to mark task as completed", zap.Error(err))
	}
}

// handleTaskError handles task failure
func (r *Runner) handleTaskError(ctx context.Context, task *Task, errMsg string) {
	r.logger.Error("Task failed",
		zap.String("task_id", task.ID),
		zap.String("error", errMsg))

	// Update task result
	if err := r.store.UpdateTaskResult(ctx, task.ID, "", nil, errMsg); err != nil {
		r.logger.Error("Failed to update task error", zap.Error(err))
	}

	// Mark as failed
	if err := r.store.UpdateTaskState(ctx, task.ID, TaskStateFailed, errMsg); err != nil {
		r.logger.Error("Failed to mark task as failed", zap.Error(err))
	}

	// Log error event
	if err := r.eventLogger.LogError(ctx, task.ID, fmt.Errorf("%s", errMsg), "task_execution"); err != nil {
		r.logger.Warn("Failed to log error event", zap.Error(err))
	}
}

// truncate truncates a string to max length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func (r *Runner) executeApprovedStep(ctx context.Context, approval *TaskApproval) error {
	step, err := r.store.GetStep(ctx, approval.StepID)
	if err != nil {
		return err
	}

	if err := r.store.UpdateStepExecutionState(ctx, step.ID, StepExecutionInProgress, false); err != nil {
		return err
	}

	task, err := r.store.GetTaskByID(ctx, approval.TaskID)
	if err != nil {
		return err
	}

	toolCall := &llm.ToolCall{}
	toolCall.Function.Name = approval.ToolName
	toolCall.Function.Arguments = string(approval.ToolInput)

	effect, err := r.beginToolEffect(ctx, approval.TaskID, step)
	if err != nil {
		return err
	}

	start := time.Now()
	result, execErr := r.executeTool(ctx, toolCall)
	duration := int(time.Since(start).Milliseconds())

	if logErr := r.eventLogger.LogToolExecuted(ctx, approval.TaskID, step.ID, approval.ToolName, execErr == nil, duration); logErr != nil {
		r.logger.Warn("Failed to log approved tool execution", zap.Error(logErr))
	}

	resultJSON := ToJSON(result)
	errMsg := ""
	if execErr != nil {
		errMsg = execErr.Error()
		resultJSON = ToJSON(map[string]string{"error": errMsg})
	}

	effectStatus := EffectStatusCompleted
	if errMsg != "" {
		effectStatus = EffectStatusFailed
	}
	if err := r.store.UpdateEffectResult(ctx, effect.ID, effectStatus, resultJSON, errMsg); err != nil {
		return err
	}

	if err := r.store.UpdateStepResult(ctx, step.ID, resultJSON, errMsg, duration); err != nil {
		return err
	}

	if err := r.store.UpdateTaskProgress(ctx, approval.TaskID, task.CurrentIteration, task.CurrentPlan, string(resultJSON)); err != nil {
		return err
	}

	return nil
}

func (r *Runner) beginToolEffect(ctx context.Context, taskID string, step *TaskStep) (*TaskEffect, error) {
	effect := &TaskEffect{
		TaskID:          taskID,
		StepID:          step.ID,
		ToolName:        step.ToolName,
		ToolFingerprint: step.ToolFingerprint,
		Status:          EffectStatusStarted,
		Input:           step.ToolInput,
	}
	if err := r.store.CreateEffect(ctx, effect); err != nil {
		return nil, err
	}
	return effect, nil
}

func (r *Runner) tryReuseToolResult(ctx context.Context, taskID string, step *TaskStep, effect *TaskEffect, toolCall *llm.ToolCall) (interface{}, bool, error) {
	fingerprint := step.ToolFingerprint
	if fingerprint == "" {
		fingerprint = toolCallFingerprint(toolCall.Function.Name, toolCall.Function.Arguments)
	}

	priorEffect, err := r.store.FindCompletedEffect(ctx, taskID, fingerprint, step.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}

	priorStep, err := r.store.GetStep(ctx, priorEffect.StepID)
	if err != nil {
		return nil, false, err
	}

	if err := r.store.MarkStepReplayed(ctx, step.ID, priorStep); err != nil {
		return nil, false, err
	}
	if err := r.store.MarkEffectReplayed(ctx, effect.ID, priorEffect); err != nil {
		return nil, false, err
	}
	step.ExecutionState = StepExecutionReplayed
	step.ReplayOfStepID = priorStep.ID

	if logErr := r.eventLogger.LogToolReplayed(ctx, taskID, step.ID, priorStep.ID, toolCall.Function.Name); logErr != nil {
		r.logger.Warn("Failed to log tool replay event", zap.Error(logErr))
	}

	if logErr := r.eventLogger.LogToolExecuted(ctx, taskID, step.ID, toolCall.Function.Name, true, 0); logErr != nil {
		r.logger.Warn("Failed to log replayed tool execution", zap.Error(logErr))
	}

	if len(priorStep.ToolOutput) == 0 {
		return nil, true, nil
	}

	var result interface{}
	if err := FromJSON(priorStep.ToolOutput, &result); err != nil {
		return string(priorStep.ToolOutput), true, nil
	}

	return result, true, nil
}

func getResultStatus(result interface{}) (string, bool) {
	resMap, ok := result.(map[string]string)
	if ok {
		status, exists := resMap["status"]
		return status, exists
	}

	resAnyMap, ok := result.(map[string]interface{})
	if !ok {
		return "", false
	}

	status, ok := resAnyMap["status"]
	if !ok {
		return "", false
	}

	statusStr, ok := status.(string)
	return statusStr, ok
}

func getResultResponse(result interface{}) string {
	resMap, ok := result.(map[string]string)
	if ok {
		return resMap["response"]
	}

	resAnyMap, ok := result.(map[string]interface{})
	if !ok {
		return ""
	}

	response, ok := resAnyMap["response"].(string)
	if !ok {
		return ""
	}

	return response
}
