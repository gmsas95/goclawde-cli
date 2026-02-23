package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/gmsas95/myrai-cli/internal/tools"
	"go.uber.org/zap"
)

// HandleChainCommand handles chain-related CLI commands
func HandleChainCommand(args []string) {
	if len(args) == 0 {
		PrintChainHelp()
		return
	}

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, err := config.Load("", "")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	st, err := store.New(cfg)
	if err != nil {
		fmt.Printf("Error initializing store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	// Create tool executor (simplified - in production would use full registry)
	toolExecutor := &SimpleToolExecutor{}

	chainExecutor, err := tools.NewChainExecutor(cfg, st, toolExecutor)
	if err != nil {
		fmt.Printf("Error initializing chain executor: %v\n", err)
		os.Exit(1)
	}

	switch args[0] {
	case "run":
		if len(args) < 2 {
			fmt.Println("Usage: myrai chain run <name> [--var key=value]")
			os.Exit(1)
		}
		chainName := args[1]

		// Parse variables
		variables := make(map[string]string)
		for i := 2; i < len(args); i++ {
			if args[i] == "--var" && i+1 < len(args) {
				parts := strings.SplitN(args[i+1], "=", 2)
				if len(parts) == 2 {
					variables[parts[0]] = parts[1]
				}
				i++
			}
		}

		fmt.Printf("Running chain: %s\n", chainName)
		if len(variables) > 0 {
			fmt.Println("Variables:")
			for k, v := range variables {
				fmt.Printf("  %s=%s\n", k, v)
			}
		}
		fmt.Println()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		result, err := chainExecutor.ExecuteChain(ctx, chainName, variables)
		if err != nil {
			fmt.Printf("❌ Chain execution failed: %v\n", err)
			os.Exit(1)
		}

		// Display results
		fmt.Printf("\n✅ Chain execution completed: %s\n", result.Status)
		fmt.Printf("Total steps: %d\n", result.TotalSteps)
		fmt.Printf("Passed: %d, Failed: %d\n", result.PassedSteps, result.FailedSteps)
		fmt.Printf("Duration: %s\n", result.CompletedAt.Sub(result.StartedAt))

		if len(result.StepResults) > 0 {
			fmt.Println("\nStep Results:")
			for _, step := range result.StepResults {
				status := "✅"
				if step.Status != "success" {
					status = "❌"
				}
				if step.Status == "skipped" {
					status = "⏭️"
				}
				fmt.Printf("  %s %s (%s)\n", status, step.StepID, step.Status)
				if step.Error != "" {
					fmt.Printf("     Error: %s\n", step.Error)
				}
			}
		}

	case "create":
		if len(args) < 2 {
			fmt.Println("Usage: myrai chain create <name>")
			os.Exit(1)
		}
		chainName := args[1]

		// Interactive creation
		reader := bufio.NewReader(os.Stdin)

		fmt.Println("Creating new tool chain")
		fmt.Println("=======================")

		fmt.Print("Description: ")
		description, _ := reader.ReadString('\n')
		description = strings.TrimSpace(description)

		fmt.Print("Type (sequential/parallel/conditional) [sequential]: ")
		chainType, _ := reader.ReadString('\n')
		chainType = strings.TrimSpace(chainType)
		if chainType == "" {
			chainType = "sequential"
		}

		fmt.Print("Version [1.0.0]: ")
		version, _ := reader.ReadString('\n')
		version = strings.TrimSpace(version)
		if version == "" {
			version = "1.0.0"
		}

		// Collect steps
		var steps []string
		stepNum := 1
		for {
			fmt.Printf("\nStep %d:\n", stepNum)
			fmt.Print("  Name (or 'done' to finish): ")
			stepName, _ := reader.ReadString('\n')
			stepName = strings.TrimSpace(stepName)
			if stepName == "done" || stepName == "" {
				break
			}

			fmt.Print("  Tool: ")
			toolName, _ := reader.ReadString('\n')
			toolName = strings.TrimSpace(toolName)

			fmt.Print("  Parameters (key=value,key2=value2): ")
			params, _ := reader.ReadString('\n')
			params = strings.TrimSpace(params)

			stepYAML := fmt.Sprintf(`  - id: step-%d
    name: %s
    tool: %s
    parameters:`, stepNum, stepName, toolName)

			if params != "" {
				paramPairs := strings.Split(params, ",")
				for _, pair := range paramPairs {
					parts := strings.SplitN(pair, "=", 2)
					if len(parts) == 2 {
						stepYAML += fmt.Sprintf("\n      %s: \"%s\"", parts[0], parts[1])
					}
				}
			}

			steps = append(steps, stepYAML)
			stepNum++
		}

		// Generate YAML
		yaml := fmt.Sprintf(`name: %s
description: %s
version: %s
author: user
type: %s

steps:
%s

variables:
  example_var: "value"
`, chainName, description, version, chainType, strings.Join(steps, "\n"))

		chain, err := chainExecutor.CreateChain(chainName, []byte(yaml))
		if err != nil {
			fmt.Printf("❌ Failed to create chain: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Created chain '%s'\n", chain.Name)
		fmt.Printf("   File: %s/chains/%s.yaml\n", cfg.Storage.DataDir, chainName)

	case "edit":
		if len(args) < 2 {
			fmt.Println("Usage: myrai chain edit <name>")
			os.Exit(1)
		}
		chainName := args[1]

		chainPath, err := chainExecutor.GetChainPath(chainName)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			os.Exit(1)
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano"
		}

		fmt.Printf("Opening %s in %s...\n", chainName, editor)
		syscall.Exec(editor, []string{editor, chainPath}, os.Environ())

	case "list":
		chains := chainExecutor.ListChains()
		if len(chains) == 0 {
			fmt.Println("No chains found.")
			return
		}

		fmt.Println("Available Tool Chains:")
		fmt.Println("=====================")
		for _, chain := range chains {
			builtin := ""
			if chain.IsBuiltIn {
				builtin = " (built-in)"
			}
			fmt.Printf("  • %s%s\n", chain.Name, builtin)
			fmt.Printf("    %s\n", chain.Description)
			fmt.Printf("    Version: %s | Steps: %d | Type: %s\n",
				chain.Version, len(chain.Steps), chain.Type)
			fmt.Println()
		}

	case "show", "view":
		if len(args) < 2 {
			fmt.Println("Usage: myrai chain show <name>")
			os.Exit(1)
		}
		chainName := args[1]

		chain, ok := chainExecutor.GetChain(chainName)
		if !ok {
			fmt.Printf("❌ Chain not found: %s\n", chainName)
			os.Exit(1)
		}

		fmt.Printf("Chain: %s\n", chain.Name)
		fmt.Printf("Description: %s\n", chain.Description)
		fmt.Printf("Version: %s | Type: %s | Built-in: %v\n",
			chain.Version, chain.Type, chain.IsBuiltIn)
		fmt.Println()

		if len(chain.Variables) > 0 {
			fmt.Println("Variables:")
			for k, v := range chain.Variables {
				fmt.Printf("  %s: %s\n", k, v)
			}
			fmt.Println()
		}

		fmt.Println("Steps:")
		for i, step := range chain.Steps {
			fmt.Printf("  %d. %s (id: %s)\n", i+1, step.Name, step.ID)
			fmt.Printf("     Tool: %s\n", step.Tool)
			if len(step.Parameters) > 0 {
				fmt.Printf("     Parameters: %v\n", step.Parameters)
			}
			if len(step.DependsOn) > 0 {
				fmt.Printf("     Depends on: %v\n", step.DependsOn)
			}
			if step.Condition != "" {
				fmt.Printf("     Condition: %s\n", step.Condition)
			}
			if step.Optional {
				fmt.Printf("     Optional: true\n")
			}
			fmt.Println()
		}

	case "delete":
		if len(args) < 2 {
			fmt.Println("Usage: myrai chain delete <name>")
			os.Exit(1)
		}
		chainName := args[1]

		chain, ok := chainExecutor.GetChain(chainName)
		if !ok {
			fmt.Printf("❌ Chain not found: %s\n", chainName)
			os.Exit(1)
		}

		if chain.IsBuiltIn {
			fmt.Println("❌ Cannot delete built-in chains")
			os.Exit(1)
		}

		fmt.Printf("Are you sure you want to delete chain '%s'? (yes/no): ", chainName)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "yes" {
			fmt.Println("Cancelled")
			return
		}

		if err := chainExecutor.DeleteChain(chainName); err != nil {
			fmt.Printf("❌ Failed to delete chain: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Deleted chain '%s'\n", chainName)

	case "publish":
		if len(args) < 2 {
			fmt.Println("Usage: myrai chain publish <name>")
			fmt.Println("Note: Publishes chain to marketplace (coming soon)")
			os.Exit(1)
		}
		chainName := args[1]

		chain, ok := chainExecutor.GetChain(chainName)
		if !ok {
			fmt.Printf("❌ Chain not found: %s\n", chainName)
			os.Exit(1)
		}

		fmt.Printf("Publishing chain '%s' to marketplace...\n", chainName)
		fmt.Println("Note: Marketplace integration coming in Phase 6")
		fmt.Printf("Chain '%s' would be shared with the community\n", chain.Name)
		fmt.Printf("  - Name: %s\n", chain.Name)
		fmt.Printf("  - Description: %s\n", chain.Description)
		fmt.Printf("  - Steps: %d\n", len(chain.Steps))

	case "validate":
		if len(args) < 2 {
			fmt.Println("Usage: myrai chain validate <name>")
			os.Exit(1)
		}
		chainName := args[1]

		chain, ok := chainExecutor.GetChain(chainName)
		if !ok {
			fmt.Printf("❌ Chain not found: %s\n", chainName)
			os.Exit(1)
		}

		if err := chain.Validate(); err != nil {
			fmt.Printf("❌ Chain validation failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Chain '%s' is valid\n", chainName)

	default:
		PrintChainHelp()
	}
}

// SimpleToolExecutor is a simple implementation for CLI usage
type SimpleToolExecutor struct{}

// Execute runs a tool by name
func (ste *SimpleToolExecutor) Execute(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	// For demonstration, execute via shell
	// In production, this would use the actual tool registry

	switch toolName {
	case "exec":
		command, _ := params["command"].(string)
		if command == "" {
			return nil, fmt.Errorf("command parameter required")
		}
		fmt.Printf("  Executing: %s\n", command)
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		output, err := cmd.CombinedOutput()
		return map[string]interface{}{
			"stdout": string(output),
			"error":  err,
		}, nil

	case "read_file":
		path, _ := params["path"].(string)
		if path == "" {
			return nil, fmt.Errorf("path parameter required")
		}
		fmt.Printf("  Reading file: %s\n", path)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return string(data), nil

	case "write_file":
		path, _ := params["path"].(string)
		content, _ := params["content"].(string)
		if path == "" {
			return nil, fmt.Errorf("path parameter required")
		}
		fmt.Printf("  Writing file: %s\n", path)
		return nil, os.WriteFile(path, []byte(content), 0644)

	case "list_dir":
		path, _ := params["path"].(string)
		if path == "" {
			path = "."
		}
		fmt.Printf("  Listing directory: %s\n", path)
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		return names, nil

	default:
		fmt.Printf("  [Simulated] Executing tool: %s\n", toolName)
		return map[string]string{"status": "simulated", "tool": toolName}, nil
	}
}

// PrintChainHelp prints help for chain commands
func PrintChainHelp() {
	fmt.Println("Tool Chain Management")
	fmt.Println("====================")
	fmt.Println()
	fmt.Println("Usage: myrai chain <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  run <name> [--var key=value]  Execute a tool chain")
	fmt.Println("  create <name>                Create a new chain interactively")
	fmt.Println("  edit <name>                  Edit a chain in your default editor")
	fmt.Println("  list                         List available chains")
	fmt.Println("  show <name>                  Show chain details")
	fmt.Println("  delete <name>                Delete a user chain")
	fmt.Println("  validate <name>              Validate chain definition")
	fmt.Println("  publish <name>               Publish chain to marketplace")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  myrai chain run deploy-application --var image_tag=myapp:v1.0")
	fmt.Println("  myrai chain create my-custom-chain")
	fmt.Println("  myrai chain list")
	fmt.Println()
	fmt.Println("Pre-built chains:")
	fmt.Println("  deploy-application    Build, push, and deploy application")
	fmt.Println("  setup-ci-cd          Setup CI/CD pipeline")
	fmt.Println("  database-migration   Run database migrations")
	fmt.Println("  code-review          Automated code review")
}

// HandleToolsInventoryCommand handles tool inventory commands
func HandleToolsInventoryCommand(args []string) {
	inventory := tools.NewInventory()

	if len(args) == 0 || args[0] == "list" {
		fmt.Println("Tool Inventory")
		fmt.Println("=============")
		fmt.Printf("Total tools: %d\n\n", inventory.GetToolCount())

		// Show count by category
		counts := inventory.GetCategoryCount()
		fmt.Println("Categories:")
		for category, count := range counts {
			fmt.Printf("  %-15s: %d tools\n", category, count)
		}
		return
	}

	switch args[0] {
	case "category":
		if len(args) < 2 {
			fmt.Println("Usage: myrai tools category <category>")
			fmt.Println("Categories: System, File, Browser, DevOps, Git, API, Database, Cloud, Communication, Cron, Memory, MCP")
			os.Exit(1)
		}

		category := tools.ToolCategory(args[1])
		tools := inventory.GetToolsByCategory(category)

		fmt.Printf("Tools in category: %s\n", category)
		fmt.Printf("Count: %d\n\n", len(tools))

		for _, tool := range tools {
			fmt.Printf("  • %s\n", tool.Name)
			fmt.Printf("    %s\n", tool.Description)
			if len(tool.Tags) > 0 {
				fmt.Printf("    Tags: %s\n", strings.Join(tool.Tags, ", "))
			}
			fmt.Println()
		}

	case "search":
		if len(args) < 2 {
			fmt.Println("Usage: myrai tools search <query>")
			os.Exit(1)
		}

		query := args[1]
		results := inventory.SearchTools(query)

		fmt.Printf("Search results for '%s':\n", query)
		fmt.Printf("Found: %d tools\n\n", len(results))

		for _, tool := range results {
			fmt.Printf("  • %s (%s)\n", tool.Name, tool.Category)
			fmt.Printf("    %s\n", tool.Description)
			fmt.Println()
		}

	case "info":
		if len(args) < 2 {
			fmt.Println("Usage: myrai tools info <tool-name>")
			os.Exit(1)
		}

		toolName := args[1]
		tool, ok := inventory.GetTool(toolName)
		if !ok {
			fmt.Printf("Tool not found: %s\n", toolName)
			os.Exit(1)
		}

		fmt.Printf("Tool: %s\n", tool.Name)
		fmt.Printf("Category: %s\n", tool.Category)
		fmt.Printf("Description: %s\n", tool.Description)
		if len(tool.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(tool.Tags, ", "))
		}
		if len(tool.Examples) > 0 {
			fmt.Println("\nExamples:")
			for _, example := range tool.Examples {
				fmt.Printf("  %s\n", example)
			}
		}

	default:
		fmt.Println("Usage: myrai tools [list|category <cat>|search <query>|info <tool>]")
	}
}

// HandleIntentCommand handles intent classification (for testing)
func HandleIntentCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai intent <input>")
		os.Exit(1)
	}

	input := strings.Join(args, " ")

	cfg, err := config.Load("", "")
	if err != nil {
		// Fallback to rule-based without LLM
		classifier := tools.NewIntentClassifier(nil)
		intent, _ := classifier.Classify(context.Background(), input)
		printIntent(intent)
		return
	}

	// Try to use LLM
	if provider, ok := cfg.LLM.Providers[cfg.LLM.DefaultProvider]; ok {
		llmClient := llm.NewClient(provider)
		classifier := tools.NewIntentClassifier(llmClient)
		intent, err := classifier.Classify(context.Background(), input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		printIntent(intent)
	} else {
		// Fallback
		classifier := tools.NewIntentClassifier(nil)
		intent, _ := classifier.Classify(context.Background(), input)
		printIntent(intent)
	}
}

func printIntent(intent *tools.Intent) {
	fmt.Println("Intent Classification")
	fmt.Println("=====================")
	fmt.Printf("Input: %s\n", intent.RawInput)
	fmt.Printf("Type: %s\n", intent.Type)
	fmt.Printf("Category: %s\n", intent.Category)
	fmt.Printf("Complexity: %s\n", intent.Complexity)
	fmt.Printf("Confidence: %.2f\n", intent.Confidence)
	if len(intent.Keywords) > 0 {
		fmt.Printf("Keywords: %s\n", strings.Join(intent.Keywords, ", "))
	}
	fmt.Printf("Needs Decomposition: %v\n", intent.NeedsDecomposition())
}

// OrchestratorIntegration provides integration with the main app
type OrchestratorIntegration struct {
	Classifier *tools.IntentClassifier
	Decomposer *tools.TaskDecomposer
	ChainExec  *tools.ChainExecutor
	Inventory  *tools.Inventory
}

// NewOrchestratorIntegration creates a new orchestrator integration
func NewOrchestratorIntegration(cfg *config.Config, store *store.Store, llmClient *llm.Client, toolExec tools.ToolExecutor) (*OrchestratorIntegration, error) {
	chainExec, err := tools.NewChainExecutor(cfg, store, toolExec)
	if err != nil {
		return nil, err
	}

	return &OrchestratorIntegration{
		Classifier: tools.NewIntentClassifier(llmClient),
		Decomposer: tools.NewTaskDecomposer(llmClient),
		ChainExec:  chainExec,
		Inventory:  tools.NewInventory(),
	}, nil
}
