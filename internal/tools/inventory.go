package tools

import (
	"fmt"
	"sort"
	"strings"
)

// ToolCategory represents a category of tools
type ToolCategory string

const (
	CategoryToolSystem        ToolCategory = "System"
	CategoryToolFile          ToolCategory = "File"
	CategoryToolBrowser       ToolCategory = "Browser"
	CategoryToolDevOps        ToolCategory = "DevOps"
	CategoryToolGit           ToolCategory = "Git"
	CategoryToolAPI           ToolCategory = "API"
	CategoryToolDatabase      ToolCategory = "Database"
	CategoryToolCloud         ToolCategory = "Cloud"
	CategoryToolCommunication ToolCategory = "Communication"
	CategoryToolCron          ToolCategory = "Cron"
	CategoryToolMemory        ToolCategory = "Memory"
	CategoryToolMCP           ToolCategory = "MCP"
)

// ToolInfo represents metadata about a tool
type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    ToolCategory           `json:"category"`
	Parameters  map[string]interface{} `json:"parameters"`
	Examples    []string               `json:"examples,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

// Inventory manages the categorization of all available tools
type Inventory struct {
	tools map[string]ToolInfo
}

// NewInventory creates a new tool inventory with all built-in tools
func NewInventory() *Inventory {
	inv := &Inventory{
		tools: make(map[string]ToolInfo),
	}
	inv.registerBuiltInTools()
	return inv
}

// registerBuiltInTools registers all 50+ built-in tools
func (i *Inventory) registerBuiltInTools() {
	// System Tools (4)
	systemTools := []ToolInfo{
		{
			Name:        "exec",
			Description: "Execute shell commands",
			Category:    CategoryToolSystem,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Command to execute",
					},
					"timeout": map[string]interface{}{
						"type":        "integer",
						"description": "Timeout in seconds",
					},
				},
				"required": []string{"command"},
			},
			Examples: []string{
				`exec {"command": "ls -la"}`,
				`exec {"command": "docker ps", "timeout": 30}`,
			},
			Tags: []string{"shell", "bash", "command"},
		},
		{
			Name:        "process_list",
			Description: "List running processes",
			Category:    CategoryToolSystem,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filter": map[string]interface{}{
						"type":        "string",
						"description": "Optional filter string",
					},
				},
			},
			Tags: []string{"process", "system", "ps"},
		},
		{
			Name:        "process_kill",
			Description: "Kill a process by PID or name",
			Category:    CategoryToolSystem,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pid": map[string]interface{}{
						"type":        "integer",
						"description": "Process ID",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Process name",
					},
				},
			},
			Tags: []string{"process", "kill", "terminate"},
		},
		{
			Name:        "system_info",
			Description: "Get system information",
			Category:    CategoryToolSystem,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"cpu", "memory", "disk", "os", "all"},
						"description": "Type of system info",
					},
				},
			},
			Tags: []string{"system", "info", "hardware"},
		},
	}

	// File Tools (6)
	fileTools := []ToolInfo{
		{
			Name:        "read_file",
			Description: "Read file contents",
			Category:    CategoryToolFile,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path",
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "Line offset",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Max lines to read",
					},
				},
				"required": []string{"path"},
			},
			Examples: []string{
				`read_file {"path": "/etc/hosts"}`,
				`read_file {"path": "main.go", "limit": 50}`,
			},
			Tags: []string{"file", "read", "cat"},
		},
		{
			Name:        "write_file",
			Description: "Write content to a file",
			Category:    CategoryToolFile,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Content to write",
					},
					"append": map[string]interface{}{
						"type":        "boolean",
						"description": "Append instead of overwrite",
					},
				},
				"required": []string{"path", "content"},
			},
			Tags: []string{"file", "write", "create"},
		},
		{
			Name:        "edit_file",
			Description: "Edit a file with find/replace",
			Category:    CategoryToolFile,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path",
					},
					"old_string": map[string]interface{}{
						"type":        "string",
						"description": "Text to replace",
					},
					"new_string": map[string]interface{}{
						"type":        "string",
						"description": "Replacement text",
					},
				},
				"required": []string{"path", "old_string", "new_string"},
			},
			Tags: []string{"file", "edit", "replace", "sed"},
		},
		{
			Name:        "search_files",
			Description: "Search for text in files",
			Category:    CategoryToolFile,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Search pattern",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory to search",
					},
					"recursive": map[string]interface{}{
						"type":        "boolean",
						"description": "Search recursively",
					},
				},
				"required": []string{"pattern"},
			},
			Tags: []string{"file", "search", "grep", "find"},
		},
		{
			Name:        "list_dir",
			Description: "List directory contents",
			Category:    CategoryToolFile,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory path",
					},
					"recursive": map[string]interface{}{
						"type":        "boolean",
						"description": "List recursively",
					},
				},
			},
			Tags: []string{"file", "directory", "ls"},
		},
		{
			Name:        "diff_files",
			Description: "Compare two files",
			Category:    CategoryToolFile,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file1": map[string]interface{}{
						"type":        "string",
						"description": "First file",
					},
					"file2": map[string]interface{}{
						"type":        "string",
						"description": "Second file",
					},
				},
				"required": []string{"file1", "file2"},
			},
			Tags: []string{"file", "diff", "compare"},
		},
	}

	// Browser Tools (7)
	browserTools := []ToolInfo{
		{
			Name:        "browser_navigate",
			Description: "Navigate to a URL",
			Category:    CategoryToolBrowser,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "URL to navigate to",
					},
					"wait": map[string]interface{}{
						"type":        "boolean",
						"description": "Wait for page load",
					},
				},
				"required": []string{"url"},
			},
			Tags: []string{"browser", "web", "navigate"},
		},
		{
			Name:        "browser_click",
			Description: "Click an element",
			Category:    CategoryToolBrowser,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector",
					},
				},
				"required": []string{"selector"},
			},
			Tags: []string{"browser", "click", "interaction"},
		},
		{
			Name:        "browser_type",
			Description: "Type text into an element",
			Category:    CategoryToolBrowser,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector",
					},
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Text to type",
					},
				},
				"required": []string{"selector", "text"},
			},
			Tags: []string{"browser", "type", "input"},
		},
		{
			Name:        "browser_screenshot",
			Description: "Take a screenshot",
			Category:    CategoryToolBrowser,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Screenshot file path",
					},
					"full_page": map[string]interface{}{
						"type":        "boolean",
						"description": "Capture full page",
					},
				},
			},
			Tags: []string{"browser", "screenshot", "image"},
		},
		{
			Name:        "browser_pdf",
			Description: "Save page as PDF",
			Category:    CategoryToolBrowser,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "PDF file path",
					},
				},
				"required": []string{"path"},
			},
			Tags: []string{"browser", "pdf", "export"},
		},
		{
			Name:        "browser_scroll",
			Description: "Scroll the page",
			Category:    CategoryToolBrowser,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"direction": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"up", "down", "left", "right"},
						"description": "Scroll direction",
					},
					"amount": map[string]interface{}{
						"type":        "integer",
						"description": "Pixels to scroll",
					},
				},
				"required": []string{"direction"},
			},
			Tags: []string{"browser", "scroll"},
		},
		{
			Name:        "browser_form_fill",
			Description: "Fill a form",
			Category:    CategoryToolBrowser,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"fields": map[string]interface{}{
						"type":        "object",
						"description": "Field selectors and values",
					},
				},
				"required": []string{"fields"},
			},
			Tags: []string{"browser", "form", "input"},
		},
	}

	// DevOps Tools (6)
	devopsTools := []ToolInfo{
		{
			Name:        "docker_ps",
			Description: "List Docker containers",
			Category:    CategoryToolDevOps,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"all": map[string]interface{}{
						"type":        "boolean",
						"description": "Show all containers",
					},
				},
			},
			Tags: []string{"docker", "container", "list"},
		},
		{
			Name:        "docker_build",
			Description: "Build Docker image",
			Category:    CategoryToolDevOps,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to Dockerfile",
					},
					"tag": map[string]interface{}{
						"type":        "string",
						"description": "Image tag",
					},
				},
				"required": []string{"path", "tag"},
			},
			Tags: []string{"docker", "build", "image"},
		},
		{
			Name:        "docker_run",
			Description: "Run Docker container",
			Category:    CategoryToolDevOps,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"image": map[string]interface{}{
						"type":        "string",
						"description": "Image name",
					},
					"ports": map[string]interface{}{
						"type":        "array",
						"description": "Port mappings",
					},
					"env": map[string]interface{}{
						"type":        "object",
						"description": "Environment variables",
					},
				},
				"required": []string{"image"},
			},
			Tags: []string{"docker", "run", "container"},
		},
		{
			Name:        "kubectl",
			Description: "Execute kubectl commands",
			Category:    CategoryToolDevOps,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "kubectl command",
					},
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Kubernetes namespace",
					},
				},
				"required": []string{"command"},
			},
			Examples: []string{
				`kubectl {"command": "get pods"}`,
				`kubectl {"command": "apply -f deployment.yaml", "namespace": "default"}`,
			},
			Tags: []string{"kubernetes", "k8s", "deploy"},
		},
		{
			Name:        "helm",
			Description: "Execute Helm commands",
			Category:    CategoryToolDevOps,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Helm command",
					},
					"release": map[string]interface{}{
						"type":        "string",
						"description": "Release name",
					},
					"chart": map[string]interface{}{
						"type":        "string",
						"description": "Chart name",
					},
				},
				"required": []string{"command"},
			},
			Tags: []string{"helm", "kubernetes", "chart"},
		},
		{
			Name:        "terraform",
			Description: "Execute Terraform commands",
			Category:    CategoryToolDevOps,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"init", "plan", "apply", "destroy", "validate"},
						"description": "Terraform command",
					},
					"dir": map[string]interface{}{
						"type":        "string",
						"description": "Working directory",
					},
				},
				"required": []string{"command"},
			},
			Tags: []string{"terraform", "infrastructure", "iac"},
		},
	}

	// Git Tools (6)
	gitTools := []ToolInfo{
		{
			Name:        "git_status",
			Description: "Check git status",
			Category:    CategoryToolGit,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"short": map[string]interface{}{
						"type":        "boolean",
						"description": "Short format",
					},
				},
			},
			Tags: []string{"git", "status"},
		},
		{
			Name:        "git_commit",
			Description: "Commit changes",
			Category:    CategoryToolGit,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Commit message",
					},
					"all": map[string]interface{}{
						"type":        "boolean",
						"description": "Stage all changes",
					},
				},
				"required": []string{"message"},
			},
			Tags: []string{"git", "commit"},
		},
		{
			Name:        "git_push",
			Description: "Push to remote",
			Category:    CategoryToolGit,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"remote": map[string]interface{}{
						"type":        "string",
						"description": "Remote name",
					},
					"branch": map[string]interface{}{
						"type":        "string",
						"description": "Branch name",
					},
				},
			},
			Tags: []string{"git", "push"},
		},
		{
			Name:        "git_branch",
			Description: "Manage branches",
			Category:    CategoryToolGit,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"list", "create", "delete", "switch"},
						"description": "Branch action",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Branch name",
					},
				},
				"required": []string{"action"},
			},
			Tags: []string{"git", "branch"},
		},
		{
			Name:        "git_diff",
			Description: "Show differences",
			Category:    CategoryToolGit,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"staged": map[string]interface{}{
						"type":        "boolean",
						"description": "Show staged changes",
					},
				},
			},
			Tags: []string{"git", "diff"},
		},
		{
			Name:        "git_log",
			Description: "Show commit history",
			Category:    CategoryToolGit,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Number of commits",
					},
					"oneline": map[string]interface{}{
						"type":        "boolean",
						"description": "One line per commit",
					},
				},
			},
			Tags: []string{"git", "log", "history"},
		},
	}

	// API Tools (3)
	apiTools := []ToolInfo{
		{
			Name:        "http_request",
			Description: "Make HTTP requests",
			Category:    CategoryToolAPI,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"method": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
						"description": "HTTP method",
					},
					"url": map[string]interface{}{
						"type":        "string",
						"description": "Request URL",
					},
					"headers": map[string]interface{}{
						"type":        "object",
						"description": "Request headers",
					},
					"body": map[string]interface{}{
						"type":        "string",
						"description": "Request body",
					},
				},
				"required": []string{"method", "url"},
			},
			Examples: []string{
				`http_request {"method": "GET", "url": "https://api.example.com/users"}`,
				`http_request {"method": "POST", "url": "https://api.example.com/users", "body": "{\"name\":\"John\"}"}`,
			},
			Tags: []string{"http", "api", "rest"},
		},
		{
			Name:        "curl",
			Description: "Execute curl command",
			Category:    CategoryToolAPI,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"args": map[string]interface{}{
						"type":        "string",
						"description": "curl arguments",
					},
				},
				"required": []string{"args"},
			},
			Tags: []string{"curl", "http"},
		},
		{
			Name:        "websocket",
			Description: "WebSocket connection",
			Category:    CategoryToolAPI,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "WebSocket URL",
					},
					"action": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"connect", "send", "close"},
						"description": "WebSocket action",
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Message to send",
					},
				},
				"required": []string{"url", "action"},
			},
			Tags: []string{"websocket", "ws", "realtime"},
		},
	}

	// Database Tools (3)
	dbTools := []ToolInfo{
		{
			Name:        "sql_query",
			Description: "Execute SQL query",
			Category:    CategoryToolDatabase,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"connection": map[string]interface{}{
						"type":        "string",
						"description": "Connection string",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "SQL query",
					},
				},
				"required": []string{"query"},
			},
			Examples: []string{
				`sql_query {"query": "SELECT * FROM users LIMIT 10"}`,
			},
			Tags: []string{"sql", "database", "query"},
		},
		{
			Name:        "mongo_query",
			Description: "Execute MongoDB query",
			Category:    CategoryToolDatabase,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"connection": map[string]interface{}{
						"type":        "string",
						"description": "MongoDB URI",
					},
					"database": map[string]interface{}{
						"type":        "string",
						"description": "Database name",
					},
					"collection": map[string]interface{}{
						"type":        "string",
						"description": "Collection name",
					},
					"operation": map[string]interface{}{
						"type":        "string",
						"description": "Query operation",
					},
				},
				"required": []string{"database", "collection", "operation"},
			},
			Tags: []string{"mongodb", "nosql", "database"},
		},
		{
			Name:        "redis_command",
			Description: "Execute Redis command",
			Category:    CategoryToolDatabase,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"connection": map[string]interface{}{
						"type":        "string",
						"description": "Redis connection",
					},
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Redis command",
					},
					"args": map[string]interface{}{
						"type":        "array",
						"description": "Command arguments",
					},
				},
				"required": []string{"command"},
			},
			Tags: []string{"redis", "cache", "database"},
		},
	}

	// Cloud Tools (3)
	cloudTools := []ToolInfo{
		{
			Name:        "aws_cli",
			Description: "Execute AWS CLI commands",
			Category:    CategoryToolCloud,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"service": map[string]interface{}{
						"type":        "string",
						"description": "AWS service",
					},
					"command": map[string]interface{}{
						"type":        "string",
						"description": "AWS command",
					},
				},
				"required": []string{"service", "command"},
			},
			Tags: []string{"aws", "cloud", "amazon"},
		},
		{
			Name:        "gcloud",
			Description: "Execute Google Cloud CLI commands",
			Category:    CategoryToolCloud,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "gcloud command",
					},
				},
				"required": []string{"command"},
			},
			Tags: []string{"gcp", "google", "cloud"},
		},
		{
			Name:        "azure_cli",
			Description: "Execute Azure CLI commands",
			Category:    CategoryToolCloud,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Azure CLI command",
					},
				},
				"required": []string{"command"},
			},
			Tags: []string{"azure", "microsoft", "cloud"},
		},
	}

	// Communication Tools (3)
	commTools := []ToolInfo{
		{
			Name:        "telegram_send",
			Description: "Send Telegram message",
			Category:    CategoryToolCommunication,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chat_id": map[string]interface{}{
						"type":        "string",
						"description": "Chat ID",
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Message text",
					},
				},
				"required": []string{"chat_id", "message"},
			},
			Tags: []string{"telegram", "message", "notification"},
		},
		{
			Name:        "discord_send",
			Description: "Send Discord message",
			Category:    CategoryToolCommunication,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"channel_id": map[string]interface{}{
						"type":        "string",
						"description": "Channel ID",
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Message text",
					},
				},
				"required": []string{"channel_id", "message"},
			},
			Tags: []string{"discord", "message", "notification"},
		},
		{
			Name:        "email_send",
			Description: "Send email",
			Category:    CategoryToolCommunication,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"to": map[string]interface{}{
						"type":        "string",
						"description": "Recipient address",
					},
					"subject": map[string]interface{}{
						"type":        "string",
						"description": "Email subject",
					},
					"body": map[string]interface{}{
						"type":        "string",
						"description": "Email body",
					},
				},
				"required": []string{"to", "subject", "body"},
			},
			Tags: []string{"email", "smtp", "notification"},
		},
	}

	// Cron Tools (3)
	cronTools := []ToolInfo{
		{
			Name:        "schedule_job",
			Description: "Schedule a recurring job",
			Category:    CategoryToolCron,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Job name",
					},
					"schedule": map[string]interface{}{
						"type":        "string",
						"description": "Cron expression",
					},
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Command to run",
					},
				},
				"required": []string{"name", "schedule", "command"},
			},
			Tags: []string{"cron", "schedule", "job"},
		},
		{
			Name:        "list_jobs",
			Description: "List scheduled jobs",
			Category:    CategoryToolCron,
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			Tags: []string{"cron", "list", "jobs"},
		},
		{
			Name:        "delete_job",
			Description: "Delete a scheduled job",
			Category:    CategoryToolCron,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Job name",
					},
				},
				"required": []string{"name"},
			},
			Tags: []string{"cron", "delete", "job"},
		},
	}

	// Memory Tools (3)
	memoryTools := []ToolInfo{
		{
			Name:        "add_memory",
			Description: "Add a memory",
			Category:    CategoryToolMemory,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Memory content",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Memory category",
					},
				},
				"required": []string{"content"},
			},
			Tags: []string{"memory", "store", "remember"},
		},
		{
			Name:        "search_memories",
			Description: "Search memories",
			Category:    CategoryToolMemory,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Max results",
					},
				},
				"required": []string{"query"},
			},
			Tags: []string{"memory", "search", "recall"},
		},
		{
			Name:        "get_neural_cluster",
			Description: "Get neural cluster by ID",
			Category:    CategoryToolMemory,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Cluster ID",
					},
				},
				"required": []string{"id"},
			},
			Tags: []string{"memory", "cluster", "neural"},
		},
	}

	// MCP Tools (dynamic - placeholder)
	mcpTools := []ToolInfo{
		{
			Name:        "mcp_call",
			Description: "Call an MCP server tool",
			Category:    CategoryToolMCP,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"server": map[string]interface{}{
						"type":        "string",
						"description": "MCP server name",
					},
					"tool": map[string]interface{}{
						"type":        "string",
						"description": "Tool name",
					},
					"args": map[string]interface{}{
						"type":        "object",
						"description": "Tool arguments",
					},
				},
				"required": []string{"server", "tool"},
			},
			Tags: []string{"mcp", "external", "plugin"},
		},
	}

	// Register all tools
	allTools := [][]ToolInfo{
		systemTools, fileTools, browserTools, devopsTools,
		gitTools, apiTools, dbTools, cloudTools, commTools,
		cronTools, memoryTools, mcpTools,
	}

	for _, category := range allTools {
		for _, tool := range category {
			i.tools[tool.Name] = tool
		}
	}
}

// GetTool retrieves a tool by name
func (i *Inventory) GetTool(name string) (ToolInfo, bool) {
	tool, ok := i.tools[name]
	return tool, ok
}

// GetToolsByCategory returns all tools in a category
func (i *Inventory) GetToolsByCategory(category ToolCategory) []ToolInfo {
	tools := make([]ToolInfo, 0)
	for _, tool := range i.tools {
		if tool.Category == category {
			tools = append(tools, tool)
		}
	}
	// Sort by name
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})
	return tools
}

// GetAllTools returns all tools
func (i *Inventory) GetAllTools() []ToolInfo {
	tools := make([]ToolInfo, 0, len(i.tools))
	for _, tool := range i.tools {
		tools = append(tools, tool)
	}
	// Sort by category then name
	sort.Slice(tools, func(i, j int) bool {
		if tools[i].Category != tools[j].Category {
			return tools[i].Category < tools[j].Category
		}
		return tools[i].Name < tools[j].Name
	})
	return tools
}

// SearchTools searches tools by name, description, or tags
func (i *Inventory) SearchTools(query string) []ToolInfo {
	query = strings.ToLower(query)
	results := make([]ToolInfo, 0)

	for _, tool := range i.tools {
		if strings.Contains(strings.ToLower(tool.Name), query) ||
			strings.Contains(strings.ToLower(tool.Description), query) {
			results = append(results, tool)
			continue
		}

		// Check tags
		for _, tag := range tool.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, tool)
				break
			}
		}
	}

	return results
}

// GetToolCount returns total number of tools
func (i *Inventory) GetToolCount() int {
	return len(i.tools)
}

// GetCategoryCount returns number of tools in each category
func (i *Inventory) GetCategoryCount() map[ToolCategory]int {
	counts := make(map[ToolCategory]int)
	for _, tool := range i.tools {
		counts[tool.Category]++
	}
	return counts
}

// RegisterTool registers a new tool (for dynamic/MCP tools)
func (i *Inventory) RegisterTool(tool ToolInfo) error {
	if _, exists := i.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}
	i.tools[tool.Name] = tool
	return nil
}

// UnregisterTool removes a tool from inventory
func (i *Inventory) UnregisterTool(name string) error {
	if _, exists := i.tools[name]; !exists {
		return fmt.Errorf("tool %s not found", name)
	}
	delete(i.tools, name)
	return nil
}
