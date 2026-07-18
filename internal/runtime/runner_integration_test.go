package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gmsas95/myrai-cli/internal/agent"
	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type sequenceLLMServer struct {
	mu        sync.Mutex
	responses []string
	index     int
}

func newSequenceLLMServer(t *testing.T, responses []string) *httptest.Server {
	t.Helper()

	server := &sequenceLLMServer{responses: responses}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.mu.Lock()
		defer server.mu.Unlock()

		require.Less(t, server.index, len(server.responses), "received more LLM requests than expected")

		response := map[string]interface{}{
			"id":      "test-response",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "test-model",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": server.responses[server.index],
					},
					"finish_reason": "stop",
				},
			},
		}
		server.index++

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
}

type countingTool struct {
	mu    sync.Mutex
	name  string
	calls int
	last  map[string]interface{}
}

func (t *countingTool) Name() string        { return t.name }
func (t *countingTool) Description() string { return "test tool" }
func (t *countingTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
	}
}
func (t *countingTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.calls++
	t.last = args
	return map[string]interface{}{"ok": true}, nil
}

func (t *countingTool) CallCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.calls
}

func newTestRunner(t *testing.T, responses []string, customTools ...tools.Tool) (*Runner, *Store, func()) {
	t.Helper()

	db := setupTestDB(t)
	store := NewStore(db)
	llmServer := newSequenceLLMServer(t, responses)
	workspace := t.TempDir()

	registry := tools.NewRegistry(nil)
	for _, tool := range customTools {
		registry.Register(tool)
	}

	client := llm.NewClient(config.Provider{
		BaseURL:   llmServer.URL,
		APIKey:    "test-key",
		Model:     "test-model",
		MaxTokens: 256,
		Timeout:   5,
	})

	runner := NewRunner(RunnerConfig{
		Logger:        zap.NewNop(),
		Store:         store,
		ToolRegistry:  registry,
		LLMClient:     client,
		WorkspaceRoot: workspace,
		AllowedCmds:   []string{"git", "go", "ls", "cat", "grep", "find"},
	})

	cleanup := func() {
		llmServer.Close()
	}

	return runner, store, cleanup
}

func waitForTaskState(t *testing.T, store *Store, taskID string, expected TaskState) *Task {
	t.Helper()

	var task *Task
	require.Eventually(t, func() bool {
		var err error
		task, err = store.GetTask(context.Background(), taskID)
		if err != nil {
			return false
		}
		return task.State == expected
	}, 5*time.Second, 50*time.Millisecond)

	return task
}

func TestRunnerApprovalFlowIntegration(t *testing.T) {
	responses := []string{
		`{"type":"tool","content":"Write the result to disk","tool_call":{"id":"call_1","type":"function","function":{"name":"write_file","arguments":"{\"path\":\"result.txt\",\"content\":\"done\"}"}}}`,
		`{"type":"respond","content":"Task finished successfully"}`,
	}

	writeTool := &countingTool{name: "write_file"}
	runner, store, cleanup := newTestRunner(t, responses, writeTool)
	defer cleanup()

	task, err := runner.StartTask(context.Background(), "write result", "", 5, true)
	require.NoError(t, err)

	blockedTask := waitForTaskState(t, store, task.ID, TaskStateBlocked)
	require.Len(t, blockedTask.Approvals, 1)
	assert.Equal(t, 0, writeTool.CallCount(), "tool should not execute before approval")

	err = runner.ApproveAction(context.Background(), blockedTask.Approvals[0].ID, "tester")
	require.NoError(t, err)

	completedTask := waitForTaskState(t, store, task.ID, TaskStateCompleted)
	assert.Equal(t, "Task finished successfully", completedTask.FinalResult)
	assert.Equal(t, 1, writeTool.CallCount(), "tool should execute exactly once after approval")

	events, err := store.ListEvents(context.Background(), task.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, events)

	var foundApprovalResolved bool
	for _, event := range events {
		if event.Type == EventTypeApprovalRes {
			foundApprovalResolved = true
			break
		}
	}
	assert.True(t, foundApprovalResolved)
}

