// Package providers provides LLM provider adapters that convert
// between Myrai's content block format and provider-specific formats.
package providers

import (
	"encoding/json"
	"fmt"

	"github.com/gmsas95/myrai-cli/internal/types"
)

// ProviderAdapter converts between Myrai message format and provider-specific formats.
// Each provider (OpenAI, Anthropic, Gemini) implements this interface.
type ProviderAdapter interface {
	// Name returns the provider name
	Name() string

	// ToProviderFormat converts Myrai messages to provider-specific format
	ToProviderFormat(messages []types.Message, tools []*ToolDefinition) (interface{}, error)

	// FromProviderFormat converts provider response to Myrai message
	FromProviderFormat(response interface{}) (*types.Message, error)

	// SupportsStreaming returns true if this provider supports streaming
	SupportsStreaming() bool

	// SupportsToolCalls returns true if this provider supports function calling
	SupportsToolCalls() bool

	// MaxContextLength returns the maximum context length for this provider
	MaxContextLength() int
}

// ToolDefinition defines a tool that can be invoked
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// OpenAIAdapter adapts Myrai messages to OpenAI format
type OpenAIAdapter struct {
	model string
}

// NewOpenAIAdapter creates a new OpenAI adapter
func NewOpenAIAdapter(model string) *OpenAIAdapter {
	return &OpenAIAdapter{model: model}
}

// Name returns "openai"
func (a *OpenAIAdapter) Name() string { return "openai" }

// SupportsStreaming returns true
func (a *OpenAIAdapter) SupportsStreaming() bool { return true }

// SupportsToolCalls returns true
func (a *OpenAIAdapter) SupportsToolCalls() bool { return true }

// MaxContextLength returns context limit based on model
func (a *OpenAIAdapter) MaxContextLength() int {
	switch a.model {
	case "gpt-4", "gpt-4-turbo":
		return 128000
	case "gpt-3.5-turbo":
		return 16385
	default:
		return 128000
	}
}

// OpenAIMessage represents OpenAI's message format
type OpenAIMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall represents OpenAI's tool call format
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function part of a tool call
type FunctionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToProviderFormat converts Myrai messages to OpenAI format
func (a *OpenAIAdapter) ToProviderFormat(messages []types.Message, tools []*ToolDefinition) (interface{}, error) {
	var openaiMessages []OpenAIMessage

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			content := extractTextFromBlocks(msg.Content)
			openaiMessages = append(openaiMessages, OpenAIMessage{
				Role:    "system",
				Content: content,
			})

		case "user":
			content := extractTextFromBlocks(msg.Content)
			openaiMessages = append(openaiMessages, OpenAIMessage{
				Role:    "user",
				Content: content,
			})

		case "assistant":
			// Extract text content and tool calls
			var textContent string
			var toolCalls []ToolCall

			for _, block := range msg.Content {
				switch b := block.(type) {
				case types.TextBlock:
					textContent += b.Text
				case types.ThinkingBlock:
					// OpenAI doesn't support thinking blocks in this format
					// Could convert to text or omit
				case types.ToolCallBlock:
					toolCalls = append(toolCalls, ToolCall{
						ID:   b.ID,
						Type: "function",
						Function: FunctionCall{
							Name:      b.Name,
							Arguments: b.Arguments,
						},
					})
				}
			}

			openaiMsg := OpenAIMessage{
				Role:    "assistant",
				Content: textContent,
			}
			if len(toolCalls) > 0 {
				openaiMsg.ToolCalls = toolCalls
			}
			openaiMessages = append(openaiMessages, openaiMsg)

		case "tool":
			// Tool results in OpenAI format
			for _, block := range msg.Content {
				if tr, ok := block.(types.ToolResultBlock); ok {
					content := extractTextFromBlocks(tr.Content)
					openaiMessages = append(openaiMessages, OpenAIMessage{
						Role:       "tool",
						Content:    content,
						ToolCallID: tr.ToolCallID,
						Name:       tr.ToolName,
					})
				}
			}
		}
	}

	// Build the full request
	request := map[string]interface{}{
		"model":    a.model,
		"messages": openaiMessages,
	}

	// Add tools if provided
	if len(tools) > 0 {
		var openaiTools []map[string]interface{}
		for _, tool := range tools {
			openaiTools = append(openaiTools, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  tool.Parameters,
				},
			})
		}
		request["tools"] = openaiTools
	}

	return request, nil
}

// FromProviderFormat converts OpenAI response to Myrai message
func (a *OpenAIAdapter) FromProviderFormat(response interface{}) (*types.Message, error) {
	respMap, ok := response.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	choices, ok := respMap["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	firstChoice, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid choice format")
	}

	message, ok := firstChoice["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no message in choice")
	}

	msg := &types.Message{
		Role:    "assistant",
		Content: []types.ContentBlock{},
	}

	// Extract text content
	if content, ok := message["content"].(string); ok && content != "" {
		msg.AddTextBlock(content)
	}

	// Extract tool calls
	if toolCalls, ok := message["tool_calls"].([]interface{}); ok {
		for _, tc := range toolCalls {
			tcMap, ok := tc.(map[string]interface{})
			if !ok {
				continue
			}

			id, _ := tcMap["id"].(string)
			function, _ := tcMap["function"].(map[string]interface{})
			name, _ := function["name"].(string)
			argsStr, _ := function["arguments"].(string)

			var args json.RawMessage
			if argsStr != "" {
				args = json.RawMessage(argsStr)
			}

			msg.AddToolCallBlock(id, name, args)
		}
	}

	// Extract usage
	if usage, ok := respMap["usage"].(map[string]interface{}); ok {
		if inputTokens, ok := usage["prompt_tokens"].(float64); ok {
			msg.Metadata.Usage.InputTokens = int(inputTokens)
		}
		if outputTokens, ok := usage["completion_tokens"].(float64); ok {
			msg.Metadata.Usage.OutputTokens = int(outputTokens)
		}
	}

	// Extract stop reason
	if finishReason, ok := firstChoice["finish_reason"].(string); ok {
		msg.Metadata.StopReason = finishReason
	}

	return msg, nil
}

// Helper function to extract text from content blocks
func extractTextFromBlocks(blocks []types.ContentBlock) string {
	var result string
	for _, block := range blocks {
		if text, ok := block.(types.TextBlock); ok {
			if result != "" {
				result += "\n"
			}
			result += text.Text
		}
	}
	return result
}

// AdapterRegistry manages provider adapters
type AdapterRegistry struct {
	adapters map[string]ProviderAdapter
}

// NewAdapterRegistry creates a new registry
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]ProviderAdapter),
	}
}

// Register adds an adapter to the registry
func (r *AdapterRegistry) Register(adapter ProviderAdapter) {
	r.adapters[adapter.Name()] = adapter
}

// Get retrieves an adapter by name
func (r *AdapterRegistry) Get(name string) (ProviderAdapter, bool) {
	adapter, ok := r.adapters[name]
	return adapter, ok
}

// List returns all registered adapter names
func (r *AdapterRegistry) List() []string {
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}

// CreateDefaultRegistry creates a registry with default adapters
func CreateDefaultRegistry() *AdapterRegistry {
	registry := NewAdapterRegistry()

	// Register OpenAI adapter with common models
	registry.Register(NewOpenAIAdapter("gpt-4"))
	registry.Register(NewOpenAIAdapter("gpt-4-turbo"))
	registry.Register(NewOpenAIAdapter("gpt-3.5-turbo"))

	// TODO: Register Anthropic, Gemini adapters

	return registry
}
