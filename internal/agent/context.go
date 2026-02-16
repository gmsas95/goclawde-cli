// Package agent implements context management for conversations
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/gmsas95/myrai-cli/internal/vector"
	"go.uber.org/zap"
)

// ContextManager manages conversation context with smart windowing
// Inspired by Claude Code's approach to long conversations
type ContextManager struct {
	store          *store.Store
	vectorSearcher *vector.Searcher
	llmClient      *llm.Client
	logger         *zap.Logger

	// Configuration
	maxTokens         int // Max tokens for context window
	maxMessages       int // Max full messages to keep
	summaryThreshold  int // Messages before summarization kicks in
	relevanceMessages int // Number of recent messages to always keep
}

// NewContextManager creates a new context manager
func NewContextManager(store *store.Store, vectorSearcher *vector.Searcher, llmClient *llm.Client, logger *zap.Logger) *ContextManager {
	return &ContextManager{
		store:             store,
		vectorSearcher:    vectorSearcher,
		llmClient:         llmClient,
		logger:            logger,
		maxTokens:         6000, // Leave room for response
		maxMessages:       50,
		summaryThreshold:  20,
		relevanceMessages: 10,
	}
}

// ConversationContext represents the built context for a conversation
type ConversationContext struct {
	Messages       []llm.Message
	Summary        string
	RelevantMemories []MemoryInfo
	TotalTokens    int
}

// MemoryInfo represents a relevant memory
type MemoryInfo struct {
	Content    string
	Type       string
	Relevance  float64
}

// BuildContext builds optimized context for a conversation
func (cm *ContextManager) BuildContext(ctx context.Context, convID string, systemPrompt string, currentQuery string) (*ConversationContext, error) {
	result := &ConversationContext{
		Messages: make([]llm.Message, 0),
	}

	// Start with system prompt
	sysMsg := llm.Message{
		Role:    "system",
		Content: systemPrompt,
	}
	result.Messages = append(result.Messages, sysMsg)
	result.TotalTokens += llm.CountTokens(systemPrompt)

	// Get message count to decide strategy
	msgCount, err := cm.store.GetMessageCount(convID)
	if err != nil {
		cm.logger.Warn("Failed to get message count", zap.Error(err))
		msgCount = 0
	}

	// Retrieve relevant memories based on current query
	if currentQuery != "" && cm.vectorSearcher != nil && cm.vectorSearcher.IsEnabled() {
		memories, err := cm.retrieveRelevantMemories(ctx, currentQuery)
		if err == nil && len(memories) > 0 {
			result.RelevantMemories = memories
			// Inject memories into system prompt or as context message
			memoryContext := cm.formatMemoriesForContext(memories)
			if memoryContext != "" {
				memoryMsg := llm.Message{
					Role:    "system",
					Content: "Relevant context from memory:\n" + memoryContext,
				}
				result.Messages = append(result.Messages, memoryMsg)
				result.TotalTokens += llm.CountTokens(memoryMsg.Content)
			}
		}
	}

	// Strategy based on conversation length
	if msgCount > int64(cm.summaryThreshold) {
		// Long conversation - use summarization
		return cm.buildSummarizedContext(ctx, convID, result)
	}

	// Short conversation - use all messages
	return cm.buildFullContext(ctx, convID, result)
}

// buildFullContext builds context with all recent messages
func (cm *ContextManager) buildFullContext(ctx context.Context, convID string, result *ConversationContext) (*ConversationContext, error) {
	storeMsgs, err := cm.store.GetMessages(convID, cm.maxMessages, 0)
	if err != nil {
		cm.logger.Warn("Failed to get messages", zap.Error(err))
		return result, nil
	}

	for _, msg := range storeMsgs {
		lmMsg := llm.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}

		// Handle tool calls
		if len(msg.ToolCalls) > 0 {
			var tcs []llm.ToolCall
			if err := json.Unmarshal(msg.ToolCalls, &tcs); err == nil {
				lmMsg.ToolCalls = tcs
			}
		}

		result.Messages = append(result.Messages, lmMsg)
		result.TotalTokens += llm.CountTokens(msg.Content)
	}

	return result, nil
}

