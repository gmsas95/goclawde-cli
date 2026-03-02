// Package pipeline provides message sanitization and transformation
// for different LLM providers, inspired by OpenClaw's architecture.
package pipeline

import (
	"fmt"

	"github.com/gmsas95/myrai-cli/internal/types"
)

// MessageSanitizer transforms a slice of messages.
// Each sanitizer should be idempotent and handle edge cases gracefully.
type MessageSanitizer interface {
	// Sanitize transforms messages and returns the result.
	// Returns error only for unrecoverable issues.
	Sanitize(messages []types.Message) ([]types.Message, error)
}

// Pipeline chains multiple sanitizers together.
// Sanitizers are executed in order, with the output of one feeding into the next.
type Pipeline struct {
	sanitizers []MessageSanitizer
	name       string
}

// NewPipeline creates a new pipeline with the given name
func NewPipeline(name string) *Pipeline {
	return &Pipeline{
		sanitizers: make([]MessageSanitizer, 0),
		name:       name,
	}
}

// Add appends a sanitizer to the pipeline
func (p *Pipeline) Add(sanitizer MessageSanitizer) *Pipeline {
	p.sanitizers = append(p.sanitizers, sanitizer)
	return p
}

// Process executes all sanitizers in sequence
func (p *Pipeline) Process(messages []types.Message) ([]types.Message, error) {
	result := messages

	for i, sanitizer := range p.sanitizers {
		var err error
		result, err = sanitizer.Sanitize(result)
		if err != nil {
			return nil, fmt.Errorf("sanitizer %d (%T) failed: %w", i, sanitizer, err)
		}
	}

	return result, nil
}

// EmptyAssistantFilter removes assistant messages with no meaningful content.
// This prevents API errors like "message with role 'assistant' must not be empty".
type EmptyAssistantFilter struct{}

// Sanitize removes empty assistant messages
func (f *EmptyAssistantFilter) Sanitize(messages []types.Message) ([]types.Message, error) {
	var result []types.Message

	for _, msg := range messages {
		// Only filter assistant messages
		if msg.Role != "assistant" {
			result = append(result, msg)
			continue
		}

		// Keep assistant messages with content
		if !msg.IsEmpty() {
			result = append(result, msg)
			continue
		}

		// Special case: keep error messages even if empty
		if msg.Metadata.StopReason == "error" {
			result = append(result, msg)
			continue
		}

		// Skip truly empty assistant messages
		// This is the key fix for the API error
	}

	return result, nil
}

// ConsecutiveAssistantMerger merges back-to-back assistant messages.
// Some APIs (like Gemini) require strict user/assistant alternation.
type ConsecutiveAssistantMerger struct{}

// Sanitize merges consecutive assistant messages
func (m *ConsecutiveAssistantMerger) Sanitize(messages []types.Message) ([]types.Message, error) {
	if len(messages) < 2 {
		return messages, nil
	}

	var result []types.Message

	for i, msg := range messages {
		if i == 0 {
			result = append(result, msg)
			continue
		}

		lastMsg := &result[len(result)-1]

		// Check if both current and last are assistant messages
		if msg.Role == "assistant" && lastMsg.Role == "assistant" {
			// Merge content blocks
			lastMsg.Content = append(lastMsg.Content, msg.Content...)

			// Merge metadata (keep most recent)
			if msg.Metadata.Usage.Total() > 0 {
				lastMsg.Metadata.Usage.InputTokens += msg.Metadata.Usage.InputTokens
				lastMsg.Metadata.Usage.OutputTokens += msg.Metadata.Usage.OutputTokens
			}
			if msg.Metadata.StopReason != "" {
				lastMsg.Metadata.StopReason = msg.Metadata.StopReason
			}
			if msg.Metadata.ErrorMessage != "" {
				lastMsg.Metadata.ErrorMessage = msg.Metadata.ErrorMessage
			}
		} else {
			result = append(result, msg)
		}
	}

	return result, nil
}

// ConsecutiveUserMerger merges back-to-back user messages.
// Required for APIs like Anthropic that need strict user/assistant alternation.
type ConsecutiveUserMerger struct{}

// Sanitize merges consecutive user messages
func (m *ConsecutiveUserMerger) Sanitize(messages []types.Message) ([]types.Message, error) {
	if len(messages) < 2 {
		return messages, nil
	}

	var result []types.Message

	for i, msg := range messages {
		if i == 0 {
			result = append(result, msg)
			continue
		}

		lastMsg := &result[len(result)-1]

		// Check if both current and last are user messages
		if msg.Role == "user" && lastMsg.Role == "user" {
			// Merge content blocks
			lastMsg.Content = append(lastMsg.Content, msg.Content...)
		} else {
			result = append(result, msg)
		}
	}

	return result, nil
}

