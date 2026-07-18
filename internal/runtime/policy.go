package runtime

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/security"
)

// PolicyEngine validates runtime tool calls before execution.
type PolicyEngine struct {
	workspaceRoot string
	allowedCmds   []string
}

// NewPolicyEngine creates a runtime policy engine.
func NewPolicyEngine(workspaceRoot string, allowedCmds []string) *PolicyEngine {
	return &PolicyEngine{
		workspaceRoot: workspaceRoot,
		allowedCmds:   allowedCmds,
	}
}

// ValidateToolCall enforces per-tool runtime policy.
func (p *PolicyEngine) ValidateToolCall(toolCall *llm.ToolCall) error {
	if toolCall == nil {
		return fmt.Errorf("missing tool call")
	}

	var args map[string]interface{}
	if toolCall.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return fmt.Errorf("invalid tool arguments: %w", err)
		}
	}

	switch strings.ToLower(toolCall.Function.Name) {
	case "read_file", "write_file", "edit_file", "append_file", "list_dir":
		return p.validateWorkspacePaths(args)
	case "exec_command":
		if err := p.validateExecCommand(args); err != nil {
			return err
		}
		return p.validateWorkspacePaths(args)
	case "fetch_url", "web_search":
		return p.validateNetworkArgs(args)
	default:
		return nil
	}
}

func (p *PolicyEngine) validateWorkspacePaths(args map[string]interface{}) error {
	if p.workspaceRoot == "" {
		return nil
	}

	pathKeys := []string{"path", "file_path", "filepath", "directory", "dir"}
	for _, key := range pathKeys {
		value, ok := args[key]
		if !ok {
			continue
		}

		path, ok := value.(string)
		if !ok || path == "" {
			continue
		}

		if _, err := security.ValidatePathInWorkspace(path, p.workspaceRoot); err != nil {
			return fmt.Errorf("path policy violation for %s: %w", key, err)
		}
	}

	return nil
}

func (p *PolicyEngine) validateExecCommand(args map[string]interface{}) error {
	command, _ := args["command"].(string)
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("command is required")
	}

	blockedFragments := []string{"&&", "||", ";", "|", ">", "<", "`", "$(", "\n", "\r"}
	for _, fragment := range blockedFragments {
		if strings.Contains(command, fragment) {
			return fmt.Errorf("command policy violation: shell control operators are not allowed")
		}
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("command is required")
	}

	cmdName := filepath.Base(parts[0])
	if len(p.allowedCmds) > 0 {
		allowed := false
		for _, candidate := range p.allowedCmds {
			if cmdName == candidate {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("command policy violation: %s is not allowlisted", cmdName)
		}
	}

	return nil
}

func (p *PolicyEngine) validateNetworkArgs(args map[string]interface{}) error {
	rawURL, _ := args["url"].(string)
	if rawURL == "" {
		return nil
	}

	lower := strings.ToLower(strings.TrimSpace(rawURL))
	if strings.HasPrefix(lower, "file://") || strings.HasPrefix(lower, "gopher://") {
		return fmt.Errorf("network policy violation: unsupported URL scheme")
	}
	if strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") || strings.Contains(lower, "0.0.0.0") {
		return fmt.Errorf("network policy violation: loopback targets are blocked")
	}

	return nil
}
