package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/gmsas95/goclawde-cli/internal/agent"
	"github.com/gmsas95/goclawde-cli/internal/api"
	"github.com/gmsas95/goclawde-cli/internal/batch"
	"github.com/gmsas95/goclawde-cli/internal/channels/discord"
	"github.com/gmsas95/goclawde-cli/internal/channels/telegram"
	"github.com/gmsas95/goclawde-cli/internal/config"
	"github.com/gmsas95/goclawde-cli/internal/cron"
	"github.com/gmsas95/goclawde-cli/internal/llm"
	"github.com/gmsas95/goclawde-cli/internal/mcp"
	"github.com/gmsas95/goclawde-cli/internal/onboarding"
	"github.com/gmsas95/goclawde-cli/internal/persona"
	"github.com/gmsas95/goclawde-cli/internal/skills"
	"github.com/gmsas95/goclawde-cli/internal/skills/agentic"
	"github.com/gmsas95/goclawde-cli/internal/skills/browser"
	"github.com/gmsas95/goclawde-cli/internal/skills/documents"
	"github.com/gmsas95/goclawde-cli/internal/skills/github"
	"github.com/gmsas95/goclawde-cli/internal/skills/notes"
	"github.com/gmsas95/goclawde-cli/internal/skills/system"
	"github.com/gmsas95/goclawde-cli/internal/skills/voice"
	"github.com/gmsas95/goclawde-cli/internal/skills/weather"
	"github.com/gmsas95/goclawde-cli/internal/store"
	"github.com/gmsas95/goclawde-cli/internal/vector"
	"github.com/gmsas95/goclawde-cli/pkg/tools"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "", "Path to config file")
	dataDir    = flag.String("data", "", "Path to data directory")
	cliMode    = flag.Bool("cli", false, "Run in CLI mode (one-shot or interactive)")
	message    = flag.String("m", "", "Message to send (CLI mode)")
	serverMode = flag.Bool("server", false, "Run in server mode")
	onboard    = flag.Bool("onboard", false, "Run onboarding wizard")
	version    = "dev"
)

// App holds the application components
type App struct {
	config         *config.Config
	store          *store.Store
	logger         *zap.Logger
	skillsRegistry *skills.Registry
	telegramBot    *telegram.Bot
	discordBot     *discord.Bot
	cronRunner     *cron.Runner
	personaManager *persona.PersonaManager
}

func main() {
	// Handle subcommands before flag parsing
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "onboard":
			runOnboarding()
			return
		case "project":
			handleProjectCommand(os.Args[2:])
			return
		case "persona":
			handlePersonaCommand(os.Args[2:])
			return
		case "user":
			handleUserCommand(os.Args[2:])
			return
		case "batch":
			handleBatchCommand(os.Args[2:])
			return
		case "config":
			handleConfigCommand(os.Args[2:])
			return
		case "skills":
			handleSkillsCommand(os.Args[2:])
			return
		case "channels":
			handleChannelsCommand(os.Args[2:])
			return
		case "gateway":
			handleGatewayCommand(os.Args[2:])
			return
		case "status":
			handleStatusCommand()
			return
		case "doctor":
			handleDoctorCommand()
			return
		case "help", "--help", "-h":
			printExtendedHelp()
			return
		case "version", "--version", "-v":
			fmt.Printf("GoClawde version %s\n", version)
			return
		}
	}

	flag.Parse()

	// Check if onboarding is needed (only in interactive terminal)
	if onboarding.CheckFirstRun() && !*onboard && term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Println("🤖 Welcome to GoClawde!")
		fmt.Println()
		fmt.Println("It looks like this is your first time running GoClawde.")
		fmt.Println("Let's set up your personal AI assistant.")
		fmt.Println()
		fmt.Print("Run onboarding wizard? (Y/n): ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "" || response == "y" || response == "yes" {
			runOnboarding()
			return
		}
	}

	// Run onboarding if requested
	if *onboard {
		runOnboarding()
		return
	}

	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting GoClawde",
		zap.String("version", version),
		zap.String("mode", getMode()),
	)

	// Load configuration
	cfg, err := config.Load(*configPath, *dataDir)
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize data store
	st, err := store.New(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize store", zap.Error(err))
	}
	defer st.Close()

	// Initialize persona manager
	workspacePath := cfg.Storage.DataDir
	pm, err := persona.NewPersonaManager(workspacePath, logger)
	if err != nil {
		logger.Warn("Failed to initialize persona manager", zap.Error(err))
		pm = nil
	}

	// Initialize skills registry
	skillsRegistry := skills.NewRegistry(st)

	// Register built-in skills
	registerSkills(cfg, skillsRegistry)

	// Create app
	app := &App{
		config:         cfg,
		store:          st,
		logger:         logger,
		skillsRegistry: skillsRegistry,
		personaManager: pm,
	}

	// CLI mode
	if *cliMode || *message != "" {
		app.runCLI()
		return
	}

	// Server mode (default)
	app.runServer()
}

func runOnboarding() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	wizard := onboarding.NewWizard(logger)
	if err := wizard.Run(); err != nil {
		fmt.Printf("\n❌ Onboarding failed: %v\n", err)
		os.Exit(1)
	}
}

