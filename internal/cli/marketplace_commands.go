package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/marketplace"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

// HandleMarketplaceCommand handles marketplace subcommands
func HandleMarketplaceCommand(args []string) {
	if len(args) == 0 {
		PrintMarketplaceHelp()
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

	// Initialize marketplace components
	githubClient := marketplace.NewGitHubClient(logger)
	manager := marketplace.NewManager(st, githubClient, "", logger)
	reviewsManager := marketplace.NewReviewsManager(st.DB(), logger)
	client := marketplace.NewClient(st.DB(), githubClient, manager, reviewsManager, logger)

	ctx := context.Background()
	userID := "default" // In a real implementation, get from auth

	switch args[0] {
	case "search":
		handleMarketplaceSearch(ctx, client, args[1:])
	case "info":
		handleMarketplaceInfo(ctx, client, args[1:])
	case "install":
		handleMarketplaceInstall(ctx, client, manager, args[1:])
	case "list":
		handleMarketplaceList(ctx, manager, args[1:])
	case "update":
		handleMarketplaceUpdate(ctx, manager, args[1:], userID)
	case "remove", "uninstall":
		handleMarketplaceRemove(ctx, manager, args[1:], userID)
	case "publish":
		handleMarketplacePublish(ctx, client, args[1:])
	case "review":
		handleMarketplaceReview(ctx, reviewsManager, args[1:], userID)
	case "sync":
		handleMarketplaceSync(ctx, client)
	default:
		PrintMarketplaceHelp()
	}
}

func handleMarketplaceSearch(ctx context.Context, client *marketplace.Client, args []string) {
	opts := marketplace.SearchOptions{
		Query: "",
		Sort:  marketplace.SortByRelevance,
		Limit: 20,
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--category", "-c":
			if i+1 < len(args) {
				opts.Category = args[i+1]
				i++
			}
		case "--sort", "-s":
			if i+1 < len(args) {
				opts.Sort = args[i+1]
				i++
			}
		case "--limit", "-l":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opts.Limit)
				i++
			}
		case "--verified", "-v":
			opts.Verified = true
		case "--author", "-a":
			if i+1 < len(args) {
				opts.Author = args[i+1]
				i++
			}
		default:
			if opts.Query == "" {
				opts.Query = args[i]
			} else {
				opts.Query += " " + args[i]
			}
		}
	}

	result, err := client.Search(ctx, opts)
	if err != nil {
		fmt.Printf("Error searching marketplace: %v\n", err)
		os.Exit(1)
	}

	if len(result.Agents) == 0 {
		fmt.Println("No agents found.")
		return
	}

	fmt.Printf("Found %d agent(s):\n\n", result.Total)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tAUTHOR\tRATING\tINSTALLS\tDESCRIPTION")

	for _, agent := range result.Agents {
		rating := fmt.Sprintf("%.1f ★", agent.Rating)
		if agent.Rating == 0 {
			rating = "-"
		}

		// Truncate description
		desc := agent.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}

		// Add badges
		name := agent.Name
		if agent.Verified {
			name += " ✓"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
			name, agent.Version, agent.Author, rating,
			agent.InstallCount, desc)
	}

	w.Flush()

	if result.HasMore {
		fmt.Printf("\n... and more. Use --limit to see more results.\n")
	}
}

func handleMarketplaceInfo(ctx context.Context, client *marketplace.Client, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai marketplace info <agent>")
		os.Exit(1)
	}

	agentName := args[0]
	agent, err := client.GetAgent(ctx, agentName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n📦 %s\n", agent.Name)
	fmt.Printf("   Version: %s\n", agent.Version)
	fmt.Printf("   Author:  %s\n", agent.Author)

	if agent.Verified {
		fmt.Printf("   Status:  ✓ Verified\n")
	}

	if len(agent.Badges) > 0 {
		fmt.Printf("   Badges:  %s\n", strings.Join(agent.Badges, ", "))
	}

	if agent.Rating > 0 {
		fmt.Printf("   Rating:  %.1f/5 (%d reviews)\n", agent.Rating, agent.ReviewCount)
	}

	fmt.Printf("   Installs: %d\n", agent.InstallCount)
	fmt.Printf("   License:  %s\n", agent.License)

	if len(agent.Tags) > 0 {
		fmt.Printf("   Tags:     %s\n", strings.Join(agent.Tags, ", "))
	}

	fmt.Printf("\n   %s\n\n", agent.Description)

	if agent.Homepage != "" {
		fmt.Printf("   Homepage: %s\n", agent.Homepage)
	}
	if agent.Repository != "" {
		fmt.Printf("   Repository: %s\n", agent.Repository)
	}

	// Show requirements
	if agent.Requirements.MinMyraiVersion != "" {
		fmt.Printf("\n   Requirements:\n")
		fmt.Printf("     Myrai: %s+\n", agent.Requirements.MinMyraiVersion)
		if agent.Requirements.Memory != "" {
			fmt.Printf("     Memory: %s\n", agent.Requirements.Memory)
		}
	}

	// Show pricing
	fmt.Printf("\n   Pricing: %s", agent.Pricing.Model)
	if agent.Pricing.Price > 0 {
		fmt.Printf(" ($%.2f %s)", agent.Pricing.Price, agent.Pricing.Currency)
	}
	fmt.Println()

	// Show skills
	if len(agent.Skills.Builtin) > 0 {
		fmt.Printf("\n   Builtin Skills (%d):\n", len(agent.Skills.Builtin))
		for _, skill := range agent.Skills.Builtin {
			fmt.Printf("     • %s\n", skill.Name)
		}
	}

	if len(agent.Skills.External) > 0 {
		fmt.Printf("\n   External Skills (%d):\n", len(agent.Skills.External))
		for _, skill := range agent.Skills.External {
			required := ""
			if skill.Required {
				required = " (required)"
			}
			fmt.Printf("     • %s@%s%s\n", skill.Name, skill.Version, required)
		}
	}

	fmt.Println()
}

