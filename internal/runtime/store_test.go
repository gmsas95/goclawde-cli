package runtime

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	// Auto-migrate tables
	err = db.AutoMigrate(
		&Task{},
		&TaskStep{},
		&TaskEffect{},
		&TaskEvent{},
		&TaskCheckpoint{},
		&TaskApproval{},
	)
	require.NoError(t, err)

	return db
}

func TestTaskStateTransitions(t *testing.T) {
	tests := []struct {
		name      string
		fromState TaskState
		toState   TaskState
		allowed   bool
	}{
		{"pending to running", TaskStatePending, TaskStateRunning, true},
		{"pending to cancelled", TaskStatePending, TaskStateCancelled, true},
		{"pending to completed", TaskStatePending, TaskStateCompleted, false},
		{"running to blocked", TaskStateRunning, TaskStateBlocked, true},
		{"running to failed", TaskStateRunning, TaskStateFailed, true},
		{"running to completed", TaskStateRunning, TaskStateCompleted, true},
		{"running to cancelled", TaskStateRunning, TaskStateCancelled, true},
		{"blocked to running", TaskStateBlocked, TaskStateRunning, true},
		{"blocked to failed", TaskStateBlocked, TaskStateFailed, true},
		{"blocked to cancelled", TaskStateBlocked, TaskStateCancelled, true},
		{"completed to running", TaskStateCompleted, TaskStateRunning, false},
		{"failed to completed", TaskStateFailed, TaskStateCompleted, false},
		{"cancelled to running", TaskStateCancelled, TaskStateRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{State: tt.fromState}
			result := task.CanTransition(tt.toState)
			assert.Equal(t, tt.allowed, result,
				"transition from %s to %s should be allowed=%v", tt.fromState, tt.toState, tt.allowed)
		})
	}
}

func TestTaskCRUD(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	t.Run("create and get task", func(t *testing.T) {
		task := &Task{
			Title:       "Test Task",
			Goal:        "Test goal",
			Description: "Test description",
			State:       TaskStatePending,
		}

		err := store.CreateTask(ctx, task)
		require.NoError(t, err)
		assert.NotEmpty(t, task.ID)
		assert.NotZero(t, task.CreatedAt)

		// Get the task
		retrieved, err := store.GetTaskByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, task.Title, retrieved.Title)
		assert.Equal(t, task.Goal, retrieved.Goal)
	})

	t.Run("list tasks", func(t *testing.T) {
		// Create multiple tasks
		for i := 0; i < 3; i++ {
			task := &Task{
				Title: "Task " + string(rune('A'+i)),
				Goal:  "Goal " + string(rune('A'+i)),
				State: TaskStateCompleted,
			}
			err := store.CreateTask(ctx, task)
			require.NoError(t, err)
		}

		tasks, err := store.ListTasks(ctx, nil, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 3)
	})

	t.Run("list with state filter", func(t *testing.T) {
		pending := TaskStatePending
		tasks, err := store.ListTasks(ctx, &pending, 10)
		require.NoError(t, err)

		for _, task := range tasks {
			assert.Equal(t, TaskStatePending, task.State)
		}
	})

	t.Run("delete task", func(t *testing.T) {
		task := &Task{
			Title: "To Delete",
			Goal:  "To be deleted",
		}
		err := store.CreateTask(ctx, task)
		require.NoError(t, err)

		err = store.DeleteTask(ctx, task.ID)
		require.NoError(t, err)

		_, err = store.GetTaskByID(ctx, task.ID)
		assert.Error(t, err) // Should not exist
	})
}

func TestTaskStateUpdate(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	task := &Task{
		Title: "State Test",
		Goal:  "Test state transitions",
		State: TaskStatePending,
	}
	err := store.CreateTask(ctx, task)
	require.NoError(t, err)

	t.Run("valid transition", func(t *testing.T) {
		err := store.UpdateTaskState(ctx, task.ID, TaskStateRunning, "starting")
		require.NoError(t, err)

		updated, err := store.GetTaskByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, TaskStateRunning, updated.State)
		assert.NotNil(t, updated.StartedAt)
	})

	t.Run("invalid transition", func(t *testing.T) {
		err := store.UpdateTaskState(ctx, task.ID, TaskStatePending, "going back")
		assert.Error(t, err) // Can't go back to pending from running
	})

	t.Run("complete task", func(t *testing.T) {
		err := store.UpdateTaskState(ctx, task.ID, TaskStateCompleted, "done")
		require.NoError(t, err)

		updated, err := store.GetTaskByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, TaskStateCompleted, updated.State)
		assert.NotNil(t, updated.CompletedAt)
	})
}

