// Package agentic provides advanced agentic capabilities for autonomous operation
// Inspired by OpenClaude's agentic features - system introspection, code understanding,
// project analysis, and task planning.
package agentic

import (
	"bufio"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/skills"
)

// AgenticSkill provides advanced agentic capabilities
type AgenticSkill struct {
	*skills.BaseSkill
	workspaceRoot string
}

// NewAgenticSkill creates a new agentic skill
func NewAgenticSkill(workspaceRoot string) *AgenticSkill {
	s := &AgenticSkill{
		BaseSkill:     skills.NewBaseSkill("agentic", "Advanced agentic capabilities for system and code analysis", "1.0.0"),
		workspaceRoot: workspaceRoot,
	}
	s.registerTools()
	return s
}

func (s *AgenticSkill) registerTools() {
	// System introspection
	s.AddTool(skills.Tool{
		Name:        "get_system_resources",
		Description: "Get detailed system resource information (CPU, memory, disk, processes)",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: s.handleGetSystemResources,
	})

	s.AddTool(skills.Tool{
		Name:        "list_processes",
		Description: "List running processes with resource usage",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of processes to return (default: 20)",
				},
				"filter": map[string]interface{}{
					"type":        "string",
					"description": "Filter by process name (optional)",
				},
			},
		},
		Handler: s.handleListProcesses,
	})

	s.AddTool(skills.Tool{
		Name:        "get_network_info",
		Description: "Get network configuration and connections",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: s.handleGetNetworkInfo,
	})

	s.AddTool(skills.Tool{
		Name:        "get_environment",
		Description: "Get environment variables and system configuration",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"filter": map[string]interface{}{
					"type":        "string",
					"description": "Filter env vars by prefix (e.g., 'GO', 'PATH')",
				},
			},
		},
		Handler: s.handleGetEnvironment,
	})

	// Code analysis
	s.AddTool(skills.Tool{
		Name:        "analyze_project_structure",
		Description: "Analyze project structure, identify language, framework, and key files",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Project root path (default: current directory)",
				},
				"depth": map[string]interface{}{
					"type":        "integer",
					"description": "Directory depth to analyze (default: 3)",
				},
			},
		},
		Handler: s.handleAnalyzeProjectStructure,
	})

	s.AddTool(skills.Tool{
		Name:        "analyze_code_file",
		Description: "Analyze a code file - extract functions, classes, imports, TODOs",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the code file",
				},
			},
			"required": []string{"path"},
		},
		Handler: s.handleAnalyzeCodeFile,
	})

	s.AddTool(skills.Tool{
		Name:        "search_code",
		Description: "Search for code patterns across the project (grep with context)",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Search pattern (regex supported)",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Root path to search (default: current directory)",
				},
				"file_pattern": map[string]interface{}{
					"type":        "string",
					"description": "File pattern to match (e.g., '*.go', '*.js')",
				},
				"context": map[string]interface{}{
					"type":        "integer",
					"description": "Lines of context (default: 2)",
				},
			},
			"required": []string{"pattern"},
		},
		Handler: s.handleSearchCode,
	})

	s.AddTool(skills.Tool{
		Name:        "find_todos",
		Description: "Find TODO, FIXME, HACK comments across the codebase",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Root path (default: current directory)",
				},
			},
		},
		Handler: s.handleFindTodos,
	})

	// Git integration
	s.AddTool(skills.Tool{
		Name:        "git_status",
		Description: "Get git repository status",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Repository path (default: current directory)",
				},
			},
		},
		Handler: s.handleGitStatus,
	})

	s.AddTool(skills.Tool{
		Name:        "git_log",
		Description: "Get recent git commit history",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Repository path (default: current directory)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Number of commits (default: 10)",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Filter to specific file (optional)",
				},
			},
		},
		Handler: s.handleGitLog,
	})

	s.AddTool(skills.Tool{
		Name:        "git_diff",
		Description: "Get git diff for current changes or specific commit",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Repository path (default: current directory)",
				},
				"commit": map[string]interface{}{
					"type":        "string",
					"description": "Specific commit hash (default: unstaged changes)",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Specific file to diff (optional)",
				},
			},
		},
		Handler: s.handleGitDiff,
	})

	s.AddTool(skills.Tool{
		Name:        "git_blame",
		Description: "Get git blame for a file",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "File path",
				},
				"line_start": map[string]interface{}{
					"type":        "integer",
					"description": "Start line (optional)",
				},
				"line_end": map[string]interface{}{
					"type":        "integer",
					"description": "End line (optional)",
				},
			},
			"required": []string{"file"},
		},
		Handler: s.handleGitBlame,
	})

	// Task planning
	s.AddTool(skills.Tool{
		Name:        "create_task_plan",
		Description: "Create a structured plan for completing a complex task",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"goal": map[string]interface{}{
					"type":        "string",
					"description": "The goal to achieve",
				},
				"steps": map[string]interface{}{
					"type":        "array",
					"description": "List of steps to complete",
				},
			},
			"required": []string{"goal"},
		},
		Handler: s.handleCreateTaskPlan,
	})

	s.AddTool(skills.Tool{
		Name:        "reflect_on_task",
		Description: "Reflect on task progress and suggest next steps or corrections",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task": map[string]interface{}{
					"type":        "string",
					"description": "Current task description",
				},
				"actions_taken": map[string]interface{}{
					"type":        "array",
					"description": "List of actions taken so far",
				},
				"current_state": map[string]interface{}{
					"type":        "string",
					"description": "Current state of the task",
				},
			},
			"required": []string{"task"},
		},
		Handler: s.handleReflectOnTask,
	})
}

