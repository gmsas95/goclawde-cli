package dashboard

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/gmsas95/myrai-cli/internal/runtime"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// TaskHandler handles task-related dashboard API endpoints
type TaskHandler struct {
	runtimeStore *runtime.Store
	runner       *runtime.Runner
	logger       *zap.Logger
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(store *runtime.Store, runner *runtime.Runner, logger *zap.Logger) *TaskHandler {
	return &TaskHandler{
		runtimeStore: store,
		runner:       runner,
		logger:       logger,
	}
}

// RegisterTaskRoutes registers task API routes
func (h *TaskHandler) RegisterTaskRoutes(api fiber.Router) {
	tasks := api.Group("/tasks")

	// Task CRUD
	tasks.Get("/", h.listTasks)
	tasks.Post("/", h.createTask)
	tasks.Get("/:id", h.getTask)
	tasks.Delete("/:id", h.deleteTask)

	// Task control
	tasks.Post("/:id/resume", h.resumeTask)
	tasks.Post("/:id/cancel", h.cancelTask)

	// Task timeline and events
	tasks.Get("/:id/timeline", h.getTaskTimeline)
	tasks.Get("/:id/steps", h.getTaskSteps)
	tasks.Get("/:id/events", h.getTaskEvents)

	// Approvals
	tasks.Get("/:id/approvals", h.getTaskApprovals)
	tasks.Post("/approvals/:approvalId/approve", h.approveAction)
	tasks.Post("/approvals/:approvalId/deny", h.denyAction)

	// Running tasks (for recovery display)
	tasks.Get("/running/all", h.getRunningTasks)
}

// TaskCreateRequest represents a request to create a task
type TaskCreateRequest struct {
	Goal            string `json:"goal"`
	Description     string `json:"description"`
	MaxIterations   int    `json:"max_iterations"`
	RequireApproval bool   `json:"require_approval"`
}

// TaskResponse represents a task response
type TaskResponse struct {
	*runtime.Task
}

// listTasks returns all tasks with optional filtering
func (h *TaskHandler) listTasks(c *fiber.Ctx) error {
	ctx := context.Background()

	// Get optional state filter
	stateStr := c.Query("state")
	var state *runtime.TaskState
	if stateStr != "" {
		s := runtime.TaskState(stateStr)
		state = &s
	}

	limit := c.QueryInt("limit", 50)

	tasks, err := h.runtimeStore.ListTasks(ctx, state, limit)
	if err != nil {
		h.logger.Error("Failed to list tasks", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list tasks",
		})
	}

	return c.JSON(fiber.Map{
		"tasks": tasks,
		"count": len(tasks),
	})
}

// createTask creates a new task
func (h *TaskHandler) createTask(c *fiber.Ctx) error {
	var req TaskCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Goal == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Goal is required",
		})
	}

	ctx := context.Background()
	task, err := h.runner.StartTask(ctx, req.Goal, req.Description, req.MaxIterations, req.RequireApproval)
	if err != nil {
		h.logger.Error("Failed to create task", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create task",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(TaskResponse{task})
}

// getTask returns a single task with all details
func (h *TaskHandler) getTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	ctx := context.Background()

	task, err := h.runtimeStore.GetTask(ctx, taskID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Task not found",
		})
	}

	return c.JSON(TaskResponse{task})
}

// deleteTask deletes a task
func (h *TaskHandler) deleteTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	ctx := context.Background()

	if err := h.runtimeStore.DeleteTask(ctx, taskID); err != nil {
		h.logger.Error("Failed to delete task", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete task",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Task deleted successfully",
	})
}