// buildSummarizedContext builds context with summary + recent messages
func (cm *ContextManager) buildSummarizedContext(ctx context.Context, convID string, result *ConversationContext) (*ConversationContext, error) {
	// Get conversation summary (or generate one)
	summary, err := cm.getOrCreateSummary(ctx, convID)
	if err != nil {
		cm.logger.Warn("Failed to get summary", zap.Error(err))
		// Fall back to full context
		return cm.buildFullContext(ctx, convID, result)
	}

	result.Summary = summary

	// Add summary as a system message
	if summary != "" {
		summaryMsg := llm.Message{
			Role:    "system",
			Content: fmt.Sprintf("Previous conversation summary:\n%s", summary),
		}
		result.Messages = append(result.Messages, summaryMsg)
		result.TotalTokens += llm.CountTokens(summaryMsg.Content)
	}

	// Get recent messages (always keep last N)
	recentMsgs, err := cm.store.GetMessages(convID, cm.relevanceMessages, 0)
	if err != nil {
		cm.logger.Warn("Failed to get recent messages", zap.Error(err))
		return result, nil
	}

	// Add recent messages in order
	for _, msg := range recentMsgs {
		lmMsg := llm.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}

		if len(msg.ToolCalls) > 0 {
			var tcs []llm.ToolCall
			if err := json.Unmarshal(msg.ToolCalls, &tcs); err == nil {
				lmMsg.ToolCalls = tcs
			}
		}

		result.Messages = append(result.Messages, lmMsg)
		result.TotalTokens += llm.CountTokens(msg.Content)

		// Check token limit
		if result.TotalTokens > cm.maxTokens {
			break
		}
	}

	return result, nil
}

// retrieveRelevantMemories searches for memories relevant to the query
func (cm *ContextManager) retrieveRelevantMemories(ctx context.Context, query string) ([]MemoryInfo, error) {
	results, err := cm.vectorSearcher.Search(query, 5)
	if err != nil {
		return nil, err
	}

	memories := make([]MemoryInfo, 0, len(results))
	for _, r := range results {
		memories = append(memories, MemoryInfo{
			Content:   r.Content,
			Type:      r.Type,
			Relevance: r.Similarity,
		})
	}

	return memories, nil
}

// formatMemoriesForContext formats memories for inclusion in context
func (cm *ContextManager) formatMemoriesForContext(memories []MemoryInfo) string {
	if len(memories) == 0 {
		return ""
	}

	var parts []string
	for _, m := range memories {
		if m.Relevance > 0.7 { // Only high relevance
			parts = append(parts, fmt.Sprintf("- [%s] %s", m.Type, m.Content))
		}
	}

	return strings.Join(parts, "\n")
}

// getOrCreateSummary gets existing summary or creates one
func (cm *ContextManager) getOrCreateSummary(ctx context.Context, convID string) (string, error) {
	// Try to get existing summary from store
	summary, err := cm.getExistingSummary(convID)
	if err == nil && summary != "" {
		return summary, nil
	}

	// Generate new summary
	return cm.generateSummary(ctx, convID)
}

// getExistingSummary retrieves an existing summary if fresh enough
func (cm *ContextManager) getExistingSummary(convID string) (string, error) {
	// Look for recent summary in BadgerDB or metadata
	// For now, return empty to regenerate
	return "", fmt.Errorf("no existing summary")
}

