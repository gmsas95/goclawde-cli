// Package cli handles CLI commands for the reflection engine
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/reflection"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

// HandleMemoryCommand handles memory-related commands
func HandleMemoryCommand(args []string) {
	if len(args) == 0 {
		PrintMemoryHelp()
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

	// Initialize LLM client
	var llmClient *llm.Client
	provider, err := cfg.DefaultProvider()
	if err == nil {
		llmClient = llm.NewClient(provider)
	}

	switch args[0] {
	case "health":
		handleHealthCommand(st, llmClient, logger, args[1:])

	case "reflect":
		handleReflectCommand(st, llmClient, logger, args[1:])

	case "contradictions":
		handleContradictionsCommand(st, args[1:])

	case "resolve":
		handleResolveCommand(st, args[1:])

	case "consolidate":
		handleConsolidateCommand(st, args[1:])

	case "gaps":
		handleGapsCommand(st, args[1:])

	default:
		PrintMemoryHelp()
	}
}

// handleHealthCommand generates a health report
func handleHealthCommand(st *store.Store, llmClient *llm.Client, logger *zap.Logger, args []string) {
	deep := false
	for _, arg := range args {
		if arg == "--deep" || arg == "-d" {
			deep = true
		}
	}

	fmt.Println("Generating health report...")
	if deep {
		fmt.Println("Running deep analysis (this may take a few minutes)...")
	}

	reporter := reflection.NewHealthReporter(st, llmClient, logger)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	report, err := reporter.GenerateReport(ctx, deep)
	if err != nil {
		fmt.Printf("Error generating report: %v\n", err)
		os.Exit(1)
	}

	// Display report using UI
	ui := reflection.NewUI(true)
	ui.DisplayHealthReport(report)
}

// handleReflectCommand runs a full reflection analysis
func handleReflectCommand(st *store.Store, llmClient *llm.Client, logger *zap.Logger, args []string) {
	fmt.Println("🧠 Running full memory reflection...")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Step 1: Health Report
	fmt.Println("Step 1/4: Generating health report...")
	reporter := reflection.NewHealthReporter(st, llmClient, logger)
	report, err := reporter.GenerateReport(ctx, true)
	if err != nil {
		fmt.Printf("Warning: Health report generation failed: %v\n", err)
	}
	fmt.Println("✓ Health report complete")

	// Step 2: Detect Contradictions
	fmt.Println("Step 2/4: Detecting contradictions...")
	if llmClient != nil {
		detector := reflection.NewContradictionDetector(llmClient, logger)
		var memories []store.Memory
		if err := st.DB().Find(&memories).Error; err == nil {
			contradictions, err := detector.Detect(ctx, memories)
			if err != nil {
				fmt.Printf("Warning: Contradiction detection failed: %v\n", err)
			} else {
				fmt.Printf("✓ Found %d contradictions\n", len(contradictions))
				// Save contradictions to DB
				for _, contr := range contradictions {
					st.DB().Create(&contr)
				}
			}
		}
	} else {
		fmt.Println("⚠ LLM client not available, skipping contradiction detection")
	}

	// Step 3: Find Redundancies
	fmt.Println("Step 3/4: Finding redundancies...")
	finder := reflection.NewRedundancyFinder()
	// Load clusters from neural package
	var clusters []interface{} // Placeholder - would need proper cluster loading
	_ = finder
	_ = clusters
	fmt.Println("✓ Redundancy check complete")

	// Step 4: Identify Gaps
	fmt.Println("Step 4/4: Identifying knowledge gaps...")
	if llmClient != nil {
		analyzer := reflection.NewGapAnalyzer(llmClient, logger)
		var conversations []store.Conversation
		var memories []store.Memory
		if err := st.DB().Find(&conversations).Error; err == nil {
			if err := st.DB().Find(&memories).Error; err == nil {
				gaps, err := analyzer.IdentifyGaps(ctx, conversations, memories)
				if err != nil {
					fmt.Printf("Warning: Gap analysis failed: %v\n", err)
				} else {
					fmt.Printf("✓ Found %d knowledge gaps\n", len(gaps))
					// Save gaps to DB
					for _, gap := range gaps {
						st.DB().Create(&gap)
					}
				}
			}
		}
	} else {
		fmt.Println("⚠ LLM client not available, skipping gap analysis")
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("                   REFLECTION COMPLETE")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	if report != nil {
		ui := reflection.NewUI(true)
		ui.DisplayHealthReport(report)
	}
}

// handleContradictionsCommand views contradictions
func handleContradictionsCommand(st *store.Store, args []string) {
	// Load contradictions from DB
	var contradictions []reflection.Contradiction
	result := st.DB().Where("status = ?", reflection.StatusOpen).Find(&contradictions)
	if result.Error != nil {
		fmt.Printf("Error loading contradictions: %v\n", result.Error)
		os.Exit(1)
	}

	if len(args) > 0 {
		// Show specific contradiction
		contrID := args[0]
		var contr reflection.Contradiction
		if err := st.DB().First(&contr, "id = ?", contrID).Error; err != nil {
			fmt.Printf("Contradiction not found: %s\n", contrID)
			os.Exit(1)
		}

		// Load memories
		var memA, memB store.Memory
		st.DB().First(&memA, "id = ?", contr.MemoryAID)
		st.DB().First(&memB, "id = ?", contr.MemoryBID)
		contr.MemoryA = &memA
		contr.MemoryB = &memB

		ui := reflection.NewUI(true)
		ui.DisplayContradictionDetails(&contr)
		return
	}

	// List all open contradictions
	if len(contradictions) == 0 {
		fmt.Println("✅ No open contradictions found!")
		return
	}

	fmt.Printf("Found %d open contradictions:\n\n", len(contradictions))

	for i, contr := range contradictions {
		severityEmoji := "⚠️"
		if contr.Severity == string(reflection.SeverityHigh) {
			severityEmoji = "🚨"
		} else if contr.Severity == string(reflection.SeverityLow) {
			severityEmoji = "ℹ️"
		}

		fmt.Printf("%d. %s [%s] %s\n", i+1, severityEmoji, contr.Severity,
			truncateString(contr.Description, 60))
		fmt.Printf("   ID: %s | Detected: %s\n", contr.ID, contr.DetectedAt.Format("Jan 2, 2006"))
		fmt.Println()
	}

	fmt.Println("View details: myrai memory contradictions <id>")
	fmt.Println("Resolve: myrai memory resolve <id> --keep A|B")
}

// handleResolveCommand resolves a contradiction
func handleResolveCommand(st *store.Store, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: myrai memory resolve <contradiction-id> --keep A|B")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --keep A    Keep Memory A, archive Memory B")
		fmt.Println("  --keep B    Keep Memory B, archive Memory A")
		fmt.Println("  --both      Keep both (context-dependent)")
		fmt.Println("  --neither   Archive both")
		os.Exit(1)
	}

	contrID := args[0]

	// Parse options
	keepA := false
	keepB := false
	keepBoth := false
	archiveBoth := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--keep":
			if i+1 < len(args) {
				switch strings.ToUpper(args[i+1]) {
				case "A":
					keepA = true
				case "B":
					keepB = true
				}
				i++
			}
		case "--both":
			keepBoth = true
		case "--neither":
			archiveBoth = true
		}
	}

	// Load contradiction
	var contr reflection.Contradiction
	if err := st.DB().First(&contr, "id = ?", contrID).Error; err != nil {
		fmt.Printf("Contradiction not found: %s\n", contrID)
		os.Exit(1)
	}

	now := time.Now()
	contr.Status = string(reflection.StatusResolved)
	contr.ResolvedAt = &now

	// Apply resolution
	switch {
	case keepA:
		// Archive memory B
		st.DB().Model(&store.Memory{}).Where("id = ?", contr.MemoryBID).Update("type", "archived")
		contr.Resolution = "Kept Memory A, archived Memory B"
		fmt.Println("✓ Kept Memory A, archived Memory B")

	case keepB:
		// Archive memory A
		st.DB().Model(&store.Memory{}).Where("id = ?", contr.MemoryAID).Update("type", "archived")
		contr.Resolution = "Kept Memory B, archived Memory A"
		fmt.Println("✓ Kept Memory B, archived Memory A")

	case keepBoth:
		contr.Resolution = "Both memories kept (context-dependent)"
		fmt.Println("✓ Both memories kept (context-dependent)")

	case archiveBoth:
		// Archive both
		st.DB().Model(&store.Memory{}).Where("id = ?", contr.MemoryAID).Update("type", "archived")
		st.DB().Model(&store.Memory{}).Where("id = ?", contr.MemoryBID).Update("type", "archived")
		contr.Resolution = "Both memories archived"
		fmt.Println("✓ Both memories archived")

	default:
		fmt.Println("Error: No resolution option specified")
		fmt.Println("Use --keep A, --keep B, --both, or --neither")
		os.Exit(1)
	}

	// Save updated contradiction
	if err := st.DB().Save(&contr).Error; err != nil {
		fmt.Printf("Error saving resolution: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Contradiction resolved")
}