func TestRecoveryResumesInterruptedRunningTask(t *testing.T) {
	responses := []string{
		`{"type":"respond","content":"Recovered and completed"}`,
	}

	runner, store, cleanup := newTestRunner(t, responses)
	defer cleanup()

	now := time.Now()
	task := &Task{
		Title:            "Recover task",
		Goal:             "resume after restart",
		State:            TaskStateRunning,
		MaxIterations:    3,
		CurrentIteration: 1,
		StartedAt:        &now,
	}
	require.NoError(t, store.CreateTask(context.Background(), task))

	checkpointState := &LoopState{
		Iteration:   1,
		CurrentPlan: "Continue after restart",
		ActionHistory: []agent.Action{
			{Iteration: 1, Type: "think", Content: "Need to continue", Timestamp: now},
		},
	}
	require.NoError(t, store.CreateCheckpoint(context.Background(), &TaskCheckpoint{
		TaskID:        task.ID,
		Iteration:     1,
		LoopState:     ToJSON(checkpointState),
		CanResumeFrom: true,
		ResumeHint:    "resume iteration 2",
	}))

	recovery := NewRecovery(store, runner, zap.NewNop())
	require.NoError(t, recovery.RecoverOnStartup(context.Background()))

	completedTask := waitForTaskState(t, store, task.ID, TaskStateCompleted)
	assert.Equal(t, "Recovered and completed", completedTask.FinalResult)
	assert.Equal(t, 2, completedTask.CurrentIteration)

	events, err := store.ListEvents(context.Background(), task.ID)
	require.NoError(t, err)

	var foundResume bool
	for _, event := range events {
		if event.Type == EventTypeResume {
			foundResume = true
			break
		}
	}
	assert.True(t, foundResume)
}

func TestRunnerDenialFlowIntegration(t *testing.T) {
	responses := []string{
		`{"type":"tool","content":"Write the result to disk","tool_call":{"id":"call_1","type":"function","function":{"name":"write_file","arguments":"{\"path\":\"result.txt\",\"content\":\"done\"}"}}}`,
		`{"type":"respond","content":"User denied the write, so I am stopping safely"}`,
	}

	writeTool := &countingTool{name: "write_file"}
	runner, store, cleanup := newTestRunner(t, responses, writeTool)
	defer cleanup()

	task, err := runner.StartTask(context.Background(), "write result", "", 5, true)
	require.NoError(t, err)

	blockedTask := waitForTaskState(t, store, task.ID, TaskStateBlocked)
	require.Len(t, blockedTask.Approvals, 1)

	err = runner.DenyAction(context.Background(), blockedTask.Approvals[0].ID, "tester", "not allowed")
	require.NoError(t, err)

	completedTask := waitForTaskState(t, store, task.ID, TaskStateCompleted)
	assert.Equal(t, "User denied the write, so I am stopping safely", completedTask.FinalResult)
	assert.Equal(t, 0, writeTool.CallCount(), "denied tool should never execute")
	assert.Contains(t, completedTask.LastToolResult, "denied")

	approvals, err := store.ListPendingApprovals(context.Background(), task.ID)
	require.NoError(t, err)
	assert.Len(t, approvals, 0)
	resolvedApproval, err := store.GetApproval(context.Background(), blockedTask.Approvals[0].ID)
	require.NoError(t, err)
	assert.Equal(t, ApprovalStatusDenied, resolvedApproval.Status)
}

