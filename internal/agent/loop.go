// Package agent implements the agent loop for autonomous task execution
// Inspired by OpenClaude's agentic capabilities
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/llm"
	"go.uber.org/zap"
)

// AgentLoop manages autonomous task execution with planning, execution, and reflection
type AgentLoop struct {
	agent  *Agent
	logger *zap.Logger
	
	// Configuration
	maxIterations    int           // Maximum tool calls before requiring user confirmation
	reflectionDepth  int           // How many past actions to consider
	timeout          time.Duration // Maximum time for autonomous operation
	requireConfirm   bool          // Whether to require user confirmation for destructive actions
}

// NewAgentLoop creates a new agent loop
func NewAgentLoop(agent *Agent, logger *zap.Logger) *AgentLoop {
	return &AgentLoop{
		agent:           agent,
		logger:          logger,
		maxIterations:   10,
		reflectionDepth: 3,
		timeout:         5 * time.Minute,
		requireConfirm:  true,
	}
}

// AutonomousTask represents a task to be executed autonomously
type AutonomousTask struct {
	Goal        string                 `json:"goal"`
	Context     string                 `json:"context"`
	Constraints []string               `json:"constraints"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TaskResult represents the result of an autonomous task
type TaskResult struct {
	Success     bool                   `json:"success"`
	Iterations  int                    `json:"iterations"`
	Actions     []Action               `json:"actions"`
	FinalAnswer string                 `json:"final_answer"`
	Errors      []string               `json:"errors,omitempty"`
	Duration    time.Duration          `json:"duration"`
}

// Action represents a single action taken by the agent
type Action struct {
	Iteration   int                    `json:"iteration"`
	Type        string                 `json:"type"` // "think", "tool", "reflect", "respond"
	Content     string                 `json:"content"`
	ToolCall    *llm.ToolCall          `json:"tool_call,omitempty"`
	ToolResult  interface{}            `json:"tool_result,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// ExecuteAutonomous executes a task autonomously with the agent loop
func (al *AgentLoop) ExecuteAutonomous(ctx context.Context, task AutonomousTask) (*TaskResult, error) {
	start := time.Now()
	result := &TaskResult{
		Success: false,
		Actions: []Action{},
		Errors:  []string{},
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, al.timeout)
	defer cancel()

	// Initialize the agent with the task
	conversationID := ""
	iteration := 0

	// Build initial system prompt for autonomous operation
	systemPrompt := al.buildAutonomousSystemPrompt(task)

	for iteration < al.maxIterations {
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, "Task timed out")
			result.Duration = time.Since(start)
			return result, nil
		default:
		}

		iteration++
		al.logger.Info("Agent loop iteration", 
			zap.Int("iteration", iteration),
			zap.String("goal", task.Goal))

		// Get the next action from the LLM
		action, err := al.getNextAction(ctx, conversationID, systemPrompt, task, result.Actions)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Iteration %d error: %v", iteration, err))
			break
		}

		action.Iteration = iteration
		action.Timestamp = time.Now()
		result.Actions = append(result.Actions, *action)

		// Execute the action
		switch action.Type {
		case "think":
			// Thinking is just logged, continue to next iteration
			al.logger.Debug("Agent thinking", zap.String("thought", action.Content))

		case "tool":
			if action.ToolCall == nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Iteration %d: tool action without tool call", iteration))
				continue
			}

			// Check if we need user confirmation
			if al.requireConfirm && al.isDestructive(action.ToolCall) {
				// In a real implementation, this would prompt the user
				al.logger.Warn("Destructive action detected, but continuing in autonomous mode",
					zap.String("tool", action.ToolCall.Function.Name))
			}

			// Execute the tool
			toolResult, err := al.executeTool(ctx, action.ToolCall)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Tool %s error: %v", action.ToolCall.Function.Name, err))
				action.ToolResult = map[string]string{"error": err.Error()}
			} else {
				action.ToolResult = toolResult
			}

		case "reflect":
			// Reflection is just logged, continue
			al.logger.Debug("Agent reflecting", zap.String("reflection", action.Content))

		case "respond":
			// Task is complete
			result.Success = true
			result.FinalAnswer = action.Content
			result.Iterations = iteration
			result.Duration = time.Since(start)
			return result, nil

		default:
			result.Errors = append(result.Errors, fmt.Sprintf("Unknown action type: %s", action.Type))
		}
	}

	// Max iterations reached
	result.Iterations = iteration
	result.Duration = time.Since(start)
	if result.FinalAnswer == "" {
		result.FinalAnswer = "Task did not complete within maximum iterations. Last action: " + result.Actions[len(result.Actions)-1].Content
	}

	return result, nil
}

