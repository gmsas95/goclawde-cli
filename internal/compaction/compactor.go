// Package compaction provides context window management and conversation summarization.
// This keeps conversations within token limits while preserving important information.
package compaction

import (
	"context"
	"fmt"
	"math"

	"github.com/gmsas95/myrai-cli/internal/types"
)

// Compactor manages conversation context window
type Compactor interface {
	// Compact reduces messages to fit within target token count
	Compact(ctx context.Context, messages []types.Message, targetTokens int) ([]types.Message, error)

	// EstimateTokens estimates token count for messages
	EstimateTokens(messages []types.Message) int
}

// Strategy defines how to compact messages
type Strategy string

const (
	// StrategySlidingWindow keeps recent messages, drops old ones
	StrategySlidingWindow Strategy = "sliding_window"

	// StrategySummarize summarizes old messages, keeps recent
	StrategySummarize Strategy = "summarize"

	// StrategySemantic groups related messages, keeps representatives
	StrategySemantic Strategy = "semantic"
)

// SlidingWindowCompactor implements sliding window compaction
type SlidingWindowCompactor struct {
	windowSize int
}

// NewSlidingWindowCompactor creates a sliding window compactor
func NewSlidingWindowCompactor(windowSize int) *SlidingWindowCompactor {
	return &SlidingWindowCompactor{windowSize: windowSize}
}

// Compact keeps only the most recent messages
func (c *SlidingWindowCompactor) Compact(ctx context.Context, messages []types.Message, targetTokens int) ([]types.Message, error) {
	if len(messages) <= c.windowSize {
		return messages, nil
	}

	// Keep system message if present
	var systemMessage *types.Message
	var startIdx int

	if len(messages) > 0 && messages[0].Role == "system" {
		systemMessage = &messages[0]
		startIdx = 1
	}

	// Calculate how many recent messages to keep
	recentCount := c.windowSize
	if systemMessage != nil {
		recentCount-- // Reserve slot for system message
	}

	// Keep system + recent messages
	result := make([]types.Message, 0, recentCount+1)
	if systemMessage != nil {
		result = append(result, *systemMessage)
	}

	// Add recent messages
	start := len(messages) - recentCount
	if start < startIdx {
		start = startIdx
	}
	result = append(result, messages[start:]...)

	return result, nil
}

// EstimateTokens estimates tokens using simple heuristic
func (c *SlidingWindowCompactor) EstimateTokens(messages []types.Message) int {
	total := 0
	for _, msg := range messages {
		total += estimateMessageTokens(msg)
	}
	return total
}

// SummarizingCompactor summarizes old messages to save tokens
type SummarizingCompactor struct {
	summarizer     Summarizer
	preserveRecent int // Number of recent messages to keep verbatim
}

// Summarizer creates summaries of conversation segments
type Summarizer interface {
	// Summarize creates a summary of messages
	Summarize(ctx context.Context, messages []types.Message) (string, error)
}

// NewSummarizingCompactor creates a summarizing compactor
func NewSummarizingCompactor(summarizer Summarizer, preserveRecent int) *SummarizingCompactor {
	return &SummarizingCompactor{
		summarizer:     summarizer,
		preserveRecent: preserveRecent,
	}
}

// Compact summarizes old messages, keeps recent ones
func (c *SummarizingCompactor) Compact(ctx context.Context, messages []types.Message, targetTokens int) ([]types.Message, error) {
	if len(messages) <= c.preserveRecent+2 { // Need at least system + 2 messages to compact
		return messages, nil
	}

	// Identify parts
	var systemMessage *types.Message
	var toSummarize []types.Message
	var toKeep []types.Message

	for i, msg := range messages {
		if i == 0 && msg.Role == "system" {
			systemMessage = &msg
			continue
		}

		// Keep recent messages verbatim
		if i >= len(messages)-c.preserveRecent {
			toKeep = append(toKeep, msg)
		} else {
			toSummarize = append(toSummarize, msg)
		}
	}

	if len(toSummarize) == 0 {
		return messages, nil
	}

	// Generate summary
	summary, err := c.summarizer.Summarize(ctx, toSummarize)
	if err != nil {
		return nil, fmt.Errorf("failed to summarize: %w", err)
	}

	// Build compacted conversation
	result := make([]types.Message, 0, len(toKeep)+2)

	if systemMessage != nil {
		result = append(result, *systemMessage)
	}

	// Add summary as system message
	summaryMsg := types.Message{
		Role:    "system",
		Content: []types.ContentBlock{types.TextBlock{Text: fmt.Sprintf("Previous conversation summary: %s", summary)}},
	}
	result = append(result, summaryMsg)

	// Add recent messages
	result = append(result, toKeep...)

	return result, nil
}

