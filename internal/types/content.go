// Package types provides the core type definitions for Myrai's content block architecture.
// This replaces the string-based message model with a flexible, extensible block system
// inspired by OpenClaw's architecture.
package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// ContentBlock is the interface for all content types in a message.
// Each block represents a distinct piece of content: text, tool call, image, etc.
type ContentBlock interface {
	// BlockType returns the type identifier for this block
	BlockType() string

	// IsEmpty returns true if this block contains no meaningful content
	IsEmpty() bool
}

// Ensure all block types implement ContentBlock
var (
	_ ContentBlock = (*TextBlock)(nil)
	_ ContentBlock = (*ToolCallBlock)(nil)
	_ ContentBlock = (*ToolResultBlock)(nil)
	_ ContentBlock = (*ThinkingBlock)(nil)
	_ ContentBlock = (*ImageBlock)(nil)
)

// TextBlock represents text content from the model or user.
type TextBlock struct {
	Text string `json:"text"`
}

// BlockType returns "text"
func (b TextBlock) BlockType() string { return "text" }

// IsEmpty returns true if text is empty or whitespace-only
func (b TextBlock) IsEmpty() bool {
	return len(b.Text) == 0 || len(trimSpace(b.Text)) == 0
}

// ToolCallBlock represents a tool invocation requested by the model.
type ToolCallBlock struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// BlockType returns "tool_call"
func (b ToolCallBlock) BlockType() string { return "tool_call" }

// IsEmpty always returns false - tool calls are never empty
func (b ToolCallBlock) IsEmpty() bool { return false }

// ToolResultBlock represents the result of executing a tool.
type ToolResultBlock struct {
	ToolCallID string         `json:"tool_call_id"`
	ToolName   string         `json:"tool_name"`
	Content    []ContentBlock `json:"content"`
	IsError    bool           `json:"is_error"`
}

// BlockType returns "tool_result"
func (b ToolResultBlock) BlockType() string { return "tool_result" }

// IsEmpty returns true if content is empty
func (b ToolResultBlock) IsEmpty() bool {
	return len(b.Content) == 0
}

// ThinkingBlock represents model reasoning/thinking content.
// This is separate from regular text to support reasoning models.
type ThinkingBlock struct {
	Thinking  string `json:"thinking"`
	Signature string `json:"signature,omitempty"`
}

// BlockType returns "thinking"
func (b ThinkingBlock) BlockType() string { return "thinking" }

// IsEmpty returns true if thinking content is empty
func (b ThinkingBlock) IsEmpty() bool {
	return len(b.Thinking) == 0
}

// ImageBlock represents image content.
type ImageBlock struct {
	Source    string `json:"source"`     // "base64" or "url"
	MediaType string `json:"media_type"` // e.g., "image/png"
	Data      string `json:"data"`       // base64 data or URL
}

// BlockType returns "image"
func (b ImageBlock) BlockType() string { return "image" }

// IsEmpty returns true if no image data
func (b ImageBlock) IsEmpty() bool {
	return len(b.Data) == 0
}

// ContentBlockWrapper is used for JSON serialization/deserialization
// of polymorphic content blocks.
type ContentBlockWrapper struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// MarshalJSON implements custom JSON marshaling for ContentBlock
func (b TextBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		TextBlock
	}{
		Type:      "text",
		TextBlock: b,
	})
}

// MarshalJSON implements custom JSON marshaling for ContentBlock
func (b ToolCallBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		ToolCallBlock
	}{
		Type:          "tool_call",
		ToolCallBlock: b,
	})
}

// MarshalJSON implements custom JSON marshaling for ContentBlock
func (b ToolResultBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		ToolResultBlock
	}{
		Type:            "tool_result",
		ToolResultBlock: b,
	})
}

// MarshalJSON implements custom JSON marshaling for ContentBlock
func (b ThinkingBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		ThinkingBlock
	}{
		Type:          "thinking",
		ThinkingBlock: b,
	})
}

// MarshalJSON implements custom JSON marshaling for ContentBlock
func (b ImageBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		ImageBlock
	}{
		Type:       "image",
		ImageBlock: b,
	})
}

