package agentic

import (
	"github.com/gmsas95/myrai-cli/internal/skills"
)

func (s *AgenticSkill) registerTools() {
	s.registerSystemTools()
	s.registerCodeTools()
	s.registerGitTools()
	s.registerTaskTools()
}

func (s *AgenticSkill) registerSystemTools() {
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
}

func (s *AgenticSkill) registerCodeTools() {
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
}

func (s *AgenticSkill) registerGitTools() {
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
}

func (s *AgenticSkill) registerTaskTools() {
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