// EstimateTokens estimates tokens
func (c *SummarizingCompactor) EstimateTokens(messages []types.Message) int {
	total := 0
	for _, msg := range messages {
		total += estimateMessageTokens(msg)
	}
	return total
}

// SimpleSummarizer uses a simple rule-based approach
// (In production, this would call an LLM)
type SimpleSummarizer struct{}

// NewSimpleSummarizer creates a new simple summarizer
func NewSimpleSummarizer() *SimpleSummarizer {
	return &SimpleSummarizer{}
}

// Summarize creates a basic summary
func (s *SimpleSummarizer) Summarize(ctx context.Context, messages []types.Message) (string, error) {
	// Count message types
	userCount := 0
	assistantCount := 0
	toolCallCount := 0

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			userCount++
		case "assistant":
			assistantCount++
			if msg.HasToolCalls() {
				toolCallCount += len(msg.GetToolCalls())
			}
		}
	}

	summary := fmt.Sprintf(
		"Conversation had %d user messages, %d assistant responses",
		userCount, assistantCount,
	)

	if toolCallCount > 0 {
		summary += fmt.Sprintf(", including %d tool calls", toolCallCount)
	}

	return summary, nil
}

// LLMSummarizer uses an LLM to create summaries
type LLMSummarizer struct {
	client LLMClient
}

// LLMClient interface for summarization
type LLMClient interface {
	Complete(ctx context.Context, messages []types.Message) (*types.Message, error)
}

// NewLLMSummarizer creates an LLM-based summarizer
func NewLLMSummarizer(client LLMClient) *LLMSummarizer {
	return &LLMSummarizer{client: client}
}

// Summarize uses LLM to create summary
func (s *LLMSummarizer) Summarize(ctx context.Context, messages []types.Message) (string, error) {
	// Build conversation text
	var conversation string
	for _, msg := range messages {
		role := msg.Role
		content := msg.GetTextContent()
		if content != "" {
			conversation += fmt.Sprintf("%s: %s\n", role, content)
		}
	}

	// Create summary prompt
	summaryPrompt := []types.Message{
		{
			Role: "system",
			Content: []types.ContentBlock{
				types.TextBlock{Text: "Summarize the following conversation concisely, preserving key information, decisions, and context needed to continue."},
			},
		},
		{
			Role: "user",
			Content: []types.ContentBlock{
				types.TextBlock{Text: conversation},
			},
		},
	}

	response, err := s.client.Complete(ctx, summaryPrompt)
	if err != nil {
		return "", fmt.Errorf("LLM summarization failed: %w", err)
	}

	return response.GetTextContent(), nil
}

// SmartCompactor intelligently manages context window
type SmartCompactor struct {
	summarizer    Summarizer
	maxTokens     int
	preserveTurns int
	minMessages   int
}

// NewSmartCompactor creates an intelligent compactor
func NewSmartCompactor(summarizer Summarizer, maxTokens, preserveTurns, minMessages int) *SmartCompactor {
	return &SmartCompactor{
		summarizer:    summarizer,
		maxTokens:     maxTokens,
		preserveTurns: preserveTurns,
		minMessages:   minMessages,
	}
}

