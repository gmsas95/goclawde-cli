// Package reflection implements the Reflection Engine for self-auditing memory system
package reflection

import (
	"context"
	"fmt"
	"time"

	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/neural"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

// Action represents a suggested action from the health report
type Action struct {
	Type        string `json:"type"`     // resolve_contradiction, consolidate, fill_gap, archive
	Priority    string `json:"priority"` // high, medium, low
	Description string `json:"description"`
	Impact      string `json:"impact"` // What will improve
	TargetID    string `json:"target_id,omitempty"`
}

// Metrics contains health metrics for the memory system
type Metrics struct {
	TotalMemories          int     `json:"total_memories"`
	NeuralClusters         int     `json:"neural_clusters"`
	OpenContradictions     int     `json:"open_contradictions"`
	ResolvedContradictions int     `json:"resolved_contradictions"`
	OpenRedundancies       int     `json:"open_redundancies"`
	OpenGaps               int     `json:"open_gaps"`
	StorageUsedMB          float64 `json:"storage_used_mb"`
	AvgClusterConfidence   float64 `json:"avg_cluster_confidence"`
	MemoriesPerCluster     float64 `json:"memories_per_cluster"`
}

// HealthReport contains a comprehensive health assessment of the memory system
type HealthReport struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	GeneratedAt  time.Time `json:"generated_at"`
	OverallScore int       `json:"overall_score"` // 0-100

	// Metrics
	TotalMemories  int `json:"total_memories"`
	NeuralClusters int `json:"neural_clusters"`
	Contradictions int `json:"contradictions"`
	Redundancies   int `json:"redundancies"`
	Gaps           int `json:"gaps"`

	// Detailed data (stored as JSON)
	DetailsJSON string `gorm:"column:details;type:text" json:"-"`
	ActionsJSON string `gorm:"column:suggested_actions;type:text" json:"-"`

	// Transient fields
	ContradictionList []Contradiction   `gorm:"-" json:"contradictions,omitempty"`
	RedundancyList    []RedundancyGroup `gorm:"-" json:"redundancies,omitempty"`
	GapList           []Gap             `gorm:"-" json:"gaps,omitempty"`
	SuggestedActions  []Action          `gorm:"-" json:"suggested_actions,omitempty"`
	Metrics           Metrics           `gorm:"-" json:"metrics"`

	// Weekly evolution tracking
	PreviousScore *int `json:"previous_score,omitempty"`
	ScoreChange   *int `json:"score_change,omitempty"`
	NewMemories   int  `json:"new_memories_this_week"`
	NewClusters   int  `json:"new_clusters_this_week"`
}

// HealthReporter generates health reports for the memory system
type HealthReporter struct {
	contradictionDetector *ContradictionDetector
	redundancyFinder      *RedundancyFinder
	gapAnalyzer           *GapAnalyzer
	logger                *zap.Logger
	store                 *store.Store
}

// NewHealthReporter creates a new health reporter
func NewHealthReporter(store *store.Store, llmClient interface{}, logger *zap.Logger) *HealthReporter {
	// Type assert the llmClient
	client, ok := llmClient.(*llm.Client)
	if !ok {
		logger.Warn("Invalid LLM client type provided to HealthReporter")
		client = nil
	}

	return &HealthReporter{
		contradictionDetector: NewContradictionDetector(client, logger),
		redundancyFinder:      NewRedundancyFinder(),
		gapAnalyzer:           NewGapAnalyzer(client, logger),
		logger:                logger,
		store:                 store,
	}
}

// GenerateReport generates a comprehensive health report
func (hr *HealthReporter) GenerateReport(ctx context.Context, deepAnalysis bool) (*HealthReport, error) {
	hr.logger.Info("Generating health report", zap.Bool("deep_analysis", deepAnalysis))

	report := &HealthReport{
		ID:          generateID("health"),
		GeneratedAt: time.Now(),
	}

	// Gather basic metrics
	if err := hr.gatherMetrics(ctx, report); err != nil {
		hr.logger.Warn("Failed to gather metrics", zap.Error(err))
	}

	// Run deep analysis if requested
	if deepAnalysis {
		if err := hr.runDeepAnalysis(ctx, report); err != nil {
			hr.logger.Warn("Deep analysis failed", zap.Error(err))
		}
	} else {
		// Quick analysis - just load existing issues
		if err := hr.loadExistingIssues(ctx, report); err != nil {
			hr.logger.Warn("Failed to load existing issues", zap.Error(err))
		}
	}

	// Calculate overall score
	report.calculateScore()

	// Generate suggested actions
	report.generateActions()

	// Get weekly evolution
	hr.loadWeeklyEvolution(report)

	hr.logger.Info("Health report generated",
		zap.Int("score", report.OverallScore),
		zap.Int("contradictions", len(report.ContradictionList)),
		zap.Int("redundancies", len(report.RedundancyList)),
		zap.Int("gaps", len(report.GapList)))

	return report, nil
}