func TestRecoveryReusesPersistedToolResultWithoutReexecution(t *testing.T) {
	responses := []string{
		`{"type":"tool","content":"Repeat the same tool call","tool_call":{"id":"call_1","type":"function","function":{"name":"write_file","arguments":"{\"path\":\"result.txt\",\"content\":\"done\"}"}}}`,
		`{"type":"respond","content":"Recovered from persisted tool memory"}`,
	}

	writeTool := &countingTool{name: "write_file"}
	runner, store, cleanup := newTestRunner(t, responses, writeTool)
	defer cleanup()

	now := time.Now()
	toolArgs := `{"path":"result.txt","content":"done"}`
	task := &Task{
		Title:            "Recover cached tool result",
		Goal:             "resume without re-running tool",
		State:            TaskStateRunning,
		MaxIterations:    4,
		CurrentIteration: 1,
		CurrentPlan:      "Reuse previous work",
		LastToolResult:   `{"ok":true}`,
		StartedAt:        &now,
	}
	require.NoError(t, store.CreateTask(context.Background(), task))

	priorStep := &TaskStep{
		TaskID:          task.ID,
		Iteration:       1,
		Type:            StepTypeTool,
		Content:         "Write the result to disk",
		ToolName:        "write_file",
		ToolInput:       json.RawMessage(toolArgs),
		ToolOutput:      ToJSON(map[string]bool{"ok": true}),
		ToolFingerprint: toolCallFingerprint("write_file", toolArgs),
		SequenceNumber:  1,
		StartedAt:       &now,
		EndedAt:         &now,
	}
	require.NoError(t, store.CreateStep(context.Background(), priorStep))
	priorEffect := &TaskEffect{
		TaskID:          task.ID,
		StepID:          priorStep.ID,
		ToolName:        priorStep.ToolName,
		ToolFingerprint: priorStep.ToolFingerprint,
		Status:          EffectStatusCompleted,
		Input:           priorStep.ToolInput,
		Output:          priorStep.ToolOutput,
		StartedAt:       now,
		FinishedAt:      &now,
	}
	require.NoError(t, store.CreateEffect(context.Background(), priorEffect))

	checkpointState := &LoopState{
		Iteration:      1,
		CurrentPlan:    task.CurrentPlan,
		LastToolResult: map[string]bool{"ok": true},
		ActionHistory: []agent.Action{
			{Iteration: 1, Type: "tool", Content: "Write the result to disk", Timestamp: now},
		},
	}
	require.NoError(t, store.CreateCheckpoint(context.Background(), &TaskCheckpoint{
		TaskID:        task.ID,
		Iteration:     1,
		LoopState:     ToJSON(checkpointState),
		CanResumeFrom: true,
		ResumeHint:    "reuse persisted tool result",
	}))

	recovery := NewRecovery(store, runner, zap.NewNop())
	require.NoError(t, recovery.RecoverOnStartup(context.Background()))

	completedTask := waitForTaskState(t, store, task.ID, TaskStateCompleted)
	assert.Equal(t, "Recovered from persisted tool memory", completedTask.FinalResult)
	assert.Equal(t, 0, writeTool.CallCount(), "recovered task should reuse persisted tool output instead of re-executing")

	steps, err := store.ListSteps(context.Background(), task.ID)
	require.NoError(t, err)

	var replayed *TaskStep
	for i := range steps {
		if steps[i].ReplayOfStepID == priorStep.ID {
			replayed = &steps[i]
			break
		}
	}
	require.NotNil(t, replayed)
	assert.Equal(t, StepExecutionReplayed, replayed.ExecutionState)
	assert.JSONEq(t, string(priorStep.ToolOutput), string(replayed.ToolOutput))

	effect, err := store.GetEffectByStep(context.Background(), replayed.ID)
	require.NoError(t, err)
	assert.Equal(t, EffectStatusReplayed, effect.Status)
	assert.Equal(t, priorEffect.ID, effect.ReplayOfEffectID)

	events, err := store.ListEvents(context.Background(), task.ID)
	require.NoError(t, err)

	var foundReplayEvent bool
	for _, event := range events {
		if event.Type == EventTypeToolReplayed {
			foundReplayEvent = true
			break
		}
	}
	assert.True(t, foundReplayEvent)
}