// ==================== System Introspection Handlers ====================

func (s *AgenticSkill) handleGetSystemResources(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	result := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"cpus":      runtime.NumCPU(),
	}

	// CPU info
	if output, err := exec.CommandContext(ctx, "cat", "/proc/cpuinfo").Output(); err == nil {
		cpus := parseCPUInfo(string(output))
		result["cpu_info"] = cpus
	}

	// Memory info
	if output, err := exec.CommandContext(ctx, "cat", "/proc/meminfo").Output(); err == nil {
		memInfo := parseMemInfo(string(output))
		result["memory"] = memInfo
	}

	// Disk usage
	if output, err := exec.CommandContext(ctx, "df", "-h").Output(); err == nil {
		result["disk_usage"] = string(output)
	}

	// Load average
	if output, err := exec.CommandContext(ctx, "cat", "/proc/loadavg").Output(); err == nil {
		result["load_average"] = strings.TrimSpace(string(output))
	}

	// Uptime
	if output, err := exec.CommandContext(ctx, "uptime", "-p").Output(); err == nil {
		result["uptime"] = strings.TrimSpace(string(output))
	}

	return result, nil
}

func (s *AgenticSkill) handleListProcesses(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	filter := ""
	if f, ok := args["filter"].(string); ok {
		filter = f
	}

	var cmd *exec.Cmd
	if filter != "" {
		cmd = exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("ps aux | grep -i %s | head -%d", filter, limit))
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("ps aux --sort=-%%mem | head -%d", limit+1))
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	processes := []map[string]string{}

	// Skip header
	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 11 {
			processes = append(processes, map[string]string{
				"user":    fields[0],
				"pid":     fields[1],
				"cpu":     fields[2],
				"mem":     fields[3],
				"command": strings.Join(fields[10:], " "),
			})
		}
	}

	return processes, nil
}

func (s *AgenticSkill) handleGetNetworkInfo(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	result := map[string]interface{}{}

	// Network interfaces
	if output, err := exec.CommandContext(ctx, "ip", "addr").Output(); err == nil {
		result["interfaces"] = string(output)
	} else if output, err := exec.CommandContext(ctx, "ifconfig").Output(); err == nil {
		result["interfaces"] = string(output)
	}

	// Routing table
	if output, err := exec.CommandContext(ctx, "ip", "route").Output(); err == nil {
		result["routes"] = string(output)
	}

	// Active connections
	if output, err := exec.CommandContext(ctx, "ss", "-tuln").Output(); err == nil {
		result["listening_ports"] = string(output)
	} else if output, err := exec.CommandContext(ctx, "netstat", "-tuln").Output(); err == nil {
		result["listening_ports"] = string(output)
	}

	// DNS
	if output, err := exec.CommandContext(ctx, "cat", "/etc/resolv.conf").Output(); err == nil {
		result["dns"] = string(output)
	}

	return result, nil
}