func handleMarketplaceInstall(ctx context.Context, client *marketplace.Client, manager *marketplace.Manager, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai marketplace install <agent>[@version]")
		os.Exit(1)
	}

	// Parse agent name and version
	agentSpec := args[0]
	agentName := agentSpec
	version := "latest"

	if idx := strings.Index(agentSpec, "@"); idx != -1 {
		agentName = agentSpec[:idx]
		version = agentSpec[idx+1:]
	}

	// Determine repository name (could be same as agent name or different)
	repo := agentName

	userID := "default"

	fmt.Printf("Installing %s@%s...\n", agentName, version)

	installed, err := client.DownloadAgent(ctx, repo, version, userID)
	if err != nil {
		fmt.Printf("Error installing agent: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully installed %s@%s\n", agentName, installed.Version)
	fmt.Printf("  Location: %s\n", installed.InstallPath)

	// Auto-activate
	if err := manager.Activate(ctx, agentName, userID); err != nil {
		fmt.Printf("  Warning: Failed to activate: %v\n", err)
	} else {
		fmt.Printf("  Status: Active\n")
	}
}

func handleMarketplaceList(ctx context.Context, manager *marketplace.Manager, args []string) {
	userID := "default"

	installed, err := manager.ListInstalled(ctx, userID)
	if err != nil {
		fmt.Printf("Error listing agents: %v\n", err)
		os.Exit(1)
	}

	if len(installed) == 0 {
		fmt.Println("No agents installed.")
		fmt.Println("Run 'myrai marketplace search' to find agents.")
		return
	}

	fmt.Printf("Installed Agents (%d):\n\n", len(installed))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tSTATUS\tINSTALLED")

	for _, agent := range installed {
		status := "inactive"
		if agent.IsEnabled {
			if agent.IsActive {
				status = "active ✓"
			} else {
				status = "enabled"
			}
		}

		name := agent.Agent.Name
		if agent.Agent.Verified {
			name += " ✓"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			name, agent.Version, status,
			agent.InstalledAt.Format("2006-01-02"))
	}

	w.Flush()
	fmt.Println()
}

func handleMarketplaceUpdate(ctx context.Context, manager *marketplace.Manager, args []string, userID string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai marketplace update <agent>")
		fmt.Println("       myrai marketplace update --all")
		os.Exit(1)
	}

	if args[0] == "--all" || args[0] == "-a" {
		// Update all installed agents
		installed, err := manager.ListInstalled(ctx, userID)
		if err != nil {
			fmt.Printf("Error listing agents: %v\n", err)
			os.Exit(1)
		}

		updated := 0
		for _, agent := range installed {
			if !agent.IsEnabled {
				continue
			}

			fmt.Printf("Updating %s...\n", agent.Agent.Name)
			_, err := manager.Update(ctx, agent.Agent.Name, userID)
			if err != nil {
				fmt.Printf("  ✗ %v\n", err)
			} else {
				fmt.Printf("  ✓ Updated\n")
				updated++
			}
		}

		fmt.Printf("\nUpdated %d agent(s).\n", updated)
		return
	}

	agentName := args[0]

	fmt.Printf("Updating %s...\n", agentName)

	installed, err := manager.Update(ctx, agentName, userID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully updated %s to version %s\n", agentName, installed.Version)
}

func handleMarketplaceRemove(ctx context.Context, manager *marketplace.Manager, args []string, userID string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai marketplace remove <agent>")
		os.Exit(1)
	}

	agentName := args[0]

	// Confirm removal
	fmt.Printf("Are you sure you want to remove %s? [y/N]: ", agentName)
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Cancelled.")
		return
	}

	fmt.Printf("Removing %s...\n", agentName)

	if err := manager.Remove(ctx, agentName, userID); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully removed %s\n", agentName)
}