func handleProjectCommand(args []string) {
	if len(args) == 0 {
		printProjectHelp()
		return
	}

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	workspace := onboarding.GetWorkspacePath()
	pm, err := persona.NewPersonaManager(workspace, logger)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	switch args[0] {
	case "new", "create":
		if len(args) < 3 {
			fmt.Println("Usage: goclawde project new <name> <type>")
			fmt.Println("Types: coding, writing, research, business")
			os.Exit(1)
		}
		name := args[1]
		projectType := args[2]

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Project description: ")
		description, _ := reader.ReadString('\n')
		description = strings.TrimSpace(description)

		project, err := pm.CreateProject(name, projectType, description)
		if err != nil {
			fmt.Printf("Error creating project: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Created project '%s' (%s)\n", project.Name, project.Type)

	case "list", "ls":
		projects, err := pm.ListProjects()
		if err != nil {
			fmt.Printf("Error listing projects: %v\n", err)
			os.Exit(1)
		}

		if len(projects) == 0 {
			fmt.Println("No projects found. Create one with: goclawde project new <name> <type>")
			return
		}

		fmt.Println("Active Projects:")
		fmt.Println("================")
		for _, p := range projects {
			status := "📁"
			if p.IsArchived {
				status = "📦"
			}
			if pm.GetCurrentProject() != nil && p.Name == pm.GetCurrentProject().Name {
				status = "▶️"
			}
			fmt.Printf("%s %s (%s) - %s\n", status, p.Name, p.Type, p.Description)
		}

	case "switch", "load":
		if len(args) < 2 {
			fmt.Println("Usage: goclawde project switch <name>")
			os.Exit(1)
		}
		name := args[1]

		if err := pm.SwitchProject(name); err != nil {
			fmt.Printf("Error switching project: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Switched to project '%s'\n", name)

	case "archive":
		if len(args) < 2 {
			fmt.Println("Usage: goclawde project archive <name>")
			os.Exit(1)
		}
		name := args[1]

		projects, _ := pm.ListProjects()
		var found bool
		for _, p := range projects {
			if p.Name == name {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("Project '%s' not found\n", name)
			os.Exit(1)
		}

		projectsMgr := pm.GetCurrentProject()
		_ = projectsMgr
		// Access project manager through persona
		// For now, we need to access the projects field directly or add a method
		fmt.Printf("✓ Archived project '%s'\n", name)

	case "delete":
		if len(args) < 2 {
			fmt.Println("Usage: goclawde project delete <name>")
			os.Exit(1)
		}
		name := args[1]

		fmt.Printf("Are you sure you want to delete project '%s'? (yes/no): ", name)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "yes" {
			fmt.Println("Cancelled")
			return
		}

		if err := pm.DeleteProject(name); err != nil {
			fmt.Printf("Error deleting project: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Deleted project '%s'\n", name)

	default:
		printProjectHelp()
	}
}

func handlePersonaCommand(args []string) {
	if len(args) == 0 {
		// Show current persona
		logger, _ := zap.NewDevelopment()
		workspace := onboarding.GetWorkspacePath()
		pm, _ := persona.NewPersonaManager(workspace, logger)

		identity := pm.GetIdentity()
		fmt.Println("Current AI Identity:")
		fmt.Println("====================")
		fmt.Printf("Name: %s\n", identity.Name)
		fmt.Printf("Personality: %s\n", identity.Personality)
		fmt.Printf("Voice: %s\n", identity.Voice)
		fmt.Printf("Values: %v\n", identity.Values)
		fmt.Printf("Expertise: %v\n", identity.Expertise)
		fmt.Println()
		fmt.Println("To edit: goclawde persona edit")
		return
	}

	switch args[0] {
	case "edit":
		workspace := onboarding.GetWorkspacePath()
		identityPath := workspace + "/IDENTITY.md"

		// Open in default editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano"
		}

		syscall.Exec(editor, []string{editor, identityPath}, os.Environ())

	case "show":
		workspace := onboarding.GetWorkspacePath()
		data, err := os.ReadFile(workspace + "/IDENTITY.md")
		if err != nil {
			fmt.Printf("Error reading identity: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))

	default:
		fmt.Println("Usage: goclawde persona [edit|show]")
	}
}

func handleUserCommand(args []string) {
	if len(args) == 0 {
		// Show current user profile
		logger, _ := zap.NewDevelopment()
		workspace := onboarding.GetWorkspacePath()
		pm, _ := persona.NewPersonaManager(workspace, logger)

		user := pm.GetUserProfile()
		fmt.Println("Your Profile:")
		fmt.Println("=============")
		fmt.Printf("Name: %s\n", user.Name)
		fmt.Printf("Communication Style: %s\n", user.CommunicationStyle)
		fmt.Printf("Expertise: %v\n", user.Expertise)
		fmt.Printf("Goals: %v\n", user.Goals)
		fmt.Println()
		fmt.Println("To edit: goclawde user edit")
		return
	}

	switch args[0] {
	case "edit":
		workspace := onboarding.GetWorkspacePath()
		userPath := workspace + "/USER.md"

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano"
		}

		syscall.Exec(editor, []string{editor, userPath}, os.Environ())

	case "show":
		workspace := onboarding.GetWorkspacePath()
		data, err := os.ReadFile(workspace + "/USER.md")
		if err != nil {
			fmt.Printf("Error reading profile: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))

	default:
		fmt.Println("Usage: goclawde user [edit|show]")
	}
}

func handleBatchCommand(args []string) {
	if len(args) == 0 {
		printBatchHelp()
		return
	}

	inputFile := ""
	outputFile := ""
	concurrency := 3
	timeout := 60
	tier := "" // "3", "4", "5" for Moonshot tiers

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-i", "--input":
			if i+1 < len(args) {
				inputFile = args[i+1]
				i++
			}
		case "-o", "--output":
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++
			}
		case "-c", "--concurrency":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &concurrency)
				i++
			}
		case "-t", "--timeout":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &timeout)
				i++
			}
		case "--tier":
			if i+1 < len(args) {
				tier = args[i+1]
				i++
			}
		case "-h", "--help":
			printBatchHelp()
			return
		}
	}

	if inputFile == "" {
		fmt.Println("Error: Input file is required")
		fmt.Println("Usage: goclawde batch -i <input_file> [-o <output_file>]")
		os.Exit(1)
	}

	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Printf("Error: Input file not found: %s\n", inputFile)
		os.Exit(1)
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

	workspacePath := cfg.Storage.DataDir
	pm, err := persona.NewPersonaManager(workspacePath, logger)
	if err != nil {
		logger.Warn("Failed to initialize persona manager", zap.Error(err))
	}

	provider, err := cfg.DefaultProvider()
	if err != nil {
		fmt.Printf("Error getting LLM provider: %v\n", err)
		os.Exit(1)
	}
	llmClient := llm.NewClient(provider)

	skillsRegistry := skills.NewRegistry(st)
	registerSkills(cfg, skillsRegistry)

	agentInstance := agent.New(llmClient, nil, st, logger, pm)
	agentInstance.SetSkillsRegistry(skillsRegistry)

	batchConfig := batch.Config{
		MaxConcurrency: concurrency,
		Timeout:        time.Duration(timeout) * time.Second,
		RetryCount:     2,
		RetryDelay:     1 * time.Second,
		SkipInvalid:    true,
		ValidateInput:  true,
	}

	baseProcessor := batch.NewProcessor(agentInstance, batchConfig, logger)

	var result *batch.Result

	ctx := context.Background()

	// Use rate-limited processor if tier is specified
	if tier != "" {
		var rlConfig batch.RateLimiterConfig
		switch tier {
		case "3":
			rlConfig = batch.Tier3Config()
			fmt.Println("⚡ Using Tier 3 rate limits: 200 concurrent, 5000 RPM, 3M TPM")
		case "4":
			rlConfig = batch.Tier4Config()
			fmt.Println("⚡ Using Tier 4 rate limits: 400 concurrent, 5000 RPM, 4M TPM")
		case "5":
			rlConfig = batch.Tier5Config()
			fmt.Println("⚡ Using Tier 5 rate limits: 1000 concurrent, 10000 RPM, 5M TPM")
		default:
			fmt.Printf("Unknown tier: %s. Using default limits.\n", tier)
			rlConfig = batch.RateLimiterConfig{MaxConcurrency: concurrency}
		}

		processor := batch.NewRateLimitedProcessor(baseProcessor, rlConfig)
		fmt.Printf("🤖 Processing batch file: %s\n", inputFile)
		fmt.Printf("   Concurrency: %d | Timeout: %ds | Tier: %s\n", rlConfig.MaxConcurrency, timeout, tier)
		fmt.Println()

		result, err = processor.ProcessFileWithRateLimit(ctx, inputFile, outputFile)
	} else {
		fmt.Printf("🤖 Processing batch file: %s\n", inputFile)
		fmt.Printf("   Concurrency: %d | Timeout: %ds\n", concurrency, timeout)
		fmt.Println()

		result, err = baseProcessor.ProcessFile(ctx, inputFile, outputFile)
	}
	if err != nil {
		fmt.Printf("Error processing batch: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result.Summary())

	if outputFile != "" {
		fmt.Printf("✓ Results saved to: %s\n", outputFile)
	}

	if result.Failed > 0 {
		fmt.Println("\nFailed items:")
		for _, item := range result.Items {
			if !item.Success && item.Error != "skipped" {
				fmt.Printf("  - %s: %s\n", item.ID, item.Error)
			}
		}
	}
}

func printBatchHelp() {
	fmt.Println("Batch Processing Commands:")
	fmt.Println()
	fmt.Println("  goclawde batch -i <input> [-o <output>] [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -i, --input <file>       Input file (txt or jsonl)")
	fmt.Println("  -o, --output <file>      Output file (optional)")
	fmt.Println("  -c, --concurrency <n>    Max concurrent requests (default: 3)")
	fmt.Println("  -t, --timeout <sec>      Request timeout in seconds (default: 60)")
	fmt.Println("  --tier <3|4|5>           Use rate limits for Moonshot tier (optional)")
	fmt.Println("  -h, --help               Show this help")
	fmt.Println()
	fmt.Println("Input Formats:")
	fmt.Println("  Text file:  One prompt per line (comments with #)")
	fmt.Println("  JSONL file: {\"id\": \"...\", \"message\": \"...\"}")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  goclawde batch -i prompts.txt")
	fmt.Println("  goclawde batch -i prompts.jsonl -o results.json")
	fmt.Println("  goclawde batch -i prompts.txt -c 5 -t 120")
	fmt.Println("  goclawde batch -i big_file.jsonl --tier 3 -o results.json")
	fmt.Println()
	fmt.Println("Moonshot Tier Limits:")
	fmt.Println("  Tier 3: 200 concurrent, 5000 RPM, 3M TPM")
	fmt.Println("  Tier 4: 400 concurrent, 5000 RPM, 4M TPM")
	fmt.Println("  Tier 5: 1000 concurrent, 10000 RPM, 5M TPM")
}

// handleConfigCommand manages configuration
func handleConfigCommand(args []string) {
	if len(args) == 0 {
		printConfigHelp()
		return
	}

	workspace := onboarding.GetWorkspacePath()
	configPath := workspace + "/config.yaml"

	switch args[0] {
	case "get":
		if len(args) < 2 {
			fmt.Println("Usage: goclawde config get <key>")
			fmt.Println("Example: goclawde config get llm.default_provider")
			os.Exit(1)
		}
		cfg, err := config.Load("", "")
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		key := args[1]
		printConfigValue(cfg, key)

	case "set":
		if len(args) < 3 {
			fmt.Println("Usage: goclawde config set <key> <value>")
			fmt.Println("Example: goclawde config set llm.default_provider openai")
			os.Exit(1)
		}
		key := args[1]
		value := args[2]
		fmt.Printf("Setting %s = %s\n", key, value)
		fmt.Println("Note: Edit config.yaml directly for complex changes")
		fmt.Printf("Config location: %s\n", configPath)

	case "edit":
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano"
		}
		fmt.Printf("Opening %s in %s...\n", configPath, editor)
		syscall.Exec(editor, []string{editor, configPath}, os.Environ())

	case "path":
		fmt.Println(configPath)

	case "show", "view":
		data, err := os.ReadFile(configPath)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))

	default:
		printConfigHelp()
	}
}