func TestTaskSteps(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	task := &Task{
		Title: "Step Test",
		Goal:  "Test steps",
	}
	err := store.CreateTask(ctx, task)
	require.NoError(t, err)

	t.Run("create steps", func(t *testing.T) {
		for i := 1; i <= 3; i++ {
			step := &TaskStep{
				TaskID:         task.ID,
				Iteration:      i,
				Type:           StepTypeThink,
				Content:        "Thinking iteration " + string(rune('0'+i)),
				SequenceNumber: i,
			}
			err := store.CreateStep(ctx, step)
			require.NoError(t, err)
			assert.NotEmpty(t, step.ID)
		}
	})

	t.Run("list steps", func(t *testing.T) {
		steps, err := store.ListSteps(ctx, task.ID)
		require.NoError(t, err)
		assert.Len(t, steps, 3)

		// Verify order
		for i, step := range steps {
			assert.Equal(t, i+1, step.SequenceNumber)
		}
	})

	t.Run("update step result", func(t *testing.T) {
		steps, err := store.ListSteps(ctx, task.ID)
		require.NoError(t, err)
		require.Greater(t, len(steps), 0)

		step := steps[0]
		output := ToJSON(map[string]string{"result": "success"})

		err = store.UpdateStepResult(ctx, step.ID, output, "", 100)
		require.NoError(t, err)

		// Reload task with steps
		updatedTask, err := store.GetTask(ctx, task.ID)
		require.NoError(t, err)

		found := false
		for _, s := range updatedTask.Steps {
			if s.ID == step.ID {
				found = true
				assert.NotNil(t, s.EndedAt)
				assert.Equal(t, 100, s.DurationMs)
			}
		}
		assert.True(t, found)
	})
}

func TestTaskApprovals(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	task := &Task{
		Title: "Approval Test",
		Goal:  "Test approvals",
	}
	err := store.CreateTask(ctx, task)
	require.NoError(t, err)

	step := &TaskStep{
		TaskID:    task.ID,
		Iteration: 1,
		Type:      StepTypeTool,
		ToolName:  "write_file",
		ToolRisk:  "write",
	}
	err = store.CreateStep(ctx, step)
	require.NoError(t, err)

	t.Run("create approval", func(t *testing.T) {
		expires := time.Now().Add(24 * time.Hour)
		approval := &TaskApproval{
			TaskID:      task.ID,
			StepID:      step.ID,
			ToolName:    "write_file",
			RiskLevel:   "write",
			Description: "Write to file.txt",
			ExpiresAt:   &expires,
		}

		err := store.CreateApproval(ctx, approval)
		require.NoError(t, err)
		assert.NotEmpty(t, approval.ID)
		assert.Equal(t, ApprovalStatusPending, approval.Status)
	})

	t.Run("list pending approvals", func(t *testing.T) {
		approvals, err := store.ListPendingApprovals(ctx, task.ID)
		require.NoError(t, err)
		assert.Len(t, approvals, 1)
	})

	t.Run("resolve approval", func(t *testing.T) {
		approvals, err := store.ListPendingApprovals(ctx, task.ID)
		require.NoError(t, err)
		require.Len(t, approvals, 1)

		err = store.ResolveApproval(ctx, approvals[0].ID, ApprovalStatusApproved, "test_user", "")
		require.NoError(t, err)

		// Verify
		approval, err := store.GetApproval(ctx, approvals[0].ID)
		require.NoError(t, err)
		assert.Equal(t, ApprovalStatusApproved, approval.Status)
		assert.Equal(t, "test_user", approval.ApprovedBy)
		assert.NotNil(t, approval.ApprovedAt)
	})
}

func TestTaskEvents(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	task := &Task{
		Title: "Event Test",
		Goal:  "Test events",
	}
	err := store.CreateTask(ctx, task)
	require.NoError(t, err)

	t.Run("create events", func(t *testing.T) {
		events := []TaskEvent{
			{
				TaskID:  task.ID,
				Type:    EventTypeStateChange,
				Message: "Task created",
			},
			{
				TaskID:  task.ID,
				Type:    EventTypeStepCreated,
				Message: "Step 1 created",
			},
			{
				TaskID:  task.ID,
				Type:    EventTypeToolExecuted,
				Message: "Tool executed successfully",
			},
		}

		for _, event := range events {
			err := store.CreateEvent(ctx, &event)
			require.NoError(t, err)
			assert.NotEmpty(t, event.ID)
		}
	})

	t.Run("list events", func(t *testing.T) {
		events, err := store.ListEvents(ctx, task.ID)
		require.NoError(t, err)
		assert.Len(t, events, 3)

		// Verify chronological order
		for i := 1; i < len(events); i++ {
			assert.True(t, events[i].CreatedAt.After(events[i-1].CreatedAt) ||
				events[i].CreatedAt.Equal(events[i-1].CreatedAt))
		}
	})
}

