// Package streaming provides OpenAI-compatible streaming client implementation.
package streaming

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gmsas95/myrai-cli/internal/types"
)

// OpenAIStreamingClient implements streaming for OpenAI-compatible APIs
type OpenAIStreamingClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      string
}

// NewOpenAIStreamingClient creates a new OpenAI streaming client
func NewOpenAIStreamingClient(apiKey, baseURL, model string) *OpenAIStreamingClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIStreamingClient{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{},
		model:      model,
	}
}

// SupportsStreaming returns true
func (c *OpenAIStreamingClient) SupportsStreaming() bool {
	return true
}

// Stream sends a streaming request to OpenAI API
func (c *OpenAIStreamingClient) Stream(ctx context.Context, request StreamRequest, handler EventHandler) error {
	// Build the request body
	body, err := c.buildRequestBody(request)
	if err != nil {
		return fmt.Errorf("failed to build request body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Process SSE stream
	return c.processStream(ctx, resp.Body, handler)
}

// buildRequestBody constructs the OpenAI API request
func (c *OpenAIStreamingClient) buildRequestBody(request StreamRequest) ([]byte, error) {
	// Convert messages to OpenAI format
	var messages []map[string]interface{}
	for _, msg := range request.Messages {
		switch msg.Role {
		case "system":
			messages = append(messages, map[string]interface{}{
				"role":    "system",
				"content": extractTextFromMessage(msg),
			})
		case "user":
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": extractTextFromMessage(msg),
			})
		case "assistant":
			openaiMsg := map[string]interface{}{
				"role":    "assistant",
				"content": extractTextFromMessage(msg),
			}
			// Add tool calls if present
			toolCalls := extractToolCallsFromMessage(msg)
			if len(toolCalls) > 0 {
				openaiMsg["tool_calls"] = toolCalls
			}
			messages = append(messages, openaiMsg)
		case "tool":
			// Tool results
			for _, block := range msg.Content {
				if tr, ok := block.(types.ToolResultBlock); ok {
					messages = append(messages, map[string]interface{}{
						"role":         "tool",
						"content":      extractTextFromBlocks(tr.Content),
						"tool_call_id": tr.ToolCallID,
					})
				}
			}
		}
	}

	reqBody := map[string]interface{}{
		"model":    c.model,
		"messages": messages,
		"stream":   true,
	}

	// Add tools if present
	if len(request.Tools) > 0 {
		var tools []map[string]interface{}
		for _, tool := range request.Tools {
			tools = append(tools, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  tool.Parameters,
				},
			})
		}
		reqBody["tools"] = tools
	}

	// Add any provider-specific options
	for key, value := range request.Options {
		reqBody[key] = value
	}

	return json.Marshal(reqBody)
}

