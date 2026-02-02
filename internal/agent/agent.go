package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/YOUR_USERNAME/jimmy.ai/internal/llm"
	"github.com/YOUR_USERNAME/jimmy.ai/internal/store"
	"github.com/YOUR_USERNAME/jimmy.ai/pkg/tools"
	"go.uber.org/zap"
)

// Agent handles conversation and tool execution
type Agent struct {
	llmClient *llm.Client
	tools     *tools.Registry
	store     *store.Store
	logger    *zap.Logger
}

// New creates a new Agent
func New(llmClient *llm.Client, toolRegistry *tools.Registry, store *store.Store, logger *zap.Logger) *Agent {
	return &Agent{
		llmClient: llmClient,
		tools:     toolRegistry,
		store:     store,
		logger:    logger,
	}
}

// ChatRequest represents a chat request
type ChatRequest struct {
	ConversationID string
	Message        string
	SystemPrompt   string
	Stream         bool
	OnStream       func(string)
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Content      string
	ToolCalls    []llm.ToolCall
	TokensUsed   int
	ResponseTime time.Duration
}

// Chat handles a single chat turn with possible tool execution
func (a *Agent) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	start := time.Now()

	// Get or create conversation
	conv, err := a.getOrCreateConversation(req.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// Save user message
	userMsg := &store.Message{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        req.Message,
		Tokens:         llm.CountTokens(req.Message),
	}
	if err := a.store.CreateMessage(userMsg); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// Build message history
	messages, err := a.buildContext(ctx, conv.ID, req.SystemPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to build context: %w", err)
	}

	// Build tool definitions
	toolDefs := a.tools.GetToolDefinitions()

	// Call LLM
	llmReq := llm.ChatRequest{
		Model:       a.llmClient.GetModel(),
		Messages:    messages,
		Tools:       a.convertTools(toolDefs),
		MaxTokens:   4096,
		Temperature: 0.7,
		Stream:      req.Stream,
	}

	var response *ChatResponse
	if req.Stream && req.OnStream != nil {
		response, err = a.chatStream(ctx, llmReq, conv.ID, req.OnStream)
	} else {
		response, err = a.chatNonStream(ctx, llmReq, conv.ID)
	}

	if err != nil {
		return nil, err
	}

	response.ResponseTime = time.Since(start)

	// Update conversation stats
	conv.TokensUsed += int64(response.TokensUsed)
	conv.MessageCount += 2 // user + assistant
	conv.UpdatedAt = time.Now()
	if err := a.store.UpdateConversation(conv); err != nil {
		a.logger.Warn("Failed to update conversation stats", zap.Error(err))
	}

	return response, nil
}

func (a *Agent) chatNonStream(ctx context.Context, req llm.ChatRequest, convID string) (*ChatResponse, error) {
	resp, err := a.llmClient.ChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	choice := resp.Choices[0]
	msg := choice.Message

	// Handle tool calls
	if len(msg.ToolCalls) > 0 {
		return a.handleToolCalls(ctx, req, convID, msg.ToolCalls)
	}

	// Save assistant message
	assistantMsg := &store.Message{
		ConversationID: convID,
		Role:           "assistant",
		Content:        msg.Content,
		Tokens:         resp.Usage.CompletionTokens,
	}
	if err := a.store.CreateMessage(assistantMsg); err != nil {
		a.logger.Warn("Failed to save assistant message", zap.Error(err))
	}

	return &ChatResponse{
		Content:    msg.Content,
		TokensUsed: resp.Usage.TotalTokens,
	}, nil
}