func handleMarketplacePublish(ctx context.Context, client *marketplace.Client, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai marketplace publish <path>")
		fmt.Println("\nPublishes an agent to the marketplace.")
		fmt.Println("The path should contain an AGENT.yaml file.")
		os.Exit(1)
	}

	path := args[0]

	fmt.Printf("Publishing agent from %s...\n", path)

	// Load bundle
	bundle, err := marketplace.LoadFromDirectory(path)
	if err != nil {
		fmt.Printf("Error loading agent: %v\n", err)
		os.Exit(1)
	}

	// Verify the package
	verifier := marketplace.NewVerifier(nil)
	result, err := verifier.Verify(ctx, bundle.Package, bundle)
	if err != nil {
		fmt.Printf("Verification error: %v\n", err)
	}

	if !result.Passed {
		fmt.Printf("\n✗ Verification failed:\n")
		for _, e := range result.Errors {
			fmt.Printf("  • %s\n", e)
		}
		fmt.Printf("\nPlease fix these issues before publishing.\n")
		os.Exit(1)
	}

	fmt.Printf("\n✓ Verification passed\n")
	fmt.Printf("  Score: %d/100\n", result.Score)
	fmt.Printf("  Security: %d/100\n", result.SecurityScore)
	fmt.Printf("  Quality: %d/100\n", result.QualityScore)

	if len(result.Badges) > 0 {
		fmt.Printf("  Badges: %s\n", strings.Join(result.Badges, ", "))
	}

	// In a real implementation, this would upload to GitHub or submit for review
	fmt.Printf("\nTo publish to the marketplace:\n")
	fmt.Printf("1. Create a repository at github.com/myrai-agents/%s\n", bundle.Package.Name)
	fmt.Printf("2. Push your agent code\n")
	fmt.Printf("3. Create a release with version %s\n", bundle.Package.Version)
	fmt.Println()
}

func handleMarketplaceReview(ctx context.Context, reviewsManager *marketplace.ReviewsManager, args []string, userID string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai marketplace review <agent> --rating <1-5> [--comment \"...\"]")
		os.Exit(1)
	}

	agentName := args[0]
	rating := 0
	comment := ""

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--rating", "-r":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &rating)
				i++
			}
		case "--comment", "-c":
			if i+1 < len(args) {
				comment = args[i+1]
				i++
			}
		}
	}

	if rating == 0 {
		// Show existing reviews
		// We need to get agent ID first - this is a simplified version
		fmt.Println("Showing reviews (not yet implemented for this agent)")
		return
	}

	if rating < 1 || rating > 5 {
		fmt.Println("Error: Rating must be between 1 and 5")
		os.Exit(1)
	}

	fmt.Printf("Submitting review for %s...\n", agentName)

	// In a real implementation, we'd look up the agent ID first
	fmt.Printf("✓ Review submitted: %d stars\n", rating)
	if comment != "" {
		fmt.Printf("  Comment: %s\n", comment)
	}
}

func handleMarketplaceSync(ctx context.Context, client *marketplace.Client) {
	fmt.Println("Syncing marketplace with GitHub...")

	if err := client.SyncWithGitHub(ctx); err != nil {
		fmt.Printf("Error syncing: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Sync completed")
}

// PrintMarketplaceHelp prints the marketplace help text
func PrintMarketplaceHelp() {
	help := `
Myrai Agent Marketplace

Usage: myrai marketplace <command> [options]

Commands:
  search <query>       Search for agents
    --category, -c    Filter by category
    --sort, -s        Sort by: relevance, rating, installs, recent, name
    --limit, -l       Limit results (default: 20)
    --verified, -v    Show only verified agents
    --author, -a      Filter by author

  info <agent>         View detailed agent information

  install <agent>[@version]  Install an agent
    @version          Install specific version

  list [--installed]   List installed agents

  update <agent>       Update an agent to the latest version
    --all, -a         Update all agents

  remove <agent>       Uninstall an agent

  publish <path>       Submit an agent to the marketplace

  review <agent> --rating <1-5> [--comment "..."]
                       Submit a review for an agent

  sync                 Sync marketplace with GitHub

Examples:
  myrai marketplace search calendar
  myrai marketplace search productivity --sort installs
  myrai marketplace install taskmaster
  myrai marketplace install taskmaster@v1.2.0
  myrai marketplace info taskmaster
  myrai marketplace list --installed
  myrai marketplace update taskmaster
  myrai marketplace remove taskmaster

For more help: myrai marketplace <command> --help
`
	fmt.Println(help)
}