// processStream reads SSE events from the response body
func (c *OpenAIStreamingClient) processStream(ctx context.Context, body io.Reader, handler EventHandler) error {
	scanner := bufio.NewScanner(body)
	accumulator := NewAccumulator()

	// Track tool call arguments being streamed
	toolCallArgs := make(map[string]string)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse SSE event
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end
		if data == "[DONE]" {
			handler.OnComplete()
			return nil
		}

		// Parse the JSON chunk
		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Non-fatal parsing error, log and continue
			continue
		}

		// Extract delta
		choices, ok := chunk["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			continue
		}

		choice, ok := choices[0].(map[string]interface{})
		if !ok {
			continue
		}

		delta, ok := choice["delta"].(map[string]interface{})
		if !ok {
			continue
		}

		// Process content
		if content, ok := delta["content"].(string); ok {
			event := NewTextDeltaEvent(content, accumulator.text+content)
			if err := handler.OnEvent(event); err != nil {
				return err
			}
			accumulator.ProcessEvent(event)
		}

		// Process tool calls
		if toolCalls, ok := delta["tool_calls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				tcMap, ok := tc.(map[string]interface{})
				if !ok {
					continue
				}

				id, _ := tcMap["id"].(string)
				function, _ := tcMap["function"].(map[string]interface{})
				name, _ := function["name"].(string)
				args, _ := function["arguments"].(string)

				if id != "" {
					// New tool call
					toolCallArgs[id] = ""
					event := NewToolCallStartEvent(id, name)
					if err := handler.OnEvent(event); err != nil {
						return err
					}
				}

				if args != "" {
					// Accumulate arguments
					toolCallArgs[id] += args
					event := NewToolCallDeltaEvent(id, args, toolCallArgs[id])
					if err := handler.OnEvent(event); err != nil {
						return err
					}
					accumulator.ProcessEvent(event)
				}

				// Check if tool call is complete
				finishReason, _ := choice["finish_reason"].(string)
				if finishReason == "tool_calls" {
					// Tool call is complete, try to parse arguments
					var argsJSON json.RawMessage
					if accumulated, ok := toolCallArgs[id]; ok {
						argsJSON = json.RawMessage(accumulated)
					}
					event := NewToolCallCompleteEvent(id, name, argsJSON)
					if err := handler.OnEvent(event); err != nil {
						return err
					}
					accumulator.ProcessEvent(event)
				}
			}
		}

		// Process finish reason
		if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" && finishReason != "null" {
			event := NewStopEvent(finishReason)
			if err := handler.OnEvent(event); err != nil {
				return err
			}
			accumulator.ProcessEvent(event)
		}

		// Process usage (usually in the last chunk)
		if usage, ok := chunk["usage"].(map[string]interface{}); ok {
			inputTokens := 0
			outputTokens := 0

			if promptTokens, ok := usage["prompt_tokens"].(float64); ok {
				inputTokens = int(promptTokens)
			}
			if completionTokens, ok := usage["completion_tokens"].(float64); ok {
				outputTokens = int(completionTokens)
			}

			event := NewUsageEvent(inputTokens, outputTokens)
			if err := handler.OnEvent(event); err != nil {
				return err
			}
			accumulator.ProcessEvent(event)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream error: %w", err)
	}

	handler.OnComplete()
	return nil
}

// extractTextFromMessage extracts text content from a message
func extractTextFromMessage(msg types.Message) string {
	var result string
	for _, block := range msg.Content {
		if text, ok := block.(types.TextBlock); ok {
			if result != "" {
				result += "\n"
			}
			result += text.Text
		}
	}
	return result
}

// extractToolCallsFromMessage extracts tool calls in OpenAI format
func extractToolCallsFromMessage(msg types.Message) []map[string]interface{} {
	var result []map[string]interface{}

	for _, block := range msg.Content {
		if tc, ok := block.(types.ToolCallBlock); ok {
			result = append(result, map[string]interface{}{
				"id":   tc.ID,
				"type": "function",
				"function": map[string]interface{}{
					"name":      tc.Name,
					"arguments": string(tc.Arguments),
				},
			})
		}
	}

	return result
}

// extractTextFromBlocks extracts text from content blocks
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

// NonStreamingClient wraps a non-streaming provider for compatibility
type NonStreamingClient struct {
	provider Provider
}

// Provider interface for non-streaming providers
type Provider interface {
	Complete(ctx context.Context, messages []types.Message, tools []*ToolDefinition) (*types.Message, error)
}

// NewNonStreamingClient creates a client that simulates streaming
func NewNonStreamingClient(provider Provider) *NonStreamingClient {
	return &NonStreamingClient{provider: provider}
}

// SupportsStreaming returns false
func (c *NonStreamingClient) SupportsStreaming() bool {
	return false
}

// Stream simulates streaming by sending the complete response as one event
func (c *NonStreamingClient) Stream(ctx context.Context, request StreamRequest, handler EventHandler) error {
	// Get complete response
	msg, err := c.provider.Complete(ctx, request.Messages, request.Tools)
	if err != nil {
		handler.OnError(err)
		return err
	}

	// Send text content as single delta
	text := msg.GetTextContent()
	if text != "" {
		event := NewTextDeltaEvent(text, text)
		if err := handler.OnEvent(event); err != nil {
			return err
		}
	}

	// Send tool calls
	for _, tc := range msg.GetToolCalls() {
		event := NewToolCallCompleteEvent(tc.ID, tc.Name, tc.Arguments)
		if err := handler.OnEvent(event); err != nil {
			return err
		}
	}

	// Send usage
	usageEvent := NewUsageEvent(msg.Metadata.Usage.InputTokens, msg.Metadata.Usage.OutputTokens)
	if err := handler.OnEvent(usageEvent); err != nil {
		return err
	}

	// Send stop event
	stopEvent := NewStopEvent(msg.Metadata.StopReason)
	if err := handler.OnEvent(stopEvent); err != nil {
		return err
	}

	handler.OnComplete()
	return nil
}
