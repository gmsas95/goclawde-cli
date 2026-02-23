// Package jobs implements background job handlers for Myrai
package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/gmsas95/myrai-cli/internal/cron"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/neural"
	"github.com/gmsas95/myrai-cli/internal/vector"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ClusterFormationJob handles weekly neural cluster formation
type ClusterFormationJob struct {
	db        *gorm.DB
	llmClient *llm.Client
	searcher  *vector.Searcher
	logger    *zap.Logger
	manager   *neural.ClusterManager
}

// NewClusterFormationJob creates a new cluster formation job handler
func NewClusterFormationJob(db *gorm.DB, llmClient *llm.Client, searcher *vector.Searcher, logger *zap.Logger) (*ClusterFormationJob, error) {
	manager, err := neural.NewClusterManager(db, llmClient, searcher, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster manager: %w", err)
	}

	return &ClusterFormationJob{
		db:        db,
		llmClient: llmClient,
		searcher:  searcher,
		logger:    logger,
		manager:   manager,
	}, nil
}

// RunClusterFormation executes the weekly cluster formation job
// It processes unclustered memories and forms new semantic clusters
func (j *ClusterFormationJob) RunClusterFormation() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	j.logger.Info("Starting weekly cluster formation job",
		zap.Time("started_at", time.Now()),
	)

	start := time.Now()

	// Use default formation options
	opts := neural.DefaultFormationOptions()

	// Run formation
	result, err := j.manager.FormClusters(ctx, opts)
	if err != nil {
		j.logger.Error("Cluster formation failed",
			zap.Error(err),
			zap.Duration("duration", time.Since(start)),
		)
		return fmt.Errorf("cluster formation failed: %w", err)
	}

	j.logger.Info("Weekly cluster formation complete",
		zap.Int("clusters_created", result.ClustersCreated),
		zap.Int("clusters_updated", result.ClustersUpdated),
		zap.Int("clusters_merged", result.ClustersMerged),
		zap.Int("clusters_deleted", result.ClustersDeleted),
		zap.Int("memories_processed", result.MemoriesProcessed),
		zap.Int("memories_clustered", result.MemoriesClustered),
		zap.Duration("duration", result.Duration),
		zap.Int("errors", len(result.Errors)),
	)

	if len(result.Errors) > 0 {
		for i, err := range result.Errors {
			if i >= 5 { // Only log first 5 errors
				j.logger.Warn("Additional errors suppressed",
					zap.Int("total", len(result.Errors)),
				)
				break
			}
			j.logger.Warn("Cluster formation error",
				zap.Int("index", i),
				zap.Error(err),
			)
		}
	}

	return nil
}

// RunDailyMaintenance executes daily lightweight maintenance tasks
// It performs cleanup and optimization without heavy processing
func (j *ClusterFormationJob) RunDailyMaintenance() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	j.logger.Info("Starting daily cluster maintenance",
		zap.Time("started_at", time.Now()),
	)

	start := time.Now()

	// Task 1: Update cluster statistics
	if err := j.updateClusterStatistics(ctx); err != nil {
		j.logger.Error("Failed to update cluster statistics",
			zap.Error(err),
		)
	}

	// Task 2: Identify and mark low-confidence clusters for refresh
	if err := j.identifyStaleClusters(ctx); err != nil {
		j.logger.Error("Failed to identify stale clusters",
			zap.Error(err),
		)
	}

	// Task 3: Clean up old query patterns
	if err := j.cleanupOldQueryPatterns(ctx); err != nil {
		j.logger.Error("Failed to cleanup query patterns",
			zap.Error(err),
		)
	}

	// Task 4: Consolidate small clusters
	if err := j.consolidateSmallClusters(ctx); err != nil {
		j.logger.Error("Failed to consolidate small clusters",
			zap.Error(err),
		)
	}

	j.logger.Info("Daily cluster maintenance complete",
		zap.Duration("duration", time.Since(start)),
	)

	return nil
}

// updateClusterStatistics updates derived statistics for all clusters
func (j *ClusterFormationJob) updateClusterStatistics(ctx context.Context) error {
	// Update cluster sizes based on actual memory associations
	result := j.db.Exec(`
		UPDATE neural_clusters 
		SET cluster_size = (
			SELECT COUNT(*) 
			FROM cluster_memories 
			WHERE cluster_memories.cluster_id = neural_clusters.id
		)
		WHERE id IN (
			SELECT DISTINCT cluster_id 
			FROM cluster_memories
		)
	`)

	if result.Error != nil {
		return fmt.Errorf("failed to update cluster sizes: %w", result.Error)
	}

	j.logger.Debug("Updated cluster statistics",
		zap.Int64("rows_affected", result.RowsAffected),
	)

	return nil
}