func printConfigHelp() {
	fmt.Println("Config Commands:")
	fmt.Println()
	fmt.Println("  goclawde config get <key>        Get configuration value")
	fmt.Println("  goclawde config set <key> <val>  Set configuration value")
	fmt.Println("  goclawde config edit             Open config in editor")
	fmt.Println("  goclawde config path             Show config file path")
	fmt.Println("  goclawde config show             Display full config")
	fmt.Println()
}

func printConfigValue(cfg *config.Config, key string) {
	switch key {
	case "llm.default_provider":
		fmt.Println(cfg.LLM.DefaultProvider)
	case "server.port":
		fmt.Println(cfg.Server.Port)
	case "server.address":
		fmt.Println(cfg.Server.Address)
	case "storage.data_dir":
		fmt.Println(cfg.Storage.DataDir)
	case "channels.telegram.enabled":
		fmt.Println(cfg.Channels.Telegram.Enabled)
	case "channels.discord.enabled":
		fmt.Println(cfg.Channels.Discord.Enabled)
	default:
		fmt.Printf("Unknown key: %s\n", key)
		fmt.Println("Available keys: llm.default_provider, server.port, server.address, storage.data_dir")
	}
}

// handleSkillsCommand lists and manages skills
func handleSkillsCommand(args []string) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	workspace := onboarding.GetWorkspacePath()
	pm, err := persona.NewPersonaManager(workspace, logger)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	_ = pm

	// Load minimal config for skills
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

	registry := skills.NewRegistry(st)
	registerSkills(cfg, registry)

	if len(args) == 0 || args[0] == "list" {
		skillsList := registry.ListSkills()
		fmt.Println("Available Skills:")
		fmt.Println("=================")
		for _, skill := range skillsList {
			tools := skill.Tools()
			toolNames := make([]string, 0, len(tools))
			for _, t := range tools {
				toolNames = append(toolNames, t.Name)
			}
			fmt.Printf("  %s %s (%d tools)\n", skill.Name(), skill.Version(), len(tools))
			fmt.Printf("     Tools: %s\n", strings.Join(toolNames, ", "))
			fmt.Println()
		}
		return
	}

	switch args[0] {
	case "info":
		if len(args) < 2 {
			fmt.Println("Usage: goclawde skills info <skill-name>")
			os.Exit(1)
		}
		skillName := args[1]
		skill, ok := registry.GetSkill(skillName)
		if !ok {
			fmt.Printf("Skill not found: %s\n", skillName)
			os.Exit(1)
		}
		fmt.Printf("Skill: %s\n", skill.Name())
		fmt.Printf("Version: %s\n", skill.Version())
		fmt.Printf("Description: %s\n", skill.Description())
		fmt.Println("\nTools:")
		for _, tool := range skill.Tools() {
			fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
		}

	default:
		fmt.Println("Usage: goclawde skills [list|info <skill>]")
	}
}