func TestCheckpoints(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	task := &Task{
		Title: "Checkpoint Test",
		Goal:  "Test checkpoints",
	}
	err := store.CreateTask(ctx, task)
	require.NoError(t, err)

	t.Run("create checkpoints", func(t *testing.T) {
		for i := 1; i <= 3; i++ {
			checkpoint := &TaskCheckpoint{
				TaskID:        task.ID,
				Iteration:     i,
				LoopState:     ToJSON(map[string]int{"iteration": i}),
				CanResumeFrom: i > 1, // Can't resume from first checkpoint
				ResumeHint:    "Iteration " + string(rune('0'+i)),
			}
			err := store.CreateCheckpoint(ctx, checkpoint)
			require.NoError(t, err)
			assert.NotEmpty(t, checkpoint.ID)
		}
	})

	t.Run("get latest resumable checkpoint", func(t *testing.T) {
		checkpoint, err := store.GetLatestCheckpoint(ctx, task.ID)
		require.NoError(t, err)
		assert.NotNil(t, checkpoint)
		assert.Equal(t, 3, checkpoint.Iteration)
		assert.True(t, checkpoint.CanResumeFrom)
	})

	t.Run("list checkpoints", func(t *testing.T) {
		checkpoints, err := store.ListCheckpoints(ctx, task.ID)
		require.NoError(t, err)
		assert.Len(t, checkpoints, 3)

		// Verify descending order (newest first)
		for i := 1; i < len(checkpoints); i++ {
			assert.True(t, checkpoints[i].CreatedAt.Before(checkpoints[i-1].CreatedAt))
		}
	})
}

func TestTaskProgress(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	task := &Task{
		Title:         "Progress Test",
		Goal:          "Test progress tracking",
		State:         TaskStateRunning,
		MaxIterations: 10,
	}
	err := store.CreateTask(ctx, task)
	require.NoError(t, err)

	t.Run("update progress", func(t *testing.T) {
		err := store.UpdateTaskProgress(ctx, task.ID, 5, "Current plan", "Last tool result")
		require.NoError(t, err)

		updated, err := store.GetTaskByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, updated.CurrentIteration)
		assert.Equal(t, "Current plan", updated.CurrentPlan)
		assert.Equal(t, "Last tool result", updated.LastToolResult)
	})

	t.Run("update result", func(t *testing.T) {
		resultData := map[string]interface{}{
			"success": true,
			"output":  "Task completed successfully",
		}

		err := store.UpdateTaskResult(ctx, task.ID, "Final answer", resultData, "")
		require.NoError(t, err)

		updated, err := store.GetTaskByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, "Final answer", updated.FinalResult)
		assert.NotNil(t, updated.ResultData)
	})
}

func TestRunningTasks(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	// Create tasks in various states
	states := []TaskState{
		TaskStateRunning,
		TaskStateBlocked,
		TaskStatePending,
		TaskStateCompleted,
		TaskStateRunning,
	}

	for i, state := range states {
		task := &Task{
			Title: "Task " + string(rune('0'+i)),
			Goal:  "Test goal",
			State: state,
		}
		if state == TaskStateRunning || state == TaskStateBlocked {
			task.CurrentIteration = i + 1
		}
		err := store.CreateTask(ctx, task)
		require.NoError(t, err)
	}

	t.Run("list running tasks", func(t *testing.T) {
		runningTasks, err := store.ListRunningTasks(ctx)
		require.NoError(t, err)
		assert.Len(t, runningTasks, 3) // 2 running + 1 blocked

		for _, task := range runningTasks {
			assert.True(t, task.State == TaskStateRunning || task.State == TaskStateBlocked)
		}
	})
}

func TestApprovalExpiration(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	task := &Task{
		Title: "Expiration Test",
		Goal:  "Test approval expiration",
	}
	err := store.CreateTask(ctx, task)
	require.NoError(t, err)

	step := &TaskStep{
		TaskID: task.ID,
		Type:   StepTypeTool,
	}
	err = store.CreateStep(ctx, step)
	require.NoError(t, err)

	t.Run("expired approval", func(t *testing.T) {
		// Create approval that expires in the past
		past := time.Now().Add(-1 * time.Hour)
		approval := &TaskApproval{
			TaskID:    task.ID,
			StepID:    step.ID,
			ToolName:  "test_tool",
			Status:    ApprovalStatusPending,
			ExpiresAt: &past,
		}

		err := store.CreateApproval(ctx, approval)
		require.NoError(t, err)

		// Check if expired
		assert.True(t, approval.IsExpired())
	})

	t.Run("non-expired approval", func(t *testing.T) {
		// Create approval that expires in the future
		future := time.Now().Add(24 * time.Hour)
		approval := &TaskApproval{
			TaskID:    task.ID,
			StepID:    step.ID,
			ToolName:  "test_tool",
			Status:    ApprovalStatusPending,
			ExpiresAt: &future,
		}

		err := store.CreateApproval(ctx, approval)
		require.NoError(t, err)

		// Check if not expired
		assert.False(t, approval.IsExpired())
	})
}
