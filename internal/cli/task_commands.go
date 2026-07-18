package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/runtime"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/gmsas95/myrai-cli/pkg/tools"
	"go.uber.org/zap"
)

// TaskCommands handles task-related CLI commands
type TaskCommands struct {
	runner *runtime.Runner
	store  *runtime.Store
}

// NewTaskCommands creates a new task commands handler
func NewTaskCommands() (*TaskCommands, error) {
	// Load config
	cfg, err := config.Load("", "")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	logger, _ := zap.NewDevelopment()

	// Initialize store
	storeInstance, err := store.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize store: %w", err)
	}

	// Create runtime store
	runtimeStore := runtime.NewStore(storeInstance.DB())
	if err := runtimeStore.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate runtime schema: %w", err)
	}

	// Initialize LLM client
	provider := cfg.LLM.Providers[cfg.LLM.DefaultProvider]
	llmClient := llm.NewClient(provider)

	// Initialize tool registry
	allowedCmds := []string{"ls", "cat", "grep", "find", "git", "go", "npm", "node"}
	toolRegistry := tools.NewRegistry(allowedCmds)
	workspaceRoot, _ := os.Getwd()

	// Create runner
	runner := runtime.NewRunner(runtime.RunnerConfig{
		Logger:        logger,
		Store:         runtimeStore,
		ToolRegistry:  toolRegistry,
		LLMClient:     llmClient,
		WorkspaceRoot: workspaceRoot,
		AllowedCmds:   allowedCmds,
	})

	return &TaskCommands{
		runner: runner,
		store:  runtimeStore,
	}, nil
}

// HandleTaskCommand handles task-related commands
func HandleTaskCommand(args []string) {
	if len(args) == 0 {
		PrintTaskHelp()
		return
	}

	cmds, err := NewTaskCommands()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	switch args[0] {
	case "create", "new":
		handleTaskCreate(ctx, cmds, args[1:])
	case "list", "ls":
		handleTaskList(ctx, cmds, args[1:])
	case "show", "get", "info":
		handleTaskShow(ctx, cmds, args[1:])
	case "resume", "continue":
		handleTaskResume(ctx, cmds, args[1:])
	case "cancel", "stop":
		handleTaskCancel(ctx, cmds, args[1:])
	case "approve":
		handleTaskApprove(ctx, cmds, args[1:])
	case "deny":
		handleTaskDeny(ctx, cmds, args[1:])
	case "logs", "events":
		handleTaskLogs(ctx, cmds, args[1:])
	case "delete", "rm":
		handleTaskDelete(ctx, cmds, args[1:])
	default:
		PrintTaskHelp()
	}
}

// PrintTaskHelp prints task command help
func PrintTaskHelp() {
	fmt.Println("Task Management Commands")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("Usage: myrai task <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  create, new <goal>      Create and start a new task")
	fmt.Println("  list, ls               List all tasks")
	fmt.Println("  show, get <id>         Show task details and timeline")
	fmt.Println("  resume <id>            Resume a blocked or running task")
	fmt.Println("  cancel <id>            Cancel a running or pending task")
	fmt.Println("  approve <approval-id>  Approve a pending action")
	fmt.Println("  deny <approval-id>     Deny a pending action")
	fmt.Println("  logs <id>              Show task event logs")
	fmt.Println("  delete <id>            Delete a task and all its data")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  myrai task create \"Fix the login bug in auth.go\"")
	fmt.Println("  myrai task ls")
	fmt.Println("  myrai task show task_20240101120000_abc123")
	fmt.Println("  myrai task resume task_20240101120000_abc123")
	fmt.Println()
}

func handleTaskCreate(ctx context.Context, cmds *TaskCommands, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai task create <goal> [options]")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --description, -d    Task description")
		fmt.Println("  --iterations, -i     Max iterations (default: 10)")
		fmt.Println("  --no-approval        Don't require approval for risky actions")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  myrai task create \"Fix the bug\" -d \"Fix the login issue\" -i 20")
		os.Exit(1)
	}

	goal := args[0]
	description := ""
	maxIterations := 10
	requireApproval := true

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--description", "-d":
			if i+1 < len(args) {
				description = args[i+1]
				i++
			}
		case "--iterations", "-i":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &maxIterations)
				i++
			}
		case "--no-approval":
			requireApproval = false
		}
	}

	task, err := cmds.runner.StartTask(ctx, goal, description, maxIterations, requireApproval)
	if err != nil {
		fmt.Printf("Error creating task: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Task created: %s\n", task.ID)
	fmt.Printf("  Goal: %s\n", task.Goal)
	fmt.Printf("  State: %s\n", task.State)
	fmt.Printf("  Max Iterations: %d\n", task.MaxIterations)
	fmt.Println()
	fmt.Printf("Task is now running. Use 'myrai task show %s' to monitor progress.\n", task.ID)
}