// gatherMetrics gathers basic metrics from the store
func (hr *HealthReporter) gatherMetrics(ctx context.Context, report *HealthReport) error {
	// Get memory count
	var memoryCount int64
	hr.store.DB().Model(&store.Memory{}).Count(&memoryCount)
	report.TotalMemories = int(memoryCount)
	report.Metrics.TotalMemories = int(memoryCount)

	// Get cluster count
	var clusterCount int64
	hr.store.DB().Model(&neural.NeuralCluster{}).Count(&clusterCount)
	report.NeuralClusters = int(clusterCount)
	report.Metrics.NeuralClusters = int(clusterCount)

	// Calculate average cluster size
	if clusterCount > 0 {
		report.Metrics.MemoriesPerCluster = float64(memoryCount) / float64(clusterCount)
	}

	return nil
}

// runDeepAnalysis performs a full analysis of the memory system
func (hr *HealthReporter) runDeepAnalysis(ctx context.Context, report *HealthReport) error {
	// Load all memories
	var memories []store.Memory
	if err := hr.store.DB().Find(&memories).Error; err != nil {
		return fmt.Errorf("failed to load memories: %w", err)
	}

	// Detect contradictions
	contradictions, err := hr.contradictionDetector.Detect(ctx, memories)
	if err != nil {
		hr.logger.Warn("Contradiction detection failed", zap.Error(err))
	} else {
		report.ContradictionList = contradictions
		report.Contradictions = len(contradictions)
	}

	// Load all clusters
	var clusters []*neural.NeuralCluster
	if err := hr.store.DB().Find(&clusters).Error; err != nil {
		hr.logger.Warn("Failed to load clusters", zap.Error(err))
	} else {
		// Find redundancies
		redundancies, err := hr.redundancyFinder.FindRedundancies(clusters)
		if err != nil {
			hr.logger.Warn("Redundancy detection failed", zap.Error(err))
		} else {
			report.RedundancyList = redundancies
			report.Redundancies = len(redundancies)
		}

		// Calculate average confidence
		var totalConfidence float64
		for _, cluster := range clusters {
			totalConfidence += cluster.ConfidenceScore
		}
		if len(clusters) > 0 {
			report.Metrics.AvgClusterConfidence = totalConfidence / float64(len(clusters))
		}
	}

	// Identify gaps
	var conversations []store.Conversation
	if err := hr.store.DB().Find(&conversations).Error; err != nil {
		hr.logger.Warn("Failed to load conversations", zap.Error(err))
	} else {
		gaps, err := hr.gapAnalyzer.IdentifyGaps(ctx, conversations, memories)
		if err != nil {
			hr.logger.Warn("Gap analysis failed", zap.Error(err))
		} else {
			report.GapList = gaps
			report.Gaps = len(gaps)
		}
	}

	return nil
}

// loadExistingIssues loads previously detected issues from the database
func (hr *HealthReporter) loadExistingIssues(ctx context.Context, report *HealthReport) error {
	// Count open contradictions
	var openContradictions int64
	hr.store.DB().Model(&Contradiction{}).Where("status = ?", StatusOpen).Count(&openContradictions)
	report.Contradictions = int(openContradictions)
	report.Metrics.OpenContradictions = int(openContradictions)

	// Count resolved contradictions
	var resolvedContradictions int64
	hr.store.DB().Model(&Contradiction{}).Where("status = ?", StatusResolved).Count(&resolvedContradictions)
	report.Metrics.ResolvedContradictions = int(resolvedContradictions)

	// Count open redundancies
	var openRedundancies int64
	hr.store.DB().Model(&RedundancyGroup{}).Where("status = ?", RedundancyStatusOpen).Count(&openRedundancies)
	report.Redundancies = int(openRedundancies)
	report.Metrics.OpenRedundancies = int(openRedundancies)

	// Count open gaps
	var openGaps int64
	hr.store.DB().Model(&Gap{}).Where("status = ?", GapStatusOpen).Count(&openGaps)
	report.Gaps = int(openGaps)
	report.Metrics.OpenGaps = int(openGaps)

	return nil
}

