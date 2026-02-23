package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/onboarding"
	"github.com/gmsas95/myrai-cli/internal/persona"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

// HandlePersonaEvolutionCommand handles persona evolution-related commands
func HandlePersonaEvolutionCommand(args []string) {
	if len(args) == 0 {
		PrintPersonaEvolutionHelp()
		return
	}

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	switch args[0] {
	case "proposals":
		if len(args) > 1 && args[1] == "pending" {
			HandlePersonaProposalsPending()
		} else {
			HandlePersonaProposalsList()
		}

	case "proposal":
		if len(args) < 2 {
			fmt.Println("Usage: myrai persona proposal [show|apply|reject] <id>")
			os.Exit(1)
		}

		if len(args) < 3 {
			fmt.Printf("Usage: myrai persona proposal %s <id>\n", args[1])
			os.Exit(1)
		}

		proposalID := args[2]

		switch args[1] {
		case "show":
			HandlePersonaProposalShow(proposalID)
		case "apply", "approve":
			HandlePersonaProposalApply(proposalID)
		case "reject":
			HandlePersonaProposalReject(proposalID)
		default:
			fmt.Printf("Unknown proposal command: %s\n", args[1])
			os.Exit(1)
		}

	case "history":
		HandlePersonaHistory()

	case "rollback":
		if len(args) < 2 {
			fmt.Println("Usage: myrai persona rollback <version-id>")
			os.Exit(1)
		}
		HandlePersonaRollback(args[1])

	case "analyze":
		HandlePersonaAnalyze()

	case "config":
		if len(args) > 1 && args[1] == "show" {
			HandlePersonaEvolutionConfigShow()
		} else {
			PrintPersonaEvolutionHelp()
		}

	default:
		PrintPersonaEvolutionHelp()
	}
}

// HandlePersonaProposalsPending shows pending proposals
func HandlePersonaProposalsPending() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, store, pm := initPersonaEvolution(logger)
	if cfg == nil {
		return
	}
	defer store.Close()

	proposalManager := persona.NewProposalManager(store.DB())
	proposals, err := proposalManager.ListPending()
	if err != nil {
		fmt.Printf("Error listing proposals: %v\n", err)
		os.Exit(1)
	}

	if len(proposals) == 0 {
		fmt.Println("No pending proposals found.")
		fmt.Println("\nTo generate new proposals, run: myrai persona analyze")
		return
	}

	fmt.Println(proposalManager.DisplayProposalsList(proposals))
	_ = pm // Use pm if needed
}

// HandlePersonaProposalsList shows all proposals
func HandlePersonaProposalsList() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, store, pm := initPersonaEvolution(logger)
	if cfg == nil {
		return
	}
	defer store.Close()

	proposalManager := persona.NewProposalManager(store.DB())
	proposals, err := proposalManager.ListAll(50)
	if err != nil {
		fmt.Printf("Error listing proposals: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(proposalManager.DisplayProposalsList(proposals))
	_ = pm
}

// HandlePersonaProposalShow shows details of a specific proposal
func HandlePersonaProposalShow(proposalID string) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, store, pm := initPersonaEvolution(logger)
	if cfg == nil {
		return
	}
	defer store.Close()

	proposalManager := persona.NewProposalManager(store.DB())
	proposal, err := proposalManager.Get(proposalID)
	if err != nil {
		fmt.Printf("Proposal not found: %s\n", proposalID)
		os.Exit(1)
	}

	fmt.Println(proposalManager.DisplayProposal(proposal))
	_ = pm
}

// HandlePersonaProposalApply approves and applies a proposal
func HandlePersonaProposalApply(proposalID string) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, store, pm := initPersonaEvolution(logger)
	if cfg == nil {
		return
	}
	defer store.Close()

	proposalManager := persona.NewProposalManager(store.DB())
	proposal, err := proposalManager.Get(proposalID)
	if err != nil {
		fmt.Printf("Proposal not found: %s\n", proposalID)
		os.Exit(1)
	}

	if proposal.Status != persona.ProposalPending {
		fmt.Printf("Proposal %s is not pending (status: %s)\n", proposalID, proposal.Status)
		os.Exit(1)
	}

	// Show proposal details and ask for confirmation
	fmt.Println(proposalManager.DisplayProposal(proposal))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Apply this change? (yes/no): ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "yes" {
		fmt.Println("Cancelled")
		return
	}

	// Apply the change
	if err := applyProposal(pm, proposal); err != nil {
		fmt.Printf("Error applying proposal: %v\n", err)
		os.Exit(1)
	}

	// Mark as applied
	if err := proposalManager.Apply(proposalID); err != nil {
		fmt.Printf("Error updating proposal status: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Applied proposal: %s\n", proposal.Title)
}