func handleTaskList(ctx context.Context, cmds *TaskCommands, args []string) {
	// Parse optional state filter
	var state *runtime.TaskState
	for _, arg := range args {
		s := runtime.TaskState(arg)
		if s == runtime.TaskStatePending || s == runtime.TaskStateRunning ||
			s == runtime.TaskStateBlocked || s == runtime.TaskStateFailed ||
			s == runtime.TaskStateCompleted || s == runtime.TaskStateCancelled {
			state = &s
			break
		}
	}

	tasks, err := cmds.store.ListTasks(ctx, state, 50)
	if err != nil {
		fmt.Printf("Error listing tasks: %v\n", err)
		os.Exit(1)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found. Create one with: myrai task create <goal>")
		return
	}

	fmt.Println("Tasks")
	fmt.Println("=====")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATE\tITERATIONS\tGOAL\tCREATED")
	fmt.Fprintln(w, "--\t-----\t----------\t----\t-------")

	for _, task := range tasks {
		stateIcon := getStateIcon(task.State)
		goal := truncate(task.Goal, 40)
		created := formatTime(task.CreatedAt)

		fmt.Fprintf(w, "%s\t%s %s\t%d/%d\t%s\t%s\n",
			task.ID,
			stateIcon,
			task.State,
			task.CurrentIteration,
			task.MaxIterations,
			goal,
			created)
	}

	w.Flush()
	fmt.Println()
	fmt.Printf("Showing %d task(s). Use 'myrai task show <id>' for details.\n", len(tasks))
}

func handleTaskShow(ctx context.Context, cmds *TaskCommands, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai task show <task-id>")
		os.Exit(1)
	}

	taskID := args[0]
	task, err := cmds.store.GetTask(ctx, taskID)
	if err != nil {
		fmt.Printf("Error: Task not found: %s\n", taskID)
		os.Exit(1)
	}

	fmt.Printf("Task: %s\n", task.ID)
	fmt.Println(strings.Repeat("=", len(task.ID)+6))
	fmt.Println()
	fmt.Printf("Goal:        %s\n", task.Goal)
	if task.Description != "" {
		fmt.Printf("Description: %s\n", task.Description)
	}
	fmt.Printf("State:       %s %s\n", getStateIcon(task.State), task.State)
	fmt.Printf("Iterations:  %d / %d\n", task.CurrentIteration, task.MaxIterations)
	fmt.Printf("Created:     %s\n", formatTime(task.CreatedAt))
	if task.StartedAt != nil {
		fmt.Printf("Started:     %s\n", formatTime(*task.StartedAt))
	}
	if task.CompletedAt != nil {
		fmt.Printf("Completed:   %s\n", formatTime(*task.CompletedAt))
	}
	fmt.Println()

	if task.CurrentPlan != "" {
		fmt.Println("Current Plan:")
		fmt.Println("-------------")
		fmt.Println(task.CurrentPlan)
		fmt.Println()
	}

	if len(task.Steps) > 0 {
		fmt.Println("Execution Steps:")
		fmt.Println("----------------")
		for _, step := range task.Steps {
			icon := getStepIcon(step.Type)
			fmt.Printf("[%s] %s %s (iter %d, state: %s)\n", formatTime(step.CreatedAt), icon, step.Type, step.Iteration, step.ExecutionState)
			if step.Content != "" {
				content := truncate(step.Content, 80)
				fmt.Printf("     %s\n", content)
			}
			if step.ToolName != "" {
				fmt.Printf("     Tool: %s", step.ToolName)
				if step.ToolRisk != "" && step.ToolRisk != "safe" {
					fmt.Printf(" [risk: %s]", step.ToolRisk)
				}
				fmt.Println()
			}
			if step.ToolError != "" {
				fmt.Printf("     Error: %s\n", step.ToolError)
			}
			if step.ReplayOfStepID != "" {
				fmt.Printf("     Replay: reused result from %s\n", step.ReplayOfStepID)
			}
		}
		fmt.Println()
	}

	if len(task.Approvals) > 0 {
		fmt.Println("Pending Approvals:")
		fmt.Println("------------------")
		for _, approval := range task.Approvals {
			if approval.Status == runtime.ApprovalStatusPending {
				fmt.Printf("  • %s: %s (risk: %s)\n", approval.ID, approval.ToolName, approval.RiskLevel)
			}
		}
		fmt.Println()
	}

	if task.Error != "" {
		fmt.Println("Error:")
		fmt.Println("------")
		fmt.Println(task.Error)
		fmt.Println()
	}

	if task.FinalResult != "" {
		fmt.Println("Result:")
		fmt.Println("-------")
		fmt.Println(task.FinalResult)
		fmt.Println()
	}
}

func handleTaskResume(ctx context.Context, cmds *TaskCommands, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai task resume <task-id>")
		os.Exit(1)
	}

	taskID := args[0]
	task, err := cmds.runner.ResumeTask(ctx, taskID)
	if err != nil {
		fmt.Printf("Error resuming task: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Task resumed: %s\n", task.ID)
	fmt.Printf("  State: %s\n", task.State)
	fmt.Printf("  Current Iteration: %d\n", task.CurrentIteration)
}

