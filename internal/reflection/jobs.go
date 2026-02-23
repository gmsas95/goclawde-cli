package reflection

import (
	"context"
	"fmt"
	"time"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

// Jobs manages background reflection jobs
type Jobs struct {
	store     *store.Store
	llmClient *llm.Client
	logger    *zap.Logger
	config    *config.Config
}

// NewJobs creates a new reflection jobs manager
func NewJobs(st *store.Store, cfg *config.Config, logger *zap.Logger) *Jobs {
	var llmClient *llm.Client
	provider, err := cfg.DefaultProvider()
	if err == nil {
		llmClient = llm.NewClient(provider)
	}

	return &Jobs{
		store:     st,
		llmClient: llmClient,
		logger:    logger,
		config:    cfg,
	}
}

// DailyLightweightJob runs the daily lightweight analysis
// - Checks new memories for contradictions
// - Quick redundancy scan
// - Duration: < 5 minutes
func (j *Jobs) DailyLightweightJob(ctx context.Context) error {
	j.logger.Info("Starting daily lightweight reflection job")
	start := time.Now()

	// Get memories from last 24 hours
	oneDayAgo := time.Now().AddDate(0, 0, -1)
	var newMemories []store.Memory
	if err := j.store.DB().Where("created_at > ?", oneDayAgo).Find(&newMemories).Error; err != nil {
		return fmt.Errorf("failed to load new memories: %w", err)
	}

	j.logger.Info("Checking new memories for contradictions", zap.Int("count", len(newMemories)))

	if len(newMemories) > 0 && j.llmClient != nil {
		// Load older memories for comparison
		var existingMemories []store.Memory
		if err := j.store.DB().Where("created_at <= ?", oneDayAgo).Find(&existingMemories).Error; err != nil {
			j.logger.Warn("Failed to load existing memories", zap.Error(err))
		}

		// Detect contradictions for new memories
		detector := NewContradictionDetector(j.llmClient, j.logger)
		contradictions, err := detector.DetectForNewMemories(ctx, newMemories, existingMemories)
		if err != nil {
			j.logger.Warn("Contradiction detection failed", zap.Error(err))
		} else if len(contradictions) > 0 {
			j.logger.Info("Found contradictions", zap.Int("count", len(contradictions)))
			// Save contradictions
			for _, contr := range contradictions {
				if err := j.store.DB().Create(&contr).Error; err != nil {
					j.logger.Warn("Failed to save contradiction", zap.Error(err))
				}
			}
		}
	}

	duration := time.Since(start)
	j.logger.Info("Daily lightweight job completed",
		zap.Duration("duration", duration),
		zap.Int("new_memories", len(newMemories)))

	return nil
}

// WeeklyDeepAudit runs the weekly deep analysis
// - Full reflection audit
// - Detect all contradictions
// - Find redundancies
// - Identify gaps
// - Create evolution proposals
// - Optimize neural clusters
// - Duration: 15-30 minutes
func (j *Jobs) WeeklyDeepAudit(ctx context.Context) error {
	j.logger.Info("Starting weekly deep reflection audit")
	start := time.Now()

	// Generate health report with deep analysis
	reporter := NewHealthReporter(j.store, j.llmClient, j.logger)
	report, err := reporter.GenerateReport(ctx, true)
	if err != nil {
		return fmt.Errorf("health report generation failed: %w", err)
	}

	// Save report to database
	if err := j.store.DB().Create(report).Error; err != nil {
		j.logger.Warn("Failed to save health report", zap.Error(err))
	}

	// Save detected issues to database
	// Save contradictions
	for _, contr := range report.ContradictionList {
		// Check if already exists
		var existing Contradiction
		result := j.store.DB().Where("memory_a_id = ? AND memory_b_id = ?",
			contr.MemoryAID, contr.MemoryBID).First(&existing)

		if result.Error != nil {
			// New contradiction - save it
			if err := j.store.DB().Create(&contr).Error; err != nil {
				j.logger.Warn("Failed to save contradiction", zap.Error(err))
			}
		}
	}

	// Save redundancy groups
	for _, group := range report.RedundancyList {
		var existing RedundancyGroup
		result := j.store.DB().Where("theme = ?", group.Theme).First(&existing)

		if result.Error != nil {
			// New redundancy - save it
			if err := j.store.DB().Create(&group).Error; err != nil {
				j.logger.Warn("Failed to save redundancy group", zap.Error(err))
			}
		}
	}

	// Save gaps
	for _, gap := range report.GapList {
		var existing Gap
		result := j.store.DB().Where("topic = ?", gap.Topic).First(&existing)

		if result.Error != nil {
			// New gap - save it
			if err := j.store.DB().Create(&gap).Error; err != nil {
				j.logger.Warn("Failed to save gap", zap.Error(err))
			}
		}
	}

	duration := time.Since(start)
	j.logger.Info("Weekly deep audit completed",
		zap.Duration("duration", duration),
		zap.Int("health_score", report.OverallScore),
		zap.Int("contradictions", report.Contradictions),
		zap.Int("redundancies", report.Redundancies),
		zap.Int("gaps", report.Gaps))

	return nil
}

// MonthlyArchival runs monthly archival tasks
// - Archive old clusters with no access in 90 days
// - Generate monthly evolution report
func (j *Jobs) MonthlyArchival(ctx context.Context) error {
	j.logger.Info("Starting monthly archival job")
	start := time.Now()

	// Find clusters not accessed in 90 days
	ninetyDaysAgo := time.Now().AddDate(0, 0, -90)

	// Archive old memories
	result := j.store.DB().Model(&store.Memory{}).
		Where("last_accessed < ? OR (last_accessed IS NULL AND created_at < ?)",
			ninetyDaysAgo, ninetyDaysAgo).
		Where("type != ?", "archived").
		Update("type", "archived")

	if result.Error != nil {
		j.logger.Warn("Failed to archive old memories", zap.Error(result.Error))
	} else {
		j.logger.Info("Archived old memories", zap.Int64("count", result.RowsAffected))
	}

	duration := time.Since(start)
	j.logger.Info("Monthly archival completed", zap.Duration("duration", duration))

	return nil
}
