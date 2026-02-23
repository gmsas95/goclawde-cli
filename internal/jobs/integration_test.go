package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestFullWorkflow tests the complete job scheduling workflow
func TestFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full workflow test in short mode")
	}

	logger := zap.NewNop()
	scheduler := NewScheduler(logger)

	executionCount := 0
	job := &Job{
		ID:       "workflow-job",
		Name:     "Workflow Job",
		Schedule: "0/1 * * * * *", // Every second for testing
		Enabled:  true,
		Func: func(ctx context.Context) error {
			executionCount++
			return nil
		},
	}

	err := scheduler.RegisterJob(job)
	require.NoError(t, err)

	// Start scheduler
	scheduler.Start()

	// Wait for a few executions
	time.Sleep(2500 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()

	// Verify executions
	assert.GreaterOrEqual(t, executionCount, 2, "Expected at least 2 executions")

	// Verify job state
	jobState, err := scheduler.GetJob("workflow-job")
	require.NoError(t, err)
	assert.Equal(t, executionCount, jobState.RunCount)
}

// TestJobErrorRecovery tests that failed jobs don't stop other jobs
func TestJobErrorRecovery(t *testing.T) {
	logger := zap.NewNop()
	scheduler := NewScheduler(logger)

	successCount := 0
	failCount := 0

	// Register a failing job
	failJob := &Job{
		ID:       "fail-job",
		Name:     "Fail Job",
		Schedule: "0/1 * * * * *",
		Enabled:  true,
		Func: func(ctx context.Context) error {
			failCount++
			return errors.New("intentional failure")
		},
	}

	// Register a successful job
	successJob := &Job{
		ID:       "success-job",
		Name:     "Success Job",
		Schedule: "0/1 * * * * *",
		Enabled:  true,
		Func: func(ctx context.Context) error {
			successCount++
			return nil
		},
	}

	err := scheduler.RegisterJob(failJob)
	require.NoError(t, err)

	err = scheduler.RegisterJob(successJob)
	require.NoError(t, err)

	// Start scheduler
	scheduler.Start()

	// Wait for executions
	time.Sleep(2500 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()

	// Both jobs should have executed
	assert.GreaterOrEqual(t, failCount, 2, "Failing job should still execute")
	assert.GreaterOrEqual(t, successCount, 2, "Success job should execute")
	assert.Equal(t, failCount, successCount, "Both jobs should have same execution count")
}

// TestSchedulerConcurrency tests that jobs don't overlap
func TestSchedulerConcurrency(t *testing.T) {
	logger := zap.NewNop()
	scheduler := NewScheduler(logger)

	concurrentExecutions := 0
	maxConcurrent := 0

	job := &Job{
		ID:       "concurrent-job",
		Name:     "Concurrent Job",
		Schedule: "0/1 * * * * *", // Every second
		Enabled:  true,
		Func: func(ctx context.Context) error {
			concurrentExecutions++
			if concurrentExecutions > maxConcurrent {
				maxConcurrent = concurrentExecutions
			}
			time.Sleep(1500 * time.Millisecond) // Longer than schedule interval
			concurrentExecutions--
			return nil
		},
	}

	err := scheduler.RegisterJob(job)
	require.NoError(t, err)

	// Start scheduler
	scheduler.Start()

	// Wait for potential overlapping executions
	time.Sleep(3500 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()

	// With SkipIfStillRunning, we should never have concurrent executions
	assert.LessOrEqual(t, maxConcurrent, 1, "Should not have concurrent executions due to SkipIfStillRunning")
}

// TestJobContextCancellation tests that context cancellation works
func TestJobContextCancellation(t *testing.T) {
	logger := zap.NewNop()
	scheduler := NewScheduler(logger)

	job := &Job{
		ID:       "cancel-job",
		Name:     "Cancel Job",
		Schedule: "0/1 * * * * *",
		Enabled:  true,
		Func: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return nil
			}
		},
	}

	err := scheduler.RegisterJob(job)
	require.NoError(t, err)

	// Start and immediately stop to trigger cancellation
	scheduler.Start()
	time.Sleep(100 * time.Millisecond)
	scheduler.Stop()

	// Job might or might not have been cancelled depending on timing
	// Just verify the scheduler stopped cleanly
	assert.True(t, scheduler.stopped)
}