// identifyStaleClusters identifies clusters that need refreshing
func (j *ClusterFormationJob) identifyStaleClusters(ctx context.Context) error {
	// Find clusters with low confidence that might need re-clustering
	threshold := time.Now().AddDate(0, 0, -30) // 30 days old

	var staleClusters []neural.NeuralCluster
	err := j.db.Where(
		"confidence_score < ? AND (last_accessed < ? OR last_accessed IS NULL)",
		0.6, threshold,
	).Find(&staleClusters).Error

	if err != nil {
		return fmt.Errorf("failed to find stale clusters: %w", err)
	}

	if len(staleClusters) > 0 {
		j.logger.Info("Found stale clusters",
			zap.Int("count", len(staleClusters)),
		)

		// Refresh a limited number of stale clusters per run
		maxRefresh := 10
		for i, cluster := range staleClusters {
			if i >= maxRefresh {
				break
			}

			if err := j.manager.RefreshCluster(ctx, cluster.ID); err != nil {
				j.logger.Warn("Failed to refresh stale cluster",
					zap.String("cluster_id", cluster.ID),
					zap.Error(err),
				)
			}
		}
	}

	return nil
}

// cleanupOldQueryPatterns removes old query pattern records
func (j *ClusterFormationJob) cleanupOldQueryPatterns(ctx context.Context) error {
	// Keep last 30 days of query patterns
	cutoff := time.Now().AddDate(0, 0, -30)

	result := j.db.Where("created_at < ?", cutoff).Delete(&neural.QueryPattern{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup query patterns: %w", result.Error)
	}

	j.logger.Debug("Cleaned up old query patterns",
		zap.Int64("rows_deleted", result.RowsAffected),
	)

	return nil
}

// consolidateSmallClusters merges very small clusters with similar larger ones
func (j *ClusterFormationJob) consolidateSmallClusters(ctx context.Context) error {
	// Find small clusters (size < 3) that could be merged
	var smallClusters []neural.NeuralCluster
	err := j.db.Where("cluster_size < ? AND confidence_score < ?", 3, 0.7).
		Limit(20).
		Find(&smallClusters).Error

	if err != nil {
		return fmt.Errorf("failed to find small clusters: %w", err)
	}

	if len(smallClusters) == 0 {
		return nil
	}

	j.logger.Info("Consolidating small clusters",
		zap.Int("count", len(smallClusters)),
	)

	for _, smallCluster := range smallClusters {
		// Try to find a similar larger cluster to merge with
		var similarCluster neural.NeuralCluster
		err := j.db.Where(
			"id != ? AND cluster_size >= ? AND confidence_score >= ?",
			smallCluster.ID, 5, 0.7,
		).
			Order("confidence_score DESC").
			First(&similarCluster).Error

		if err != nil {
			continue // No suitable merge target found
		}

		// Merge the clusters
		_, err = j.manager.MergeClusters(ctx, smallCluster.ID, similarCluster.ID)
		if err != nil {
			j.logger.Warn("Failed to merge clusters",
				zap.String("small_cluster", smallCluster.ID),
				zap.String("target_cluster", similarCluster.ID),
				zap.Error(err),
			)
		}
	}

	return nil
}

// ScheduleClusterJobs schedules the cluster formation jobs with the cron runner
func ScheduleClusterJobs(runner *cron.Runner, db *gorm.DB, llmClient *llm.Client, searcher *vector.Searcher, logger *zap.Logger) error {
	_, err := NewClusterFormationJob(db, llmClient, searcher, logger)
	if err != nil {
		return fmt.Errorf("failed to create cluster formation job handler: %w", err)
	}

	// Schedule weekly cluster formation (Sunday at 2 AM)
	// Using cron expression: "0 2 * * 0" (minute hour day month weekday)
	_, err = runner.AddJob(
		"cluster-formation-weekly",
		"0 2 * * 0",
		"Run neural cluster formation on unclustered memories",
	)
	if err != nil {
		return fmt.Errorf("failed to schedule weekly formation job: %w", err)
	}

	// Schedule daily maintenance (every day at 3 AM)
	_, err = runner.AddJob(
		"cluster-maintenance-daily",
		"0 3 * * *",
		"Run daily cluster maintenance tasks",
	)
	if err != nil {
		return fmt.Errorf("failed to schedule daily maintenance job: %w", err)
	}

	logger.Info("Scheduled cluster formation jobs",
		zap.String("weekly", "0 2 * * 0 (Sunday 2 AM)"),
		zap.String("daily", "0 3 * * * (Daily 3 AM)"),
	)

	return nil
}

// ManualTrigger provides a way to manually trigger cluster formation
// Useful for CLI commands and testing
func ManualTrigger(db *gorm.DB, llmClient *llm.Client, searcher *vector.Searcher, logger *zap.Logger, fullFormation bool) error {
	jobHandler, err := NewClusterFormationJob(db, llmClient, searcher, logger)
	if err != nil {
		return err
	}

	if fullFormation {
		return jobHandler.RunClusterFormation()
	}

	return jobHandler.RunDailyMaintenance()
}
