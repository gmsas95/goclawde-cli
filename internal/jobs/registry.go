// Package jobs provides background job registration and initialization
package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/neural"
	"github.com/gmsas95/myrai-cli/internal/reflection"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/gmsas95/myrai-cli/internal/vector"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Registry manages all background jobs for Myrai
type Registry struct {
	scheduler      *Scheduler
	store          *store.Store
	db             *gorm.DB
	llmClient      *llm.Client
	vectorSearcher *vector.Searcher
	config         *config.Config
	logger         *zap.Logger
	initialized    bool
}

// NewRegistry creates a new job registry
func NewRegistry(store *store.Store, cfg *config.Config, logger *zap.Logger) (*Registry, error) {
	db := store.DB()

	// Initialize LLM client
	provider, err := cfg.DefaultProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM provider: %w", err)
	}
	llmClient := llm.NewClient(provider)

	// Initialize vector searcher with proper config
	vectorConfig := &config.VectorConfig{
		Enabled:   true,
		Provider:  "local",
		Dimension: 1536,
	}
	vectorSearcher, err := vector.NewSearcher(vectorConfig, store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector searcher: %w", err)
	}

	return &Registry{
		scheduler:      NewScheduler(logger),
		store:          store,
		db:             db,
		llmClient:      llmClient,
		vectorSearcher: vectorSearcher,
		config:         cfg,
		logger:         logger,
	}, nil
}

// Initialize registers and schedules all background jobs
func (r *Registry) Initialize() error {
	if r.initialized {
		return fmt.Errorf("registry already initialized")
	}

	r.logger.Info("Initializing job registry...")

	// 1. Neural Cluster Formation - Weekly (Sunday 2 AM)
	if err := r.registerNeuralClusterJobs(); err != nil {
		return fmt.Errorf("failed to register neural cluster jobs: %w", err)
	}

	// 2. Reflection Engine Jobs - Daily and Weekly
	if err := r.registerReflectionJobs(); err != nil {
		return fmt.Errorf("failed to register reflection jobs: %w", err)
	}

	// 3. Persona Evolution - Weekly (Monday 1 AM)
	if err := r.registerPersonaEvolutionJob(); err != nil {
		r.logger.Warn("Failed to register persona evolution job, skipping", zap.Error(err))
		// Don't fail initialization for persona evolution
	}

	r.initialized = true
	r.logger.Info("Job registry initialized successfully",
		zap.Int("job_count", len(r.scheduler.ListJobs())),
	)

	return nil
}

// registerNeuralClusterJobs registers neural cluster formation and maintenance jobs
func (r *Registry) registerNeuralClusterJobs() error {
	// Create cluster manager
	clusterManager, err := neural.NewClusterManager(r.db, r.llmClient, r.vectorSearcher, r.logger)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// Job 1: Weekly Cluster Formation
	formationJob := &Job{
		ID:          "neural-formation-weekly",
		Name:        "Neural Cluster Formation",
		Description: "Forms semantic clusters from unclustered memories",
		Schedule:    SchedulePresets.WeeklySunday2AM,
		Enabled:     true,
		Func: func(ctx context.Context) error {
			start := time.Now()
			opts := neural.DefaultFormationOptions()
			result, err := clusterManager.FormClusters(ctx, opts)
			if err != nil {
				return fmt.Errorf("cluster formation failed: %w", err)
			}

			r.logger.Info("Weekly cluster formation complete",
				zap.Int("clusters_created", result.ClustersCreated),
				zap.Int("clusters_updated", result.ClustersUpdated),
				zap.Int("memories_processed", result.MemoriesProcessed),
				zap.Duration("duration", time.Since(start)),
			)
			return nil
		},
	}

	if err := r.scheduler.RegisterJob(formationJob); err != nil {
		return fmt.Errorf("failed to register formation job: %w", err)
	}

	// Job 2: Daily Cluster Maintenance
	maintenanceJob := &Job{
		ID:          "neural-maintenance-daily",
		Name:        "Neural Cluster Maintenance",
		Description: "Performs daily maintenance on neural clusters",
		Schedule:    SchedulePresets.DailyAt3AM,
		Enabled:     true,
		Func: func(ctx context.Context) error {
			start := time.Now()

			// Update cluster statistics
			if err := r.updateClusterStatistics(ctx); err != nil {
				r.logger.Warn("Failed to update cluster statistics", zap.Error(err))
			}

			// Identify stale clusters
			if err := r.identifyStaleClusters(ctx, clusterManager); err != nil {
				r.logger.Warn("Failed to identify stale clusters", zap.Error(err))
			}

			// Cleanup old query patterns
			if err := r.cleanupQueryPatterns(ctx); err != nil {
				r.logger.Warn("Failed to cleanup query patterns", zap.Error(err))
			}

			r.logger.Info("Daily cluster maintenance complete",
				zap.Duration("duration", time.Since(start)),
			)
			return nil
		},
	}

	if err := r.scheduler.RegisterJob(maintenanceJob); err != nil {
		return fmt.Errorf("failed to register maintenance job: %w", err)
	}

	return nil
}

