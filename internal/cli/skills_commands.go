package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/mcp"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

// HandleEnhancedSkillsCommand handles enhanced skill commands
func HandleEnhancedSkillsCommand(args []string) {
	if len(args) == 0 {
		PrintEnhancedSkillsHelp()
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

	// Create enhanced registry
	registry := skills.NewEnhancedRegistry(st)

	// Get skills directory
	skillsDir := getSkillsDir(cfg)

	switch args[0] {
	case "install":
		if len(args) < 2 {
			fmt.Println("Usage: myrai skills install <github.com/user/repo>")
			os.Exit(1)
		}
		repo := args[1]
		installFromGitHub(registry, skillsDir, repo, logger)

	case "update":
		if len(args) < 2 {
			fmt.Println("Usage: myrai skills update <skill-name>")
			os.Exit(1)
		}
		skillName := args[1]
		updateSkill(registry, skillsDir, skillName, logger)

	case "search":
		if len(args) < 2 {
			fmt.Println("Usage: myrai skills search <query>")
			os.Exit(1)
		}
		query := args[1]
		searchSkills(registry, query)

	case "watch":
		if len(args) < 2 {
			fmt.Println("Usage: myrai skills watch <path>")
			os.Exit(1)
		}
		path := args[1]
		watchSkills(registry, skillsDir, path, logger)

	case "enable":
		if len(args) < 2 {
			fmt.Println("Usage: myrai skills enable <skill-name>")
			os.Exit(1)
		}
		skillName := args[1]
		enableSkill(registry, skillName)

	case "disable":
		if len(args) < 2 {
			fmt.Println("Usage: myrai skills disable <skill-name>")
			os.Exit(1)
		}
		skillName := args[1]
		disableSkill(registry, skillName)

	case "uninstall", "remove":
		if len(args) < 2 {
			fmt.Println("Usage: myrai skills uninstall <skill-name>")
			os.Exit(1)
		}
		skillName := args[1]
		uninstallSkill(registry, skillsDir, skillName)

	case "validate":
		if len(args) < 2 {
			fmt.Println("Usage: myrai skills validate <path-to-skill.md>")
			os.Exit(1)
		}
		skillPath := args[1]
		validateSkill(skillPath)

	case "list":
		listSkills(registry)

	case "stats":
		showStats(registry)

	default:
		PrintEnhancedSkillsHelp()
	}
}

// HandleMCPCommand handles MCP server commands
func HandleMCPCommand(args []string) {
	if len(args) == 0 {
		PrintMCPHelp()
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

	// Load existing MCP config
	mcpConfigPath := getMCPConfigPath(cfg)
	serverConfigs, _ := mcp.LoadMCPConfig(mcpConfigPath)

	registry := skills.NewEnhancedRegistry(st)
	adapter := mcp.NewAdapter(registry, mcpConfigPath)
	for _, config := range serverConfigs {
		adapter.AddServer(config)
	}

	switch args[0] {
	case "list":
		listMCPServers(adapter)

	case "tools":
		listMCPTools(registry)

	case "add":
		if len(args) < 2 {
			fmt.Println("Usage: myrai mcp add --name <name> --command <command> [--args <args>]")
			os.Exit(1)
		}
		addMCPServer(adapter, mcpConfigPath, args[1:])

	case "remove", "delete":
		if len(args) < 2 {
			fmt.Println("Usage: myrai mcp remove <server-name>")
			os.Exit(1)
		}
		serverName := args[1]
		removeMCPServer(adapter, mcpConfigPath, serverName)

	case "start":
		if len(args) < 2 {
			fmt.Println("Usage: myrai mcp start <server-name>")
			os.Exit(1)
		}
		serverName := args[1]
		startMCPServer(adapter, serverName)

	case "stop":
		if len(args) < 2 {
			fmt.Println("Usage: myrai mcp stop <server-name>")
			os.Exit(1)
		}
		serverName := args[1]
		stopMCPServer(adapter, serverName)

	case "discover":
		if len(args) > 1 {
			discoverMCPServers(args[1])
		} else {
			discoverMCPServers("")
		}

	case "discover-add":
		if len(args) < 2 {
			fmt.Println("Usage: myrai mcp discover-add <server-name>")
			os.Exit(1)
		}
		serverName := args[1]
		discoverAndAddServer(adapter, mcpConfigPath, serverName)

	default:
		PrintMCPHelp()
	}
}

// ==================== Skills Commands ====================

func installFromGitHub(registry *skills.EnhancedRegistry, skillsDir, repo string, logger *zap.Logger) {
	loader := skills.NewSkillLoader(registry, skillsDir)
	installer := skills.NewGitHubInstaller(loader, skillsDir)

	fmt.Printf("📦 Installing skill from %s...\n", repo)

	skill, err := installer.InstallFromGitHub(repo)
	if err != nil {
		fmt.Printf("❌ Failed to install skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Skill '%s' installed successfully!\n", skill.Manifest.Name)
	fmt.Printf("   Version: %s\n", skill.Manifest.Version)
	fmt.Printf("   Tools: %d\n", len(skill.Tools))
	if len(skill.Manifest.Tags) > 0 {
		fmt.Printf("   Tags: %s\n", strings.Join(skill.Manifest.Tags, ", "))
	}
}

func updateSkill(registry *skills.EnhancedRegistry, skillsDir, skillName string, logger *zap.Logger) {
	loader := skills.NewSkillLoader(registry, skillsDir)
	installer := skills.NewGitHubInstaller(loader, skillsDir)

	fmt.Printf("🔄 Updating skill '%s'...\n", skillName)

	skill, err := installer.UpdateSkill(skillName)
	if err != nil {
		fmt.Printf("❌ Failed to update skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Skill '%s' updated to version %s!\n", skill.Manifest.Name, skill.Manifest.Version)
}

func searchSkills(registry *skills.EnhancedRegistry, query string) {
	// First search local registry
	localResults := registry.SearchSkills(query)

	fmt.Printf("🔍 Search results for '%s':\n\n", query)

	if len(localResults) > 0 {
		fmt.Println("📁 Local Skills:")
		for _, skill := range localResults {
			status := "enabled"
			if !skill.IsEnabled() {
				status = "disabled"
			}
			fmt.Printf("  • %s@%s (%s) - %s\n", skill.Manifest.Name, skill.Manifest.Version, status, skill.Manifest.Description)
		}
		fmt.Println()
	}

	// Search GitHub
	loader := skills.NewSkillLoader(registry, "")
	installer := skills.NewGitHubInstaller(loader, "")
	githubResults, _ := installer.SearchGitHub(query)

	if len(githubResults) > 0 {
		fmt.Println("🌐 Available on GitHub:")
		for _, result := range githubResults {
			fmt.Printf("  • %s@%s - %s (⭐ %d)\n", result.Name, result.Version, result.Description, result.Stars)
			fmt.Printf("    Install: myrai skills install github.com/%s\n", result.Repo)
		}
	}

	if len(localResults) == 0 && len(githubResults) == 0 {
		fmt.Println("No skills found matching your query.")
	}
}

func watchSkills(registry *skills.EnhancedRegistry, skillsDir, path string, logger *zap.Logger) {
	loader := skills.NewSkillLoader(registry, skillsDir)

	watcher, err := skills.NewWatcher(loader)
	if err != nil {
		fmt.Printf("❌ Failed to create watcher: %v\n", err)
		os.Exit(1)
	}

	// Set callbacks
	watcher.SetCallbacks(
		func(skillPath string) {
			fmt.Printf("🔄 Reloaded: %s\n", skillPath)
		},
		func(skillPath string, err error) {
			fmt.Printf("❌ Error reloading %s: %v\n", skillPath, err)
		},
	)

	// Watch directory
	if err := watcher.WatchDirectory(path); err != nil {
		fmt.Printf("❌ Failed to watch directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("👁️  Watching %s for changes...\n", path)
	fmt.Println("Press Ctrl+C to stop watching")

	// Keep running
	select {}
}

func enableSkill(registry *skills.EnhancedRegistry, skillName string) {
	if err := registry.Enable(skillName); err != nil {
		fmt.Printf("❌ Failed to enable skill: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Skill '%s' enabled\n", skillName)
}

func disableSkill(registry *skills.EnhancedRegistry, skillName string) {
	if err := registry.Disable(skillName); err != nil {
		fmt.Printf("❌ Failed to disable skill: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Skill '%s' disabled\n", skillName)
}

func uninstallSkill(registry *skills.EnhancedRegistry, skillsDir, skillName string) {
	skill, ok := registry.GetRuntimeSkill(skillName)
	if !ok {
		fmt.Printf("❌ Skill '%s' not found\n", skillName)
		os.Exit(1)
	}

	if skill.Source == skills.SourceGitHub {
		loader := skills.NewSkillLoader(registry, skillsDir)
		installer := skills.NewGitHubInstaller(loader, skillsDir)
		if err := installer.UninstallSkill(skillName); err != nil {
			fmt.Printf("❌ Failed to uninstall skill: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Just unregister for local/builtin skills
		if err := registry.UnregisterSkill(skillName); err != nil {
			fmt.Printf("❌ Failed to unregister skill: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("✅ Skill '%s' uninstalled\n", skillName)
}

func validateSkill(skillPath string) {
	content, err := os.ReadFile(skillPath)
	if err != nil {
		fmt.Printf("❌ Failed to read skill file: %v\n", err)
		os.Exit(1)
	}

	validator := skills.NewValidator()
	result := validator.ValidateSkillFile(string(content))

	if result.Valid {
		fmt.Printf("✅ Skill '%s' is valid\n", result.SkillName)
		if len(result.Warnings) > 0 {
			fmt.Println("\n⚠️  Warnings:")
			fmt.Print(result.FormatWarnings())
		}
	} else {
		fmt.Printf("❌ Skill validation failed\n\n")
		fmt.Print(result.FormatErrors())
		os.Exit(1)
	}
}

func listSkills(registry *skills.EnhancedRegistry) {
	skills := registry.ListRuntimeSkills()

	if len(skills) == 0 {
		fmt.Println("No skills installed.")
		fmt.Println("Install one with: myrai skills install github.com/user/repo")
		return
	}

	fmt.Println("Installed Skills:")
	fmt.Println("=================")

	for _, skill := range skills {
		status := "✅"
		if !skill.IsEnabled() {
			status = "⏸️"
		}
		if skill.Status == "error" {
			status = "❌"
		}

		source := ""
		switch skill.Source {
		case "github":
			source = "(GitHub)"
		case "local":
			source = "(local)"
		case "builtin":
			source = "(builtin)"
		}

		fmt.Printf("\n%s %s %s\n", status, skill.Manifest.Name, source)
		fmt.Printf("   Version: %s\n", skill.Manifest.Version)
		fmt.Printf("   Description: %s\n", skill.Manifest.Description)
		fmt.Printf("   Tools: %d\n", len(skill.Tools))
		if skill.UseCount > 0 {
			fmt.Printf("   Uses: %d\n", skill.UseCount)
		}
	}
}

func showStats(registry *skills.EnhancedRegistry) {
	stats := registry.GetStats()

	fmt.Println("Skill Registry Statistics:")
	fmt.Println("==========================")
	fmt.Printf("Total Skills: %d\n", stats["total_skills"])
	fmt.Printf("  Enabled: %d\n", stats["enabled"])
	fmt.Printf("  Disabled: %d\n", stats["disabled"])
	fmt.Printf("  Errors: %d\n", stats["errors"])
	fmt.Printf("\nTotal Tools: %d\n", stats["total_tools"])
	fmt.Printf("  MCP Tools: %d\n", stats["mcp_tools"])
}

// ==================== MCP Commands ====================

func listMCPServers(adapter *mcp.Adapter) {
	servers := adapter.ListServers()

	if len(servers) == 0 {
		fmt.Println("No MCP servers configured.")
		fmt.Println("Add one with: myrai mcp add --name <name> --command <command>")
		return
	}

	fmt.Println("Configured MCP Servers:")
	fmt.Println("=======================")

	for _, server := range servers {
		status := "⏹️"
		if server.Enabled {
			status = "▶️"
		}

		fmt.Printf("\n%s %s\n", status, server.Name)
		fmt.Printf("   Command: %s %s\n", server.Command, strings.Join(server.Args, " "))
		if len(server.Env) > 0 {
			fmt.Printf("   Environment: %d variables\n", len(server.Env))
		}
	}
}

func listMCPTools(registry *skills.EnhancedRegistry) {
	tools := registry.ListMCPTools()

	if len(tools) == 0 {
		fmt.Println("No MCP tools available.")
		fmt.Println("Start an MCP server to see available tools.")
		return
	}

	fmt.Printf("Available MCP Tools (%d):\n", len(tools))
	fmt.Println("=====================")

	// Group by server
	toolsByServer := make(map[string][]*skills.MCPTool)
	for _, tool := range tools {
		toolsByServer[tool.ServerName] = append(toolsByServer[tool.ServerName], tool)
	}

	for serverName, serverTools := range toolsByServer {
		fmt.Printf("\n📦 %s:\n", serverName)
		for _, tool := range serverTools {
			fmt.Printf("   • %s - %s\n", tool.Name, tool.Description)
		}
	}
}

func addMCPServer(adapter *mcp.Adapter, configPath string, args []string) {
	// Parse arguments
	config := mcp.MCPServerConfig{Enabled: true}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			if i+1 < len(args) {
				config.Name = args[i+1]
				i++
			}
		case "--command":
			if i+1 < len(args) {
				config.Command = args[i+1]
				i++
			}
		case "--args":
			if i+1 < len(args) {
				config.Args = strings.Split(args[i+1], " ")
				i++
			}
		}
	}

	if config.Name == "" || config.Command == "" {
		fmt.Println("❌ Name and command are required")
		os.Exit(1)
	}

	if err := adapter.AddServer(config); err != nil {
		fmt.Printf("❌ Failed to add server: %v\n", err)
		os.Exit(1)
	}

	// Save configuration
	servers := adapter.ListServers()
	if err := mcp.SaveMCPConfig(configPath, servers); err != nil {
		fmt.Printf("⚠️  Failed to save config: %v\n", err)
	}

	fmt.Printf("✅ MCP server '%s' added\n", config.Name)
}

func removeMCPServer(adapter *mcp.Adapter, configPath, serverName string) {
	if err := adapter.RemoveServer(serverName); err != nil {
		fmt.Printf("❌ Failed to remove server: %v\n", err)
		os.Exit(1)
	}

	// Save configuration
	servers := adapter.ListServers()
	if err := mcp.SaveMCPConfig(configPath, servers); err != nil {
		fmt.Printf("⚠️  Failed to save config: %v\n", err)
	}

	fmt.Printf("✅ MCP server '%s' removed\n", serverName)
}

func startMCPServer(adapter *mcp.Adapter, serverName string) {
	fmt.Printf("🚀 Starting MCP server '%s'...\n", serverName)

	ctx := context.Background()
	if err := adapter.ConnectServer(ctx, serverName); err != nil {
		fmt.Printf("❌ Failed to start server: %v\n", err)
		os.Exit(1)
	}

	// Get tools
	client, _ := adapter.GetClient(serverName)
	tools, _ := client.ListTools(ctx)

	fmt.Printf("✅ MCP server '%s' started\n", serverName)
	fmt.Printf("   Available tools: %d\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("   • %s\n", tool.Name)
	}
}

func stopMCPServer(adapter *mcp.Adapter, serverName string) {
	if err := adapter.DisconnectServer(serverName); err != nil {
		fmt.Printf("❌ Failed to stop server: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("⏹️  MCP server '%s' stopped\n", serverName)
}

func discoverMCPServers(query string) {
	var servers []mcp.DiscoveredServer
	if query != "" {
		servers = mcp.SearchDiscoveredServers(query)
	} else {
		servers = mcp.AutoDiscover()
	}

	if len(servers) == 0 {
		fmt.Println("No MCP servers found.")
		return
	}

	fmt.Println("Available MCP Servers:")
	fmt.Println("=====================")

	for _, server := range servers {
		badge := ""
		if server.Official {
			badge = "[Official]"
		}

		fmt.Printf("\n📦 %s %s\n", server.Name, badge)
		fmt.Printf("   Description: %s\n", server.Description)
		fmt.Printf("   Category: %s\n", server.Category)
		fmt.Printf("   Command: %s %s\n", server.Command, strings.Join(server.Args, " "))
		fmt.Printf("   Add: myrai mcp discover-add %s\n", server.Name)
	}
}

func discoverAndAddServer(adapter *mcp.Adapter, configPath, serverName string) {
	servers := mcp.AutoDiscover()

	var found *mcp.DiscoveredServer
	for _, s := range servers {
		if s.Name == serverName {
			found = &s
			break
		}
	}

	if found == nil {
		fmt.Printf("❌ Server '%s' not found in discovery\n", serverName)
		os.Exit(1)
	}

	config := mcp.MCPServerConfig{
		Name:    found.Name,
		Command: found.Command,
		Args:    found.Args,
		Enabled: true,
	}

	if err := adapter.AddServer(config); err != nil {
		fmt.Printf("❌ Failed to add server: %v\n", err)
		os.Exit(1)
	}

	// Save configuration
	serverConfigs := adapter.ListServers()
	if err := mcp.SaveMCPConfig(configPath, serverConfigs); err != nil {
		fmt.Printf("⚠️  Failed to save config: %v\n", err)
	}

	fmt.Printf("✅ MCP server '%s' added from discovery\n", serverName)
}

// ==================== Helper Functions ====================

func getSkillsDir(cfg *config.Config) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".myrai", "skills")
}

func getMCPConfigPath(cfg *config.Config) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".myrai", "mcp.yaml")
}

// PrintEnhancedSkillsHelp prints help for enhanced skills commands
func PrintEnhancedSkillsHelp() {
	fmt.Println("Myrai Skills Commands:")
	fmt.Println("=====================")
	fmt.Println()
	fmt.Println("Usage: myrai skills <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  install <repo>       Install skill from GitHub (e.g., github.com/user/repo@v1.0.0)")
	fmt.Println("  update <name>        Update skill to latest version")
	fmt.Println("  search <query>       Search for skills locally and on GitHub")
	fmt.Println("  watch <path>         Watch directory for SKILL.md changes (hot-reload)")
	fmt.Println("  enable <name>        Enable a skill")
	fmt.Println("  disable <name>       Disable a skill")
	fmt.Println("  uninstall <name>     Remove a skill")
	fmt.Println("  validate <path>       Validate a SKILL.md file")
	fmt.Println("  list                 List all installed skills")
	fmt.Println("  stats                Show skill registry statistics")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  myrai skills install github.com/myrai-agents/docker-helper")
	fmt.Println("  myrai skills install github.com/myrai-agents/docker-helper@v1.2.0")
	fmt.Println("  myrai skills search docker")
	fmt.Println("  myrai skills watch ./my-custom-skills/")
}

// PrintMCPHelp prints help for MCP commands
func PrintMCPHelp() {
	fmt.Println("Myrai MCP Commands:")
	fmt.Println("==================")
	fmt.Println()
	fmt.Println("Usage: myrai mcp <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list                 List configured MCP servers")
	fmt.Println("  tools                List available MCP tools")
	fmt.Println("  add                  Add MCP server (--name, --command, --args)")
	fmt.Println("  remove <name>        Remove MCP server")
	fmt.Println("  start <name>         Start MCP server and register tools")
	fmt.Println("  stop <name>          Stop MCP server")
	fmt.Println("  discover [query]     Discover available MCP servers")
	fmt.Println("  discover-add <name>  Add discovered MCP server")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  myrai mcp discover")
	fmt.Println("  myrai mcp discover-add filesystem")
	fmt.Println("  myrai mcp add --name myserver --command npx --args \"-y,@modelcontextprotocol/server-filesystem,/home\"")
}