func (a *Agent) chatStream(ctx context.Context, req llm.ChatRequest, convID string, onChunk func(string)) (*ChatResponse, error) {
	var fullContent strings.Builder
	var toolCalls []llm.ToolCall

	err := a.llmClient.ChatCompletionStream(ctx, req, func(chunk llm.StreamResponse) error {
		if len(chunk.Choices) == 0 {
			return nil
		}

		delta := chunk.Choices[0].Delta

		// Accumulate content
		if delta.Content != "" {
			fullContent.WriteString(delta.Content)
			onChunk(delta.Content)
		}

		// Accumulate tool calls
		if len(delta.ToolCalls) > 0 {
			toolCalls = append(toolCalls, delta.ToolCalls...)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("streaming error: %w", err)
	}

	content := fullContent.String()

	// Handle tool calls if any
	if len(toolCalls) > 0 {
		return a.handleToolCalls(ctx, req, convID, toolCalls)
	}

	// Save assistant message
	assistantMsg := &store.Message{
		ConversationID: convID,
		Role:           "assistant",
		Content:        content,
		Tokens:         llm.CountTokens(content),
	}
	if err := a.store.CreateMessage(assistantMsg); err != nil {
		a.logger.Warn("Failed to save assistant message", zap.Error(err))
	}

	return &ChatResponse{
		Content:    content,
		TokensUsed: llm.CountTokens(content),
	}, nil
}

func (a *Agent) handleToolCalls(ctx context.Context, req llm.ChatRequest, convID string, toolCalls []llm.ToolCall) (*ChatResponse, error) {
	// Execute tools
	toolResults := make([]map[string]interface{}, 0, len(toolCalls))

	for _, tc := range toolCalls {
		a.logger.Info("Executing tool",
			zap.String("tool", tc.Function.Name),
			zap.String("args", tc.Function.Arguments),
		)

		result, err := a.tools.ExecuteJSON(ctx, tc.Function.Name, tc.Function.Arguments)
		
		resultObj := map[string]interface{}{
			"tool_call_id": tc.ID,
			"role":         "tool",
			"name":         tc.Function.Name,
		}

		if err != nil {
			resultObj["content"] = fmt.Sprintf("Error: %v", err)
			a.logger.Warn("Tool execution failed",
				zap.String("tool", tc.Function.Name),
				zap.Error(err),
			)
		} else {
			resultStr := fmt.Sprintf("%v", result)
			resultObj["content"] = resultStr
		}

		toolResults = append(toolResults, resultObj)

		// Save tool call and result
		toolMsg := &store.Message{
			ConversationID: convID,
			Role:           "tool",
			Content:        resultObj["content"].(string),
			ToolCalls:      store.ToJSON(tc),
			ToolResults:    store.ToJSON(result),
		}
		if err := a.store.CreateMessage(toolMsg); err != nil {
			a.logger.Warn("Failed to save tool message", zap.Error(err))
		}
	}

	// Build follow-up request with tool results
	followUpMessages := append(req.Messages, llm.Message{
		Role:      "assistant",
		Content:   "",
		ToolCalls: toolCalls,
	})

	for _, tr := range toolResults {
		followUpMessages = append(followUpMessages, llm.Message{
			Role:       "tool",
			Content:    tr["content"].(string),
			ToolCallID: tr["tool_call_id"].(string),
		})
	}

	followUpReq := llm.ChatRequest{
		Model:       req.Model,
		Messages:    followUpMessages,
		Tools:       req.Tools,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	// Get final response
	resp, err := a.llmClient.ChatCompletion(ctx, followUpReq)
	if err != nil {
		return nil, fmt.Errorf("LLM error after tool calls: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM after tool calls")
	}

	finalContent := resp.Choices[0].Message.Content

	// Save final assistant message
	assistantMsg := &store.Message{
		ConversationID: convID,
		Role:           "assistant",
		Content:        finalContent,
		Tokens:         resp.Usage.CompletionTokens,
	}
	if err := a.store.CreateMessage(assistantMsg); err != nil {
		a.logger.Warn("Failed to save assistant message", zap.Error(err))
	}

	return &ChatResponse{
		Content:    finalContent,
		ToolCalls:  toolCalls,
		TokensUsed: resp.Usage.TotalTokens,
	}, nil
}

func (a *Agent) getOrCreateConversation(id string) (*store.Conversation, error) {
	if id == "" {
		// Create new conversation
		conv := &store.Conversation{
			Title: "New Conversation",
		}
		if err := a.store.CreateConversation(conv); err != nil {
			return nil, err
		}
		return conv, nil
	}

	conv, err := a.store.GetConversation(id)
	if err != nil {
		// Create new if not found
		conv = &store.Conversation{
			Title: "New Conversation",
		}
		if err := a.store.CreateConversation(conv); err != nil {
			return nil, err
		}
	}

	return conv, nil
}

func (a *Agent) buildContext(ctx context.Context, convID string, systemPrompt string) ([]llm.Message, error) {
	messages := make([]llm.Message, 0)

	// Add system prompt
	if systemPrompt == "" {
		systemPrompt = a.defaultSystemPrompt()
	}
	messages = append(messages, llm.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// Get recent messages (last 20)
	storeMsgs, err := a.store.GetMessages(convID, 20, 0)
	if err != nil {
		return messages, nil // Return just system prompt if no history
	}

	for _, msg := range storeMsgs {
		lmMsg := llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}

		// Handle tool calls from history
		if len(msg.ToolCalls) > 0 {
			var tcs []llm.ToolCall
			if err := json.Unmarshal(msg.ToolCalls, &tcs); err == nil {
				lmMsg.ToolCalls = tcs
			}
		}

		messages = append(messages, lmMsg)
	}

	return messages, nil
}

func (a *Agent) defaultSystemPrompt() string {
	return `You are Jimmy.ai, a helpful AI assistant running locally on the user's machine.
You have access to tools for file operations, command execution, and web search.

When asked to perform tasks:
1. Use the appropriate tool when needed
2. Explain what you're doing before using tools
3. Be concise but thorough

You can:
- Read and write files (read_file, write_file)
- List directories (list_dir)
- Execute safe shell commands (exec_command)
- Search the web (web_search)
- Fetch URL content (fetch_url)

Always prioritize user privacy and safety.`
}

func (a *Agent) convertTools(defs []map[string]interface{}) []llm.Tool {
	tools := make([]llm.Tool, 0, len(defs))
	for _, def := range defs {
		tool := llm.Tool{
			Type: "function",
		}
		
		if fn, ok := def["function"].(map[string]interface{}); ok {
			tool.Function.Name = getString(fn, "name")
			tool.Function.Description = getString(fn, "description")
			if params, ok := fn["parameters"].(map[string]interface{}); ok {
				tool.Function.Parameters = params
			}
		}
		
		tools = append(tools, tool)
	}
	return tools
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// GenerateTitle generates a title for a conversation
func (a *Agent) GenerateTitle(ctx context.Context, firstMessage string) (string, error) {
	prompt := fmt.Sprintf("Generate a short, concise title (3-5 words) for a conversation that starts with this message: \"%s\". Respond with ONLY the title, no quotes.", firstMessage)
	
	title, err := a.llmClient.SimpleChat(ctx, "You are a helpful assistant.", prompt)
	if err != nil {
		return "New Conversation", nil
	}
	
	title = strings.TrimSpace(title)
	title = strings.Trim(title, `"'`)
	
	if len(title) > 50 {
		title = title[:50] + "..."
	}
	
	return title, nil
}