func TestRecoveryMarksInterruptedInProgressStep(t *testing.T) {
	responses := []string{
		`{"type":"respond","content":"Recovered after interrupted step"}`,
	}

	runner, store, cleanup := newTestRunner(t, responses)
	defer cleanup()

	now := time.Now()
	task := &Task{
		Title:            "Interrupted task",
		Goal:             "recover after crash",
		State:            TaskStateRunning,
		MaxIterations:    3,
		CurrentIteration: 1,
		StartedAt:        &now,
	}
	require.NoError(t, store.CreateTask(context.Background(), task))

	interruptedStep := &TaskStep{
		TaskID:         task.ID,
		Iteration:      1,
		Type:           StepTypeTool,
		ExecutionState: StepExecutionInProgress,
		Content:        "Run shell command",
		ToolName:       "exec_command",
		SequenceNumber: 1,
		StartedAt:      &now,
	}
	require.NoError(t, store.CreateStep(context.Background(), interruptedStep))

	checkpointState := &LoopState{
		Iteration:   1,
		CurrentPlan: "Recover from crash",
	}
	require.NoError(t, store.CreateCheckpoint(context.Background(), &TaskCheckpoint{
		TaskID:        task.ID,
		Iteration:     1,
		LoopState:     ToJSON(checkpointState),
		CanResumeFrom: true,
		ResumeHint:    "resume after interruption",
	}))

	recovery := NewRecovery(store, runner, zap.NewNop())
	require.NoError(t, recovery.RecoverOnStartup(context.Background()))

	_ = waitForTaskState(t, store, task.ID, TaskStateCompleted)
	reloadedStep, err := store.GetStep(context.Background(), interruptedStep.ID)
	require.NoError(t, err)
	assert.Equal(t, StepExecutionInterrupted, reloadedStep.ExecutionState)
	assert.NotNil(t, reloadedStep.EndedAt)
}

func TestRecoveryMarksInterruptedStartedEffect(t *testing.T) {
	responses := []string{
		`{"type":"respond","content":"Recovered after interrupted effect"}`,
	}

	runner, store, cleanup := newTestRunner(t, responses)
	defer cleanup()

	now := time.Now()
	task := &Task{
		Title:            "Interrupted effect task",
		Goal:             "recover journal",
		State:            TaskStateRunning,
		MaxIterations:    3,
		CurrentIteration: 1,
		StartedAt:        &now,
	}
	require.NoError(t, store.CreateTask(context.Background(), task))

	step := &TaskStep{
		TaskID:          task.ID,
		Iteration:       1,
		Type:            StepTypeTool,
		ExecutionState:  StepExecutionInProgress,
		Content:         "Write file",
		ToolName:        "write_file",
		ToolFingerprint: toolCallFingerprint("write_file", `{"path":"result.txt","content":"done"}`),
		SequenceNumber:  1,
		StartedAt:       &now,
	}
	require.NoError(t, store.CreateStep(context.Background(), step))

	effect := &TaskEffect{
		TaskID:          task.ID,
		StepID:          step.ID,
		ToolName:        step.ToolName,
		ToolFingerprint: step.ToolFingerprint,
		Status:          EffectStatusStarted,
		Input:           json.RawMessage(`{"path":"result.txt","content":"done"}`),
		StartedAt:       now,
	}
	require.NoError(t, store.CreateEffect(context.Background(), effect))

	require.NoError(t, store.CreateCheckpoint(context.Background(), &TaskCheckpoint{
		TaskID:        task.ID,
		Iteration:     1,
		LoopState:     ToJSON(&LoopState{Iteration: 1, CurrentPlan: "Recover effect"}),
		CanResumeFrom: true,
		ResumeHint:    "resume after interrupted effect",
	}))

	recovery := NewRecovery(store, runner, zap.NewNop())
	require.NoError(t, recovery.RecoverOnStartup(context.Background()))

	_ = waitForTaskState(t, store, task.ID, TaskStateCompleted)
	reloadedEffect, err := store.GetEffectByStep(context.Background(), step.ID)
	require.NoError(t, err)
	assert.Equal(t, EffectStatusInterrupted, reloadedEffect.Status)
	assert.NotNil(t, reloadedEffect.FinishedAt)
}
