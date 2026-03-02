// Package conversation provides high-level conversation management
// including context window management, compaction, and lifecycle.
package conversation

import (
	"context"
	"fmt"
	"time"

	"github.com/gmsas95/myrai-cli/internal/compaction"
	"github.com/gmsas95/myrai-cli/internal/pipeline"
	"github.com/gmsas95/myrai-cli/internal/storev2"
	"github.com/gmsas95/myrai-cli/internal/types"
)

// Manager handles conversation lifecycle and context management
type Manager struct {
	store       *storev2.ConversationStore
	pipeline    *pipeline.Pipeline
	contextMgr  *compaction.ContextManager
	maxMessages int
	maxTokens   int
}

// ManagerOptions configures the conversation manager
type ManagerOptions struct {
	Store       *storev2.ConversationStore
	Pipeline    *pipeline.Pipeline
	Compactor   compaction.Compactor
	MaxMessages int
	MaxTokens   int
}

// NewManager creates a new conversation manager
func NewManager(opts ManagerOptions) *Manager {
	ctxMgr := compaction.NewContextManager(
		opts.Compactor,
		opts.MaxTokens,
		opts.MaxMessages,
	)

	return &Manager{
		store:       opts.Store,
		pipeline:    opts.Pipeline,
		contextMgr:  ctxMgr,
		maxMessages: opts.MaxMessages,
		maxTokens:   opts.MaxTokens,
	}
}

// CreateConversation creates a new conversation for a user
func (m *Manager) CreateConversation(ctx context.Context, userID string) (*types.Conversation, error) {
	return m.store.CreateConversation(ctx, userID)
}

// GetConversation retrieves a conversation by ID
func (m *Manager) GetConversation(ctx context.Context, id string) (*types.Conversation, error) {
	return m.store.GetConversation(ctx, id)
}

// GetOrCreateConversation gets existing conversation or creates new one
func (m *Manager) GetOrCreateConversation(ctx context.Context, id, userID string) (*types.Conversation, error) {
	conv, err := m.store.GetConversation(ctx, id)
	if err != nil {
		// Create new conversation
		return m.store.CreateConversation(ctx, userID)
	}
	return conv, nil
}

// AddMessage adds a message to a conversation
func (m *Manager) AddMessage(ctx context.Context, conversationID string, msg *types.Message) error {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	return m.store.SaveMessage(ctx, conversationID, msg)
}

// AddUserMessage adds a user message to a conversation
func (m *Manager) AddUserMessage(ctx context.Context, conversationID, text string) (*types.Message, error) {
	msg := &types.Message{
		Role:      "user",
		Content:   []types.ContentBlock{types.TextBlock{Text: text}},
		Timestamp: time.Now(),
	}

	if err := m.store.SaveMessage(ctx, conversationID, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// AddAssistantMessage adds an assistant message to a conversation
func (m *Manager) AddAssistantMessage(ctx context.Context, conversationID string, msg *types.Message) error {
	msg.Role = "assistant"
	msg.Timestamp = time.Now()
	return m.store.SaveMessage(ctx, conversationID, msg)
}

// GetContext retrieves and prepares conversation context for LLM
func (m *Manager) GetContext(ctx context.Context, conversationID string) ([]types.Message, error) {
	// Get all messages
	messages, err := m.store.GetMessages(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Prepare context (compact if needed)
	prepared, err := m.contextMgr.PrepareContext(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare context: %w", err)
	}

	// Apply sanitization pipeline
	sanitized, err := m.pipeline.Process(prepared)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize messages: %w", err)
	}

	return sanitized, nil
}

// GetRecentContext gets the most recent n messages, prepared for LLM
func (m *Manager) GetRecentContext(ctx context.Context, conversationID string, limit int) ([]types.Message, error) {
	// Get recent messages
	messages, err := m.store.GetRecentMessages(ctx, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent messages: %w", err)
	}

	// Apply sanitization pipeline
	sanitized, err := m.pipeline.Process(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize messages: %w", err)
	}

	return sanitized, nil
}

// ShouldCompact returns true if conversation needs compaction
func (m *Manager) ShouldCompact(conversationID string) (bool, error) {
	// This would require a method to get token count without loading all messages
	// For now, estimate based on message count
	return false, nil // Placeholder
}

// CompactConversation manually compacts a conversation
func (m *Manager) CompactConversation(ctx context.Context, conversationID string) error {
	// Get all messages
	messages, err := m.store.GetMessages(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}

	// Compact
	compacted, err := m.contextMgr.PrepareContext(ctx, messages)
	if err != nil {
		return fmt.Errorf("failed to compact: %w", err)
	}

	// If compaction happened, we need to replace messages
	// This is complex - we'd need to delete old and insert new
	// For now, just log that compaction would happen
	if len(compacted) < len(messages) {
		// In a real implementation, we'd:
		// 1. Delete old messages
		// 2. Insert summary message
		// 3. Insert recent messages
		_ = compacted
	}

	return nil
}

// GetTokenEstimate estimates total tokens in conversation
func (m *Manager) GetTokenEstimate(ctx context.Context, conversationID string) (int, error) {
	messages, err := m.store.GetMessages(ctx, conversationID)
	if err != nil {
		return 0, err
	}

	tokens := 0
	for _, msg := range messages {
		tokens += estimateTokens(msg)
	}

	return tokens, nil
}

// DeleteConversation deletes a conversation
func (m *Manager) DeleteConversation(ctx context.Context, conversationID string) error {
	return m.store.DeleteConversation(ctx, conversationID)
}

// ListUserConversations lists all conversations for a user
func (m *Manager) ListUserConversations(ctx context.Context, userID string) ([]*types.Conversation, error) {
	// This would require a method in storev2
	// For now, return empty
	return []*types.Conversation{}, nil
}

// ConversationStats provides statistics about a conversation
type ConversationStats struct {
	MessageCount          int
	TokenEstimate         int
	UserMessageCount      int
	AssistantMessageCount int
	ToolCallCount         int
	Duration              time.Duration
}

// GetStats returns statistics for a conversation
func (m *Manager) GetStats(ctx context.Context, conversationID string) (*ConversationStats, error) {
	messages, err := m.store.GetMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	stats := &ConversationStats{
		MessageCount: len(messages),
	}

	if len(messages) > 0 {
		firstMsg := messages[0]
		lastMsg := messages[len(messages)-1]
		stats.Duration = lastMsg.Timestamp.Sub(firstMsg.Timestamp)
	}

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			stats.UserMessageCount++
		case "assistant":
			stats.AssistantMessageCount++
			stats.ToolCallCount += len(msg.GetToolCalls())
		}
	}

	// Estimate tokens
	tokenCount := 0
	for _, msg := range messages {
		tokenCount += estimateTokens(msg)
	}
	stats.TokenEstimate = tokenCount

	return stats, nil
}

// estimateTokens estimates token count for a message
func estimateTokens(msg types.Message) int {
	// Simple estimation: ~4 chars per token
	tokens := 4 // Base overhead

	for _, block := range msg.Content {
		switch b := block.(type) {
		case types.TextBlock:
			tokens += len(b.Text) / 4
		case types.ToolCallBlock:
			tokens += 10 + len(b.Name)/4 + len(b.Arguments)/4
		case types.ToolResultBlock:
			tokens += 10
			for _, cb := range b.Content {
				if tb, ok := cb.(types.TextBlock); ok {
					tokens += len(tb.Text) / 4
				}
			}
		}
	}

	return tokens
}
