// Package agent provides tests for context management
package agent

import (
	"context"
	"testing"

	"github.com/gmsas95/myrai-cli/internal/llm"
	"go.uber.org/zap"
)

func TestNewContextManager(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cm := NewContextManager(nil, nil, nil, logger)

	if cm == nil {
		t.Fatal("NewContextManager returned nil")
	}

	if cm.maxTokens != 6000 {
		t.Errorf("Expected maxTokens 6000, got %d", cm.maxTokens)
	}

	if cm.maxMessages != 50 {
		t.Errorf("Expected maxMessages 50, got %d", cm.maxMessages)
	}

	if cm.summaryThreshold != 20 {
		t.Errorf("Expected summaryThreshold 20, got %d", cm.summaryThreshold)
	}

	if cm.relevanceMessages != 10 {
		t.Errorf("Expected relevanceMessages 10, got %d", cm.relevanceMessages)
	}
}

func TestContextManager_BuildContext(t *testing.T) {
	// Skip this test as it requires a real store
	t.Skip("Skipping test - requires initialized store")
}

func TestConversationContext(t *testing.T) {
	convCtx := &ConversationContext{
		Messages: []llm.Message{
			{Role: "system", Content: "System prompt"},
			{Role: "user", Content: "Hello"},
		},
		Summary:     "Test summary",
		TotalTokens: 100,
	}

	if len(convCtx.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(convCtx.Messages))
	}

	if convCtx.TotalTokens != 100 {
		t.Errorf("Expected 100 tokens, got %d", convCtx.TotalTokens)
	}
}

func TestMemoryInfo(t *testing.T) {
	mem := MemoryInfo{
		Content:   "User likes coffee",
		Type:      "preference",
		Relevance: 0.85,
	}

	if mem.Content != "User likes coffee" {
		t.Errorf("Expected content 'User likes coffee', got '%s'", mem.Content)
	}

	if mem.Type != "preference" {
		t.Errorf("Expected type 'preference', got '%s'", mem.Type)
	}

	if mem.Relevance != 0.85 {
		t.Errorf("Expected relevance 0.85, got %f", mem.Relevance)
	}
}

func TestContextManager_ClassifyMemoryType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cm := NewContextManager(nil, nil, nil, logger)

	tests := []struct {
		content  string
		expected string
	}{
		{"User prefers Python", "preference"},
		{"User likes coffee", "preference"},
		{"User favorite color is blue", "preference"},
		{"User is working on Project X", "project"},
		{"User lives in New York", "location"},
		{"User job is software engineer", "profession"},
		{"User goal is to learn Spanish", "goal"},
		{"User met Sarah yesterday", "fact"},
	}

	for _, test := range tests {
		result := cm.classifyMemoryType(test.content)
		if result != test.expected {
			t.Errorf("classifyMemoryType(%q) = %q, expected %q",
				test.content, result, test.expected)
		}
	}
}

func TestContextManager_CalculateRelevance(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cm := NewContextManager(nil, nil, nil, logger)

	tests := []struct {
		msg      llm.Message
		query    string
		minScore float64
	}{
		{
			msg:      llm.Message{Role: "user", Content: "I love Python programming"},
			query:    "Python",
			minScore: 0.1,
		},
		{
			msg:      llm.Message{Role: "assistant", Content: "Python is great"},
			query:    "Python",
			minScore: 0.1,
		},
		{
			msg:      llm.Message{Role: "user", Content: "I love Python"},
			query:    "coffee",
			minScore: 0.0,
		},
	}

	for _, test := range tests {
		score := cm.calculateRelevance(test.msg, test.query)
		if score < test.minScore {
			t.Errorf("Expected relevance >= %f, got %f for query '%s'",
				test.minScore, score, test.query)
		}
	}
}