// EmptyTextBlockFilter removes empty text blocks from assistant messages.
// This keeps the message but cleans up useless content blocks.
type EmptyTextBlockFilter struct{}

// Sanitize removes empty text blocks from assistant messages
func (f *EmptyTextBlockFilter) Sanitize(messages []types.Message) ([]types.Message, error) {
	for i := range messages {
		if messages[i].Role != "assistant" {
			continue
		}

		// Filter out empty text blocks, keep everything else
		var filtered []types.ContentBlock
		for _, block := range messages[i].Content {
			if textBlock, ok := block.(types.TextBlock); ok {
				if textBlock.IsEmpty() {
					continue // Skip empty text
				}
			}
			filtered = append(filtered, block)
		}

		messages[i].Content = filtered
	}

	return messages, nil
}

// SystemMessageNormalizer ensures there's exactly one system message at the start.
type SystemMessageNormalizer struct{}

// Sanitize ensures proper system message handling
func (n *SystemMessageNormalizer) Sanitize(messages []types.Message) ([]types.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	var systemMessages []types.Message
	var otherMessages []types.Message

	for _, msg := range messages {
		if msg.Role == "system" {
			systemMessages = append(systemMessages, msg)
		} else {
			otherMessages = append(otherMessages, msg)
		}
	}

	// If no system messages, return as-is
	if len(systemMessages) == 0 {
		return messages, nil
	}

	// Merge all system messages into one
	mergedSystem := types.Message{
		Role:      "system",
		Content:   []types.ContentBlock{},
		Timestamp: systemMessages[0].Timestamp,
	}

	for _, sysMsg := range systemMessages {
		mergedSystem.Content = append(mergedSystem.Content, sysMsg.Content...)
	}

	// Return merged system message followed by other messages
	return append([]types.Message{mergedSystem}, otherMessages...), nil
}

// ToolResultValidator ensures tool results have matching tool calls.
type ToolResultValidator struct{}

// Sanitize validates tool results
func (v *ToolResultValidator) Sanitize(messages []types.Message) ([]types.Message, error) {
	// Build set of tool call IDs
	toolCallIDs := make(map[string]bool)

	for _, msg := range messages {
		if msg.Role == "assistant" {
			for _, call := range msg.GetToolCalls() {
				toolCallIDs[call.ID] = true
			}
		}
	}

	// Filter out orphaned tool results
	var result []types.Message
	for _, msg := range messages {
		if msg.Role == "tool" {
			// Check if this tool result has a matching call
			for _, block := range msg.Content {
				if tr, ok := block.(types.ToolResultBlock); ok {
					if !toolCallIDs[tr.ToolCallID] {
						// Orphaned tool result, skip it
						continue
					}
				}
			}
		}
		result = append(result, msg)
	}

	return result, nil
}

// Provider-specific pipelines

// OpenAIPipeline returns a pipeline optimized for OpenAI API
func OpenAIPipeline() *Pipeline {
	return NewPipeline("openai").
		Add(&SystemMessageNormalizer{}).
		Add(&EmptyTextBlockFilter{}).
		Add(&EmptyAssistantFilter{}).
		Add(&ConsecutiveAssistantMerger{}).
		Add(&ToolResultValidator{})
}

// AnthropicPipeline returns a pipeline optimized for Anthropic API
func AnthropicPipeline() *Pipeline {
	return NewPipeline("anthropic").
		Add(&SystemMessageNormalizer{}).
		Add(&EmptyTextBlockFilter{}).
		Add(&EmptyAssistantFilter{}).
		Add(&ConsecutiveUserMerger{}). // Anthropic needs strict alternation
		Add(&ConsecutiveAssistantMerger{}).
		Add(&ToolResultValidator{})
}

// GeminiPipeline returns a pipeline optimized for Google Gemini API
func GeminiPipeline() *Pipeline {
	return NewPipeline("gemini").
		Add(&SystemMessageNormalizer{}).
		Add(&EmptyTextBlockFilter{}).
		Add(&EmptyAssistantFilter{}).
		Add(&ConsecutiveUserMerger{}).
		Add(&ConsecutiveAssistantMerger{})
}

// UniversalPipeline returns a general-purpose pipeline that works for most providers
func UniversalPipeline() *Pipeline {
	return NewPipeline("universal").
		Add(&SystemMessageNormalizer{}).
		Add(&EmptyTextBlockFilter{}).
		Add(&EmptyAssistantFilter{}).
		Add(&ConsecutiveAssistantMerger{}).
		Add(&ToolResultValidator{})
}