// loadWeeklyEvolution loads metrics from the previous week
func (hr *HealthReporter) loadWeeklyEvolution(report *HealthReport) {
	oneWeekAgo := time.Now().AddDate(0, 0, -7)

	// Count new memories this week
	var newMemories int64
	hr.store.DB().Model(&store.Memory{}).Where("created_at > ?", oneWeekAgo).Count(&newMemories)
	report.NewMemories = int(newMemories)

	// Count new clusters this week
	var newClusters int64
	hr.store.DB().Model(&neural.NeuralCluster{}).Where("created_at > ?", oneWeekAgo).Count(&newClusters)
	report.NewClusters = int(newClusters)

	// Get previous report for comparison
	var previousReport HealthReport
	err := hr.store.DB().Where("generated_at < ?", report.GeneratedAt).
		Order("generated_at DESC").
		First(&previousReport).Error

	if err == nil {
		report.PreviousScore = &previousReport.OverallScore
		change := report.OverallScore - previousReport.OverallScore
		report.ScoreChange = &change
	}
}

// calculateScore calculates the overall health score
func (r *HealthReport) calculateScore() {
	score := 100

	// Deduct for contradictions
	score -= r.Contradictions * 5

	// Deduct for redundancies
	score -= r.Redundancies * 3

	// Deduct for gaps
	score -= r.Gaps * 2

	// Boost for good clustering
	if r.Metrics.MemoriesPerCluster > 5 && r.Metrics.MemoriesPerCluster < 20 {
		score += 5
	}

	if r.Metrics.AvgClusterConfidence > 0.7 {
		score += 5
	}

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	r.OverallScore = score
}

// generateActions generates suggested actions based on the report
func (r *HealthReport) generateActions() {
	var actions []Action

	// Actions for contradictions
	for _, contr := range r.ContradictionList {
		if contr.Status == string(StatusOpen) {
			actions = append(actions, Action{
				Type:        "resolve_contradiction",
				Priority:    contr.Severity,
				Description: fmt.Sprintf("Resolve contradiction: %s", contr.Description),
				Impact:      "Improve accuracy by clarifying conflicting information",
				TargetID:    contr.ID,
			})
		}
	}

	// Actions for redundancies
	for _, redun := range r.RedundancyList {
		if redun.Status == string(RedundancyStatusOpen) {
			priority := redun.GetPriority()
			actions = append(actions, Action{
				Type:        "consolidate",
				Priority:    priority,
				Description: fmt.Sprintf("Consolidate %d redundant memories about '%s'", len(redun.MemoryIDs), redun.Theme),
				Impact:      fmt.Sprintf("Save ~%d bytes of storage", redun.CalculateStorageSavings()),
				TargetID:    redun.ID,
			})
		}
	}

	// Actions for gaps
	for _, gap := range r.GapList {
		if gap.Status == string(GapStatusOpen) {
			actions = append(actions, Action{
				Type:        "fill_gap",
				Priority:    gap.GetSeverity(),
				Description: fmt.Sprintf("Fill knowledge gap on '%s' (mentioned %d times, %d memories)", gap.Topic, gap.MentionCount, gap.MemoryCount),
				Impact:      gap.GetImpactEstimate(),
				TargetID:    gap.ID,
			})
		}
	}

	// Storage optimization suggestion
	if r.Redundancies > 10 {
		actions = append(actions, Action{
			Type:        "optimize_storage",
			Priority:    "medium",
			Description: fmt.Sprintf("Run full storage optimization to consolidate %d redundancy groups", r.Redundancies),
			Impact:      "Significant storage savings and improved query performance",
		})
	}

	r.SuggestedActions = actions
}

// GetScoreCategory returns a human-readable category for the score
func (r *HealthReport) GetScoreCategory() string {
	switch {
	case r.OverallScore >= 90:
		return "Excellent"
	case r.OverallScore >= 80:
		return "Good"
	case r.OverallScore >= 60:
		return "Fair"
	case r.OverallScore >= 40:
		return "Poor"
	default:
		return "Critical"
	}
}

// GetScoreEmoji returns an emoji representing the score
func (r *HealthReport) GetScoreEmoji() string {
	switch {
	case r.OverallScore >= 90:
		return "✅"
	case r.OverallScore >= 80:
		return "✨"
	case r.OverallScore >= 60:
		return "⚠️"
	case r.OverallScore >= 40:
		return "🔶"
	default:
		return "🚨"
	}
}