// registerReflectionJobs registers reflection engine jobs
func (r *Registry) registerReflectionJobs() error {
	// Create reflection jobs manager
	reflectionJobs := reflection.NewJobs(r.store, r.config, r.logger)

	// Job 1: Daily Lightweight Reflection
	dailyJob := &Job{
		ID:          "reflection-daily",
		Name:        "Daily Reflection Analysis",
		Description: "Performs lightweight daily analysis of new memories",
		Schedule:    SchedulePresets.DailyAt2AM,
		Enabled:     true,
		Func: func(ctx context.Context) error {
			return reflectionJobs.DailyLightweightJob(ctx)
		},
	}

	if err := r.scheduler.RegisterJob(dailyJob); err != nil {
		return fmt.Errorf("failed to register daily reflection job: %w", err)
	}

	// Job 2: Weekly Deep Audit
	weeklyJob := &Job{
		ID:          "reflection-weekly",
		Name:        "Weekly Deep Reflection Audit",
		Description: "Performs comprehensive weekly memory audit",
		Schedule:    SchedulePresets.WeeklySunday3AM,
		Enabled:     true,
		Func: func(ctx context.Context) error {
			return reflectionJobs.WeeklyDeepAudit(ctx)
		},
	}

	if err := r.scheduler.RegisterJob(weeklyJob); err != nil {
		return fmt.Errorf("failed to register weekly reflection job: %w", err)
	}

	// Job 3: Monthly Archival
	monthlyJob := &Job{
		ID:          "reflection-monthly",
		Name:        "Monthly Memory Archival",
		Description: "Archives old memories and generates monthly report",
		Schedule:    SchedulePresets.Monthly1st4AM,
		Enabled:     true,
		Func: func(ctx context.Context) error {
			return reflectionJobs.MonthlyArchival(ctx)
		},
	}

	if err := r.scheduler.RegisterJob(monthlyJob); err != nil {
		return fmt.Errorf("failed to register monthly reflection job: %w", err)
	}

	return nil
}

// registerPersonaEvolutionJob registers the persona evolution analysis job
func (r *Registry) registerPersonaEvolutionJob() error {
	// For now, we skip persona evolution as it requires different initialization
	// This can be added later when the persona package API is stabilized
	r.logger.Info("Skipping persona evolution job - not yet integrated")
	return nil
}

// Helper functions for cluster maintenance

func (r *Registry) updateClusterStatistics(ctx context.Context) error {
	result := r.db.Exec(`
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

	r.logger.Debug("Updated cluster statistics",
		zap.Int64("rows_affected", result.RowsAffected),
	)

	return nil
}

func (r *Registry) identifyStaleClusters(ctx context.Context, manager *neural.ClusterManager) error {
	threshold := time.Now().AddDate(0, 0, -30)

	var staleClusters []neural.NeuralCluster
	err := r.db.Where(
		"confidence_score < ? AND (last_accessed < ? OR last_accessed IS NULL)",
		0.6, threshold,
	).Limit(10).Find(&staleClusters).Error

	if err != nil {
		return fmt.Errorf("failed to find stale clusters: %w", err)
	}

	for _, cluster := range staleClusters {
		if err := manager.RefreshCluster(ctx, cluster.ID); err != nil {
			r.logger.Warn("Failed to refresh stale cluster",
				zap.String("cluster_id", cluster.ID),
				zap.Error(err),
			)
		}
	}

	return nil
}

func (r *Registry) cleanupQueryPatterns(ctx context.Context) error {
	cutoff := time.Now().AddDate(0, 0, -30)

	result := r.db.Where("created_at < ?", cutoff).Delete(&neural.QueryPattern{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup query patterns: %w", result.Error)
	}

	r.logger.Debug("Cleaned up old query patterns",
		zap.Int64("rows_deleted", result.RowsAffected),
	)

	return nil
}

// Start starts the scheduler and all registered jobs
func (r *Registry) Start() {
	r.scheduler.Start()
}

// Stop stops the scheduler gracefully
func (r *Registry) Stop() {
	r.scheduler.Stop()
}

// Wait waits for all running jobs to complete
func (r *Registry) Wait(ctx context.Context) error {
	return r.scheduler.Wait(ctx)
}

// GetScheduler returns the underlying scheduler
func (r *Registry) GetScheduler() *Scheduler {
	return r.scheduler
}

// IsInitialized returns true if the registry has been initialized
func (r *Registry) IsInitialized() bool {
	return r.initialized
}

// ManualJobTrigger allows manual execution of a job by ID
func (r *Registry) ManualJobTrigger(jobID string) error {
	return r.scheduler.RunJob(jobID)
}

// ListJobs returns all registered jobs
func (r *Registry) ListJobs() []*Job {
	return r.scheduler.ListJobs()
}