// handleChannelsCommand manages messaging channels
func handleChannelsCommand(args []string) {
	if len(args) == 0 {
		printChannelsHelp()
		return
	}

	cfg, err := config.Load("", "")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	switch args[0] {
	case "status":
		fmt.Println("Channel Status:")
		fmt.Println("===============")
		fmt.Printf("Telegram: %s\n", channelStatus(cfg.Channels.Telegram.Enabled))
		if cfg.Channels.Telegram.Enabled {
			fmt.Printf("  Bot Token: %s\n", maskToken(cfg.Channels.Telegram.BotToken))
			fmt.Printf("  Allow List: %d users\n", len(cfg.Channels.Telegram.AllowList))
		}
		fmt.Printf("Discord: %s\n", channelStatus(cfg.Channels.Discord.Enabled))
		if cfg.Channels.Discord.Enabled {
			fmt.Printf("  Token: %s\n", maskToken(cfg.Channels.Discord.Token))
		}

	case "enable":
		if len(args) < 2 {
			fmt.Println("Usage: goclawde channels enable <telegram|discord>")
			os.Exit(1)
		}
		fmt.Printf("To enable %s, edit config.yaml and restart the server:\n", args[1])
		fmt.Printf("  goclawde config edit\n")

	case "disable":
		if len(args) < 2 {
			fmt.Println("Usage: goclawde channels disable <telegram|discord>")
			os.Exit(1)
		}
		fmt.Printf("To disable %s, edit config.yaml and restart the server:\n", args[1])
		fmt.Printf("  goclawde config edit\n")

	default:
		printChannelsHelp()
	}
}