// UnmarshalContentBlock deserializes a content block from JSON
func UnmarshalContentBlock(data []byte) (ContentBlock, error) {
	var wrapper struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content block wrapper: %w", err)
	}

	switch wrapper.Type {
	case "text":
		var block TextBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, fmt.Errorf("failed to unmarshal text block: %w", err)
		}
		return block, nil

	case "tool_call":
		var block ToolCallBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tool call block: %w", err)
		}
		return block, nil

	case "tool_result":
		var block ToolResultBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tool result block: %w", err)
		}
		return block, nil

	case "thinking":
		var block ThinkingBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, fmt.Errorf("failed to unmarshal thinking block: %w", err)
		}
		return block, nil

	case "image":
		var block ImageBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, fmt.Errorf("failed to unmarshal image block: %w", err)
		}
		return block, nil

	default:
		return nil, fmt.Errorf("unknown content block type: %s", wrapper.Type)
	}
}

// Usage represents token usage for a message or conversation
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	CacheRead    int `json:"cache_read,omitempty"`
	CacheWrite   int `json:"cache_write,omitempty"`
}

// Total returns the total token count
func (u Usage) Total() int {
	return u.InputTokens + u.OutputTokens
}

// MessageMetadata contains additional message information
type MessageMetadata struct {
	StopReason   string `json:"stop_reason,omitempty"`
	Usage        Usage  `json:"usage,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	Model        string `json:"model,omitempty"`
}

// Message represents a single message in a conversation.
// This replaces the old store.Message with content blocks.
type Message struct {
	ID        string          `json:"id"`
	Role      string          `json:"role"` // "system", "user", "assistant", "tool"
	Content   []ContentBlock  `json:"content"`
	Metadata  MessageMetadata `json:"metadata,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// IsEmpty returns true if the message has no meaningful content
func (m *Message) IsEmpty() bool {
	if len(m.Content) == 0 {
		return true
	}

	// Check if all blocks are empty
	for _, block := range m.Content {
		if !block.IsEmpty() {
			return false
		}
	}
	return true
}

// HasToolCalls returns true if this message contains tool calls
func (m *Message) HasToolCalls() bool {
	for _, block := range m.Content {
		if _, ok := block.(ToolCallBlock); ok {
			return true
		}
	}
	return false
}

// GetTextContent extracts all text content from the message
func (m *Message) GetTextContent() string {
	var result string
	for _, block := range m.Content {
		if text, ok := block.(TextBlock); ok {
			if result != "" {
				result += "\n"
			}
			result += text.Text
		}
	}
	return result
}

// GetToolCalls returns all tool call blocks from the message
func (m *Message) GetToolCalls() []ToolCallBlock {
	var calls []ToolCallBlock
	for _, block := range m.Content {
		if call, ok := block.(ToolCallBlock); ok {
			calls = append(calls, call)
		}
	}
	return calls
}

// GetThinkingContent returns all thinking blocks from the message
func (m *Message) GetThinkingContent() string {
	var result string
	for _, block := range m.Content {
		if thinking, ok := block.(ThinkingBlock); ok {
			if result != "" {
				result += "\n"
			}
			result += thinking.Thinking
		}
	}
	return result
}

// AddTextBlock adds a text block to the message
func (m *Message) AddTextBlock(text string) {
	m.Content = append(m.Content, TextBlock{Text: text})
}

// AddToolCallBlock adds a tool call block to the message
func (m *Message) AddToolCallBlock(id, name string, args json.RawMessage) {
	m.Content = append(m.Content, ToolCallBlock{
		ID:        id,
		Name:      name,
		Arguments: args,
	})
}

// AddThinkingBlock adds a thinking block to the message
func (m *Message) AddThinkingBlock(thinking, signature string) {
	m.Content = append(m.Content, ThinkingBlock{
		Thinking:  thinking,
		Signature: signature,
	})
}

// Conversation represents a conversation with metadata
type Conversation struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Helper function to trim whitespace (simple implementation)
func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end {
		c := s[start]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		start++
	}

	for end > start {
		c := s[end-1]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		end--
	}

	return s[start:end]
}
