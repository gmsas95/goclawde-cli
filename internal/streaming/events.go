// Package streaming provides streaming event handling for LLM responses.
// This enables real-time response processing, tool call streaming, and progress updates.
package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gmsas95/myrai-cli/internal/types"
)

// ToolDefinition represents a tool that can be used in streaming requests
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// StreamEvent represents an event in the response stream.
// Events are sent as the LLM generates content, enabling real-time processing.
type StreamEvent interface {
	EventType() string
	Timestamp() time.Time
}

// BaseEvent provides common fields for all stream events
type BaseEvent struct {
	Type string    `json:"type"`
	Time time.Time `json:"timestamp"`
}

// EventType returns the event type
func (e BaseEvent) EventType() string { return e.Type }

// Timestamp returns the event timestamp
func (e BaseEvent) Timestamp() time.Time { return e.Time }

// --- Event Types ---

// TextDeltaEvent represents incremental text content
type TextDeltaEvent struct {
	BaseEvent
	Delta string `json:"delta"` // The incremental text
	Text  string `json:"text"`  // Accumulated text so far
}

// NewTextDeltaEvent creates a new text delta event
func NewTextDeltaEvent(delta, text string) *TextDeltaEvent {
	return &TextDeltaEvent{
		BaseEvent: BaseEvent{Type: "text_delta", Time: time.Now()},
		Delta:     delta,
		Text:      text,
	}
}

// ThinkingDeltaEvent represents incremental reasoning/thinking content
type ThinkingDeltaEvent struct {
	BaseEvent
	Delta     string `json:"delta"`
	Thinking  string `json:"thinking"` // Accumulated thinking
	Signature string `json:"signature,omitempty"`
}

// NewThinkingDeltaEvent creates a new thinking delta event
func NewThinkingDeltaEvent(delta, thinking string) *ThinkingDeltaEvent {
	return &ThinkingDeltaEvent{
		BaseEvent: BaseEvent{Type: "thinking_delta", Time: time.Now()},
		Delta:     delta,
		Thinking:  thinking,
	}
}

// ToolCallStartEvent signals that a tool call is beginning
type ToolCallStartEvent struct {
	BaseEvent
	ID   string `json:"id"`
	Name string `json:"name"`
}

// NewToolCallStartEvent creates a new tool call start event
func NewToolCallStartEvent(id, name string) *ToolCallStartEvent {
	return &ToolCallStartEvent{
		BaseEvent: BaseEvent{Type: "tool_call_start", Time: time.Now()},
		ID:        id,
		Name:      name,
	}
}

// ToolCallDeltaEvent represents incremental tool call arguments
type ToolCallDeltaEvent struct {
	BaseEvent
	ID        string `json:"id"`
	Delta     string `json:"delta"`     // Incremental JSON
	Arguments string `json:"arguments"` // Accumulated arguments JSON
}

// NewToolCallDeltaEvent creates a new tool call delta event
func NewToolCallDeltaEvent(id, delta, arguments string) *ToolCallDeltaEvent {
	return &ToolCallDeltaEvent{
		BaseEvent: BaseEvent{Type: "tool_call_delta", Time: time.Now()},
		ID:        id,
		Delta:     delta,
		Arguments: arguments,
	}
}

