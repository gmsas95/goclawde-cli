package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/gmsas95/myrai-cli/internal/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCLITestStore(t *testing.T) *runtime.Store {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:cli_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	store := runtime.NewStore(db)
	require.NoError(t, store.Migrate())
	return store
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	outputCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outputCh <- buf.String()
	}()

	fn()
	_ = w.Close()
	os.Stdout = oldStdout
	output := <-outputCh
	_ = r.Close()
	return output
}

func TestHandleTaskShowDisplaysReplayMetadata(t *testing.T) {
	store := setupCLITestStore(t)
	now := time.Now()
	task := &runtime.Task{
		ID:               "task_test_show",
		Title:            "Replay Task",
		Goal:             "Show replay metadata",
		State:            runtime.TaskStateCompleted,
		CurrentIteration: 2,
		MaxIterations:    4,
		CreatedAt:        now,
		StartedAt:        &now,
		CompletedAt:      &now,
	}
	require.NoError(t, store.CreateTask(context.Background(), task))

	sourceStep := &runtime.TaskStep{
		ID:             "step_source",
		TaskID:         task.ID,
		Iteration:      1,
		Type:           runtime.StepTypeTool,
		ExecutionState: runtime.StepExecutionCompleted,
		Content:        "Original tool run",
		ToolName:       "write_file",
		SequenceNumber: 1,
		StartedAt:      &now,
		EndedAt:        &now,
		CreatedAt:      now,
	}
	require.NoError(t, store.CreateStep(context.Background(), sourceStep))

	replayedStep := &runtime.TaskStep{
		ID:             "step_replayed",
		TaskID:         task.ID,
		Iteration:      2,
		Type:           runtime.StepTypeTool,
		ExecutionState: runtime.StepExecutionReplayed,
		Content:        "Reused tool result",
		ToolName:       "write_file",
		ReplayOfStepID: sourceStep.ID,
		SequenceNumber: 2,
		StartedAt:      &now,
		EndedAt:        &now,
		CreatedAt:      now,
	}
	require.NoError(t, store.CreateStep(context.Background(), replayedStep))

	output := captureStdout(t, func() {
		handleTaskShow(context.Background(), &TaskCommands{store: store}, []string{task.ID})
	})

	assert.Contains(t, output, "state: replayed")
	assert.Contains(t, output, "Replay: reused result from step_source")
}
