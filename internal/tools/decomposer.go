package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/llm"
)

// Task represents a single step in a decomposed task
type Task struct {
	ID            string                 `json:"id"`
	Description   string                 `json:"description"`
	Dependencies  []string               `json:"dependencies"`
	Tools         []string               `json:"tools"`
	EstimatedTime time.Duration          `json:"estimated_time"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	Condition     string                 `json:"condition,omitempty"`
	Optional      bool                   `json:"optional,omitempty"`
}

// TaskDecomposer breaks down complex intents into executable tasks
type TaskDecomposer struct {
	llmClient *llm.Client
}

// NewTaskDecomposer creates a new task decomposer
func NewTaskDecomposer(llmClient *llm.Client) *TaskDecomposer {
	return &TaskDecomposer{
		llmClient: llmClient,
	}
}

// Decompose breaks down an intent into a series of tasks
func (td *TaskDecomposer) Decompose(ctx context.Context, intent *Intent, availableTools []ToolMetadata) ([]Task, error) {
	if intent.Complexity == ComplexitySimple {
		// Simple intents don't need decomposition
		return []Task{
			{
				ID:            "task-001",
				Description:   intent.RawInput,
				Dependencies:  []string{},
				Tools:         suggestTools(intent, availableTools),
				EstimatedTime: 30 * time.Second,
			},
		}, nil
	}

	if td.llmClient == nil {
		return td.ruleBasedDecompose(intent, availableTools)
	}

	// Use LLM for decomposition
	toolsJSON, _ := json.Marshal(availableTools)

	systemPrompt := fmt.Sprintf(`You are a task decomposition expert. Break down the following task into sequential steps.

Intent Type: %s
Category: %s
Complexity: %s

Available Tools:
%s

Respond with a JSON array of tasks. Each task must have:
- id: unique identifier (task-001, task-002, etc.)
- description: clear description of what to do
- dependencies: array of task IDs this depends on (can be empty)
- tools: array of suggested tool names from the available tools
- estimated_time: estimated duration in seconds (as number)
- parameters: optional parameters for the tools
- condition: optional condition for execution (for conditional chains)
- optional: boolean indicating if this step is optional

Example response:
[
  {
    "id": "task-001",
    "description": "Build Docker image",
    "dependencies": [],
    "tools": ["docker_build"],
    "estimated_time": 120,
    "parameters": {"path": ".", "tag": "myapp:latest"}
  },
  {
    "id": "task-002",
    "description": "Push to registry",
    "dependencies": ["task-001"],
    "tools": ["docker_push"],
    "estimated_time": 60
  }
]`, intent.Type, intent.Category, intent.Complexity, string(toolsJSON))

	response, err := td.llmClient.SimpleChat(ctx, systemPrompt, intent.RawInput)
	if err != nil {
		return nil, fmt.Errorf("failed to decompose task: %w", err)
	}

	// Parse JSON response
	var tasks []Task
	if err := json.Unmarshal([]byte(extractJSON(response)), &tasks); err != nil {
		// Fallback to rule-based if parsing fails
		return td.ruleBasedDecompose(intent, availableTools)
	}

	// Validate and set defaults
	for i := range tasks {
		if tasks[i].ID == "" {
			tasks[i].ID = fmt.Sprintf("task-%03d", i+1)
		}
		if tasks[i].EstimatedTime == 0 {
			tasks[i].EstimatedTime = 30 * time.Second
		}
		if tasks[i].Dependencies == nil {
			tasks[i].Dependencies = []string{}
		}
		if tasks[i].Tools == nil {
			tasks[i].Tools = []string{}
		}
	}

	return tasks, nil
}

// ruleBasedDecompose provides fallback decomposition without LLM
func (td *TaskDecomposer) ruleBasedDecompose(intent *Intent, availableTools []ToolMetadata) ([]Task, error) {
	tasks := make([]Task, 0)
	inputLower := strings.ToLower(intent.RawInput)

	// Pattern-based decomposition for common tasks

	// Deployment workflow
	if strings.Contains(inputLower, "deploy") && (strings.Contains(inputLower, "docker") || strings.Contains(inputLower, "container")) {
		tasks = append(tasks, Task{
			ID:            "build",
			Description:   "Build Docker image",
			Dependencies:  []string{},
			Tools:         []string{"docker_build", "exec"},
			EstimatedTime: 120 * time.Second,
		})
		tasks = append(tasks, Task{
			ID:            "push",
			Description:   "Push image to registry",
			Dependencies:  []string{"build"},
			Tools:         []string{"docker_push", "exec"},
			EstimatedTime: 60 * time.Second,
		})
		tasks = append(tasks, Task{
			ID:            "deploy",
			Description:   "Deploy to Kubernetes",
			Dependencies:  []string{"push"},
			Tools:         []string{"kubectl", "exec"},
			EstimatedTime: 60 * time.Second,
		})
		return tasks, nil
	}

	// CI/CD setup workflow
	if strings.Contains(inputLower, "ci/cd") || strings.Contains(inputLower, "pipeline") {
		tasks = append(tasks, Task{
			ID:            "analyze",
			Description:   "Analyze project structure",
			Dependencies:  []string{},
			Tools:         []string{"list_dir", "read_file"},
			EstimatedTime: 30 * time.Second,
		})
		tasks = append(tasks, Task{
			ID:            "config",
			Description:   "Create CI/CD configuration",
			Dependencies:  []string{"analyze"},
			Tools:         []string{"write_file", "exec"},
			EstimatedTime: 60 * time.Second,
		})
		tasks = append(tasks, Task{
			ID:            "verify",
			Description:   "Verify configuration",
			Dependencies:  []string{"config"},
			Tools:         []string{"exec"},
			EstimatedTime: 30 * time.Second,
		})
		return tasks, nil
	}

	// Database migration workflow
	if strings.Contains(inputLower, "migration") || strings.Contains(inputLower, "database") {
		tasks = append(tasks, Task{
			ID:            "backup",
			Description:   "Backup current database",
			Dependencies:  []string{},
			Tools:         []string{"sql_query", "exec"},
			EstimatedTime: 120 * time.Second,
		})
		tasks = append(tasks, Task{
			ID:            "migrate",
			Description:   "Run database migrations",
			Dependencies:  []string{"backup"},
			Tools:         []string{"sql_query", "exec"},
			EstimatedTime: 180 * time.Second,
		})
		tasks = append(tasks, Task{
			ID:            "verify",
			Description:   "Verify migration success",
			Dependencies:  []string{"migrate"},
			Tools:         []string{"sql_query"},
			EstimatedTime: 30 * time.Second,
		})
		return tasks, nil
	}

	// Default: single task
	tasks = append(tasks, Task{
		ID:            "task-001",
		Description:   intent.RawInput,
		Dependencies:  []string{},
		Tools:         suggestTools(intent, availableTools),
		EstimatedTime: 60 * time.Second,
	})

	return tasks, nil
}

// ToolMetadata represents metadata about an available tool
type ToolMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// suggestTools suggests appropriate tools for an intent
func suggestTools(intent *Intent, availableTools []ToolMetadata) []string {
	suggestions := make([]string, 0)
	inputLower := strings.ToLower(intent.RawInput)

	// Map keywords to tools
	toolKeywords := map[string][]string{
		"exec":         {"run", "execute", "command", "shell", "bash"},
		"read_file":    {"read", "file", "content", "view"},
		"write_file":   {"write", "create", "file", "save"},
		"list_dir":     {"list", "directory", "folder", "ls"},
		"docker":       {"docker", "container", "image"},
		"kubectl":      {"kubernetes", "k8s", "pod", "deployment"},
		"git":          {"git", "commit", "push", "pull", "branch"},
		"search":       {"search", "find", "lookup"},
		"browser":      {"browser", "web", "url", "page"},
		"sql_query":    {"database", "sql", "query", "db"},
		"http_request": {"http", "api", "request", "endpoint"},
	}

	for tool, keywords := range toolKeywords {
		for _, keyword := range keywords {
			if strings.Contains(inputLower, keyword) {
				suggestions = append(suggestions, tool)
				break
			}
		}
	}

	return uniqueStrings(suggestions)
}

// uniqueStrings removes duplicates from a string slice
func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// GetTotalTime calculates the total estimated time for all tasks
func GetTotalTime(tasks []Task) time.Duration {
	var total time.Duration
	for _, task := range tasks {
		total += task.EstimatedTime
	}
	return total
}

// ValidateDependencies checks if all task dependencies exist
func ValidateDependencies(tasks []Task) error {
	taskIDs := make(map[string]bool)
	for _, task := range tasks {
		taskIDs[task.ID] = true
	}

	for _, task := range tasks {
		for _, dep := range task.Dependencies {
			if !taskIDs[dep] {
				return fmt.Errorf("task %s depends on non-existent task %s", task.ID, dep)
			}
		}
	}

	return nil
}

// GetExecutionOrder returns tasks in dependency-resolved order
func GetExecutionOrder(tasks []Task) ([]Task, error) {
	if err := ValidateDependencies(tasks); err != nil {
		return nil, err
	}

	// Build dependency graph
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, task := range tasks {
		if _, exists := inDegree[task.ID]; !exists {
			inDegree[task.ID] = 0
		}
		for _, dep := range task.Dependencies {
			graph[dep] = append(graph[dep], task.ID)
			inDegree[task.ID]++
		}
	}

	// Kahn's algorithm for topological sort
	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	ordered := make([]string, 0)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		ordered = append(ordered, current)

		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(ordered) != len(tasks) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	// Map back to tasks
	taskMap := make(map[string]Task)
	for _, task := range tasks {
		taskMap[task.ID] = task
	}

	result := make([]Task, len(ordered))
	for i, id := range ordered {
		result[i] = taskMap[id]
	}

	return result, nil
}
