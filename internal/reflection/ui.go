// Package reflection implements the Reflection Engine UI components
package reflection

import (
	"fmt"
	"strings"
)

// UI provides formatted display functions for reflection reports
type UI struct {
	useColors bool
}

// NewUI creates a new UI instance
func NewUI(useColors bool) *UI {
	return &UI{useColors: useColors}
}

// DisplayHealthReport displays a formatted health report
func (ui *UI) DisplayHealthReport(report *HealthReport) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("                   📊 MEMORY HEALTH REPORT")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("Generated: %s\n\n", report.GeneratedAt.Format("January 2, 2006 3:04 PM"))

	// Overall Score
	fmt.Printf("Overall Score: %d/100 %s %s\n",
		report.OverallScore,
		report.GetScoreEmoji(),
		report.GetScoreCategory())

	if report.ScoreChange != nil {
		change := *report.ScoreChange
		if change > 0 {
			fmt.Printf("  (↑ +%d from last week)\n", change)
		} else if change < 0 {
			fmt.Printf("  (↓ %d from last week)\n", change)
		} else {
			fmt.Println("  (→ No change from last week)")
		}
	}
	fmt.Println()

	// Statistics
	fmt.Println("📈 Statistics:")
	fmt.Printf("  • Total Memories: %d\n", report.TotalMemories)
	fmt.Printf("  • Neural Clusters: %d\n", report.NeuralClusters)
	if report.Metrics.MemoriesPerCluster > 0 {
		fmt.Printf("  • Avg Memories per Cluster: %.1f\n", report.Metrics.MemoriesPerCluster)
	}
	fmt.Println()

	// Issues
	fmt.Println("⚠️  Issues Requiring Attention:")
	fmt.Println()

	// Contradictions
	if report.Contradictions > 0 {
		fmt.Printf("  1. Contradictions (%d found)\n", report.Contradictions)
		ui.displayContradictionSummary(report.ContradictionList)
		fmt.Println()
	}

	// Redundancies
	if report.Redundancies > 0 {
		fmt.Printf("  2. Redundancies (%d groups)\n", report.Redundancies)
		ui.displayRedundancySummary(report.RedundancyList)
		fmt.Println()
	}

	// Gaps
	if report.Gaps > 0 {
		fmt.Printf("  3. Knowledge Gaps (%d found)\n", report.Gaps)
		ui.displayGapSummary(report.GapList)
		fmt.Println()
	}

	if report.Contradictions == 0 && report.Redundancies == 0 && report.Gaps == 0 {
		fmt.Println("   ✅ No issues found! Your memory system is healthy.")
		fmt.Println()
	}

	// Weekly Evolution
	fmt.Println("📊 Weekly Evolution:")
	fmt.Printf("  • %d new memories added\n", report.NewMemories)
	fmt.Printf("  • %d new neural clusters formed\n", report.NewClusters)
	fmt.Println()

	// Suggested Actions
	if len(report.SuggestedActions) > 0 {
		fmt.Println("💡 Suggested Actions:")
		ui.displayActions(report.SuggestedActions[:min(5, len(report.SuggestedActions))])
		if len(report.SuggestedActions) > 5 {
			fmt.Printf("  ... and %d more actions\n", len(report.SuggestedActions)-5)
		}
		fmt.Println()
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
}