// Compact intelligently manages the conversation
func (c *SmartCompactor) Compact(ctx context.Context, messages []types.Message, targetTokens int) ([]types.Message, error) {
	// If under limit, return as-is
	currentTokens := c.EstimateTokens(messages)
	if currentTokens <= targetTokens {
		return messages, nil
	}

	// If too few messages, can't compact
	if len(messages) <= c.minMessages {
		return messages, fmt.Errorf("cannot compact: minimum message count reached (%d)", len(messages))
	}

	// Strategy: Keep recent turns, summarize older ones
	// A "turn" is user + assistant (possibly with tool calls)

	// Find the cutoff point for recent turns
	recentCount := 0
	turnCount := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			turnCount++
			if turnCount >= c.preserveTurns {
				recentCount = len(messages) - i
				break
			}
		}
	}

	if recentCount == 0 {
		recentCount = c.preserveTurns * 2 // Fallback
	}

	// Split into old and recent
	var systemMsg *types.Message
	var oldMessages []types.Message
	var recentMessages []types.Message

	for i, msg := range messages {
		if i == 0 && msg.Role == "system" {
			systemMsg = &msg
			continue
		}

		if i >= len(messages)-recentCount {
			recentMessages = append(recentMessages, msg)
		} else {
			oldMessages = append(oldMessages, msg)
		}
	}

	// Summarize old messages if any
	var summary string
	if len(oldMessages) > 0 {
		var err error
		summary, err = c.summarizer.Summarize(ctx, oldMessages)
		if err != nil {
			// Fallback to sliding window on summarization failure
			return c.fallbackToSlidingWindow(ctx, messages, targetTokens)
		}
	}

	// Build result
	result := make([]types.Message, 0, len(recentMessages)+2)

	if systemMsg != nil {
		result = append(result, *systemMsg)
	}

	if summary != "" {
		summaryMsg := types.Message{
			Role: "system",
			Content: []types.ContentBlock{
				types.TextBlock{Text: fmt.Sprintf("Context from earlier in conversation: %s", summary)},
			},
		}
		result = append(result, summaryMsg)
	}

	result = append(result, recentMessages...)

	return result, nil
}

// fallbackToSlidingWindow falls back to simple windowing
func (c *SmartCompactor) fallbackToSlidingWindow(ctx context.Context, messages []types.Message, targetTokens int) ([]types.Message, error) {
	compactor := NewSlidingWindowCompactor(c.preserveTurns * 2)
	return compactor.Compact(ctx, messages, targetTokens)
}

// EstimateTokens estimates total tokens
func (c *SmartCompactor) EstimateTokens(messages []types.Message) int {
	total := 0
	for _, msg := range messages {
		total += estimateMessageTokens(msg)
	}
	return total
}

// Helper function to estimate tokens for a message
// Uses a simple heuristic: ~4 characters per token
func estimateMessageTokens(msg types.Message) int {
	tokenCount := 0

	for _, block := range msg.Content {
		switch b := block.(type) {
		case types.TextBlock:
			// Rough estimate: 4 characters per token
			tokenCount += int(math.Ceil(float64(len(b.Text)) / 4.0))
		case types.ToolCallBlock:
			// Tool calls: name + arguments
			tokenCount += 10 // Base overhead
			tokenCount += int(math.Ceil(float64(len(b.Name)) / 4.0))
			tokenCount += int(math.Ceil(float64(len(b.Arguments)) / 4.0))
		case types.ToolResultBlock:
			// Tool results: content
			tokenCount += 10 // Base overhead
			for _, contentBlock := range b.Content {
				if text, ok := contentBlock.(types.TextBlock); ok {
					tokenCount += int(math.Ceil(float64(len(text.Text)) / 4.0))
				}
			}
		case types.ThinkingBlock:
			tokenCount += int(math.Ceil(float64(len(b.Thinking)) / 4.0))
		}
	}

	// Add overhead per message
	tokenCount += 4 // Role and format overhead

	return tokenCount
}

// ContextManager manages conversation context with automatic compaction
type ContextManager struct {
	compactor   Compactor
	maxTokens   int
	maxMessages int
}

// NewContextManager creates a context manager
func NewContextManager(compactor Compactor, maxTokens, maxMessages int) *ContextManager {
	return &ContextManager{
		compactor:   compactor,
		maxTokens:   maxTokens,
		maxMessages: maxMessages,
	}
}

// PrepareContext prepares messages for LLM, compacting if necessary
func (m *ContextManager) PrepareContext(ctx context.Context, messages []types.Message) ([]types.Message, error) {
	// Check if compaction needed
	currentTokens := m.compactor.EstimateTokens(messages)

	if currentTokens <= m.maxTokens && len(messages) <= m.maxMessages {
		return messages, nil
	}

	// Need to compact
	return m.compactor.Compact(ctx, messages, m.maxTokens)
}

// ShouldCompact returns true if messages exceed limits
func (m *ContextManager) ShouldCompact(messages []types.Message) bool {
	tokens := m.compactor.EstimateTokens(messages)
	return tokens > m.maxTokens || len(messages) > m.maxMessages
}