func handleTaskCancel(ctx context.Context, cmds *TaskCommands, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai task cancel <task-id> [reason]")
		os.Exit(1)
	}

	taskID := args[0]
	reason := "User cancelled"
	if len(args) > 1 {
		reason = strings.Join(args[1:], " ")
	}

	if err := cmds.runner.CancelTask(ctx, taskID, reason); err != nil {
		fmt.Printf("Error cancelling task: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Task cancelled: %s\n", taskID)
	fmt.Printf("  Reason: %s\n", reason)
}

func handleTaskApprove(ctx context.Context, cmds *TaskCommands, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai task approve <approval-id>")
		os.Exit(1)
	}

	approvalID := args[0]
	approvedBy := os.Getenv("USER")
	if approvedBy == "" {
		approvedBy = "cli-user"
	}

	if err := cmds.runner.ApproveAction(ctx, approvalID, approvedBy); err != nil {
		fmt.Printf("Error approving action: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Action approved: %s\n", approvalID)
}

func handleTaskDeny(ctx context.Context, cmds *TaskCommands, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai task deny <approval-id> [reason]")
		os.Exit(1)
	}

	approvalID := args[0]
	reason := "Denied by user"
	if len(args) > 1 {
		reason = strings.Join(args[1:], " ")
	}

	deniedBy := os.Getenv("USER")
	if deniedBy == "" {
		deniedBy = "cli-user"
	}

	if err := cmds.runner.DenyAction(ctx, approvalID, deniedBy, reason); err != nil {
		fmt.Printf("Error denying action: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Action denied: %s\n", approvalID)
	fmt.Printf("  Reason: %s\n", reason)
}

func handleTaskLogs(ctx context.Context, cmds *TaskCommands, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai task logs <task-id>")
		os.Exit(1)
	}

	taskID := args[0]
	events, err := cmds.store.ListEvents(ctx, taskID)
	if err != nil {
		fmt.Printf("Error getting task logs: %v\n", err)
		os.Exit(1)
	}

	if len(events) == 0 {
		fmt.Println("No events found for this task.")
		return
	}

	fmt.Printf("Task Logs: %s\n", taskID)
	fmt.Println(strings.Repeat("-", 50))

	for _, event := range events {
		icon := getEventIcon(event.Type)
		fmt.Printf("[%s] %s %s\n", formatTime(event.CreatedAt), icon, event.Type)
		fmt.Printf("      %s\n", event.Message)
		if len(event.Data) > 0 {
			var data map[string]interface{}
			if err := json.Unmarshal(event.Data, &data); err == nil {
				dataStr, _ := json.MarshalIndent(data, "      ", "  ")
				fmt.Printf("      Data: %s\n", string(dataStr))
			}
		}
		fmt.Println()
	}
}

func handleTaskDelete(ctx context.Context, cmds *TaskCommands, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai task delete <task-id>")
		fmt.Println()
		fmt.Println("⚠️  Warning: This will permanently delete the task and all its data.")
		os.Exit(1)
	}

	taskID := args[0]

	// Check if task exists
	_, err := cmds.store.GetTaskByID(ctx, taskID)
	if err != nil {
		fmt.Printf("Error: Task not found: %s\n", taskID)
		os.Exit(1)
	}

	fmt.Printf("⚠️  Are you sure you want to delete task %s? [y/N]: ", taskID)
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Cancelled.")
		return
	}

	if err := cmds.store.DeleteTask(ctx, taskID); err != nil {
		fmt.Printf("Error deleting task: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Task deleted: %s\n", taskID)
}

// Helper functions

func getStateIcon(state runtime.TaskState) string {
	switch state {
	case runtime.TaskStatePending:
		return "⏳"
	case runtime.TaskStateRunning:
		return "▶️"
	case runtime.TaskStateBlocked:
		return "🛑"
	case runtime.TaskStateFailed:
		return "❌"
	case runtime.TaskStateCompleted:
		return "✅"
	case runtime.TaskStateCancelled:
		return "🚫"
	default:
		return "❓"
	}
}

func getStepIcon(stepType runtime.StepType) string {
	switch stepType {
	case runtime.StepTypeThink:
		return "💭"
	case runtime.StepTypePlan:
		return "📋"
	case runtime.StepTypeTool:
		return "🔧"
	case runtime.StepTypeReflect:
		return "🤔"
	case runtime.StepTypeRespond:
		return "💬"
	case runtime.StepTypeError:
		return "⚠️"
	case runtime.StepTypeApproval:
		return "⏸️"
	default:
		return "•"
	}
}

func getEventIcon(eventType runtime.EventType) string {
	switch eventType {
	case runtime.EventTypeStateChange:
		return "🔄"
	case runtime.EventTypeStepCreated:
		return "📝"
	case runtime.EventTypeToolExecuted:
		return "🔧"
	case runtime.EventTypeApprovalReq:
		return "⏸️"
	case runtime.EventTypeApprovalRes:
		return "✓"
	case runtime.EventTypeError:
		return "⚠️"
	case runtime.EventTypeCheckpoint:
		return "💾"
	case runtime.EventTypeResume:
		return "▶️"
	case runtime.EventTypeCancel:
		return "🚫"
	default:
		return "•"
	}
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// truncate truncates a string to max length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