// generateSummary creates a summary of the conversation
func (cm *ContextManager) generateSummary(ctx context.Context, convID string) (string, error) {
	// Get older messages (skip recent ones that we keep in full)
	offset := cm.relevanceMessages
	limit := 30 // Messages to summarize

	storeMsgs, err := cm.store.GetMessages(convID, limit, offset)
	if err != nil {
		return "", err
	}

	if len(storeMsgs) == 0 {
		return "", nil
	}

	// Build conversation text for summarization
	var convoParts []string
	for _, msg := range storeMsgs {
		if msg.Role == "user" || msg.Role == "assistant" {
			convoParts = append(convoParts, fmt.Sprintf("%s: %s", msg.Role, msg.Content))
		}
	}

	convoText := strings.Join(convoParts, "\n")
	if len(convoText) > 4000 {
		convoText = convoText[:4000] + "..."
	}

	// Use LLM to summarize
	prompt := fmt.Sprintf(`Summarize the following conversation concisely. Focus on:
- Key facts and information shared
- Decisions made
- User preferences revealed
- Tasks or action items

Conversation:
%s

Provide a brief summary (2-4 sentences):`, convoText)

	summary, err := cm.llmClient.SimpleChat(ctx, 
		"You are a helpful assistant that summarizes conversations accurately.",
		prompt)
	if err != nil {
		return "", err
	}

	// Store summary for future use
	cm.storeSummary(convID, summary)

	return summary, nil
}

// storeSummary saves a conversation summary
func (cm *ContextManager) storeSummary(convID string, summary string) {
	// Store in BadgerDB with TTL or as conversation metadata
	// This is a placeholder - implement based on your storage needs
}

// ExtractAndStoreMemories extracts memories from a conversation turn
func (cm *ContextManager) ExtractAndStoreMemories(ctx context.Context, convID string, userMsg, assistantMsg string) error {
	// Skip if no content
	if userMsg == "" || assistantMsg == "" {
		return nil
	}

	// Use LLM to extract potential memories
	prompt := fmt.Sprintf(`Analyze this conversation and extract any facts, preferences, or important information about the user that should be remembered for future conversations.

User: %s
Assistant: %s

Extract 0-3 key facts/preferences. Format each as a simple statement. If nothing important to remember, respond with "NONE".

Examples of good extractions:
- "User prefers Python over JavaScript"
- "User is working on a project called GoClawde"
- "User lives in Kuala Lumpur"

Extractions:`, userMsg, assistantMsg)

	extraction, err := cm.llmClient.SimpleChat(ctx,
		"You extract important facts and preferences from conversations.",
		prompt)
	if err != nil {
		return err
	}

	// Parse and store memories
	if extraction == "" || strings.TrimSpace(extraction) == "NONE" {
		return nil
	}

	lines := strings.Split(extraction, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "NONE" {
			continue
		}

		// Clean up the extraction
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimPrefix(line, ".")
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Determine memory type
		memType := cm.classifyMemoryType(line)

		// Create memory
		mem := &store.Memory{
			Type:    memType,
			Content: line,
			Source:  convID,
		}

		if err := cm.store.CreateMemory(mem); err != nil {
			cm.logger.Warn("Failed to create memory", zap.Error(err))
			continue
		}

		// Index for vector search
		if cm.vectorSearcher != nil && cm.vectorSearcher.IsEnabled() {
			if err := cm.vectorSearcher.IndexMemory(mem.ID, mem.Content); err != nil {
				cm.logger.Warn("Failed to index memory", zap.Error(err))
			}
		}

		cm.logger.Debug("Extracted and stored memory",
			zap.String("content", line),
			zap.String("type", memType),
		)
	}

	return nil
}

// classifyMemoryType classifies the type of memory
func (cm *ContextManager) classifyMemoryType(content string) string {
	content = strings.ToLower(content)

	if strings.Contains(content, "prefer") || strings.Contains(content, "like") || strings.Contains(content, "favorite") {
		return "preference"
	}
	if strings.Contains(content, "working on") || strings.Contains(content, "project") || strings.Contains(content, "building") {
		return "project"
	}
	if strings.Contains(content, "live") || strings.Contains(content, "from") || strings.Contains(content, "location") {
		return "location"
	}
	if strings.Contains(content, "job") || strings.Contains(content, "work as") || strings.Contains(content, "profession") {
		return "profession"
	}
	if strings.Contains(content, "goal") || strings.Contains(content, "want to") || strings.Contains(content, "plan") {
		return "goal"
	}

	return "fact"
}