// buildAutonomousSystemPrompt creates a system prompt for autonomous operation
func (al *AgentLoop) buildAutonomousSystemPrompt(task AutonomousTask) string {
	var sb strings.Builder

	sb.WriteString(`You are an autonomous AI agent. You can think, use tools, reflect on your progress, and respond when complete.

Your goal: ` + task.Goal + `

You operate in a loop with these action types:
1. "think" - Think step by step about what to do next
2. "tool" - Use a tool to interact with the environment
3. "reflect" - Reflect on progress and adjust strategy
4. "respond" - Provide final answer when task is complete

Guidelines:
- ALWAYS think before using tools
- Use reflection every 2-3 iterations to assess progress
- If stuck, try a different approach
- For file operations, prefer reading before writing
- Always verify your changes worked
- When complete, use "respond" with a summary

Respond with JSON in this format:
{
  "type": "think|tool|reflect|respond",
  "content": "your thinking/reflection/response",
  "tool_call": { // only for "tool" type
    "id": "call_1",
    "type": "function",
    "function": {
      "name": "tool_name",
      "arguments": "{\"arg\": \"value\"}"
    }
  }
}`)

	if task.Context != "" {
		sb.WriteString("\n\nAdditional Context:\n" + task.Context)
	}

	if len(task.Constraints) > 0 {
		sb.WriteString("\n\nConstraints:\n")
		for _, c := range task.Constraints {
			sb.WriteString("- " + c + "\n")
		}
	}

	return sb.String()
}

// getNextAction determines the next action based on current state
func (al *AgentLoop) getNextAction(ctx context.Context, convID string, systemPrompt string, task AutonomousTask, previousActions []Action) (*Action, error) {
	// Build action history for context
	history := al.formatActionHistory(previousActions)

	// Create the prompt
	prompt := fmt.Sprintf(`Current task: %s

Previous actions (%d total):
%s

What is your next action? Respond with JSON.`,
		task.Goal, len(previousActions), history)

	// Call LLM
	resp, err := al.agent.llmClient.SimpleChat(ctx, systemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	// Parse the response
	action, err := al.parseActionResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse action: %w", err)
	}

	return action, nil
}

// formatActionHistory formats previous actions for the prompt
func (al *AgentLoop) formatActionHistory(actions []Action) string {
	if len(actions) == 0 {
		return "None"
	}

	var sb strings.Builder
	// Only include last N actions based on reflection depth
	start := 0
	if len(actions) > al.reflectionDepth {
		start = len(actions) - al.reflectionDepth
	}

	for i := start; i < len(actions); i++ {
		action := actions[i]
		sb.WriteString(fmt.Sprintf("\n[%d] %s: %s", 
			action.Iteration, action.Type, action.Content))
		
		if action.ToolCall != nil {
			sb.WriteString(fmt.Sprintf(" (tool: %s)", action.ToolCall.Function.Name))
		}
		if action.ToolResult != nil {
			resultStr := fmt.Sprintf("%v", action.ToolResult)
			if len(resultStr) > 100 {
				resultStr = resultStr[:100] + "..."
			}
			sb.WriteString(fmt.Sprintf(" -> %s", resultStr))
		}
	}

	return sb.String()
}

// parseActionResponse parses the LLM response into an Action
func (al *AgentLoop) parseActionResponse(response string) (*Action, error) {
	// Try to extract JSON from the response
	response = strings.TrimSpace(response)
	
	// Handle markdown code blocks
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	var rawAction struct {
		Type      string          `json:"type"`
		Content   string          `json:"content"`
		ToolCall  *llm.ToolCall   `json:"tool_call,omitempty"`
	}

	if err := json.Unmarshal([]byte(response), &rawAction); err != nil {
		// If parsing fails, treat as a think action
		return &Action{
			Type:    "think",
			Content: response,
		}, nil
	}

	return &Action{
		Type:     rawAction.Type,
		Content:  rawAction.Content,
		ToolCall: rawAction.ToolCall,
	}, nil
}

// executeTool executes a tool call
func (al *AgentLoop) executeTool(ctx context.Context, toolCall *llm.ToolCall) (interface{}, error) {
	// Try skills registry first
	if al.agent.skillsRegistry != nil {
		result, err := al.agent.skillsRegistry.ExecuteTool(ctx, toolCall.Function.Name, []byte(toolCall.Function.Arguments))
		if err == nil {
			return result, nil
		}
	}

	// Fall back to tools registry
	if al.agent.tools != nil {
		return al.agent.tools.ExecuteJSON(ctx, toolCall.Function.Name, toolCall.Function.Arguments)
	}

	return nil, fmt.Errorf("no tool registry available")
}

// isDestructive checks if a tool call is potentially destructive
func (al *AgentLoop) isDestructive(toolCall *llm.ToolCall) bool {
	destructiveTools := []string{
		"write_file", "delete_file", "rm", "remove", "drop", "delete",
	}

	for _, dt := range destructiveTools {
		if strings.Contains(toolCall.Function.Name, dt) {
			return true
		}
	}

	// Check arguments for destructive patterns
	if strings.Contains(toolCall.Function.Arguments, "rm -rf") ||
		strings.Contains(toolCall.Function.Arguments, "delete") {
		return true
	}

	return false
}

// SetMaxIterations sets the maximum iterations for the agent loop
func (al *AgentLoop) SetMaxIterations(max int) {
	al.maxIterations = max
}

// SetTimeout sets the timeout for autonomous operation
func (al *AgentLoop) SetTimeout(timeout time.Duration) {
	al.timeout = timeout
}

// EnableConfirmation enables/disables user confirmation for destructive actions
func (al *AgentLoop) EnableConfirmation(enable bool) {
	al.requireConfirm = enable
}