// HandlePersonaProposalReject rejects a proposal
func HandlePersonaProposalReject(proposalID string) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, store, pm := initPersonaEvolution(logger)
	if cfg == nil {
		return
	}
	defer store.Close()

	proposalManager := persona.NewProposalManager(store.DB())
	proposal, err := proposalManager.Get(proposalID)
	if err != nil {
		fmt.Printf("Proposal not found: %s\n", proposalID)
		os.Exit(1)
	}

	if proposal.Status != persona.ProposalPending {
		fmt.Printf("Proposal %s is not pending (status: %s)\n", proposalID, proposal.Status)
		os.Exit(1)
	}

	// Ask for rejection reason (optional)
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Rejection reason (optional): ")
	reason, _ := reader.ReadString('\n')
	reason = strings.TrimSpace(reason)

	if err := proposalManager.Reject(proposalID, reason); err != nil {
		fmt.Printf("Error rejecting proposal: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Rejected proposal: %s\n", proposal.Title)
	_ = pm
}

// HandlePersonaHistory shows evolution history
func HandlePersonaHistory() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, store, pm := initPersonaEvolution(logger)
	if cfg == nil {
		return
	}
	defer store.Close()

	versionManager := persona.NewVersionManager(store.DB(), pm.GetWorkspacePath())
	versions, err := versionManager.ListVersions(20)
	if err != nil {
		fmt.Printf("Error listing versions: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(versionManager.DisplayVersionList(versions))
}

// HandlePersonaRollback rolls back to a specific version
func HandlePersonaRollback(versionID string) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, store, pm := initPersonaEvolution(logger)
	if cfg == nil {
		return
	}
	defer store.Close()

	versionManager := persona.NewVersionManager(store.DB(), pm.GetWorkspacePath())
	version, err := versionManager.GetVersion(versionID)
	if err != nil {
		fmt.Printf("Version not found: %s\n", versionID)
		os.Exit(1)
	}

	// Show version details
	fmt.Println(versionManager.DisplayVersion(version))
	fmt.Println()

	// Confirm rollback
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Rollback to this version? (yes/no): ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "yes" {
		fmt.Println("Cancelled")
		return
	}

	// Perform rollback
	rollbackVersion, identity, _, err := versionManager.Rollback(versionID)
	if err != nil {
		fmt.Printf("Error rolling back: %v\n", err)
		os.Exit(1)
	}

	// Apply the restored persona
	if identity != nil {
		if err := pm.SetIdentity(identity); err != nil {
			fmt.Printf("Error restoring identity: %v\n", err)
			os.Exit(1)
		}
	}

	// Note: UserProfile restoration would require additional methods in PersonaManager
	// For now, we just save the identity

	fmt.Printf("✓ Rolled back to version: %s\n", versionID)
	fmt.Printf("  New version ID: %s\n", rollbackVersion.ID)
}

// HandlePersonaAnalyze runs pattern analysis and generates proposals
func HandlePersonaAnalyze() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, store, pm := initPersonaEvolution(logger)
	if cfg == nil {
		return
	}
	defer store.Close()

	fmt.Println("Analyzing patterns...")
	fmt.Println()

	// Create evolution engine
	evolutionEngine := persona.NewEvolutionEngine(store, nil, logger)

	// Run analysis
	result, err := evolutionEngine.AnalyzePatterns(nil) // Context not needed for this implementation
	if err != nil {
		fmt.Printf("Error analyzing patterns: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Analysis complete in %v\n", result.Duration)
	fmt.Printf("Patterns found: %d\n", result.PatternsFound)
	fmt.Printf("Proposals generated: %d\n", len(result.Proposals))
	fmt.Println()

	// Save proposals
	proposalManager := persona.NewProposalManager(store.DB())
	for _, proposal := range result.Proposals {
		if err := proposalManager.Create(proposal); err != nil {
			logger.Warn("Failed to save proposal", zap.Error(err))
		}
	}

	if len(result.Proposals) > 0 {
		fmt.Println("New proposals created:")
		for i, p := range result.Proposals {
			fmt.Printf("%d. [%s] %s (%.0f%% confidence)\n", i+1, p.Type, p.Title, p.Confidence*100)
		}
		fmt.Println()
		fmt.Println("View proposals with: myrai persona proposals pending")
	} else {
		fmt.Println("No new proposals generated.")
	}
	_ = pm
}

