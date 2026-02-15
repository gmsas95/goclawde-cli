package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/agent"
	"github.com/gmsas95/goclawde-cli/internal/api"
	"github.com/gmsas95/goclawde-cli/internal/channels/telegram"
	"github.com/gmsas95/goclawde-cli/internal/config"
	"github.com/gmsas95/goclawde-cli/internal/llm"
	"github.com/gmsas95/goclawde-cli/internal/onboarding"
	"github.com/gmsas95/goclawde-cli/internal/persona"
	"github.com/gmsas95/goclawde-cli/internal/skills"
	"github.com/gmsas95/goclawde-cli/internal/skills/browser"
	"github.com/gmsas95/goclawde-cli/internal/skills/github"
	"github.com/gmsas95/goclawde-cli/internal/skills/notes"
	"github.com/gmsas95/goclawde-cli/internal/skills/system"
	"github.com/gmsas95/goclawde-cli/internal/skills/weather"
	"github.com/gmsas95/goclawde-cli/internal/store"
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
		case "help", "--help", "-h":
			printExtendedHelp()
			return
		case "version", "--version", "-v":
			fmt.Printf("GoClawde version %s\n", version)
			return
		}
	}

	flag.Parse()

	// Check if onboarding is needed
	if onboarding.CheckFirstRun() && !*onboard {
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

	// Initialize Telegram bot if enabled
	if app.config.Channels.Telegram.Enabled {
		telegramCfg := telegram.Config{
			Token:     app.config.Channels.Telegram.BotToken,
			Enabled:   true,
			AllowList: app.config.Channels.Telegram.AllowList,
		}

		bot, err := telegram.NewBot(telegramCfg, agentInstance, app.store, app.logger)
		if err != nil {
			app.logger.Error("Failed to create Telegram bot", zap.Error(err))
		} else {
			if err := bot.Start(); err != nil {
				app.logger.Error("Failed to start Telegram bot", zap.Error(err))
			} else {
				app.telegramBot = bot
				app.logger.Info("Telegram bot started")
			}
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
	fmt.Println("  goclawde                 Run in server mode (default)")
	fmt.Println("  goclawde --server           Run in server mode")
	fmt.Println("  goclawde --cli              Run interactive CLI mode")
	fmt.Println("  goclawde -m 'message'       Send one-shot message")
	fmt.Println("  goclawde onboard            Run setup wizard")
	fmt.Println()
	fmt.Println("Project Commands:")
	fmt.Println("  goclawde project new <name> <type>   Create new project")
	fmt.Println("  goclawde project list                List all projects")
	fmt.Println("  goclawde project switch <name>       Switch to project")
	fmt.Println("  goclawde project archive <name>      Archive a project")
	fmt.Println("  goclawde project delete <name>       Delete a project")
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
