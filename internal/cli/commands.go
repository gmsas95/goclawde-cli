package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/gmsas95/myrai-cli/internal/app"
	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/onboarding"
	"github.com/gmsas95/myrai-cli/internal/persona"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

var Version = "dev"

func HandleProjectCommand(args []string) {
	if len(args) == 0 {
		PrintProjectHelp()
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
			fmt.Println("Usage: myrai project new <name> <type>")
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
			fmt.Println("No projects found. Create one with: myrai project new <name> <type>")
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
			fmt.Println("Usage: myrai project switch <name>")
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
			fmt.Println("Usage: myrai project archive <name>")
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

		fmt.Printf("✓ Archived project '%s'\n", name)

	case "delete":
		if len(args) < 2 {
			fmt.Println("Usage: myrai project delete <name>")
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
		PrintProjectHelp()
	}
}

func HandlePersonaCommand(args []string) {
	if len(args) == 0 {
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
		fmt.Println("To edit: myrai persona edit")
		return
	}

	switch args[0] {
	case "edit":
		workspace := onboarding.GetWorkspacePath()
		identityPath := workspace + "/IDENTITY.md"

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
		fmt.Println("Usage: myrai persona [edit|show]")
	}
}

func HandleUserCommand(args []string) {
	if len(args) == 0 {
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
		fmt.Println("To edit: myrai user edit")
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
		fmt.Println("Usage: myrai user [edit|show]")
	}
}

func HandleConfigCommand(args []string) {
	if len(args) == 0 {
		PrintConfigHelp()
		return
	}

	workspace := onboarding.GetWorkspacePath()
	configPath := workspace + "/config.yaml"

	switch args[0] {
	case "get":
		if len(args) < 2 {
			fmt.Println("Usage: myrai config get <key>")
			fmt.Println("Example: myrai config get llm.default_provider")
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
			fmt.Println("Usage: myrai config set <key> <value>")
			fmt.Println("Example: myrai config set llm.default_provider openai")
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
		PrintConfigHelp()
	}
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

func HandleSkillsCommand(args []string) {
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

	registry := skills.NewRegistry(st)
	app.RegisterSkills(cfg, st, registry, logger)

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
			fmt.Println("Usage: myrai skills info <skill-name>")
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
		fmt.Println("Usage: myrai skills [list|info <skill>]")
	}
}

func HandleChannelsCommand(args []string) {
	if len(args) == 0 {
		PrintChannelsHelp()
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
			fmt.Println("Usage: myrai channels enable <telegram|discord>")
			os.Exit(1)
		}
		fmt.Printf("To enable %s, edit config.yaml and restart the server:\n", args[1])
		fmt.Printf("  myrai config edit\n")

	case "disable":
		if len(args) < 2 {
			fmt.Println("Usage: myrai channels disable <telegram|discord>")
			os.Exit(1)
		}
		fmt.Printf("To disable %s, edit config.yaml and restart the server:\n", args[1])
		fmt.Printf("  myrai config edit\n")

	default:
		PrintChannelsHelp()
	}
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

func HandleGatewayCommand(args []string, application *app.App) {
	if len(args) == 0 {
		PrintGatewayHelp()
		return
	}

	switch args[0] {
	case "run", "start":
		fmt.Println("Starting Myrai server...")
		fmt.Printf("URL: http://localhost:%d\n", application.Config.Server.Port)
		application.RunServer()

	case "stop":
		fmt.Println("To stop the server, press Ctrl+C in the terminal where it's running")
		fmt.Println("Or use: pkill -f myrai")

	case "status":
		fmt.Println("Gateway Status:")
		fmt.Println("==============")
		fmt.Printf("Address: %s:%d\n", application.Config.Server.Address, application.Config.Server.Port)
		fmt.Printf("URL: http://localhost:%d\n", application.Config.Server.Port)
		fmt.Printf("Data Directory: %s\n", application.Config.Storage.DataDir)

	case "logs":
		fmt.Println("Logs are written to stdout/stderr")
		fmt.Println("To save logs to a file: myrai gateway run > myrai.log 2>&1")

	default:
		PrintGatewayHelp()
	}
}

func HandleStatusCommand() {
	cfg, err := config.Load("", "")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Myrai Status")
	fmt.Println("===============")
	fmt.Println()
	fmt.Printf("Version: %s\n", Version)
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
	fmt.Println("Run 'myrai doctor' for diagnostics")
}

func HandleDoctorCommand() {
	fmt.Println("Myrai Diagnostics")
	fmt.Println("====================")
	fmt.Println()

	issues := 0

	cfg, err := config.Load("", "")
	if err != nil {
		fmt.Println("❌ Config: Error loading configuration")
		fmt.Printf("   %v\n", err)
		issues++
	} else {
		fmt.Println("✅ Config: Loaded successfully")
	}

	if _, err := os.Stat(cfg.Storage.DataDir); os.IsNotExist(err) {
		fmt.Println("❌ Data Directory: Does not exist")
		issues++
	} else {
		fmt.Println("✅ Data Directory: Exists")
	}

	if cfg.LLM.DefaultProvider == "" {
		fmt.Println("⚠️  LLM Provider: Not configured")
		fmt.Println("   Run: myrai onboard")
		issues++
	} else {
		fmt.Printf("✅ LLM Provider: %s\n", cfg.LLM.DefaultProvider)
	}

	if _, err := exec.LookPath("curl"); err != nil {
		fmt.Println("⚠️  curl: Not found (required for weather skill)")
		fmt.Println("   Install: sudo apt-get install curl")
		issues++
	} else {
		fmt.Println("✅ curl: Found")
	}

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
		fmt.Printf("⚠️  Found %d issue(s). Run 'myrai onboard' to fix configuration.\n", issues)
	}
}