// resumeTask resumes a task
func (h *TaskHandler) resumeTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	ctx := context.Background()

	task, err := h.runner.ResumeTask(ctx, taskID)
	if err != nil {
		h.logger.Error("Failed to resume task", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(TaskResponse{task})
}

// cancelTask cancels a task
func (h *TaskHandler) cancelTask(c *fiber.Ctx) error {
	taskID := c.Params("id")

	var req struct {
		Reason string `json:"reason"`
	}
	c.BodyParser(&req)

	if req.Reason == "" {
		req.Reason = "Cancelled by user"
	}

	ctx := context.Background()
	if err := h.runner.CancelTask(ctx, taskID, req.Reason); err != nil {
		h.logger.Error("Failed to cancel task", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Task cancelled",
	})
}

// getTaskTimeline returns the complete timeline for a task
func (h *TaskHandler) getTaskTimeline(c *fiber.Ctx) error {
	taskID := c.Params("id")
	ctx := context.Background()

	task, err := h.runtimeStore.GetTask(ctx, taskID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Task not found",
		})
	}

	// Combine steps and events into timeline
	type TimelineItem struct {
		Type      string          `json:"type"`
		Timestamp interface{}     `json:"timestamp"`
		Data      json.RawMessage `json:"data"`
	}

	var timeline []TimelineItem

	// Add steps
	for _, step := range task.Steps {
		timeline = append(timeline, TimelineItem{
			Type:      "step",
			Timestamp: step.CreatedAt,
			Data:      mustJSON(step),
		})
	}

	// Add events
	for _, event := range task.Events {
		timeline = append(timeline, TimelineItem{
			Type:      "event",
			Timestamp: event.CreatedAt,
			Data:      mustJSON(event),
		})
	}

	sort.Slice(timeline, func(i, j int) bool {
		return timelineTimestamp(timeline[i].Timestamp).Before(timelineTimestamp(timeline[j].Timestamp))
	})

	return c.JSON(fiber.Map{
		"task_id":  taskID,
		"timeline": timeline,
		"count":    len(timeline),
	})
}

// getTaskSteps returns all steps for a task
func (h *TaskHandler) getTaskSteps(c *fiber.Ctx) error {
	taskID := c.Params("id")
	ctx := context.Background()

	steps, err := h.runtimeStore.ListSteps(ctx, taskID)
	if err != nil {
		h.logger.Error("Failed to list steps", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list steps",
		})
	}

	return c.JSON(fiber.Map{
		"steps": steps,
		"count": len(steps),
	})
}

// getTaskEvents returns all events for a task
func (h *TaskHandler) getTaskEvents(c *fiber.Ctx) error {
	taskID := c.Params("id")
	ctx := context.Background()

	events, err := h.runtimeStore.ListEvents(ctx, taskID)
	if err != nil {
		h.logger.Error("Failed to list events", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list events",
		})
	}

	return c.JSON(fiber.Map{
		"events": events,
		"count":  len(events),
	})
}

// getTaskApprovals returns all approvals for a task
func (h *TaskHandler) getTaskApprovals(c *fiber.Ctx) error {
	taskID := c.Params("id")
	ctx := context.Background()

	approvals, err := h.runtimeStore.ListPendingApprovals(ctx, taskID)
	if err != nil {
		h.logger.Error("Failed to list approvals", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list approvals",
		})
	}

	return c.JSON(fiber.Map{
		"approvals": approvals,
		"count":     len(approvals),
	})
}

// approveAction approves a pending action
func (h *TaskHandler) approveAction(c *fiber.Ctx) error {
	approvalID := c.Params("approvalId")

	var req struct {
		ApprovedBy string `json:"approved_by"`
	}
	c.BodyParser(&req)

	if req.ApprovedBy == "" {
		req.ApprovedBy = "dashboard"
	}

	ctx := context.Background()
	if err := h.runner.ApproveAction(ctx, approvalID, req.ApprovedBy); err != nil {
		h.logger.Error("Failed to approve action", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Action approved",
	})
}

// denyAction denies a pending action
func (h *TaskHandler) denyAction(c *fiber.Ctx) error {
	approvalID := c.Params("approvalId")

	var req struct {
		DeniedBy string `json:"denied_by"`
		Reason   string `json:"reason"`
	}
	c.BodyParser(&req)

	if req.DeniedBy == "" {
		req.DeniedBy = "dashboard"
	}
	if req.Reason == "" {
		req.Reason = "Denied by user"
	}

	ctx := context.Background()
	if err := h.runner.DenyAction(ctx, approvalID, req.DeniedBy, req.Reason); err != nil {
		h.logger.Error("Failed to deny action", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Action denied",
	})
}

// getRunningTasks returns all running or blocked tasks (for recovery display)
func (h *TaskHandler) getRunningTasks(c *fiber.Ctx) error {
	ctx := context.Background()

	tasks, err := h.runtimeStore.ListRunningTasks(ctx)
	if err != nil {
		h.logger.Error("Failed to list running tasks", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list running tasks",
		})
	}

	return c.JSON(fiber.Map{
		"tasks": tasks,
		"count": len(tasks),
	})
}

// mustJSON marshals v to JSON or returns empty bytes
func mustJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return json.RawMessage(b)
}

func timelineTimestamp(v interface{}) time.Time {
	t, ok := v.(time.Time)
	if !ok {
		return time.Time{}
	}
	return t
}
