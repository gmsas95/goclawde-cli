// Package jobs implements background jobs for the reflection engine
package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/reflection"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

// ReflectionJobs manages background reflection jobs
type ReflectionJobs struct {
	store     *store.Store
	llmClient *llm.Client
	logger    *zap.Logger
	config    *config.Config
}

// NewReflectionJobs creates a new reflection jobs manager
func NewReflectionJobs(st *store.Store, cfg *config.Config, logger *zap.Logger) *ReflectionJobs {
	var llmClient *llm.Client
	provider, err := cfg.DefaultProvider()
	if err == nil {
		llmClient = llm.NewClient(provider)
	}

	return &ReflectionJobs{
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
func (rj *ReflectionJobs) DailyLightweightJob(ctx context.Context) error {
	rj.logger.Info("Starting daily lightweight reflection job")
	start := time.Now()

	// Get memories from last 24 hours
	oneDayAgo := time.Now().AddDate(0, 0, -1)
	var newMemories []store.Memory
	if err := rj.store.DB().Where("created_at > ?", oneDayAgo).Find(&newMemories).Error; err != nil {
		return fmt.Errorf("failed to load new memories: %w", err)
	}

	rj.logger.Info("Checking new memories for contradictions", zap.Int("count", len(newMemories)))

	if len(newMemories) > 0 && rj.llmClient != nil {
		// Load older memories for comparison
		var existingMemories []store.Memory
		if err := rj.store.DB().Where("created_at <= ?", oneDayAgo).Find(&existingMemories).Error; err != nil {
			rj.logger.Warn("Failed to load existing memories", zap.Error(err))
		}

		// Detect contradictions for new memories
		detector := reflection.NewContradictionDetector(rj.llmClient, rj.logger)
		contradictions, err := detector.DetectForNewMemories(ctx, newMemories, existingMemories)
		if err != nil {
			rj.logger.Warn("Contradiction detection failed", zap.Error(err))
		} else if len(contradictions) > 0 {
			rj.logger.Info("Found contradictions", zap.Int("count", len(contradictions)))
			// Save contradictions
			for _, contr := range contradictions {
				if err := rj.store.DB().Create(&contr).Error; err != nil {
					rj.logger.Warn("Failed to save contradiction", zap.Error(err))
				}
			}
		}
	}

	duration := time.Since(start)
	rj.logger.Info("Daily lightweight job completed",
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
func (rj *ReflectionJobs) WeeklyDeepAudit(ctx context.Context) error {
	rj.logger.Info("Starting weekly deep reflection audit")
	start := time.Now()

	// Generate health report with deep analysis
	reporter := reflection.NewHealthReporter(rj.store, rj.llmClient, rj.logger)
	report, err := reporter.GenerateReport(ctx, true)
	if err != nil {
		return fmt.Errorf("health report generation failed: %w", err)
	}

	// Save report to database
	if err := rj.store.DB().Create(report).Error; err != nil {
		rj.logger.Warn("Failed to save health report", zap.Error(err))
	}

	// Save detected issues to database
	// Save contradictions
	for _, contr := range report.ContradictionList {
		// Check if already exists
		var existing reflection.Contradiction
		result := rj.store.DB().Where("memory_a_id = ? AND memory_b_id = ?",
			contr.MemoryAID, contr.MemoryBID).First(&existing)

		if result.Error != nil {
			// New contradiction - save it
			if err := rj.store.DB().Create(&contr).Error; err != nil {
				rj.logger.Warn("Failed to save contradiction", zap.Error(err))
			}
		}
	}

	// Save redundancy groups
	for _, group := range report.RedundancyList {
		var existing reflection.RedundancyGroup
		result := rj.store.DB().Where("theme = ?", group.Theme).First(&existing)

		if result.Error != nil {
			// New redundancy - save it
			if err := rj.store.DB().Create(&group).Error; err != nil {
				rj.logger.Warn("Failed to save redundancy group", zap.Error(err))
			}
		}
	}

	// Save gaps
	for _, gap := range report.GapList {
		var existing reflection.Gap
		result := rj.store.DB().Where("topic = ?", gap.Topic).First(&existing)

		if result.Error != nil {
			// New gap - save it
			if err := rj.store.DB().Create(&gap).Error; err != nil {
				rj.logger.Warn("Failed to save gap", zap.Error(err))
			}
		}
	}

	duration := time.Since(start)
	rj.logger.Info("Weekly deep audit completed",
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
func (rj *ReflectionJobs) MonthlyArchival(ctx context.Context) error {
	rj.logger.Info("Starting monthly archival job")
	start := time.Now()

	// Find clusters not accessed in 90 days
	ninetyDaysAgo := time.Now().AddDate(0, 0, -90)

	// Archive old memories
	result := rj.store.DB().Model(&store.Memory{}).
		Where("last_accessed < ? OR (last_accessed IS NULL AND created_at < ?)",
			ninetyDaysAgo, ninetyDaysAgo).
		Where("type != ?", "archived").
		Update("type", "archived")

	if result.Error != nil {
		rj.logger.Warn("Failed to archive old memories", zap.Error(result.Error))
	} else {
		rj.logger.Info("Archived old memories", zap.Int64("count", result.RowsAffected))
	}

	duration := time.Since(start)
	rj.logger.Info("Monthly archival completed", zap.Duration("duration", duration))

	return nil
}

// ScheduleReflectionJobs schedules the reflection jobs with the cron runner
func ScheduleReflectionJobs(cronRunner interface{}, jobs *ReflectionJobs) error {
	// Note: This would integrate with the existing cron system
	// For now, we return instructions

	fmt.Println("To schedule reflection jobs, add these entries to your cron configuration:")
	fmt.Println()
	fmt.Println("# Daily lightweight job (02:00 AM)")
	fmt.Println("0 2 * * * /usr/local/bin/myrai job daily-reflection")
	fmt.Println()
	fmt.Println("# Weekly deep audit (Sunday 03:00 AM)")
	fmt.Println("0 3 * * 0 /usr/local/bin/myrai job weekly-reflection")
	fmt.Println()
	fmt.Println("# Monthly archival (1st of month at 04:00 AM)")
	fmt.Println("0 4 1 * * /usr/local/bin/myrai job monthly-reflection")

	return nil
}

// RunDailyJob executes the daily lightweight job
func RunDailyJob(store *store.Store, cfg *config.Config, logger *zap.Logger) error {
	jobs := NewReflectionJobs(store, cfg, logger)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return jobs.DailyLightweightJob(ctx)
}

// RunWeeklyJob executes the weekly deep audit
func RunWeeklyJob(store *store.Store, cfg *config.Config, logger *zap.Logger) error {
	jobs := NewReflectionJobs(store, cfg, logger)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	return jobs.WeeklyDeepAudit(ctx)
}

// RunMonthlyJob executes the monthly archival
func RunMonthlyJob(store *store.Store, cfg *config.Config, logger *zap.Logger) error {
	jobs := NewReflectionJobs(store, cfg, logger)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return jobs.MonthlyArchival(ctx)
}
