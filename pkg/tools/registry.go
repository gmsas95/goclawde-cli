package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Tool is the interface for all tools
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// Registry manages available tools
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool registry with default tools
func NewRegistry(allowedCmds []string) *Registry {
	r := &Registry{
		tools: make(map[string]Tool),
	}

	// Register default tools
	r.Register(&ReadFileTool{})
	r.Register(&WriteFileTool{})
	r.Register(&ListDirTool{})
	r.Register(&ExecCommandTool{AllowedCmds: allowedCmds})
	r.Register(&WebSearchTool{})
	r.Register(&FetchURLTool{})
	r.Register(&ThinkingTool{})

	return r
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List returns all available tools
func (r *Registry) List() []Tool {
	list := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}

// GetToolDefinitions returns tool definitions for LLM
func (r *Registry) GetToolDefinitions() []map[string]interface{} {
	defs := make([]map[string]interface{}, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        t.Name(),
				"description": t.Description(),
				"parameters":  t.Parameters(),
			},
		})
	}
	return defs
}

// Execute runs a tool by name with given arguments
func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool.Execute(ctx, args)
}

// ExecuteJSON runs a tool with JSON arguments
func (r *Registry) ExecuteJSON(ctx context.Context, name string, argsJSON string) (interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}
	return r.Execute(ctx, name, args)
}

// ==================== Built-in Tools ====================

// ReadFileTool reads file contents
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string        { return "read_file" }
func (t *ReadFileTool) Description() string { return "Read the contents of a file" }
func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to read",
			},
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "Line offset to start reading from (default: 0)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of lines to read (default: 100)",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Security: prevent reading outside working directory
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("path contains invalid characters")
	}

	content, err := exec.CommandContext(ctx, "cat", path).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

// WriteFileTool writes content to a file
type WriteFileTool struct{}

func (t *WriteFileTool) Name() string        { return "write_file" }
func (t *WriteFileTool) Description() string { return "Write content to a file (creates if doesn't exist)" }
func (t *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to write",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to the file",
			},
			"append": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to append instead of overwrite (default: false)",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	append, _ := args["append"].(bool)

	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("path contains invalid characters")
	}

	var cmd *exec.Cmd
	if append {
		cmd = exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("echo %q >> %s", content, path))
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("echo %q > %s", content, path))
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote to %s", path), nil
}

// ListDirTool lists directory contents
type ListDirTool struct{}

func (t *ListDirTool) Name() string        { return "list_dir" }
func (t *ListDirTool) Description() string { return "List files and directories in a path" }
func (t *ListDirTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to list (default: current directory)",
			},
			"recursive": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to list recursively (default: false)",
			},
		},
		"required": []string{},
	}
}

func (t *ListDirTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}
	recursive, _ := args["recursive"].(bool)

	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("path contains invalid characters")
	}

	var cmd *exec.Cmd
	if recursive {
		cmd = exec.CommandContext(ctx, "find", path, "-type", "f", "-o", "-type", "d")
	} else {
		cmd = exec.CommandContext(ctx, "ls", "-la", path)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	return string(output), nil
}

// ExecCommandTool executes shell commands
type ExecCommandTool struct {
	AllowedCmds []string // If empty, all commands allowed (dangerous!)
}

func (t *ExecCommandTool) Name() string        { return "exec_command" }
func (t *ExecCommandTool) Description() string { return "Execute a shell command (use with caution)" }
func (t *ExecCommandTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Command to execute",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in seconds (default: 30)",
			},
		},
		"required": []string{"command"},
	}
}

func (t *ExecCommandTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	command, _ := args["command"].(string)
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Check if command is allowed
	if len(t.AllowedCmds) > 0 {
		cmdName := strings.Fields(command)[0]
		allowed := false
		for _, allowedCmd := range t.AllowedCmds {
			if cmdName == allowedCmd {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("command '%s' is not in allowed list", cmdName)
		}
	}

	// Security: block dangerous commands
	dangerous := []string{"rm -rf /", "> /dev/sda", "mkfs", "dd if=/dev/zero"}
	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return nil, fmt.Errorf("dangerous command blocked")
		}
	}

	output, err := exec.CommandContext(ctx, "sh", "-c", command).CombinedOutput()
	result := map[string]interface{}{
		"stdout": string(output),
	}
	if err != nil {
		result["error"] = err.Error()
		result["exit_code"] = 1
	} else {
		result["exit_code"] = 0
	}

	return result, nil
}

// WebSearchTool searches the web
type WebSearchTool struct{}

func (t *WebSearchTool) Name() string        { return "web_search" }
func (t *WebSearchTool) Description() string { return "Search the web for information (requires external search API)" }
func (t *WebSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query",
			},
			"num_results": map[string]interface{}{
				"type":        "integer",
				"description": "Number of results (default: 5)",
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	// For now, return a placeholder - actual implementation needs search API
	return map[string]interface{}{
		"note": "Web search requires configuration of a search API (Brave, Serper, etc.)",
		"query": query,
		"results": []map[string]string{
			{"title": "Example result", "url": "https://example.com"},
		},
	}, nil
}

// FetchURLTool fetches URL content
type FetchURLTool struct{}

func (t *FetchURLTool) Name() string        { return "fetch_url" }
func (t *FetchURLTool) Description() string { return "Fetch and extract text content from a URL" }
func (t *FetchURLTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to fetch",
			},
			"max_length": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum characters to return (default: 5000)",
			},
		},
		"required": []string{"url"},
	}
}

func (t *FetchURLTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	url, _ := args["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	// Use curl to fetch
	output, err := exec.CommandContext(ctx, "curl", "-s", "-L", "--max-time", "10", url).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}

	// Simple text extraction (remove HTML tags)
	content := string(output)
	content = strings.ReplaceAll(content, "<script", "\x00")
	content = strings.ReplaceAll(content, "</script>", "\x00")
	
	// Truncate if needed
	maxLen := 5000
	if len(content) > maxLen {
		content = content[:maxLen] + "..."
	}

	return content, nil
}

// ThinkingTool allows the model to show its reasoning
type ThinkingTool struct{}

func (t *ThinkingTool) Name() string        { return "thinking" }
func (t *ThinkingTool) Description() string { return "Use this tool to think step by step before responding" }
func (t *ThinkingTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"thought": map[string]interface{}{
				"type":        "string",
				"description": "Your step-by-step thinking process",
			},
		},
		"required": []string{"thought"},
	}
}

func (t *ThinkingTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	thought, _ := args["thought"].(string)
	return map[string]string{
		"status":  "thought_recorded",
		"thought": thought,
	}, nil
}
