package llm

import (
	"context"
	"encoding/json"
	"strings"
)

// ToolCallAccumulator handles streaming tool call chunks
type ToolCallAccumulator struct {
	toolCalls map[int]*accumulatingToolCall
}

type accumulatingToolCall struct {
	ID        string
	Type      string
	Function  accumulatingFunction
}

type accumulatingFunction struct {
	Name      string
	Arguments strings.Builder
}

// NewToolCallAccumulator creates a new accumulator
func NewToolCallAccumulator() *ToolCallAccumulator {
	return &ToolCallAccumulator{
		toolCalls: make(map[int]*accumulatingToolCall),
	}
}

// AddChunk processes a streaming chunk and accumulates tool calls
// Returns true when all tool calls are complete
func (acc *ToolCallAccumulator) AddChunk(chunk StreamResponse) (complete bool, toolCalls []ToolCall) {
	if len(chunk.Choices) == 0 {
		return false, nil
	}

	delta := chunk.Choices[0].Delta

	// Accumulate tool calls
	for _, tc := range delta.ToolCalls {
		idx := chunk.Choices[0].Index
		
		if _, exists := acc.toolCalls[idx]; !exists {
			acc.toolCalls[idx] = &accumulatingToolCall{
				ID:   tc.ID,
				Type: tc.Type,
			}
		}
		
		atc := acc.toolCalls[idx]
		
		if tc.Function.Name != "" {
			atc.Function.Name = tc.Function.Name
		}
		
		if tc.Function.Arguments != "" {
			atc.Function.Arguments.WriteString(tc.Function.Arguments)
		}
	}

	// Check if finish_reason indicates completion
	finishReason := chunk.Choices[0].FinishReason
	if finishReason != "" && finishReason != "null" {
		// Convert accumulated tool calls to final format
		result := make([]ToolCall, 0, len(acc.toolCalls))
		for _, atc := range acc.toolCalls {
			result = append(result, ToolCall{
				ID:   atc.ID,
				Type: atc.Type,
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      atc.Function.Name,
					Arguments: atc.Function.Arguments.String(),
				},
			})
		}
		return true, result
	}

	return false, nil
}

// StreamHandler manages streaming responses with tool support
type StreamHandler struct {
	accumulator    *ToolCallAccumulator
	onContent      func(string)
	onToolCall     func(ToolCall)
	contentBuffer  strings.Builder
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(
	onContent func(string),
	onToolCall func(ToolCall),
) *StreamHandler {
	return &StreamHandler{
		accumulator: NewToolCallAccumulator(),
		onContent:   onContent,
		onToolCall:  onToolCall,
	}
}

// HandleChunk processes a chunk from the stream
// Returns (done, toolCalls, error)
func (sh *StreamHandler) HandleChunk(chunk StreamResponse) (bool, []ToolCall, error) {
	if len(chunk.Choices) == 0 {
		return false, nil, nil
	}

	choice := chunk.Choices[0]
	delta := choice.Delta

	// Handle content
	if delta.Content != "" {
		sh.contentBuffer.WriteString(delta.Content)
		if sh.onContent != nil {
			sh.onContent(delta.Content)
		}
	}

	// Handle tool calls
	if len(delta.ToolCalls) > 0 {
		complete, toolCalls := sh.accumulator.AddChunk(chunk)
		if complete && len(toolCalls) > 0 {
			if sh.onToolCall != nil {
				for _, tc := range toolCalls {
					sh.onToolCall(tc)
				}
			}
			return true, toolCalls, nil
		}
	}

	// Check for completion
	if choice.FinishReason != "" && choice.FinishReason != "null" {
		// Check if we have accumulated tool calls
		if len(sh.accumulator.toolCalls) > 0 {
			toolCalls := make([]ToolCall, 0, len(sh.accumulator.toolCalls))
			for _, atc := range sh.accumulator.toolCalls {
				toolCalls = append(toolCalls, ToolCall{
					ID:   atc.ID,
					Type: atc.Type,
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      atc.Function.Name,
						Arguments: atc.Function.Arguments.String(),
					},
				})
			}
			return true, toolCalls, nil
		}
		return true, nil, nil
	}

	return false, nil, nil
}

// GetContent returns the accumulated content
func (sh *StreamHandler) GetContent() string {
	return sh.contentBuffer.String()
}

// IsValidJSON checks if a string is valid JSON
func IsValidJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

// ParseToolArguments parses tool call arguments safely
func ParseToolArguments(args string) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(args), &result); err != nil {
		// Try to fix common JSON issues
		// Sometimes the model returns partial JSON
		fixed := fixPartialJSON(args)
		if err := json.Unmarshal([]byte(fixed), &result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// fixPartialJSON attempts to fix incomplete JSON
func fixPartialJSON(s string) string {
	// Remove trailing commas
	s = strings.TrimRight(s, ",")
	
	// Balance braces
	openBraces := strings.Count(s, "{")
	closeBraces := strings.Count(s, "}")
	for i := 0; i < openBraces-closeBraces; i++ {
		s += "}"
	}
	
	// Balance brackets
	openBrackets := strings.Count(s, "[")
	closeBrackets := strings.Count(s, "]")
	for i := 0; i < openBrackets-closeBrackets; i++ {
		s += "]"
	}
	
	return s
}

// ChatCompletionStreamWithTools handles streaming with proper tool call accumulation
func (c *Client) ChatCompletionStreamWithTools(
	ctx context.Context,
	req ChatRequest,
	onContent func(string),
	onComplete func(content string, toolCalls []ToolCall),
) error {
	req.Stream = true

	handler := NewStreamHandler(onContent, nil)
	
	return c.ChatCompletionStream(ctx, req, func(chunk StreamResponse) error {
		done, toolCalls, err := handler.HandleChunk(chunk)
		if err != nil {
			return err
		}
		if done {
			if onComplete != nil {
				onComplete(handler.GetContent(), toolCalls)
			}
		}
		return nil
	})
}