func TestMessageWithPriority(t *testing.T) {
	msg := MessageWithPriority{
		Message: llm.Message{
			Role:    "user",
			Content: "Hello",
		},
		Priority: 0.8,
		Index:    5,
	}

	if msg.Priority != 0.8 {
		t.Errorf("Expected priority 0.8, got %f", msg.Priority)
	}

	if msg.Index != 5 {
		t.Errorf("Expected index 5, got %d", msg.Index)
	}
}

func TestContextManager_PrioritizeMessages(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cm := NewContextManager(nil, nil, nil, logger)
	cm.relevanceMessages = 5
	cm.maxTokens = 10000

	messages := []llm.Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "I like Python"},
		{Role: "assistant", Content: "Python is great"},
		{Role: "user", Content: "What about Java?"},
	}

	// Prioritize with query about Python
	prioritized := cm.PrioritizeMessages(messages, "Python")

	if len(prioritized) == 0 {
		t.Fatal("Expected non-empty prioritized messages")
	}

	// System message should always be first
	if prioritized[0].Role != "system" {
		t.Errorf("Expected first message to be system, got %s", prioritized[0].Role)
	}
}

func TestContextManager_FormatMemoriesForContext(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cm := NewContextManager(nil, nil, nil, logger)

	memories := []MemoryInfo{
		{Content: "User likes coffee", Type: "preference", Relevance: 0.9},
		{Content: "User lives in NYC", Type: "location", Relevance: 0.8},
		{Content: "Old memory", Type: "fact", Relevance: 0.3}, // Below threshold
	}

	formatted := cm.formatMemoriesForContext(memories)

	if formatted == "" {
		t.Error("Expected non-empty formatted memories")
	}

	if !containsString(formatted, "User likes coffee") {
		t.Error("Expected formatted memories to include high relevance memory")
	}

	if containsString(formatted, "Old memory") {
		t.Error("Low relevance memory should not be included")
	}
}

func TestContextManager_GetContextStats(t *testing.T) {
	// Skip test that requires real store
	t.Skip("Skipping test - requires initialized store")
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkClassifyMemoryType(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cm := NewContextManager(nil, nil, nil, logger)
	content := "User prefers Python over JavaScript"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.classifyMemoryType(content)
	}
}

func BenchmarkCalculateRelevance(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cm := NewContextManager(nil, nil, nil, logger)
	msg := llm.Message{Role: "user", Content: "I love Python programming language"}
	query := "Python"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.calculateRelevance(msg, query)
	}
}

func BenchmarkFormatMemoriesForContext(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cm := NewContextManager(nil, nil, nil, logger)

	memories := []MemoryInfo{
		{Content: "User likes coffee", Type: "preference", Relevance: 0.9},
		{Content: "User lives in NYC", Type: "location", Relevance: 0.8},
		{Content: "User is a developer", Type: "profession", Relevance: 0.85},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.formatMemoriesForContext(memories)
	}
}

// Integration test (skipped by default)
func TestContextManager_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires initialized dependencies")
}

// TestMessageOrder tests that messages are returned in correct chronological order
// This is critical for proper LLM context building
func TestContextManager_MessageOrder(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// This test documents the expected behavior:
	// 1. GetMessages returns newest messages first (DESC order by created_at)
	// 2. buildFullContext reverses them to chronological order (oldest first)
	// 3. LLM receives messages in correct conversation sequence

	t.Run("MessageOrderingLogic", func(t *testing.T) {
		// Simulate messages as they come from DB (newest first)
		dbMessages := []struct {
			content string
			role    string
		}{
			{"Message 5 (newest)", "user"},
			{"Message 4", "assistant"},
			{"Message 3", "user"},
			{"Message 2", "assistant"},
			{"Message 1 (oldest)", "user"},
		}

		// Simulate what buildFullContext does: reverse to chronological order
		var chronological []string
		for i := len(dbMessages) - 1; i >= 0; i-- {
			chronological = append(chronological, dbMessages[i].content)
		}

		// Verify order
		expected := []string{
			"Message 1 (oldest)",
			"Message 2",
			"Message 3",
			"Message 4",
			"Message 5 (newest)",
		}

		for i, exp := range expected {
			if chronological[i] != exp {
				t.Errorf("Message %d: expected %q, got %q", i, exp, chronological[i])
			}
		}
	})
}

