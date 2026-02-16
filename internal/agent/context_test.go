// Package agent provides tests for context management
package agent

import (
	"testing"

	"github.com/gmsas95/goclawde-cli/internal/llm"
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
		msg   llm.Message
		query string
		minScore float64
	}{
		{
			msg:   llm.Message{Role: "user", Content: "I love Python programming"},
			query: "Python",
			minScore: 0.1,
		},
		{
			msg:   llm.Message{Role: "assistant", Content: "Python is great"},
			query: "Python",
			minScore: 0.1,
		},
		{
			msg:   llm.Message{Role: "user", Content: "I love Python"},
			query: "coffee",
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