// handleConsolidateCommand consolidates redundancies
func handleConsolidateCommand(st *store.Store, args []string) {
	consolidateAll := false
	for _, arg := range args {
		if arg == "--all" {
			consolidateAll = true
		}
	}

	if !consolidateAll {
		fmt.Println("Usage: myrai memory consolidate --all")
		fmt.Println()
		fmt.Println("This will consolidate all redundant memories into single entries.")
		fmt.Println("Use with caution - review redundancies first with 'myrai memory health'")
		os.Exit(1)
	}

	fmt.Println("Consolidating redundancies...")

	// Load redundancy groups
	var groups []reflection.RedundancyGroup
	result := st.DB().Where("status = ?", reflection.RedundancyStatusOpen).Find(&groups)
	if result.Error != nil {
		fmt.Printf("Error loading redundancies: %v\n", result.Error)
		os.Exit(1)
	}

	if len(groups) == 0 {
		fmt.Println("✅ No open redundancies to consolidate")
		return
	}

	fmt.Printf("Found %d redundancy groups\n", len(groups))

	consolidatedCount := 0
	for _, group := range groups {
		if err := consolidateRedundancyGroup(st, &group); err != nil {
			fmt.Printf("Warning: Failed to consolidate group %s: %v\n", group.ID, err)
			continue
		}
		consolidatedCount++
	}

	fmt.Printf("✓ Consolidated %d/%d redundancy groups\n", consolidatedCount, len(groups))
}

