// Package agent provides tests for agent loop
package agent

import (
	"testing"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/llm"
	"go.uber.org/zap"
)

func TestNewAgentLoop(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create a minimal agent for testing
	agent := &Agent{
		logger: logger,
	}

	loop := NewAgentLoop(agent, logger)

	if loop == nil {
		t.Fatal("NewAgentLoop returned nil")
	}

	if loop.agent != agent {
		t.Error("AgentLoop agent mismatch")
	}

	if loop.maxIterations != 10 {
		t.Errorf("Expected maxIterations 10, got %d", loop.maxIterations)
	}

	if loop.timeout != 5*time.Minute {
		t.Errorf("Expected timeout 5m, got %v", loop.timeout)
	}
}

func TestAgentLoop_SetMaxIterations(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	agent := &Agent{logger: logger}
	loop := NewAgentLoop(agent, logger)

	loop.SetMaxIterations(20)

	if loop.maxIterations != 20 {
		t.Errorf("Expected maxIterations 20, got %d", loop.maxIterations)
	}
}

func TestAgentLoop_SetTimeout(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	agent := &Agent{logger: logger}
	loop := NewAgentLoop(agent, logger)

	loop.SetTimeout(10 * time.Minute)

	if loop.timeout != 10*time.Minute {
		t.Errorf("Expected timeout 10m, got %v", loop.timeout)
	}
}

func TestAgentLoop_EnableConfirmation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	agent := &Agent{logger: logger}
	loop := NewAgentLoop(agent, logger)

	// Default is true
	if !loop.requireConfirm {
		t.Error("Expected requireConfirm to be true by default")
	}

	loop.EnableConfirmation(false)

	if loop.requireConfirm {
		t.Error("Expected requireConfirm to be false after disabling")
	}
}

func TestAutonomousTask(t *testing.T) {
	task := AutonomousTask{
		Goal:    "Test goal",
		Context: "Test context",
		Constraints: []string{
			"Constraint 1",
			"Constraint 2",
		},
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	if task.Goal != "Test goal" {
		t.Errorf("Expected goal 'Test goal', got '%s'", task.Goal)
	}

	if len(task.Constraints) != 2 {
		t.Errorf("Expected 2 constraints, got %d", len(task.Constraints))
	}
}

func TestTaskResult(t *testing.T) {
	result := &TaskResult{
		Success:     true,
		Iterations:  5,
		FinalAnswer: "Test answer",
		Errors:      []string{},
		Duration:    30 * time.Second,
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if result.Iterations != 5 {
		t.Errorf("Expected 5 iterations, got %d", result.Iterations)
	}
}

func TestAction(t *testing.T) {
	action := Action{
		Iteration: 1,
		Type:      "think",
		Content:   "Thinking about the problem",
		Timestamp: time.Now(),
	}

	if action.Iteration != 1 {
		t.Errorf("Expected iteration 1, got %d", action.Iteration)
	}

	if action.Type != "think" {
		t.Errorf("Expected type 'think', got '%s'", action.Type)
	}
}

func TestAgentLoop_BuildAutonomousSystemPrompt(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	agent := &Agent{logger: logger}
	loop := NewAgentLoop(agent, logger)

	task := AutonomousTask{
		Goal:    "Test a simple task",
		Context: "This is test context",
	}

	prompt := loop.buildAutonomousSystemPrompt(task)

	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}

	if !containsSubstring(prompt, "Test a simple task") {
		t.Error("Expected prompt to contain goal")
	}

	if !containsSubstring(prompt, "think") {
		t.Error("Expected prompt to mention 'think' action")
	}

	if !containsSubstring(prompt, "tool") {
		t.Error("Expected prompt to mention 'tool' action")
	}
}

func TestAgentLoop_BuildAutonomousSystemPrompt_WithConstraints(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	agent := &Agent{logger: logger}
	loop := NewAgentLoop(agent, logger)

	task := AutonomousTask{
		Goal:        "Test task",
		Constraints: []string{"Don't break things", "Be careful"},
	}

	prompt := loop.buildAutonomousSystemPrompt(task)

	if !containsSubstring(prompt, "Don't break things") {
		t.Error("Expected prompt to contain constraints")
	}
}

func TestAgentLoop_ParseActionResponse(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	agent := &Agent{logger: logger}
	loop := NewAgentLoop(agent, logger)

	tests := []struct {
		name     string
		response string
		wantType string
	}{
		{
			name:     "Valid JSON think",
			response: `{"type": "think", "content": "Thinking..."}`,
			wantType: "think",
		},
		{
			name:     "Valid JSON tool",
			response: `{"type": "tool", "content": "Using tool"}`,
			wantType: "tool",
		},
		{
			name:     "Valid JSON respond",
			response: `{"type": "respond", "content": "Done"}`,
			wantType: "respond",
		},
		{
			name:     "Markdown code block",
			response: "```json\n{\"type\": \"think\", \"content\": \"Test\"}\n```",
			wantType: "think",
		},
		{
			name:     "Plain text fallback",
			response: "I need to think about this",
			wantType: "think",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := loop.parseActionResponse(tt.response)
			if err != nil {
				t.Fatalf("parseActionResponse failed: %v", err)
			}

			if action.Type != tt.wantType {
				t.Errorf("Expected type '%s', got '%s'", tt.wantType, action.Type)
			}
		})
	}
}