// HandlePersonaEvolutionConfigShow shows evolution configuration
func HandlePersonaEvolutionConfigShow() {
	config := persona.DefaultEvolutionConfig()

	fmt.Println("Adaptive Persona Configuration")
	fmt.Println("==============================\n")

	fmt.Printf("Enabled: %v\n", config.Enabled)
	fmt.Printf("Notifications: %v\n", config.Notifications)
	fmt.Printf("Auto-apply high confidence: %v\n\n", config.AutoApplyHighConfidence)

	fmt.Println("Thresholds:")
	fmt.Printf("  Min Frequency: %d\n", config.Thresholds.MinFrequency)
	fmt.Printf("  Min Confidence: %.0f%%\n", config.Thresholds.MinConfidence*100)
	fmt.Printf("  Skill Usage Threshold: %d\n", config.Thresholds.SkillUsageThreshold)
	fmt.Printf("  Time Window: %d days\n", config.Thresholds.TimeWindowDays)
	fmt.Printf("  Auto-apply Threshold: %.0f%%\n\n", config.Thresholds.AutoApplyThreshold*100)

	fmt.Println("Weekly Analysis:")
	fmt.Printf("  Enabled: %v\n", config.WeeklyAnalysis.Enabled)
	fmt.Printf("  Day: %d (0=Sunday)\n", config.WeeklyAnalysis.DayOfWeek)
	fmt.Printf("  Time: %02d:%02d\n", config.WeeklyAnalysis.Hour, config.WeeklyAnalysis.Minute)
	fmt.Printf("  Max Proposals: %d\n", config.WeeklyAnalysis.MaxProposals)
}

// Helper function to initialize evolution dependencies

// Helper function to initialize evolution dependencies
func initPersonaEvolution(logger *zap.Logger) (*config.Config, *store.Store, *persona.PersonaManager) {
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

	workspace := onboarding.GetWorkspacePath()
	pm, err := persona.NewPersonaManager(workspace, logger)
	if err != nil {
		fmt.Printf("Error initializing persona manager: %v\n", err)
		os.Exit(1)
	}

	return cfg, st, pm
}

// applyProposal applies a proposal's changes to the persona
func applyProposal(pm *persona.PersonaManager, proposal *persona.EvolutionProposal) error {
	if proposal.Change == nil {
		return fmt.Errorf("proposal has no change defined")
	}

	change := proposal.Change

	switch change.Field {
	case "identity.expertise":
		identity := pm.GetIdentity()
		if identity != nil {
			// Add expertise if not already present
			expertise, ok := change.Value.(string)
			if !ok {
				return fmt.Errorf("invalid expertise value type")
			}

			// Check if already exists
			for _, e := range identity.Expertise {
				if strings.EqualFold(e, expertise) {
					return nil // Already exists
				}
			}

			identity.Expertise = append(identity.Expertise, expertise)
			return pm.SetIdentity(identity)
		}

	case "identity.values":
		identity := pm.GetIdentity()
		if identity != nil {
			value, ok := change.Value.(string)
			if !ok {
				return fmt.Errorf("invalid value type")
			}

			for _, v := range identity.Values {
				if strings.EqualFold(v, value) {
					return nil
				}
			}

			identity.Values = append(identity.Values, value)
			return pm.SetIdentity(identity)
		}

	case "user.preferences.active_hours":
		// This would require updating user profile
		// For now, just log it
		logger, _ := zap.NewDevelopment()
		logger.Info("Would update user preference",
			zap.String("field", change.Field),
			zap.Any("value", change.Value))
		return nil

	default:
		return fmt.Errorf("unknown field: %s", change.Field)
	}

	return nil
}