func (s *AgenticSkill) handleGetEnvironment(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	filter := ""
	if f, ok := args["filter"].(string); ok {
		filter = f
	}

	env := map[string]string{}
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			if filter == "" || strings.HasPrefix(parts[0], filter) {
				env[parts[0]] = parts[1]
			}
		}
	}

	// Add Go-specific info
	env["GO_VERSION"] = runtime.Version()
	env["GO_OS"] = runtime.GOOS
	env["GO_ARCH"] = runtime.GOARCH

	return env, nil
}

// ==================== Code Analysis Handlers ====================

func (s *AgenticSkill) handleAnalyzeProjectStructure(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	depth := 3
	if d, ok := args["depth"].(float64); ok {
		depth = int(d)
	}

	result := map[string]interface{}{
		"path":         path,
		"languages":    detectLanguages(path),
		"frameworks":   detectFrameworks(path),
		"entry_points": findEntryPoints(path),
		"structure":    buildDirTree(path, depth),
	}

	// Count files by type
	fileCounts := countFileTypes(path)
	result["file_counts"] = fileCounts

	return result, nil
}

func (s *AgenticSkill) handleAnalyzeCodeFile(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	result := map[string]interface{}{
		"path":         path,
		"size_bytes":   len(content),
		"line_count":   strings.Count(string(content), "\n") + 1,
		"todos":        extractTODOs(string(content)),
		"imports":      []string{},
		"functions":    []map[string]interface{}{},
		"classes":      []map[string]interface{}{},
	}

	// Language-specific analysis
	if strings.HasSuffix(path, ".go") {
		analyzeGoFile(content, result)
	} else if strings.HasSuffix(path, ".py") {
		analyzePythonFile(string(content), result)
	} else if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".ts") {
		analyzeJSFile(string(content), result)
	}

	return result, nil
}

func (s *AgenticSkill) handleSearchCode(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}

	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	filePattern := "*"
	if fp, ok := args["file_pattern"].(string); ok && fp != "" {
		filePattern = fp
	}

	context := 2
	if c, ok := args["context"].(float64); ok {
		context = int(c)
	}

	// Use grep for searching
	cmd := exec.CommandContext(ctx, "sh", "-c",
		fmt.Sprintf("grep -rn --include=%q -C %d %q %s 2>/dev/null | head -100",
			filePattern, context, pattern, path))

	output, err := cmd.Output()
	if err != nil {
		// No matches isn't an error
		return []map[string]interface{}{}, nil
	}

	return map[string]string{
		"results": string(output),
	}, nil
}

func (s *AgenticSkill) handleFindTodos(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	patterns := []string{"TODO", "FIXME", "HACK", "XXX", "BUG", "NOTE"}
	allResults := []map[string]string{}

	for _, pattern := range patterns {
		cmd := exec.CommandContext(ctx, "sh", "-c",
			fmt.Sprintf("grep -rn --include='*.go' --include='*.py' --include='*.js' --include='*.ts' --include='*.md' %q %s 2>/dev/null | head -20",
				pattern, path))

		output, _ := cmd.Output()
		lines := strings.Split(string(output), "\n")

		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 3)
			if len(parts) >= 3 {
				allResults = append(allResults, map[string]string{
					"type":    pattern,
					"file":    parts[0],
					"line":    parts[1],
					"content": strings.TrimSpace(parts[2]),
				})
			}
		}
	}

	return allResults, nil
}

// ==================== Git Handlers ====================

