package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/persona"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/gmsas95/myrai-cli/pkg/tools"
	"go.uber.org/zap"
)

// Agent handles conversation and tool execution
type Agent struct {
	llmClient      *llm.Client
	tools          *tools.Registry
	skillsRegistry *skills.Registry
	store          *store.Store
	logger         *zap.Logger
	personaManager *persona.PersonaManager
	contextManager *ContextManager
	agentLoop      *AgentLoop
}

// New creates a new Agent
func New(llmClient *llm.Client, toolRegistry *tools.Registry, store *store.Store, logger *zap.Logger, personaManager *persona.PersonaManager) *Agent {
	return &Agent{
		llmClient:      llmClient,
		tools:          toolRegistry,
		store:          store,
		logger:         logger,
		personaManager: personaManager,
	}
}

// SetContextManager sets the context manager for smart context handling
func (a *Agent) SetContextManager(cm *ContextManager) {
	a.contextManager = cm
}

// SetAgentLoop sets the agent loop for autonomous task execution
func (a *Agent) SetAgentLoop(al *AgentLoop) {
	a.agentLoop = al
}

// ExecuteAutonomous executes a task autonomously using the agent loop
func (a *Agent) ExecuteAutonomous(ctx context.Context, task AutonomousTask) (*TaskResult, error) {
	if a.agentLoop == nil {
		// Create agent loop on demand if not set
		a.agentLoop = NewAgentLoop(a, a.logger)
	}
	return a.agentLoop.ExecuteAutonomous(ctx, task)
}

// SetSkillsRegistry sets the skills registry
func (a *Agent) SetSkillsRegistry(registry *skills.Registry) {
	a.skillsRegistry = registry
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
	Content        string
	ConversationID string
	ToolCalls      []llm.ToolCall
	TokensUsed     int
	ResponseTime   time.Duration
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

	// Build message history using context manager if available
	var messages []llm.Message
	if a.contextManager != nil {
		convCtx, err := a.contextManager.BuildContext(ctx, conv.ID, req.SystemPrompt, req.Message)
		if err != nil {
			a.logger.Warn("Context manager failed, falling back to default", zap.Error(err))
			messages, _ = a.buildContext(ctx, conv.ID, req.SystemPrompt)
		} else {
			messages = convCtx.Messages
		}
	} else {
		messages, err = a.buildContext(ctx, conv.ID, req.SystemPrompt)
		if err != nil {
			return nil, fmt.Errorf("failed to build context: %w", err)
		}
	}

	// Build tool definitions from both tools and skills registries
	var toolDefs []map[string]interface{}
	if a.tools != nil {
		toolDefs = a.tools.GetToolDefinitions()
	}
	if a.skillsRegistry != nil {
		toolDefs = append(toolDefs, a.skillsRegistry.GetToolDefinitions()...)
	}

	// Call LLM
	tools := a.convertTools(toolDefs)
	llmReq := llm.ChatRequest{
		Model:             a.llmClient.GetModel(),
		Messages:          messages,
		Tools:             tools,
		MaxTokens:         4096,
		Stream:            req.Stream,
		ParallelToolCalls: len(tools) > 0, // Disable parallel tool calls for better reliability
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
		return a.handleToolCalls(ctx, req, convID, msg)
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

	// Extract memories from this conversation turn (async)
	if a.contextManager != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Get the user message from the conversation
			userMsgs, _ := a.store.GetMessages(convID, 1, 0)
			var userContent string
			if len(userMsgs) > 0 && userMsgs[0].Role == "user" {
				userContent = userMsgs[0].Content
			}

			if err := a.contextManager.ExtractAndStoreMemories(ctx, convID, userContent, msg.Content); err != nil {
				a.logger.Debug("Failed to extract memories", zap.Error(err))
			}
		}()
	}

	return &ChatResponse{
		Content:        msg.Content,
		ConversationID: convID,
		TokensUsed:     resp.Usage.TotalTokens,
	}, nil
}