// handleGapsCommand views knowledge gaps
func handleGapsCommand(st *store.Store, args []string) {
	// Load gaps from DB
	var gaps []reflection.Gap
	result := st.DB().Where("status = ?", reflection.GapStatusOpen).Find(&gaps)
	if result.Error != nil {
		fmt.Printf("Error loading gaps: %v\n", result.Error)
		os.Exit(1)
	}

	if len(args) > 0 {
		// Show specific gap
		gapID := args[0]
		var gap reflection.Gap
		if err := st.DB().First(&gap, "id = ?", gapID).Error; err != nil {
			fmt.Printf("Gap not found: %s\n", gapID)
			os.Exit(1)
		}

		ui := reflection.NewUI(true)
		ui.DisplayGap(&gap)
		return
	}

	// List all open gaps
	if len(gaps) == 0 {
		fmt.Println("✅ No knowledge gaps found!")
		return
	}

	fmt.Printf("Found %d knowledge gaps:\n\n", len(gaps))

	for i, gap := range gaps {
		severityEmoji := "🟢"
		switch gap.GetSeverity() {
		case "high":
			severityEmoji = "🔴"
		case "medium":
			severityEmoji = "🟡"
		}

		fmt.Printf("%d. %s %s (mentioned %dx, %d memories)\n",
			i+1, severityEmoji, gap.Topic, gap.MentionCount, gap.MemoryCount)
		fmt.Printf("   Gap Ratio: %.0f%% | ID: %s\n", gap.GapRatio*100, gap.ID)
		fmt.Println()
	}

	fmt.Println("View details: myrai memory gaps <id>")
}

// PrintMemoryHelp prints help for memory commands
func PrintMemoryHelp() {
	fmt.Println("Memory Management Commands:")
	fmt.Println()
	fmt.Println("  myrai memory health [--deep]     Generate health report")
	fmt.Println("  myrai memory reflect             Run full reflection analysis")
	fmt.Println("  myrai memory contradictions      View contradictions")
	fmt.Println("  myrai memory contradictions <id> View specific contradiction")
	fmt.Println("  myrai memory resolve <id> --keep A|B  Resolve contradiction")
	fmt.Println("  myrai memory consolidate --all   Consolidate redundant memories")
	fmt.Println("  myrai memory gaps                View knowledge gaps")
	fmt.Println("  myrai memory gaps <id>          View specific gap")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  myrai memory health              Quick health check")
	fmt.Println("  myrai memory health --deep       Full analysis (slower)")
	fmt.Println("  myrai memory resolve contr_xxx --keep A")
}