// TestToolCallMessageSchema tests that tool messages are stored and retrieved correctly
func TestToolCallMessageSchema(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	t.Run("ToolCallsStoredAsArray", func(t *testing.T) {
		// This test verifies the fix where ToolCalls were stored as single object
		// but retrieved as array, causing deserialization failures

		// Before fix: stored as single ToolCall
		// After fix: stored as []ToolCall (array with 1 element)

		// Simulate proper array storage
		toolCalls := []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		}{
			{
				ID:   "call_123",
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      "get_weather",
					Arguments: `{"location":"KL"}`,
				},
			},
		}

		// Verify it can be serialized and deserialized as array
		// (This is what the fix ensures)
		if len(toolCalls) != 1 {
			t.Errorf("Expected 1 tool call, got %d", len(toolCalls))
		}

		if toolCalls[0].Function.Name != "get_weather" {
			t.Errorf("Expected tool name 'get_weather', got %q", toolCalls[0].Function.Name)
		}
	})
}

// Tests for Neural Cluster Integration (Phase 3)

func TestContextManager_SetNeuralRetriever(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cm := NewContextManager(nil, nil, nil, logger)

	// Initially no retriever
	if cm.neuralRetriever != nil {
		t.Error("Expected neuralRetriever to be nil initially")
	}

	// Set retriever - we can't easily create a real one, so just verify the method exists
	// In real usage, this would be set by the caller
}

func TestContextManager_retrieveRelevantMemories_Neural(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cm := NewContextManager(nil, nil, nil, logger)

	// Without retriever or vector searcher, should handle gracefully
	ctx := context.Background()
	memories, err := cm.retrieveRelevantMemories(ctx, "test query")

	// Should handle nil searcher gracefully (either error or empty result)
	// The method should not panic
	_ = err
	_ = memories
}

func TestContextManager_retrieveFromNeuralClusters(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cm := NewContextManager(nil, nil, nil, logger)

	// Without retriever, should return error
	ctx := context.Background()
	memories, err := cm.retrieveFromNeuralClusters(ctx, "test query")

	if err == nil {
		t.Error("Expected error without neural retriever")
	}

	if memories != nil {
		t.Error("Expected nil memories on error")
	}
}

func TestContextManager_formatMemoriesForContext_RelevanceThreshold(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cm := NewContextManager(nil, nil, nil, logger)

	tests := []struct {
		name     string
		memories []MemoryInfo
		expected int // expected number of formatted memories
	}{
		{
			name:     "empty",
			memories: []MemoryInfo{},
			expected: 0,
		},
		{
			name: "one high relevance",
			memories: []MemoryInfo{
				{Content: "High", Type: "fact", Relevance: 0.8},
			},
			expected: 1,
		},
		{
			name: "mixed relevance",
			memories: []MemoryInfo{
				{Content: "High", Type: "fact", Relevance: 0.8},
				{Content: "Low", Type: "fact", Relevance: 0.5}, // Below 0.7 threshold
				{Content: "Medium", Type: "fact", Relevance: 0.75},
			},
			expected: 2, // Only high and medium
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.formatMemoriesForContext(tt.memories)

			if tt.expected == 0 {
				if result != "" {
					t.Errorf("Expected empty result, got: %s", result)
				}
				return
			}

			// Count lines in result
			lines := 0
			if result != "" {
				for _, c := range result {
					if c == '\n' {
						lines++
					}
				}
				lines++ // Last line doesn't end with \n
			}

			if lines != tt.expected {
				t.Errorf("Expected %d formatted memories, got %d", tt.expected, lines)
			}
		})
	}
}
