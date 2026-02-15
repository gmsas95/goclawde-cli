package system

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/skills"
)

// SystemSkill provides system-level tools
type SystemSkill struct {
	*skills.BaseSkill
	allowedCommands []string
}

// NewSystemSkill creates a new system skill
func NewSystemSkill(allowedCommands []string) *SystemSkill {
	if len(allowedCommands) == 0 {
		allowedCommands = []string{"ls", "cat", "grep", "find", "pwd", "echo", "mkdir", "touch", "head", "tail", "wc", "df", "du", "ps", "top", "htop"}
	}

	s := &SystemSkill{
		BaseSkill:       skills.NewBaseSkill("system", "System commands and file operations", "1.0.0"),
		allowedCommands: allowedCommands,
	}

	s.registerTools()
	return s
}

func (s *SystemSkill) registerTools() {
	s.AddTool(skills.Tool{
		Name:        "execute_command",
		Description: "Execute a shell command safely",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The command to execute",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Timeout in seconds (default: 30)",
				},
			},
			"required": []string{"command"},
		},
		Handler: s.handleExecuteCommand,
	})

	s.AddTool(skills.Tool{
		Name:        "read_file",
		Description: "Read the contents of a file",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file",
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"description": "Line offset to start from",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum lines to read",
				},
			},
			"required": []string{"path"},
		},
		Handler: s.handleReadFile,
	})

	s.AddTool(skills.Tool{
		Name:        "write_file",
		Description: "Write content to a file",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file",
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
		Handler: s.handleWriteFile,
	})

	s.AddTool(skills.Tool{
		Name:        "list_directory",
		Description: "List files in a directory",
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
		Handler: s.handleListDirectory,
	})

	s.AddTool(skills.Tool{
		Name:        "system_info",
		Description: "Get system information",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: s.handleSystemInfo,
	})
}

func (s *SystemSkill) handleExecuteCommand(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	command, _ := args["command"].(string)
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	cmdParts := strings.Fields(command)
	if len(cmdParts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	baseCmd := cmdParts[0]
	allowed := false
	for _, allowedCmd := range s.allowedCommands {
		if baseCmd == allowedCmd {
			allowed = true
			break
		}
	}

	if !allowed {
		return nil, fmt.Errorf("command '%s' is not in allowed list", baseCmd)
	}

	dangerousPatterns := []string{"rm -rf /", "> /dev/sda", "mkfs", "dd if=/dev/zero"}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return nil, fmt.Errorf("dangerous command blocked")
		}
	}

	timeout := 30
	if t, ok := args["timeout"].(float64); ok {
		timeout = int(t)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()

	result := map[string]interface{}{
		"stdout":    string(output),
		"exit_code": 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		} else {
			result["exit_code"] = 1
			result["error"] = err.Error()
		}
	}

	return result, nil
}

func (s *SystemSkill) handleReadFile(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("path contains invalid characters")
	}

	offset := 0
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}

	limit := 100
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	cmd := exec.Command("cat", path)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(output), "\n")

	if offset >= len(lines) {
		return "", nil
	}

	end := offset + limit
	if end > len(lines) {
		end = len(lines)
	}

	return strings.Join(lines[offset:end], "\n"), nil
}

func (s *SystemSkill) handleWriteFile(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	append, _ := args["append"].(bool)

	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("path contains invalid characters")
	}

	flag := ">"
	if append {
		flag = ">>"
	}

	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo %q %s %s", content, flag, path))
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote to %s", path), nil
}

func (s *SystemSkill) handleListDirectory(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}

	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("path contains invalid characters")
	}

	recursive, _ := args["recursive"].(bool)

	var cmd *exec.Cmd
	if recursive {
		cmd = exec.Command("find", path, "-type", "f", "-o", "-type", "d")
	} else {
		cmd = exec.Command("ls", "-la", path)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	return string(output), nil
}

func (s *SystemSkill) handleSystemInfo(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	info := map[string]interface{}{
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
		"cpus":    runtime.NumCPU(),
		"version": runtime.Version(),
	}

	if hostname, err := exec.Command("hostname").Output(); err == nil {
		info["hostname"] = strings.TrimSpace(string(hostname))
	}

	if output, err := exec.Command("df", "-h").Output(); err == nil {
		info["disk_usage"] = string(output)
	}

	if output, err := exec.Command("free", "-h").Output(); err == nil {
		info["memory"] = string(output)
	}

	return info, nil
}