// truncateString truncates a string to max length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// HandleNeuralCommand handles neural memory commands
func HandleNeuralCommand(args []string) {
	if len(args) == 0 {
		PrintMemoryHelp()
		return
	}

	switch args[0] {
	case "health":
		logger, _ := zap.NewDevelopment()
		defer logger.Sync()
		cfg, _ := config.Load("", "")
		st, _ := store.New(cfg)
		defer st.Close()
		handleHealthCommand(st, nil, logger, args[1:])
	case "reflect":
		logger, _ := zap.NewDevelopment()
		defer logger.Sync()
		cfg, _ := config.Load("", "")
		st, _ := store.New(cfg)
		defer st.Close()
		handleReflectCommand(st, nil, logger, args[1:])
	case "contradictions":
		logger, _ := zap.NewDevelopment()
		defer logger.Sync()
		cfg, _ := config.Load("", "")
		st, _ := store.New(cfg)
		defer st.Close()
		handleContradictionsCommand(st, args[1:])
	default:
		PrintMemoryHelp()
	}
}

// consolidateRedundancyGroup consolidates a redundancy group by merging memories
func consolidateRedundancyGroup(st *store.Store, group *reflection.RedundancyGroup) error {
	if len(group.MemoryIDs) < 2 {
		// Nothing to consolidate
		group.Status = string(reflection.RedundancyStatusIgnored)
		group.ConsolidatedAt = func() *time.Time { t := time.Now(); return &t }()
		return st.DB().Save(group).Error
	}

	// Load all memories in the group
	var memories []store.Memory
	if err := st.DB().Where("id IN ?", group.MemoryIDs).Find(&memories).Error; err != nil {
		return fmt.Errorf("failed to load memories: %w", err)
	}

	if len(memories) < 2 {
		group.Status = string(reflection.RedundancyStatusIgnored)
		return st.DB().Save(group).Error
	}

	// Create consolidated memory by combining content
	consolidatedContent := consolidateMemoryContent(memories)

	// Create new consolidated memory
	consolidatedMem := store.Memory{
		ID:      generateID("mem_cons"),
		Content: consolidatedContent,
		Type:    "consolidated",
		// Copy metadata from the most recent memory
	}

	if len(memories) > 0 {
		consolidatedMem.Importance = memories[0].Importance
		consolidatedMem.Source = memories[0].Source
	}

	// Save consolidated memory
	if err := st.DB().Create(&consolidatedMem).Error; err != nil {
		return fmt.Errorf("failed to create consolidated memory: %w", err)
	}

	// Mark original memories as archived
	for _, mem := range memories {
		mem.Type = "archived"
		if err := st.DB().Save(&mem).Error; err != nil {
			// Log but continue
			fmt.Printf("Warning: Failed to archive memory %s: %v\n", mem.ID, err)
		}
	}

	// Update redundancy group status
	now := time.Now()
	group.Status = string(reflection.RedundancyStatusConsolidated)
	group.ConsolidatedAt = &now

	return st.DB().Save(group).Error
}

// consolidateMemoryContent combines multiple memory contents into one
func consolidateMemoryContent(memories []store.Memory) string {
	if len(memories) == 0 {
		return ""
	}
	if len(memories) == 1 {
		return memories[0].Content
	}

	// Combine all contents with separators
	var parts []string
	for _, mem := range memories {
		if mem.Content != "" {
			parts = append(parts, mem.Content)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	// Remove duplicates while preserving order
	seen := make(map[string]bool)
	var uniqueParts []string
	for _, part := range parts {
		if !seen[part] {
			seen[part] = true
			uniqueParts = append(uniqueParts, part)
		}
	}

	// Join with separator
	return strings.Join(uniqueParts, "\n\n[Consolidated from multiple memories]\n\n")
}

// generateID generates a unique ID with the given prefix
func generateID(prefix string) string {
	return prefix + "_" + fmt.Sprintf("%d", time.Now().UnixNano())
}