func (a *Agent) chatStream(ctx context.Context, req llm.ChatRequest, convID string, onChunk func(string)) (*ChatResponse, error) {
	handler := llm.NewStreamHandler(onChunk, nil)

	err := a.llmClient.ChatCompletionStream(ctx, req, func(chunk llm.StreamResponse) error {
		done, toolCalls, err := handler.HandleChunk(chunk)
		if err != nil {
			return err
		}
		if done {
			// Handle completion in the callback
			if len(toolCalls) > 0 {
				// Tool calls detected - will be handled after stream ends
				return nil
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("streaming error: %w", err)
	}

	content := handler.GetContent()

	// Check for tool calls from handler
	// Note: We need to access the accumulator's tool calls
	// For now, let's check if there were any tool calls in the request
	// and handle them properly

	// Handle tool calls if any (simplified - in real implementation,
	// we'd track if tool calls were detected)
	// For now, return the content

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

func (a *Agent) handleToolCalls(ctx context.Context, req llm.ChatRequest, convID string, msg llm.Message) (*ChatResponse, error) {
	toolCalls := msg.ToolCalls
	// Execute tools
	toolResults := make([]map[string]interface{}, 0, len(toolCalls))

	for _, tc := range toolCalls {
		a.logger.Info("Executing tool",
			zap.String("tool", tc.Function.Name),
			zap.String("args", tc.Function.Arguments),
		)

		// Try skills registry first
		var result interface{}
		var err error

		if a.skillsRegistry != nil {
			result, err = a.skillsRegistry.ExecuteTool(ctx, tc.Function.Name, []byte(tc.Function.Arguments))
		} else if a.tools != nil {
			result, err = a.tools.ExecuteJSON(ctx, tc.Function.Name, tc.Function.Arguments)
		} else {
			err = fmt.Errorf("no tool registry available")
		}

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
			ToolCallID:     tc.ID, // Save the tool_call_id
		}
		if err := a.store.CreateMessage(toolMsg); err != nil {
			a.logger.Warn("Failed to save tool message", zap.Error(err))
		}
	}

	// Build follow-up request with tool results
	// Note: Content must be omitted (not empty string) when ToolCalls are present
	// Include reasoning_content if present (required by some LLM APIs with thinking enabled)
	followUpMessages := append(req.Messages, llm.Message{
		Role:             "assistant",
		ToolCalls:        toolCalls,
		ReasoningContent: msg.ReasoningContent, // Preserve reasoning content
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
		Content:        finalContent,
		ConversationID: convID,
		ToolCalls:      toolCalls,
		TokensUsed:     resp.Usage.TotalTokens,
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

	// Build system prompt with persona context (cached internally by personaManager)
	if systemPrompt == "" {
		systemPrompt = a.buildSystemPrompt()
	}

	// Preallocate message slice with capacity for efficiency
	messages = make([]llm.Message, 0, 25)
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
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID, // Restore tool_call_id from history
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

func (a *Agent) buildSystemPrompt() string {
	// Add persona context if available, otherwise use default
	if a.personaManager != nil {
		return a.personaManager.GetSystemPrompt()
	}
	return a.defaultSystemPrompt()
}

func (a *Agent) defaultSystemPrompt() string {
	return `You are Myrai, a helpful AI assistant running locally on the user's machine with access to real-time information.

IMPORTANT - WHEN TO USE WEB SEARCH:
You MUST use the web_search tool when:
- The user asks about current events, news, or recent developments
- You need up-to-date information (your training data has a cutoff date)
- The user asks about time-sensitive topics (weather, stock prices, sports scores, etc.)
- You need to verify facts that may have changed recently
- The user asks "What's the latest...", "Current...", "Recent...", "Today...", "News..."

Guidelines:
- Be concise but thorough
- Explain what you're doing before using tools
- Confirm destructive operations before proceeding
- Prioritize user privacy and safety
- Use web_search proactively for real-time information needs
- After searching, summarize findings clearly and cite sources

Available tools:
- read_file, write_file - File operations
- list_dir - List directory contents  
- exec_command - Execute safe shell commands
- web_search - Search the web for current information (USE THIS for real-time data!)
- fetch_url - Fetch URL content
- get_weather - Weather information
- github_search_repos, github_get_file, etc. - GitHub operations`
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