func (s *AgenticSkill) handleGitStatus(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	result := map[string]interface{}{}

	// Check if git repo
	if _, err := exec.CommandContext(ctx, "git", "-C", path, "rev-parse", "--git-dir").Output(); err != nil {
		return nil, fmt.Errorf("not a git repository")
	}

	// Status
	if output, err := exec.CommandContext(ctx, "git", "-C", path, "status", "--porcelain").Output(); err == nil {
		result["status"] = string(output)
		result["has_changes"] = len(output) > 0
	}

	// Branch
	if output, err := exec.CommandContext(ctx, "git", "-C", path, "branch", "--show-current").Output(); err == nil {
		result["branch"] = strings.TrimSpace(string(output))
	}

	// Remote URL
	if output, err := exec.CommandContext(ctx, "git", "-C", path, "remote", "-v").Output(); err == nil {
		result["remotes"] = string(output)
	}

	// Last commit
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

// ==================== Task Planning Handlers ====================

func (s *AgenticSkill) handleCreateTaskPlan(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	goal, _ := args["goal"].(string)
	if goal == "" {
		return nil, fmt.Errorf("goal is required")
	}

	steps := []string{}
	if s, ok := args["steps"].([]interface{}); ok {
		for _, step := range s {
			if str, ok := step.(string); ok {
				steps = append(steps, str)
			}
		}
	}

	return map[string]interface{}{
		"goal":       goal,
		"steps":      steps,
		"created_at": time.Now().Format(time.RFC3339),
		"status":     "planning",
	}, nil
}

func (s *AgenticSkill) handleReflectOnTask(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	task, _ := args["task"].(string)
	currentState, _ := args["current_state"].(string)

	actions := []string{}
	if a, ok := args["actions_taken"].([]interface{}); ok {
		for _, action := range a {
			if str, ok := action.(string); ok {
				actions = append(actions, str)
			}
		}
	}

	// This is a template for reflection - the LLM should actually do the reflection
	return map[string]interface{}{
		"task":           task,
		"actions_taken":  len(actions),
		"current_state":  currentState,
		"suggestions":    []string{
			"Consider verifying the current state",
			"Check if all prerequisites are met",
			"Document progress so far",
		},
	}, nil
}

// ==================== Helper Functions ====================

func parseCPUInfo(data string) []map[string]string {
	cpus := []map[string]string{}
	current := map[string]string{}

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if len(current) > 0 {
				cpus = append(cpus, current)
				current = map[string]string{}
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			current[key] = value
		}
	}

	return cpus
}

func parseMemInfo(data string) map[string]string {
	memInfo := map[string]string{}

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			memInfo[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	return memInfo
}

func detectLanguages(path string) []string {
	languages := map[string]bool{}
	exts := map[string]string{
		".go":   "Go",
		".py":   "Python",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".jsx":  "React",
		".tsx":  "React TypeScript",
		".rs":   "Rust",
		".java": "Java",
		".kt":   "Kotlin",
		".cpp":  "C++",
		".c":    "C",
		".h":    "C/C++",
		".rb":   "Ruby",
		".php":  "PHP",
		".swift": "Swift",
	}

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := filepath.Ext(p)
		if lang, ok := exts[ext]; ok {
			languages[lang] = true
		}
		return nil
	})

	result := []string{}
	for lang := range languages {
		result = append(result, lang)
	}
	sort.Strings(result)
	return result
}

func detectFrameworks(path string) []string {
	frameworks := []string{}

	// Check for framework indicators
	indicators := map[string]string{
		"go.mod":         "Go Modules",
		"package.json":   "Node.js",
		"requirements.txt": "Python pip",
		"Cargo.toml":     "Rust Cargo",
		"pom.xml":        "Maven",
		"build.gradle":   "Gradle",
		"Dockerfile":     "Docker",
		"docker-compose.yml": "Docker Compose",
		".github":        "GitHub Actions",
		"main.go":        "Go CLI",
	}

	for file, framework := range indicators {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			frameworks = append(frameworks, framework)
		}
	}

	return frameworks
}

func findEntryPoints(path string) []string {
	entryPoints := []string{}

	// Common entry point patterns
	patterns := []string{
		"main.go", "main.py", "index.js", "index.ts", "app.py",
		"cmd/*/main.go", "src/main.rs", "src/lib.rs",
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(path, pattern))
		entryPoints = append(entryPoints, matches...)
	}

	return entryPoints
}

func buildDirTree(path string, maxDepth int) map[string]interface{} {
	result := map[string]interface{}{
		"name":  filepath.Base(path),
		"type":  "directory",
		"items": []map[string]interface{}{},
	}

	if maxDepth <= 0 {
		return result
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return result
	}

	items := []map[string]interface{}{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		item := map[string]interface{}{
			"name": entry.Name(),
			"type": "file",
		}

		if entry.IsDir() {
			item["type"] = "directory"
			if maxDepth > 1 {
				subPath := filepath.Join(path, entry.Name())
				item["items"] = buildDirTree(subPath, maxDepth-1)["items"]
			}
		}

		items = append(items, item)
	}

	result["items"] = items
	return result
}

func countFileTypes(path string) map[string]int {
	counts := map[string]int{}

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := filepath.Ext(info.Name())
		if ext != "" {
			counts[ext]++
		}
		return nil
	})

	return counts
}