// DisplayContradictionDetails displays detailed information about a contradiction
func (ui *UI) DisplayContradictionDetails(contr *Contradiction) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("                   ⚠️  CONTRADICTION DETECTED")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	// Severity indicator
	severityEmoji := "⚠️"
	if contr.Severity == string(SeverityHigh) {
		severityEmoji = "🚨"
	} else if contr.Severity == string(SeverityLow) {
		severityEmoji = "ℹ️"
	}

	fmt.Printf("Severity: %s %s\n", severityEmoji, strings.ToUpper(contr.Severity))
	fmt.Printf("Detected: %s\n", contr.DetectedAt.Format("Jan 2, 2006"))
	fmt.Printf("Status: %s\n", contr.Status)
	fmt.Println()

	// Memory A
	fmt.Println("Memory A:")
	if contr.MemoryA != nil {
		fmt.Printf("  ID: %s\n", contr.MemoryA.ID)
		fmt.Printf("  Content: \"%s\"\n", truncateString(contr.MemoryA.Content, 200))
		fmt.Printf("  Created: %s\n", contr.MemoryA.CreatedAt.Format("Jan 2, 2006"))
	} else {
		fmt.Printf("  ID: %s\n", contr.MemoryAID)
	}
	fmt.Println()

	// Memory B
	fmt.Println("Memory B:")
	if contr.MemoryB != nil {
		fmt.Printf("  ID: %s\n", contr.MemoryB.ID)
		fmt.Printf("  Content: \"%s\"\n", truncateString(contr.MemoryB.Content, 200))
		fmt.Printf("  Created: %s\n", contr.MemoryB.CreatedAt.Format("Jan 2, 2006"))
	} else {
		fmt.Printf("  ID: %s\n", contr.MemoryBID)
	}
	fmt.Println()

	// Description
	fmt.Println("Contradiction:")
	fmt.Printf("  %s\n", contr.Description)
	fmt.Println()

	// Resolution suggestions
	if contr.SuggestedResolution != "" {
		fmt.Println("Suggested Resolution:")
		fmt.Printf("  %s\n", contr.SuggestedResolution)
		fmt.Println()
	}

	fmt.Println("Resolution Options:")
	fmt.Println("  [A is correct] - Keep Memory A, archive Memory B")
	fmt.Println("  [B is correct] - Keep Memory B, archive Memory A")
	fmt.Println("  [Context-dependent] - Both are true in different contexts")
	fmt.Println("  [Neither] - Both are incorrect")
	fmt.Println()

	fmt.Println("═══════════════════════════════════════════════════════════════")
}

// DisplayRedundancyGroup displays details of a redundancy group
func (ui *UI) DisplayRedundancyGroup(group *RedundancyGroup) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("              🗂️   REDUNDANCY GROUP: %s\n", strings.ToUpper(group.Theme))
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Printf("Theme: %s\n", group.Theme)
	fmt.Printf("Memories: %d\n", len(group.MemoryIDs))
	fmt.Printf("Reason: %s\n", group.Reason)
	fmt.Printf("Suggested Action: %s\n", group.SuggestedAction)
	fmt.Printf("Detected: %s\n", group.DetectedAt.Format("Jan 2, 2006"))
	fmt.Println()

	if len(group.Clusters) > 0 {
		fmt.Println("Related Clusters:")
		for _, cluster := range group.Clusters {
			fmt.Printf("  • %s (%d memories)\n", cluster.Theme, cluster.ClusterSize)
		}
		fmt.Println()
	}

	fmt.Printf("Storage Savings: ~%d bytes\n", group.CalculateStorageSavings())
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
}

// DisplayGap displays details of a knowledge gap
func (ui *UI) DisplayGap(gap *Gap) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("              🔍 KNOWLEDGE GAP: %s\n", strings.ToUpper(gap.Topic))
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Printf("Topic: %s\n", gap.Topic)
	fmt.Printf("Times Mentioned: %d\n", gap.MentionCount)
	fmt.Printf("Memory Entries: %d\n", gap.MemoryCount)
	fmt.Printf("Gap Ratio: %.0f%% (lower is worse)\n", gap.GapRatio*100)
	fmt.Printf("Severity: %s\n", gap.GetSeverity())
	fmt.Printf("Detected: %s\n", gap.DetectedAt.Format("Jan 2, 2006"))
	fmt.Println()

	if len(gap.SampleMentions) > 0 {
		fmt.Println("Sample Mentions:")
		for i, mention := range gap.SampleMentions[:min(3, len(gap.SampleMentions))] {
			fmt.Printf("  %d. %s\n", i+1, truncateString(mention, 100))
		}
		fmt.Println()
	}

	fmt.Println("Impact:")
	fmt.Printf("  %s\n", gap.GetImpactEstimate())
	fmt.Println()

	fmt.Println("Suggested Action:")
	fmt.Println("  Ask the user about their knowledge/preferences on this topic")
	fmt.Println("  to create more comprehensive memory entries.")
	fmt.Println()

	fmt.Println("═══════════════════════════════════════════════════════════════")
}

