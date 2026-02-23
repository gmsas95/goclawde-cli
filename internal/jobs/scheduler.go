// Package jobs provides background job scheduling and management
package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// JobFunc is the function signature for jobs
type JobFunc func(ctx context.Context) error

// Job represents a scheduled background job
type Job struct {
	ID          string
	Name        string
	Description string
	Schedule    string // Cron expression
	Func        JobFunc
	Enabled     bool
	LastRun     *time.Time
	NextRun     *time.Time
	RunCount    int
	ErrorCount  int
	LastError   error
}

// Scheduler manages background job execution
type Scheduler struct {
	cron    *cron.Cron
	jobs    map[string]*Job
	entries map[string]cron.EntryID
	logger  *zap.Logger
	mu      sync.RWMutex
	wg      sync.WaitGroup
	stopCh  chan struct{}
	stopped bool
}

// cronLogger adapts zap.Logger to cron.Logger
type cronLogger struct {
	logger *zap.Logger
}

func (c *cronLogger) Info(msg string, keysAndValues ...interface{}) {
	c.logger.Info(msg, zap.Any("data", keysAndValues))
}

func (c *cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	c.logger.Error(msg, zap.Error(err), zap.Any("data", keysAndValues))
}

// NewScheduler creates a new job scheduler
func NewScheduler(logger *zap.Logger) *Scheduler {
	cronLog := &cronLogger{logger: logger}
	return &Scheduler{
		cron: cron.New(
			cron.WithSeconds(),
			cron.WithChain(
				cron.Recover(cronLog),
				cron.SkipIfStillRunning(cronLog),
			),
			cron.WithLogger(cron.VerbosePrintfLogger(&cronPrintfAdapter{logger: logger})),
		),
		jobs:    make(map[string]*Job),
		entries: make(map[string]cron.EntryID),
		logger:  logger,
		stopCh:  make(chan struct{}),
	}
}

// cronPrintfAdapter adapts zap.Logger to printf-style logger
type cronPrintfAdapter struct {
	logger *zap.Logger
}

func (c *cronPrintfAdapter) Printf(format string, args ...interface{}) {
	c.logger.Info(fmt.Sprintf(format, args...))
}

// RegisterJob registers a new job with the scheduler
func (s *Scheduler) RegisterJob(job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return fmt.Errorf("scheduler is stopped")
	}

	if _, exists := s.jobs[job.ID]; exists {
		return fmt.Errorf("job with ID %s already registered", job.ID)
	}

	if !job.Enabled {
		s.jobs[job.ID] = job
		s.logger.Info("Job registered (disabled)",
			zap.String("id", job.ID),
			zap.String("name", job.Name),
		)
		return nil
	}

	// Wrap the job function with logging and metrics
	wrappedFunc := func() {
		start := time.Now()
		s.wg.Add(1)
		defer s.wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()

		s.logger.Info("Starting job execution",
			zap.String("job_id", job.ID),
			zap.String("job_name", job.Name),
		)

		job.RunCount++
		now := time.Now()
		job.LastRun = &now

		if err := job.Func(ctx); err != nil {
			job.ErrorCount++
			job.LastError = err
			s.logger.Error("Job execution failed",
				zap.String("job_id", job.ID),
				zap.String("job_name", job.Name),
				zap.Error(err),
				zap.Duration("duration", time.Since(start)),
			)
		} else {
			s.logger.Info("Job execution completed",
				zap.String("job_id", job.ID),
				zap.String("job_name", job.Name),
				zap.Duration("duration", time.Since(start)),
			)
		}
	}

	// Add to cron
	entryID, err := s.cron.AddFunc(job.Schedule, wrappedFunc)
	if err != nil {
		return fmt.Errorf("failed to schedule job %s: %w", job.ID, err)
	}

	s.jobs[job.ID] = job
	s.entries[job.ID] = entryID

	// Get next run time
	entry := s.cron.Entry(entryID)
	job.NextRun = &entry.Next

	s.logger.Info("Job registered and scheduled",
		zap.String("id", job.ID),
		zap.String("name", job.Name),
		zap.String("schedule", job.Schedule),
		zap.Time("next_run", entry.Next),
	)

	return nil
}

// UnregisterJob removes a job from the scheduler
func (s *Scheduler) UnregisterJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}

	if entryID, hasEntry := s.entries[jobID]; hasEntry {
		s.cron.Remove(entryID)
		delete(s.entries, jobID)
	}

	delete(s.jobs, jobID)

	s.logger.Info("Job unregistered",
		zap.String("id", jobID),
		zap.String("name", job.Name),
	)

	return nil
}