// ToolCallCompleteEvent signals that a tool call is complete
type ToolCallCompleteEvent struct {
	BaseEvent
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// NewToolCallCompleteEvent creates a new tool call complete event
func NewToolCallCompleteEvent(id, name string, args json.RawMessage) *ToolCallCompleteEvent {
	return &ToolCallCompleteEvent{
		BaseEvent: BaseEvent{Type: "tool_call_complete", Time: time.Now()},
		ID:        id,
		Name:      name,
		Arguments: args,
	}
}

// ToolResultStartEvent signals that tool execution is starting
type ToolResultStartEvent struct {
	BaseEvent
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
}

// NewToolResultStartEvent creates a new tool result start event
func NewToolResultStartEvent(toolCallID, toolName string) *ToolResultStartEvent {
	return &ToolResultStartEvent{
		BaseEvent:  BaseEvent{Type: "tool_result_start", Time: time.Now()},
		ToolCallID: toolCallID,
		ToolName:   toolName,
	}
}

// ToolResultDeltaEvent represents incremental tool result content
type ToolResultDeltaEvent struct {
	BaseEvent
	ToolCallID string `json:"tool_call_id"`
	Delta      string `json:"delta"`
	Content    string `json:"content"`
}

// NewToolResultDeltaEvent creates a new tool result delta event
func NewToolResultDeltaEvent(toolCallID, delta, content string) *ToolResultDeltaEvent {
	return &ToolResultDeltaEvent{
		BaseEvent:  BaseEvent{Type: "tool_result_delta", Time: time.Now()},
		ToolCallID: toolCallID,
		Delta:      delta,
		Content:    content,
	}
}

// ToolResultCompleteEvent signals that tool execution is complete
type ToolResultCompleteEvent struct {
	BaseEvent
	ToolCallID string          `json:"tool_call_id"`
	Content    json.RawMessage `json:"content"` // Array of content blocks
	IsError    bool            `json:"is_error"`
}

// NewToolResultCompleteEvent creates a new tool result complete event
func NewToolResultCompleteEvent(toolCallID string, content json.RawMessage, isError bool) *ToolResultCompleteEvent {
	return &ToolResultCompleteEvent{
		BaseEvent:  BaseEvent{Type: "tool_result_complete", Time: time.Now()},
		ToolCallID: toolCallID,
		Content:    content,
		IsError:    isError,
	}
}

// UsageEvent contains token usage information
type UsageEvent struct {
	BaseEvent
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	CacheRead    int `json:"cache_read,omitempty"`
	CacheWrite   int `json:"cache_write,omitempty"`
	Total        int `json:"total"`
}

// NewUsageEvent creates a new usage event
func NewUsageEvent(inputTokens, outputTokens int) *UsageEvent {
	return &UsageEvent{
		BaseEvent:    BaseEvent{Type: "usage", Time: time.Now()},
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Total:        inputTokens + outputTokens,
	}
}

// StopEvent signals the end of the stream
type StopEvent struct {
	BaseEvent
	Reason string `json:"reason"` // "stop", "length", "content_filter", "tool_calls", "error"
}

// NewStopEvent creates a new stop event
func NewStopEvent(reason string) *StopEvent {
	return &StopEvent{
		BaseEvent: BaseEvent{Type: "stop", Time: time.Now()},
		Reason:    reason,
	}
}

// ErrorEvent signals an error occurred
type ErrorEvent struct {
	BaseEvent
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// NewErrorEvent creates a new error event
func NewErrorEvent(err error) *ErrorEvent {
	return &ErrorEvent{
		BaseEvent: BaseEvent{Type: "error", Time: time.Now()},
		Error:     err.Error(),
		Message:   err.Error(),
	}
}

// ProgressEvent provides progress updates for long operations
type ProgressEvent struct {
	BaseEvent
	Stage       string  `json:"stage"`       // e.g., "thinking", "tool_call", "processing"
	Description string  `json:"description"` // Human-readable description
	Percent     float64 `json:"percent"`     // 0-100, -1 if unknown
}

// NewProgressEvent creates a new progress event
func NewProgressEvent(stage, description string, percent float64) *ProgressEvent {
	return &ProgressEvent{
		BaseEvent:   BaseEvent{Type: "progress", Time: time.Now()},
		Stage:       stage,
		Description: description,
		Percent:     percent,
	}
}

// --- Event Handler ---

// EventHandler processes streaming events
type EventHandler interface {
	// OnEvent is called for each stream event
	OnEvent(event StreamEvent) error

	// OnError is called when an error occurs
	OnError(err error)

	// OnComplete is called when the stream ends successfully
	OnComplete()
}

// EventHandlerFunc is a function type that implements EventHandler
type EventHandlerFunc struct {
	OnEventFunc    func(StreamEvent) error
	OnErrorFunc    func(error)
	OnCompleteFunc func()
}

// OnEvent implements EventHandler
func (h EventHandlerFunc) OnEvent(event StreamEvent) error {
	if h.OnEventFunc != nil {
		return h.OnEventFunc(event)
	}
	return nil
}

// OnError implements EventHandler
func (h EventHandlerFunc) OnError(err error) {
	if h.OnErrorFunc != nil {
		h.OnErrorFunc(err)
	}
}

// OnComplete implements EventHandler
func (h EventHandlerFunc) OnComplete() {
	if h.OnCompleteFunc != nil {
		h.OnCompleteFunc()
	}
}

// --- Streaming Client ---

// Client interface for streaming LLM providers
type Client interface {
	// Stream sends a request and streams the response
	Stream(ctx context.Context, request StreamRequest, handler EventHandler) error

	// SupportsStreaming returns true if this client supports streaming
	SupportsStreaming() bool
}

// StreamRequest contains parameters for a streaming request
type StreamRequest struct {
	Messages []types.Message
	Tools    []*ToolDefinition
	Model    string
	// Provider-specific options
	Options map[string]interface{}
}

// --- Accumulator ---

// Accumulator collects stream events into a complete message
type Accumulator struct {
	message         *types.Message
	toolCalls       map[string]*types.ToolCallBlock
	accumulatedArgs map[string]string
	thinking        string
	text            string
	usage           types.Usage
}

// NewAccumulator creates a new event accumulator
func NewAccumulator() *Accumulator {
	return &Accumulator{
		message: &types.Message{
			Role:    "assistant",
			Content: []types.ContentBlock{},
		},
		toolCalls:       make(map[string]*types.ToolCallBlock),
		accumulatedArgs: make(map[string]string),
	}
}

// ProcessEvent processes a single stream event
func (a *Accumulator) ProcessEvent(event StreamEvent) error {
	switch e := event.(type) {
	case *TextDeltaEvent:
		a.text += e.Delta

	case *ThinkingDeltaEvent:
		a.thinking += e.Delta

	case *ToolCallStartEvent:
		// Initialize tool call tracking
		a.toolCalls[e.ID] = &types.ToolCallBlock{
			ID:   e.ID,
			Name: e.Name,
		}
		a.accumulatedArgs[e.ID] = ""

	case *ToolCallDeltaEvent:
		// Accumulate arguments
		if current, ok := a.accumulatedArgs[e.ID]; ok {
			a.accumulatedArgs[e.ID] = current + e.Delta
		}

	case *ToolCallCompleteEvent:
		// Store completed tool call
		a.toolCalls[e.ID] = &types.ToolCallBlock{
			ID:        e.ID,
			Name:      e.Name,
			Arguments: e.Arguments,
		}

	case *UsageEvent:
		a.usage.InputTokens = e.InputTokens
		a.usage.OutputTokens = e.OutputTokens
		a.usage.CacheRead = e.CacheRead
		a.usage.CacheWrite = e.CacheWrite

	case *StopEvent:
		// Finalize message
		a.message.Metadata.StopReason = e.Reason
	}

	return nil
}

// Finalize builds the final message from accumulated events
func (a *Accumulator) Finalize() *types.Message {
	// Add thinking if present
	if a.thinking != "" {
		a.message.AddThinkingBlock(a.thinking, "")
	}

	// Add text content
	if a.text != "" {
		a.message.AddTextBlock(a.text)
	}

	// Add tool calls
	for _, toolCall := range a.toolCalls {
		a.message.AddToolCallBlock(toolCall.ID, toolCall.Name, toolCall.Arguments)
	}

	// Set usage
	a.message.Metadata.Usage = a.usage

	return a.message
}

// GetMessage returns the current accumulated message state
func (a *Accumulator) GetMessage() *types.Message {
	// Return a copy with current state
	msg := &types.Message{
		Role:     "assistant",
		Content:  []types.ContentBlock{},
		Metadata: a.message.Metadata,
	}

	if a.thinking != "" {
		msg.AddThinkingBlock(a.thinking, "")
	}
	if a.text != "" {
		msg.AddTextBlock(a.text)
	}

	return msg
}

// --- Utility Functions ---

// IsStreamingError checks if an error should stop the stream
func IsStreamingError(err error) bool {
	// Context cancellation should stop the stream
	if err == context.Canceled || err == context.DeadlineExceeded {
		return true
	}
	return false
}

// EventToJSON converts a stream event to JSON
func EventToJSON(event StreamEvent) ([]byte, error) {
	return json.Marshal(event)
}

// JSONToEvent converts JSON back to a stream event (for deserialization)
func JSONToEvent(data []byte) (StreamEvent, error) {
	var base BaseEvent
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}

	switch base.Type {
	case "text_delta":
		var e TextDeltaEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return &e, nil
	case "thinking_delta":
		var e ThinkingDeltaEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return &e, nil
	case "tool_call_start":
		var e ToolCallStartEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return &e, nil
	case "tool_call_delta":
		var e ToolCallDeltaEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return &e, nil
	case "tool_call_complete":
		var e ToolCallCompleteEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return &e, nil
	case "usage":
		var e UsageEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return &e, nil
	case "stop":
		var e StopEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return &e, nil
	case "error":
		var e ErrorEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return &e, nil
	default:
		return nil, fmt.Errorf("unknown event type: %s", base.Type)
	}
}