// DisplayProgress shows a progress indicator for long operations
func (ui *UI) DisplayProgress(current, total int, operation string) {
	percent := float64(current) / float64(total) * 100
	barWidth := 40
	filled := int(float64(barWidth) * float64(current) / float64(total))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	fmt.Printf("\r%s [%s] %.1f%% (%d/%d)", operation, bar, percent, current, total)

	if current == total {
		fmt.Println() // New line when complete
	}
}

// DisplayNotification shows a gap notification
func (ui *UI) DisplayNotification(gap *Gap) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   💡 KNOWLEDGE GAP DETECTED                   ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Topic: %-52s ║\n", truncateString(gap.Topic, 50))
	fmt.Printf("║  Mentioned %d times but only %d memories exist.              ║\n",
		gap.MentionCount, gap.MemoryCount)
	fmt.Println("║                                                               ║")
	fmt.Printf("║  Would you like to share more about %s?                    ║\n",
		truncateString(gap.Topic, 30))
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// displayContradictionSummary displays a summary of contradictions
func (ui *UI) displayContradictionSummary(contradictions []Contradiction) {
	// Count by severity
	severityCount := make(map[string]int)
	for _, contr := range contradictions {
		if contr.Status == string(StatusOpen) {
			severityCount[contr.Severity]++
		}
	}

	if severityCount[string(SeverityHigh)] > 0 {
		fmt.Printf("     🚨 High: %d\n", severityCount[string(SeverityHigh)])
	}
	if severityCount[string(SeverityMedium)] > 0 {
		fmt.Printf("     ⚠️  Medium: %d\n", severityCount[string(SeverityMedium)])
	}
	if severityCount[string(SeverityLow)] > 0 {
		fmt.Printf("     ℹ️  Low: %d\n", severityCount[string(SeverityLow)])
	}

	fmt.Println("     [View details] / [Auto-resolve]")
}

// displayRedundancySummary displays a summary of redundancies
func (ui *UI) displayRedundancySummary(groups []RedundancyGroup) {
	totalMemories := 0
	for _, group := range groups {
		if group.Status == string(RedundancyStatusOpen) {
			totalMemories += len(group.MemoryIDs)
		}
	}

	fmt.Printf("     %d memories can be consolidated\n", totalMemories)

	if len(groups) <= 3 {
		for _, group := range groups {
			if group.Status == string(RedundancyStatusOpen) {
				fmt.Printf("     • %d memories about '%s'\n", len(group.MemoryIDs), group.Theme)
			}
		}
	}

	fmt.Println("     [Consolidate all] / [Review individually]")
}

// displayGapSummary displays a summary of gaps
func (ui *UI) displayGapSummary(gaps []Gap) {
	for _, gap := range gaps {
		if gap.Status == string(GapStatusOpen) {
			fmt.Printf("     • %s (mentioned %dx, %d memories)\n",
				gap.Topic, gap.MentionCount, gap.MemoryCount)
		}
	}

	fmt.Println("     [Ask about gaps] / [Dismiss]")
}

// displayActions displays suggested actions
func (ui *UI) displayActions(actions []Action) {
	for i, action := range actions {
		priorityEmoji := "⚪"
		switch action.Priority {
		case "high":
			priorityEmoji = "🔴"
		case "medium":
			priorityEmoji = "🟡"
		case "low":
			priorityEmoji = "🟢"
		}

		fmt.Printf("  %d. %s [%s] %s\n", i+1, priorityEmoji, action.Priority, action.Description)
		fmt.Printf("     Impact: %s\n", action.Impact)
	}
}