// EnableJob enables a disabled job
func (s *Scheduler) EnableJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}

	if job.Enabled {
		return nil // Already enabled
	}

	job.Enabled = true

	// Re-register with cron
	wrappedFunc := func() {
		start := time.Now()
		s.wg.Add(1)
		defer s.wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()

		s.logger.Info("Starting job execution",
			zap.String("job_id", job.ID),
			zap.String("job_name", job.Name),
		)

		job.RunCount++
		now := time.Now()
		job.LastRun = &now

		if err := job.Func(ctx); err != nil {
			job.ErrorCount++
			job.LastError = err
			s.logger.Error("Job execution failed",
				zap.String("job_id", job.ID),
				zap.String("job_name", job.Name),
				zap.Error(err),
				zap.Duration("duration", time.Since(start)),
			)
		} else {
			s.logger.Info("Job execution completed",
				zap.String("job_id", job.ID),
				zap.String("job_name", job.Name),
				zap.Duration("duration", time.Since(start)),
			)
		}
	}

	entryID, err := s.cron.AddFunc(job.Schedule, wrappedFunc)
	if err != nil {
		return fmt.Errorf("failed to schedule job %s: %w", jobID, err)
	}

	s.entries[jobID] = entryID
	entry := s.cron.Entry(entryID)
	job.NextRun = &entry.Next

	s.logger.Info("Job enabled",
		zap.String("id", jobID),
		zap.String("name", job.Name),
		zap.Time("next_run", entry.Next),
	)

	return nil
}

// DisableJob disables a job without removing it
func (s *Scheduler) DisableJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}

	if !job.Enabled {
		return nil // Already disabled
	}

	if entryID, hasEntry := s.entries[jobID]; hasEntry {
		s.cron.Remove(entryID)
		delete(s.entries, jobID)
	}

	job.Enabled = false
	job.NextRun = nil

	s.logger.Info("Job disabled",
		zap.String("id", jobID),
		zap.String("name", job.Name),
	)

	return nil
}

// GetJob returns a job by ID
func (s *Scheduler) GetJob(jobID string) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}

	// Update next run time
	if entryID, hasEntry := s.entries[jobID]; hasEntry {
		entry := s.cron.Entry(entryID)
		job.NextRun = &entry.Next
	}

	return job, nil
}

// ListJobs returns all registered jobs
func (s *Scheduler) ListJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		// Update next run time
		if entryID, hasEntry := s.entries[job.ID]; hasEntry {
			entry := s.cron.Entry(entryID)
			job.NextRun = &entry.Next
		}
		jobs = append(jobs, job)
	}

	return jobs
}

// RunJob manually triggers a job execution
func (s *Scheduler) RunJob(jobID string) error {
	s.mu.RLock()
	job, exists := s.jobs[jobID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}

	if job.Func == nil {
		return fmt.Errorf("job %s has no function", jobID)
	}

	// Run in background
	go func() {
		start := time.Now()
		s.wg.Add(1)
		defer s.wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()

		s.logger.Info("Manually triggering job",
			zap.String("job_id", jobID),
			zap.String("job_name", job.Name),
		)

		if err := job.Func(ctx); err != nil {
			s.logger.Error("Manual job execution failed",
				zap.String("job_id", jobID),
				zap.String("job_name", job.Name),
				zap.Error(err),
				zap.Duration("duration", time.Since(start)),
			)
		} else {
			s.logger.Info("Manual job execution completed",
				zap.String("job_id", jobID),
				zap.String("job_name", job.Name),
				zap.Duration("duration", time.Since(start)),
			)
		}
	}()

	return nil
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		s.logger.Warn("Cannot start scheduler, it has been stopped")
		return
	}

	s.cron.Start()
	s.logger.Info("Job scheduler started")
}

// Stop stops the scheduler and waits for running jobs
func (s *Scheduler) Stop() {
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()

	s.logger.Info("Stopping job scheduler...")

	// Stop cron (stops scheduling new jobs)
	ctx := s.cron.Stop()

	// Wait for cron to stop
	<-ctx.Done()

	// Wait for all running jobs to complete
	waitCh := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		s.logger.Info("All jobs completed")
	case <-time.After(30 * time.Second):
		s.logger.Warn("Timeout waiting for jobs to complete")
	}

	s.logger.Info("Job scheduler stopped")
}

// Wait waits for all running jobs to complete with timeout
func (s *Scheduler) Wait(ctx context.Context) error {
	waitCh := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IsRunning returns true if the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.stopped
}

// SchedulePresets contains common cron schedules
var SchedulePresets = struct {
	EveryMinute     string
	Every5Minutes   string
	Every15Minutes  string
	EveryHour       string
	Daily           string
	DailyAt2AM      string
	DailyAt3AM      string
	WeeklySunday2AM string
	WeeklySunday3AM string
	WeeklyMonday1AM string
	Monthly1st4AM   string
}{
	EveryMinute:     "0 * * * * *",
	Every5Minutes:   "0 */5 * * * *",
	Every15Minutes:  "0 */15 * * * *",
	EveryHour:       "0 0 * * * *",
	Daily:           "0 0 0 * * *",
	DailyAt2AM:      "0 0 2 * * *",
	DailyAt3AM:      "0 0 3 * * *",
	WeeklySunday2AM: "0 0 2 * * 0",
	WeeklySunday3AM: "0 0 3 * * 0",
	WeeklyMonday1AM: "0 0 1 * * 1",
	Monthly1st4AM:   "0 0 4 1 * *",
}
