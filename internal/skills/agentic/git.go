package agentic

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func (s *AgenticSkill) handleGitStatus(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	result := map[string]interface{}{}

	if _, err := exec.CommandContext(ctx, "git", "-C", path, "rev-parse", "--git-dir").Output(); err != nil {
		return nil, fmt.Errorf("not a git repository")
	}

	if output, err := exec.CommandContext(ctx, "git", "-C", path, "status", "--porcelain").Output(); err == nil {
		result["status"] = string(output)
		result["has_changes"] = len(output) > 0
	}

	if output, err := exec.CommandContext(ctx, "git", "-C", path, "branch", "--show-current").Output(); err == nil {
		result["branch"] = strings.TrimSpace(string(output))
	}

	if output, err := exec.CommandContext(ctx, "git", "-C", path, "remote", "-v").Output(); err == nil {
		result["remotes"] = string(output)
	}

	if output, err := exec.CommandContext(ctx, "git", "-C", path, "log", "-1", "--oneline").Output(); err == nil {
		result["last_commit"] = strings.TrimSpace(string(output))
	}

	return result, nil
}

func (s *AgenticSkill) handleGitLog(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	file := ""
	if f, ok := args["file"].(string); ok {
		file = f
	}

	var cmd *exec.Cmd
	if file != "" {
		cmd = exec.CommandContext(ctx, "git", "-C", path, "log", fmt.Sprintf("-%d", limit), "--oneline", file)
	} else {
		cmd = exec.CommandContext(ctx, "git", "-C", path, "log", fmt.Sprintf("-%d", limit), "--oneline")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git log: %w", err)
	}

	commits := []map[string]string{}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			commits = append(commits, map[string]string{
				"hash":    parts[0],
				"message": parts[1],
			})
		}
	}

	return commits, nil
}

func (s *AgenticSkill) handleGitDiff(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	commit := ""
	if c, ok := args["commit"].(string); ok {
		commit = c
	}

	file := ""
	if f, ok := args["file"].(string); ok {
		file = f
	}

	var cmd *exec.Cmd
	if commit != "" {
		if file != "" {
			cmd = exec.CommandContext(ctx, "git", "-C", path, "diff", commit, "--", file)
		} else {
			cmd = exec.CommandContext(ctx, "git", "-C", path, "diff", commit)
		}
	} else {
		if file != "" {
			cmd = exec.CommandContext(ctx, "git", "-C", path, "diff", "--", file)
		} else {
			cmd = exec.CommandContext(ctx, "git", "-C", path, "diff")
		}
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	return map[string]string{
		"diff": string(output),
	}, nil
}

func (s *AgenticSkill) handleGitBlame(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	file, _ := args["file"].(string)
	if file == "" {
		return nil, fmt.Errorf("file is required")
	}

	lineStart := 0
	if ls, ok := args["line_start"].(float64); ok {
		lineStart = int(ls)
	}

	lineEnd := 0
	if le, ok := args["line_end"].(float64); ok {
		lineEnd = int(le)
	}

	var cmd *exec.Cmd
	if lineStart > 0 && lineEnd > 0 {
		cmd = exec.CommandContext(ctx, "git", "blame", "-L", fmt.Sprintf("%d,%d", lineStart, lineEnd), file)
	} else {
		cmd = exec.CommandContext(ctx, "git", "blame", file)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get blame: %w", err)
	}

	return map[string]string{
		"blame": string(output),
	}, nil
}