func extractTODOs(content string) []map[string]string {
	todos := []map[string]string{}
	pattern := regexp.MustCompile(`(?i)(TODO|FIXME|HACK|XXX|BUG|NOTE)[\s:]*(.+)`)

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if matches := pattern.FindStringSubmatch(line); matches != nil {
			todos = append(todos, map[string]string{
				"type":    strings.ToUpper(matches[1]),
				"content": strings.TrimSpace(matches[2]),
				"line":    strconv.Itoa(i + 1),
			})
		}
	}

	return todos
}

func analyzeGoFile(content []byte, result map[string]interface{}) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return
	}

	imports := []string{}
	for _, imp := range f.Imports {
		if imp.Path != nil {
			imports = append(imports, strings.Trim(imp.Path.Value, `"`))
		}
	}
	result["imports"] = imports

	functions := []map[string]interface{}{}
	classes := []map[string]interface{}{}

	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			fn := map[string]interface{}{
				"name": x.Name.Name,
				"line": fset.Position(x.Pos()).Line,
			}
			if x.Recv != nil {
				fn["type"] = "method"
				for _, recv := range x.Recv.List {
					if len(recv.Names) > 0 {
						fn["receiver"] = recv.Names[0].Name
					}
				}
			} else {
				fn["type"] = "function"
			}
			functions = append(functions, fn)

		case *ast.TypeSpec:
			if _, ok := x.Type.(*ast.StructType); ok {
				classes = append(classes, map[string]interface{}{
					"name": x.Name.Name,
					"type": "struct",
					"line": fset.Position(x.Pos()).Line,
				})
			}
			if _, ok := x.Type.(*ast.InterfaceType); ok {
				classes = append(classes, map[string]interface{}{
					"name": x.Name.Name,
					"type": "interface",
					"line": fset.Position(x.Pos()).Line,
				})
			}
		}
		return true
	})

	result["functions"] = functions
	result["classes"] = classes
	result["package"] = f.Name.Name
}

func analyzePythonFile(content string, result map[string]interface{}) {
	functions := []map[string]interface{}{}
	classes := []map[string]interface{}{}
	imports := []string{}

	// Simple regex-based analysis (AST parsing would be better but requires Python)
	importPattern := regexp.MustCompile(`^(import|from)\s+(\S+)`)
	funcPattern := regexp.MustCompile(`^def\s+(\w+)\s*\(`)
	classPattern := regexp.MustCompile(`^class\s+(\w+)`)

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if matches := importPattern.FindStringSubmatch(line); matches != nil {
			imports = append(imports, matches[2])
		}
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			functions = append(functions, map[string]interface{}{
				"name": matches[1],
				"type": "function",
				"line": i + 1,
			})
		}
		if matches := classPattern.FindStringSubmatch(line); matches != nil {
			classes = append(classes, map[string]interface{}{
				"name": matches[1],
				"type": "class",
				"line": i + 1,
			})
		}
	}

	result["imports"] = imports
	result["functions"] = functions
	result["classes"] = classes
}

func analyzeJSFile(content string, result map[string]interface{}) {
	functions := []map[string]interface{}{}
	classes := []map[string]interface{}{}
	imports := []string{}

	// Simple regex-based analysis
	importPattern := regexp.MustCompile(`import\s+.*?\s+from\s+['"]([^'"]+)['"]`)
	funcPattern := regexp.MustCompile(`(?:async\s+)?function\s+(\w+)\s*\(`)
	classPattern := regexp.MustCompile(`class\s+(\w+)`)
	arrowFuncPattern := regexp.MustCompile(`(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(`)

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if matches := importPattern.FindStringSubmatch(line); matches != nil {
			imports = append(imports, matches[1])
		}
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			functions = append(functions, map[string]interface{}{
				"name": matches[1],
				"type": "function",
				"line": i + 1,
			})
		}
		if matches := classPattern.FindStringSubmatch(line); matches != nil {
			classes = append(classes, map[string]interface{}{
				"name": matches[1],
				"type": "class",
				"line": i + 1,
			})
		}
		if matches := arrowFuncPattern.FindStringSubmatch(line); matches != nil {
			functions = append(functions, map[string]interface{}{
				"name": matches[1],
				"type": "arrow_function",
				"line": i + 1,
			})
		}
	}

	result["imports"] = imports
	result["functions"] = functions
	result["classes"] = classes
}