// MessageWithPriority represents a message with relevance priority
type MessageWithPriority struct {
	Message  llm.Message
	Priority float64
	Index    int
}

// PrioritizeMessages sorts messages by relevance to current query
func (cm *ContextManager) PrioritizeMessages(messages []llm.Message, query string) []llm.Message {
	if query == "" || len(messages) <= cm.relevanceMessages {
		return messages
	}

	// Score messages by relevance
	scored := make([]MessageWithPriority, len(messages))
	for i, msg := range messages {
		scored[i] = MessageWithPriority{
			Message:  msg,
			Priority: cm.calculateRelevance(msg, query),
			Index:    i,
		}
	}

	// Sort by priority (higher first)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Priority > scored[j].Priority
	})

	// Take top messages up to token limit
	result := make([]llm.Message, 0, len(messages))
	totalTokens := 0

	for _, s := range scored {
		tokens := llm.CountTokens(s.Message.Content)
		if totalTokens+tokens > cm.maxTokens && len(result) >= cm.relevanceMessages {
			break
		}
		result = append(result, s.Message)
		totalTokens += tokens
	}

	// Re-sort by original index to maintain chronological order
	sort.Slice(result, func(i, j int) bool {
		return scored[i].Index < scored[j].Index
	})

	return result
}

// calculateRelevance calculates relevance score between message and query
func (cm *ContextManager) calculateRelevance(msg llm.Message, query string) float64 {
	// Simple keyword overlap scoring
	// In production, use embeddings for better relevance
	queryWords := strings.Fields(strings.ToLower(query))
	contentWords := strings.Fields(strings.ToLower(msg.Content))

	if len(queryWords) == 0 {
		return 0.5
	}

	contentWordSet := make(map[string]bool)
	for _, w := range contentWords {
		contentWordSet[w] = true
	}

	matches := 0
	for _, qw := range queryWords {
		if len(qw) > 3 && contentWordSet[qw] {
			matches++
		}
	}

	score := float64(matches) / float64(len(queryWords))

	// Boost recent messages and user messages
	if msg.Role == "user" {
		score += 0.1
	}

	return score
}

// CompactContext compacts context when it gets too large
func (cm *ContextManager) CompactContext(ctx context.Context, convID string) error {
	msgCount, err := cm.store.GetMessageCount(convID)
	if err != nil {
		return err
	}

	// Only compact if we have many messages
	if msgCount < int64(cm.summaryThreshold*2) {
		return nil
	}

	cm.logger.Info("Compacting conversation context",
		zap.String("conversation_id", convID),
		zap.Int64("message_count", msgCount),
	)

	// Generate summary of older messages
	_, err = cm.generateSummary(ctx, convID)
	if err != nil {
		return err
	}

	return nil
}

// GetContextStats returns statistics about context usage
func (cm *ContextManager) GetContextStats(convID string) (map[string]interface{}, error) {
	msgCount, err := cm.store.GetMessageCount(convID)
	if err != nil {
		return nil, err
	}

	// Get recent messages to estimate tokens
	recentMsgs, _ := cm.store.GetMessages(convID, 20, 0)
	totalTokens := 0
	for _, msg := range recentMsgs {
		totalTokens += llm.CountTokens(msg.Content)
	}

	return map[string]interface{}{
		"message_count":      msgCount,
		"recent_tokens":      totalTokens,
		"max_tokens":         cm.maxTokens,
		"summary_threshold":  cm.summaryThreshold,
		"relevance_messages": cm.relevanceMessages,
	}, nil
}