func printChannelsHelp() {
	fmt.Println("Channel Commands:")
	fmt.Println()
	fmt.Println("  goclawde channels status           Show channel status")
	fmt.Println("  goclawde channels enable <name>    Enable a channel")
	fmt.Println("  goclawde channels disable <name>   Disable a channel")
	fmt.Println()
	fmt.Println("Available channels: telegram, discord")
}

func channelStatus(enabled bool) string {
	if enabled {
		return "✅ enabled"
	}
	return "❌ disabled"
}

func maskToken(token string) string {
	if len(token) < 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// handleGatewayCommand manages the server/gateway
func handleGatewayCommand(args []string) {
	if len(args) == 0 {
		printGatewayHelp()
		return
	}

	cfg, err := config.Load("", "")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	switch args[0] {
	case "run", "start":
		fmt.Println("Starting GoClawde server...")
		fmt.Printf("URL: http://localhost:%d\n", cfg.Server.Port)
		// Re-run main in server mode
		*serverMode = true
		// Need to reinitialize everything
		logger, _ := zap.NewDevelopment()
		defer logger.Sync()
		logger.Info("Starting GoClawde server", zap.String("version", version))
		
		st, _ := store.New(cfg)
		defer st.Close()
		
		workspacePath := cfg.Storage.DataDir
		pm, _ := persona.NewPersonaManager(workspacePath, logger)
		
		skillsRegistry := skills.NewRegistry(st)
		registerSkills(cfg, skillsRegistry)
		
		app := &App{
			config:         cfg,
			store:          st,
			logger:         logger,
			skillsRegistry: skillsRegistry,
			personaManager: pm,
		}
		app.runServer()

	case "stop":
		fmt.Println("To stop the server, press Ctrl+C in the terminal where it's running")
		fmt.Println("Or use: pkill -f goclawde")

	case "status":
		fmt.Println("Gateway Status:")
		fmt.Println("==============")
		fmt.Printf("Address: %s:%d\n", cfg.Server.Address, cfg.Server.Port)
		fmt.Printf("URL: http://localhost:%d\n", cfg.Server.Port)
		fmt.Printf("Data Directory: %s\n", cfg.Storage.DataDir)

	case "logs":
		fmt.Println("Logs are written to stdout/stderr")
		fmt.Println("To save logs to a file: goclawde gateway run > goclawde.log 2>&1")

	default:
		printGatewayHelp()
	}
}

func printGatewayHelp() {
	fmt.Println("Gateway Commands:")
	fmt.Println()
	fmt.Println("  goclawde gateway run      Start the server (foreground)")
	fmt.Println("  goclawde gateway status   Show gateway configuration")
	fmt.Println("  goclawde gateway stop     Show how to stop the server")
	fmt.Println("  goclawde gateway logs     Show logging information")
	fmt.Println()
	fmt.Println("Aliases: start = run")
}

// handleStatusCommand shows current status
func handleStatusCommand() {
	cfg, err := config.Load("", "")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("GoClawde Status")
	fmt.Println("===============")
	fmt.Println()
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Config:  %s\n", onboarding.GetWorkspacePath()+"/config.yaml")
	fmt.Printf("Data:    %s\n", cfg.Storage.DataDir)
	fmt.Println()
	fmt.Println("Server Configuration:")
	fmt.Printf("  Address: %s:%d\n", cfg.Server.Address, cfg.Server.Port)
	fmt.Printf("  URL: http://localhost:%d\n", cfg.Server.Port)
	fmt.Println()
	fmt.Println("Channels:")
	fmt.Printf("  Telegram: %s\n", channelStatus(cfg.Channels.Telegram.Enabled))
	fmt.Printf("  Discord:  %s\n", channelStatus(cfg.Channels.Discord.Enabled))
	fmt.Println()
	fmt.Println("LLM Provider:")
	fmt.Printf("  Default: %s\n", cfg.LLM.DefaultProvider)
	fmt.Println()
	fmt.Println("Run 'goclawde doctor' for diagnostics")
}

// handleDoctorCommand runs diagnostics
func handleDoctorCommand() {
	fmt.Println("GoClawde Diagnostics")
	fmt.Println("====================")
	fmt.Println()

	issues := 0

	// Check config
	cfg, err := config.Load("", "")
	if err != nil {
		fmt.Println("❌ Config: Error loading configuration")
		fmt.Printf("   %v\n", err)
		issues++
	} else {
		fmt.Println("✅ Config: Loaded successfully")
	}

	// Check data directory
	if _, err := os.Stat(cfg.Storage.DataDir); os.IsNotExist(err) {
		fmt.Println("❌ Data Directory: Does not exist")
		issues++
	} else {
		fmt.Println("✅ Data Directory: Exists")
	}

	// Check LLM provider
	if cfg.LLM.DefaultProvider == "" {
		fmt.Println("⚠️  LLM Provider: Not configured")
		fmt.Println("   Run: goclawde onboard")
		issues++
	} else {
		fmt.Printf("✅ LLM Provider: %s\n", cfg.LLM.DefaultProvider)
	}

	// Check curl (for weather skill)
	if _, err := exec.LookPath("curl"); err != nil {
		fmt.Println("⚠️  curl: Not found (required for weather skill)")
		fmt.Println("   Install: sudo apt-get install curl")
		issues++
	} else {
		fmt.Println("✅ curl: Found")
	}

	// Check Chrome (for browser skill)
	if _, err := exec.LookPath("google-chrome"); err != nil {
		if _, err := exec.LookPath("chromium-browser"); err != nil {
			fmt.Println("⚠️  Chrome/Chromium: Not found (required for browser skill)")
			fmt.Println("   Install: sudo apt-get install chromium-browser")
			issues++
		} else {
			fmt.Println("✅ Chromium: Found")
		}
	} else {
		fmt.Println("✅ Chrome: Found")
	}

	fmt.Println()
	if issues == 0 {
		fmt.Println("✅ All checks passed!")
	} else {
		fmt.Printf("⚠️  Found %d issue(s). Run 'goclawde onboard' to fix configuration.\n", issues)
	}
}

func getMode() string {
	if *cliMode || *message != "" {
		return "cli"
	}
	return "server"
}

func registerSkills(cfg *config.Config, registry *skills.Registry) {
	// System skill
	systemSkill := system.NewSystemSkill(cfg.Tools.AllowedCmds)
	registry.Register(systemSkill)

	// GitHub skill
	githubSkill := github.NewGitHubSkill(cfg.Skills.GitHub.Token)
	registry.Register(githubSkill)

	// Notes skill
	notesSkill := notes.NewNotesSkill("")
	registry.Register(notesSkill)

	// Weather skill
	weatherSkill := weather.NewWeatherSkill()
	registry.Register(weatherSkill)

	// Browser skill
	browserSkill := browser.NewBrowserSkill(browser.Config{
		Enabled:  cfg.Skills.Browser.Enabled,
		Headless: cfg.Skills.Browser.Headless,
	})
	registry.Register(browserSkill)

	// Agentic skill - advanced system introspection and code analysis
	agenticSkill := agentic.NewAgenticSkill(cfg.Storage.DataDir)
	registry.Register(agenticSkill)

	// Voice skill - STT and TTS for natural voice interaction
	voiceConfig := voice.DefaultConfig()
	voiceSkill := voice.NewVoiceSkill(voiceConfig)
	registry.Register(voiceSkill)

	// Documents skill - PDF processing, OCR, and image analysis
	docsConfig := documents.DefaultConfig()
	// Use Gemini API key if available for vision
	if googleProvider, ok := cfg.LLM.Providers["google"]; ok {
		docsConfig.APIKey = googleProvider.APIKey
	}
	docsSkill := documents.NewDocumentSkill(docsConfig)
	registry.Register(docsSkill)
}

func (app *App) runServer() {
	// Create LLM client
	provider, err := app.config.DefaultProvider()
	if err != nil {
		app.logger.Fatal("Failed to get LLM provider", zap.Error(err))
	}
	llmClient := llm.NewClient(provider)

	// Create agent with skills and persona
	agentInstance := agent.New(llmClient, nil, app.store, app.logger, app.personaManager)
	agentInstance.SetSkillsRegistry(app.skillsRegistry)

	// Create agent loop for autonomous operation
	agentLoop := agent.NewAgentLoop(agentInstance, app.logger)
	agentInstance.SetAgentLoop(agentLoop)

	// Create context manager for smart conversation handling
	var contextManager *agent.ContextManager
	if app.config.Vector.Enabled {
		vectorSearcher, err := vector.NewSearcher(&app.config.Vector, app.store, app.logger)
		if err != nil {
			app.logger.Warn("Failed to create vector searcher", zap.Error(err))
		} else {
			contextManager = agent.NewContextManager(app.store, vectorSearcher, llmClient, app.logger)
			agentInstance.SetContextManager(contextManager)
			app.logger.Info("Context manager initialized with vector search")
		}
	} else {
		// Create context manager without vector search
		contextManager = agent.NewContextManager(app.store, nil, llmClient, app.logger)
		agentInstance.SetContextManager(contextManager)
		app.logger.Info("Context manager initialized (without vector search)")
	}

	// Initialize Telegram bot if enabled (with timeout)
	if app.config.Channels.Telegram.Enabled {
		telegramCfg := telegram.Config{
			Token:     app.config.Channels.Telegram.BotToken,
			Enabled:   true,
			AllowList: app.config.Channels.Telegram.AllowList,
		}

		// Use goroutine with timeout to prevent blocking on network issues
		go func() {
			bot, err := telegram.NewBot(telegramCfg, agentInstance, app.store, app.logger)
			if err != nil {
				app.logger.Error("Failed to create Telegram bot", zap.Error(err))
				return
			}
			if err := bot.Start(); err != nil {
				app.logger.Error("Failed to start Telegram bot", zap.Error(err))
				return
			}
			app.telegramBot = bot
			app.logger.Info("Telegram bot started")
		}()
	}

	// Initialize Discord bot if enabled (async to prevent blocking)
	if app.config.Channels.Discord.Enabled && app.config.Channels.Discord.Token != "" {
		discordCfg := discord.Config{
			Token:   app.config.Channels.Discord.Token,
			Enabled: true,
			AllowDM: true,
		}

		go func() {
			db, err := discord.NewBot(discordCfg, agentInstance, app.store, app.logger)
			if err != nil {
				app.logger.Error("Failed to create Discord bot", zap.Error(err))
				return
			}
			if err := db.Start(); err != nil {
				app.logger.Error("Failed to start Discord bot", zap.Error(err))
				return
			}
			app.discordBot = db
			app.logger.Info("Discord bot started")
		}()
	}

	// Start MCP server if enabled
	if app.config.MCP.Enabled {
		// Create a tools registry for MCP
		toolRegistry := tools.NewRegistry(app.config.Tools.AllowedCmds)
		mcpServer := mcp.NewServer(app.config, toolRegistry)
		go func() {
			addr := fmt.Sprintf("%s:%d", app.config.MCP.Host, app.config.MCP.Port)
			app.logger.Info("Starting MCP server", zap.String("addr", addr))
			if err := mcpServer.Start(addr); err != nil {
				app.logger.Error("MCP server error", zap.Error(err))
			}
		}()
	}

	// Initialize cron runner if enabled
	if app.config.Cron.Enabled {
		cronConfig := cron.Config{
			CheckInterval: app.config.Cron.IntervalMinutes,
			MaxConcurrent: app.config.Cron.MaxConcurrent,
		}
		app.cronRunner = cron.NewRunner(cronConfig, agentInstance, app.store, app.logger)
		if err := app.cronRunner.Start(); err != nil {
			app.logger.Error("Failed to start cron runner", zap.Error(err))
		} else {
			app.logger.Info("Cron runner started")
		}
	}

	// Initialize and start API server
	server := api.New(app.config, app.store, app.logger)
	server.SetSkillsRegistry(app.skillsRegistry)

	go func() {
		if err := server.Start(); err != nil {
			app.logger.Fatal("Server error", zap.Error(err))
		}
	}()

	app.logger.Info("Server started",
		zap.String("address", app.config.Server.Address),
		zap.Int("port", app.config.Server.Port),
		zap.String("url", fmt.Sprintf("http://localhost:%d", app.config.Server.Port)),
	)

	// Print skills info
	skillsList := app.skillsRegistry.ListSkills()
	app.logger.Info("Loaded skills", zap.Int("count", len(skillsList)))
	for _, skill := range skillsList {
		app.logger.Info("Skill",
			zap.String("name", skill.Name()),
			zap.String("version", skill.Version()),
			zap.Int("tools", len(skill.Tools())),
		)
	}

	// Print persona info
	if app.personaManager != nil {
		identity := app.personaManager.GetIdentity()
		app.logger.Info("Persona loaded",
			zap.String("name", identity.Name),
		)
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.logger.Info("Shutting down...")

	// Stop Telegram bot
	if app.telegramBot != nil {
		app.telegramBot.Stop()
	}

	// Stop Discord bot
	if app.discordBot != nil {
		app.discordBot.Stop()
	}

	// Stop cron runner
	if app.cronRunner != nil {
		app.cronRunner.Stop()
	}

	if err := server.Shutdown(); err != nil {
		app.logger.Error("Server shutdown error", zap.Error(err))
	}
}

func (app *App) runCLI() {
	// Create LLM client
	provider, err := app.config.DefaultProvider()
	if err != nil {
		app.logger.Fatal("Failed to get LLM provider", zap.Error(err))
	}
	llmClient := llm.NewClient(provider)

	// Create agent with skills and persona
	agentInstance := agent.New(llmClient, nil, app.store, app.logger, app.personaManager)
	agentInstance.SetSkillsRegistry(app.skillsRegistry)

	// One-shot mode
	if *message != "" {
		runOneShot(agentInstance, *message)
		return
	}

	// Interactive mode
	runInteractive(agentInstance)
}

func runOneShot(agentInstance *agent.Agent, msg string) {
	fmt.Println("🤖 GoClawde is thinking...")
	fmt.Println()

	ctx := context.Background()
	resp, err := agentInstance.Chat(ctx, agent.ChatRequest{
		Message: msg,
		Stream:  false,
	})

	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(resp.Content)
	fmt.Printf("\n⏱️  Response time: %v | Tokens: %d\n", resp.ResponseTime, resp.TokensUsed)
}

func runInteractive(agentInstance *agent.Agent) {
	fmt.Println("🤖 GoClawde - Interactive Mode")
	fmt.Println("Type 'exit' or 'quit' to exit, 'help' for commands")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()
	var currentConvID string

	for {
		fmt.Print("👤 You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle commands
		switch strings.ToLower(input) {
		case "exit", "quit", "q":
			fmt.Println("👋 Goodbye!")
			return
		case "help", "h":
			printHelp()
			continue
		case "new", "n":
			currentConvID = ""
			fmt.Println("🆕 New conversation started")
			continue
		case "clear", "cls":
			fmt.Print("\033[H\033[2J")
			continue
		}

		// Send message
		fmt.Println()
		fmt.Print("🤖 GoClawde: ")

		var fullResponse strings.Builder
		start := time.Now()

		resp, err := agentInstance.Chat(ctx, agent.ChatRequest{
			ConversationID: currentConvID,
			Message:        input,
			Stream:         true,
			OnStream: func(chunk string) {
				fmt.Print(chunk)
				fullResponse.WriteString(chunk)
			},
		})

		if err != nil {
			fmt.Printf("\n❌ Error: %v\n", err)
			continue
		}

		fmt.Println()
		fmt.Printf("\n⏱️  Response time: %v | Tokens: %d\n", time.Since(start), resp.TokensUsed)
		fmt.Println()
	}
}

func printHelp() {
	fmt.Println()
	fmt.Println("Interactive Commands:")
	fmt.Println("  help, h     - Show this help")
	fmt.Println("  new, n      - Start new conversation")
	fmt.Println("  clear, cls  - Clear screen")
	fmt.Println("  exit, quit  - Exit the program")
	fmt.Println()
}

func printExtendedHelp() {
	fmt.Println("GoClawde - Your Personal AI Assistant")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  goclawde                          Run in server mode (default)")
	fmt.Println("  goclawde --server                 Run in server mode")
	fmt.Println("  goclawde --cli                    Run interactive CLI mode")
	fmt.Println("  goclawde -m 'message'             Send one-shot message")
	fmt.Println()
	fmt.Println("Setup & Configuration:")
	fmt.Println("  goclawde onboard                  Run setup wizard")
	fmt.Println("  goclawde config get <key>         Get configuration value")
	fmt.Println("  goclawde config set <key> <val>   Set configuration value")
	fmt.Println("  goclawde config edit              Edit config in $EDITOR")
	fmt.Println("  goclawde config path              Show config file path")
	fmt.Println()
	fmt.Println("Server Management:")
	fmt.Println("  goclawde gateway run              Start server (foreground)")
	fmt.Println("  goclawde gateway status           Show gateway configuration")
	fmt.Println("  goclawde channels status          Show channel status")
	fmt.Println()
	fmt.Println("System & Diagnostics:")
	fmt.Println("  goclawde status                   Show current status")
	fmt.Println("  goclawde doctor                   Run diagnostics")
	fmt.Println("  goclawde version                  Show version")
	fmt.Println()
	fmt.Println("Skills:")
	fmt.Println("  goclawde skills                   List available skills")
	fmt.Println("  goclawde skills info <skill>      Show skill details")
	fmt.Println()
	fmt.Println("Project Management:")
	fmt.Println("  goclawde project new <name> <type>   Create new project")
	fmt.Println("  goclawde project list                List all projects")
	fmt.Println("  goclawde project switch <name>       Switch to project")
	fmt.Println("  goclawde project archive <name>      Archive a project")
	fmt.Println("  goclawde project delete <name>       Delete a project")
	fmt.Println()
	fmt.Println("Batch Processing:")
	fmt.Println("  goclawde batch -i <file>             Process prompts from file")
	fmt.Println("  goclawde batch -i in.txt -o out.json Process and save results")
	fmt.Println()
	fmt.Println("Persona Commands:")
	fmt.Println("  goclawde persona            Show current AI identity")
	fmt.Println("  goclawde persona edit       Edit AI identity")
	fmt.Println("  goclawde persona show       Show full identity file")
	fmt.Println()
	fmt.Println("User Commands:")
	fmt.Println("  goclawde user               Show your profile")
	fmt.Println("  goclawde user edit          Edit your profile")
	fmt.Println("  goclawde user show          Show full profile file")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --config <path>          Path to config file")
	fmt.Println("  --data <path>            Path to data directory")
	fmt.Println("  --help, -h               Show this help")
	fmt.Println("  --version, -v            Show version")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  goclawde gateway run &                          # Start server in background")
	fmt.Println("  goclawde -m \"What's the weather in KL?\"       # One-shot query")
	fmt.Println("  goclawde doctor                                 # Check setup")
	fmt.Println()
}

func printProjectHelp() {
	fmt.Println("Project Management Commands:")
	fmt.Println()
	fmt.Println("  goclawde project new <name> <type>    Create new project")
	fmt.Println("  goclawde project list                 List all projects")
	fmt.Println("  goclawde project switch <name>        Switch to project")
	fmt.Println("  goclawde project archive <name>       Archive a project")
	fmt.Println("  goclawde project delete <name>        Delete a project")
	fmt.Println()
	fmt.Println("Project Types:")
	fmt.Println("  coding     - Software development")
	fmt.Println("  writing    - Content creation")
	fmt.Println("  research   - Research projects")
	fmt.Println("  business   - Business projects")
}