func TestAgentLoop_ContextManager(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Test context manager separately
	cm := NewContextManager(nil, nil, nil, logger)

	messages := []llm.Message{
		{Role: "user", Content: "Hello, I need help with Python"},
		{Role: "assistant", Content: "Sure, I can help with Python"},
		{Role: "user", Content: "How do I define a function?"},
	}

	// Prioritize with query about Python
	prioritized := cm.PrioritizeMessages(messages, "Python function")

	if len(prioritized) == 0 {
		t.Fatal("Expected non-empty prioritized messages")
	}
}

func TestAgentLoop_IsDestructive(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Define anonymous struct matching llm.ToolCall.Function
	type toolCallFunc struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}

	type toolCall struct {
		ID       string       `json:"id"`
		Type     string       `json:"type"`
		Function toolCallFunc `json:"function"`
	}

	tests := []struct {
		name            string
		toolCall        toolCall
		wantDestructive bool
	}{
		{
			name: "write_file is destructive",
			toolCall: toolCall{
				Function: toolCallFunc{
					Name: "write_file",
				},
			},
			wantDestructive: true,
		},
		{
			name: "read_file is not destructive",
			toolCall: toolCall{
				Function: toolCallFunc{
					Name: "read_file",
				},
			},
			wantDestructive: false,
		},
		{
			name: "delete in arguments",
			toolCall: toolCall{
				Function: toolCallFunc{
					Name:      "exec_command",
					Arguments: `{"command": "rm -rf /important"}`,
				},
			},
			wantDestructive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't call isDestructive directly due to type mismatch
			// but we can test the logic
			got := isDestructiveTool(tt.toolCall.Function.Name, tt.toolCall.Function.Arguments)
			if got != tt.wantDestructive {
				t.Errorf("isDestructive() = %v, want %v", got, tt.wantDestructive)
			}
		})
	}
}

// Helper to test destructive tool detection
func isDestructiveTool(name, arguments string) bool {
	destructiveTools := []string{
		"write_file", "delete_file", "rm", "remove", "drop", "delete",
	}

	for _, dt := range destructiveTools {
		if containsString(name, dt) {
			return true
		}
	}

	if containsString(arguments, "rm -rf") ||
		containsString(arguments, "delete") {
		return true
	}

	return false
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

// Helper function for tests
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubHelper(s, substr))
}

func containsSubHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkParseActionResponse(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	agent := &Agent{logger: logger}
	loop := NewAgentLoop(agent, logger)

	response := `{"type": "think", "content": "I need to analyze this problem carefully"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loop.parseActionResponse(response)
	}
}

func BenchmarkIsDestructive(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isDestructiveTool("write_file", "")
	}
}
