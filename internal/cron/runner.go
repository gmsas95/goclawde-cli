// Package cron implements a job scheduler for recurring tasks
package cron

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/agent"
	"github.com/gmsas95/goclawde-cli/internal/store"
	"go.uber.org/zap"
)

// Config holds cron runner configuration
type Config struct {
	CheckInterval int // Minutes between job checks
	MaxConcurrent int // Maximum concurrent job executions
}

// Runner manages scheduled job execution
type Runner struct {
	config    Config
	agent     *agent.Agent
	store     *store.Store
	logger    *zap.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	running   bool
	mu        sync.RWMutex
}

// NewRunner creates a new cron runner
func NewRunner(config Config, agentInstance *agent.Agent, st *store.Store, logger *zap.Logger) *Runner {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set defaults
	if config.CheckInterval <= 0 {
		config.CheckInterval = 1 // Check every minute by default
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 3
	}

	return &Runner{
		config: config,
		agent:  agentInstance,
		store:  st,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the cron runner
func (r *Runner) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return fmt.Errorf("cron runner already running")
	}

	r.running = true
	r.wg.Add(1)
	go r.run()

	return nil
}

// Stop stops the cron runner
func (r *Runner) Stop() {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return
	}
	r.running = false
	r.mu.Unlock()

	r.cancel()
	r.wg.Wait()
	r.logger.Info("Cron runner stopped")
}

// IsRunning returns whether the runner is active
func (r *Runner) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.running
}

// run is the main loop
func (r *Runner) run() {
	defer r.wg.Done()

	ticker := time.NewTicker(time.Duration(r.config.CheckInterval) * time.Minute)
	defer ticker.Stop()

	// Check immediately on start
	r.checkAndRunJobs()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.checkAndRunJobs()
		}
	}
}

// checkAndRunJobs checks for due jobs and executes them
func (r *Runner) checkAndRunJobs() {
	jobs, err := r.store.GetDueJobs(50)
	if err != nil {
		r.logger.Error("Failed to get due jobs", zap.Error(err))
		return
	}

	if len(jobs) == 0 {
		return
	}

	r.logger.Info("Found scheduled jobs to run", zap.Int("count", len(jobs)))

	// Execute jobs with semaphore for concurrency control
	sem := make(chan struct{}, r.config.MaxConcurrent)
	var wg sync.WaitGroup

	for _, job := range jobs {
		wg.Add(1)
		sem <- struct{}{} // Acquire

		go func(j *store.ScheduledJob) {
			defer wg.Done()
			defer func() { <-sem }() // Release

			r.executeJob(j)
		}(job)
	}

	wg.Wait()
}

// executeJob runs a single scheduled job
func (r *Runner) executeJob(job *store.ScheduledJob) {
	r.logger.Info("Executing scheduled job",
		zap.String("job_id", job.ID),
		zap.String("name", job.Name),
	)

	// Update last run time
	now := time.Now()
	job.LastRunAt = &now
	job.RunCount++

	// Calculate next run
	nextRun, err := r.calculateNextRun(job.CronExpression, now)
	if err != nil {
		r.logger.Error("Failed to calculate next run",
			zap.String("job_id", job.ID),
			zap.Error(err),
		)
		job.IsActive = false // Disable job on error
	} else {
		job.NextRunAt = &nextRun
	}

	// Save job state before execution
	if err := r.store.UpdateJob(job); err != nil {
		r.logger.Error("Failed to update job state",
			zap.String("job_id", job.ID),
			zap.Error(err),
		)
	}

	// Execute the job prompt via agent
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Minute)
	defer cancel()

	resp, err := r.agent.Chat(ctx, agent.ChatRequest{
		Message:      job.Prompt,
		SystemPrompt: "You are executing a scheduled task. Be concise but thorough.",
		Stream:       false,
	})

	if err != nil {
		r.logger.Error("Job execution failed",
			zap.String("job_id", job.ID),
			zap.Error(err),
		)
	} else {
		r.logger.Info("Job completed",
			zap.String("job_id", job.ID),
			zap.Int("tokens_used", resp.TokensUsed),
		)
	}
}

// calculateNextRun parses cron expression and returns next run time
func (r *Runner) calculateNextRun(cronExpr string, from time.Time) (time.Time, error) {
	// Parse and calculate next run from cron expression
	// For now, support simple intervals and standard cron
	
	// Handle special intervals
	switch cronExpr {
	case "@hourly":
		return from.Add(1 * time.Hour), nil
	case "@daily":
		return from.Add(24 * time.Hour), nil
	case "@weekly":
		return from.Add(7 * 24 * time.Hour), nil
	case "@monthly":
		return from.AddDate(0, 1, 0), nil
	}

	// Parse standard cron expression (simplified)
	// Format: "minute hour day month dow"
	// For now, support simple cases like "0 9 * * *" (daily at 9am)
	
	// Try to parse as simple interval (e.g., "30m", "1h")
	if duration, err := parseSimpleInterval(cronExpr); err == nil {
		return from.Add(duration), nil
	}

	// Fallback: use simple cron parsing
	return r.parseCronExpression(cronExpr, from)
}

// parseSimpleInterval parses strings like "30m", "1h", "2h30m"
func parseSimpleInterval(expr string) (time.Duration, error) {
	return time.ParseDuration(expr)
}

// parseCronExpression parses standard cron expressions
func (r *Runner) parseCronExpression(expr string, from time.Time) (time.Time, error) {
	// Simplified cron parser - supports: "minute hour * * *" format
	// For full cron support, consider using github.com/robfig/cron
	
	var minute, hour int
	var day, month, dow string
	
	_, err := fmt.Sscanf(expr, "%d %d %s %s %s", &minute, &hour, &day, &month, &dow)
	if err != nil {
		return from, fmt.Errorf("unsupported cron format: %s", expr)
	}

	// Calculate next occurrence
	next := time.Date(from.Year(), from.Month(), from.Day(), hour, minute, 0, 0, from.Location())
	
	if next.Before(from) || next.Equal(from) {
		next = next.Add(24 * time.Hour)
	}

	return next, nil
}

// AddJob adds a new scheduled job programmatically
func (r *Runner) AddJob(name, cronExpr, prompt string) (*store.ScheduledJob, error) {
	nextRun, err := r.calculateNextRun(cronExpr, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	job := &store.ScheduledJob{
		Name:           name,
		CronExpression: cronExpr,
		Prompt:         prompt,
		IsActive:       true,
		NextRunAt:      &nextRun,
	}

	if err := r.store.CreateJob(job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	r.logger.Info("Scheduled job added",
		zap.String("job_id", job.ID),
		zap.String("name", name),
		zap.Time("next_run", nextRun),
	)

	return job, nil
}

// RemoveJob removes a scheduled job
func (r *Runner) RemoveJob(jobID string) error {
	return r.store.DeleteJob(jobID)
}

// ListJobs returns all scheduled jobs
func (r *Runner) ListJobs() ([]*store.ScheduledJob, error) {
	return r.store.ListJobs()
}
