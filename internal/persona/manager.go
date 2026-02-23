package persona

import (
	"context"
	"fmt"
	"time"

	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// EvolutionManager coordinates the evolution system components
type EvolutionManager struct {
	pm              *PersonaManager
	engine          *EvolutionEngine
	proposalManager *ProposalManager
	versionManager  *VersionManager
	store           *store.Store
	llmClient       *llm.Client
	logger          *zap.Logger
	config          EvolutionConfig
}

// NewEvolutionManager creates a new evolution manager
func NewEvolutionManager(pm *PersonaManager, store *store.Store, llmClient *llm.Client, logger *zap.Logger) *EvolutionManager {
	return &EvolutionManager{
		pm:        pm,
		store:     store,
		llmClient: llmClient,
		logger:    logger,
		config:    DefaultEvolutionConfig(),
	}
}

// Initialize initializes the evolution system
func (em *EvolutionManager) Initialize() error {
	em.logger.Info("Initializing adaptive persona evolution system")

	// Initialize evolution engine
	em.engine = NewEvolutionEngine(em.store, em.llmClient, em.logger)
	em.engine.SetConfig(em.config)

	// Initialize proposal manager
	em.proposalManager = NewProposalManager(em.store.DB())

	// Initialize version manager
	em.versionManager = NewVersionManager(em.store.DB(), em.pm.GetWorkspacePath())

	// Set in persona manager
	em.pm.SetEvolutionEngine(em.engine)
	em.pm.SetProposalManager(em.proposalManager)
	em.pm.SetVersionManager(em.versionManager)
	em.pm.SetEvolutionConfig(em.config)

	// Enable evolution
	if err := em.pm.EnableEvolution(em.store); err != nil {
		return fmt.Errorf("failed to enable evolution: %w", err)
	}

	em.logger.Info("Evolution system initialized successfully")
	return nil
}

// SetConfig updates the evolution configuration
func (em *EvolutionManager) SetConfig(config EvolutionConfig) {
	em.config = config
	if em.engine != nil {
		em.engine.SetConfig(config)
	}
	if em.pm != nil {
		em.pm.SetEvolutionConfig(config)
	}
}

// GetConfig returns the current evolution configuration
func (em *EvolutionManager) GetConfig() EvolutionConfig {
	return em.config
}

// RunAnalysis runs pattern analysis and generates proposals
func (em *EvolutionManager) RunAnalysis(ctx context.Context) (*AnalysisResult, error) {
	if em.engine == nil {
		return nil, fmt.Errorf("evolution engine not initialized")
	}

	em.logger.Info("Running pattern analysis")

	result, err := em.engine.AnalyzePatterns(ctx)
	if err != nil {
		return nil, fmt.Errorf("pattern analysis failed: %w", err)
	}

	// Save proposals
	for _, proposal := range result.Proposals {
		if err := em.proposalManager.Create(proposal); err != nil {
			em.logger.Warn("Failed to save proposal", zap.Error(err))
		}
	}

	em.logger.Info("Analysis complete",
		zap.Int("patterns_found", result.PatternsFound),
		zap.Int("proposals_created", len(result.Proposals)),
	)

	return result, nil
}

// ApplyProposal applies an approved proposal
func (em *EvolutionManager) ApplyProposal(proposalID string) error {
	if em.proposalManager == nil {
		return fmt.Errorf("proposal manager not initialized")
	}

	proposal, err := em.proposalManager.Get(proposalID)
	if err != nil {
		return fmt.Errorf("proposal not found: %w", err)
	}

	if proposal.Status != ProposalPending {
		return fmt.Errorf("proposal is not pending (status: %s)", proposal.Status)
	}

	// Apply the change
	if err := em.pm.ApplyProposalChange(proposal); err != nil {
		return fmt.Errorf("failed to apply proposal: %w", err)
	}

	// Mark as applied
	if err := em.proposalManager.Apply(proposalID); err != nil {
		return fmt.Errorf("failed to update proposal status: %w", err)
	}

	em.logger.Info("Proposal applied successfully",
		zap.String("proposal_id", proposalID),
		zap.String("title", proposal.Title),
	)

	return nil
}

// RejectProposal rejects a proposal
func (em *EvolutionManager) RejectProposal(proposalID string, reason string) error {
	if em.proposalManager == nil {
		return fmt.Errorf("proposal manager not initialized")
	}

	return em.proposalManager.Reject(proposalID, reason)
}

// GetPendingProposals returns all pending proposals
func (em *EvolutionManager) GetPendingProposals() ([]*EvolutionProposal, error) {
	if em.proposalManager == nil {
		return nil, fmt.Errorf("proposal manager not initialized")
	}

	return em.proposalManager.ListPending()
}

// HasPendingProposals returns true if there are pending proposals
func (em *EvolutionManager) HasPendingProposals() bool {
	if em.proposalManager == nil {
		return false
	}

	return em.proposalManager.HasPendingProposals()
}

// GetPendingProposalCount returns the number of pending proposals
func (em *EvolutionManager) GetPendingProposalCount() (int64, error) {
	if em.proposalManager == nil {
		return 0, fmt.Errorf("proposal manager not initialized")
	}

	return em.proposalManager.GetPendingCount()
}

// NotifyNewProposals creates notifications for new proposals
func (em *EvolutionManager) NotifyNewProposals(proposals []*EvolutionProposal) error {
	if !em.config.Notifications {
		return nil
	}

	for _, proposal := range proposals {
		notification := &Notification{
			ID:         fmt.Sprintf("notif_%d", time.Now().UnixNano()),
			Type:       "proposal",
			Title:      "New Evolution Proposal",
			Message:    proposal.Summary(),
			ProposalID: &proposal.ID,
			Read:       false,
			CreatedAt:  time.Now(),
		}

		if err := em.store.DB().Create(notification).Error; err != nil {
			em.logger.Warn("Failed to create notification", zap.Error(err))
		}
	}

	em.logger.Info("Created notifications for new proposals", zap.Int("count", len(proposals)))
	return nil
}

// GetUnreadNotifications returns unread notifications
func (em *EvolutionManager) GetUnreadNotifications(limit int) ([]*Notification, error) {
	var notifications []*Notification
	err := em.store.DB().Where("read = ?", false).
		Order("created_at DESC").
		Limit(limit).
		Find(&notifications).Error
	return notifications, err
}

// MarkNotificationRead marks a notification as read
func (em *EvolutionManager) MarkNotificationRead(notificationID string) error {
	return em.store.DB().Model(&Notification{}).
		Where("id = ?", notificationID).
		Update("read", true).Error
}

// WeeklyAnalysisJob represents the weekly analysis background job
type WeeklyAnalysisJob struct {
	em       *EvolutionManager
	config   WeeklyAnalysisConfig
	stopChan chan struct{}
	running  bool
}

// NewWeeklyAnalysisJob creates a new weekly analysis job
func NewWeeklyAnalysisJob(em *EvolutionManager) *WeeklyAnalysisJob {
	return &WeeklyAnalysisJob{
		em:       em,
		config:   em.config.WeeklyAnalysis,
		stopChan: make(chan struct{}),
	}
}

// Start starts the weekly analysis job
func (j *WeeklyAnalysisJob) Start() {
	if !j.config.Enabled {
		j.em.logger.Info("Weekly analysis job is disabled")
		return
	}

	if j.running {
		return
	}

	j.running = true
	j.em.logger.Info("Starting weekly analysis job",
		zap.Int("day_of_week", j.config.DayOfWeek),
		zap.Int("hour", j.config.Hour),
		zap.Int("minute", j.config.Minute),
	)

	go j.run()
}

// Stop stops the weekly analysis job
func (j *WeeklyAnalysisJob) Stop() {
	if !j.running {
		return
	}

	j.running = false
	close(j.stopChan)
}

// run is the main loop for the weekly analysis job
func (j *WeeklyAnalysisJob) run() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-j.stopChan:
			return
		case <-ticker.C:
			if j.shouldRun() {
				j.runAnalysis()
			}
		}
	}
}

// shouldRun checks if analysis should run now
func (j *WeeklyAnalysisJob) shouldRun() bool {
	now := time.Now()

	// Check day of week
	if int(now.Weekday()) != j.config.DayOfWeek {
		return false
	}

	// Check hour (run within the hour window)
	if now.Hour() != j.config.Hour {
		return false
	}

	// Check if already ran today
	lastRun, _ := j.getLastRunTime()
	if lastRun != nil {
		// Don't run more than once per day
		if lastRun.Year() == now.Year() &&
			lastRun.YearDay() == now.YearDay() {
			return false
		}
	}

	return true
}

// runAnalysis executes the analysis
func (j *WeeklyAnalysisJob) runAnalysis() {
	j.em.logger.Info("Running scheduled weekly analysis")

	ctx := context.Background()
	result, err := j.em.RunAnalysis(ctx)
	if err != nil {
		j.em.logger.Error("Weekly analysis failed", zap.Error(err))
		return
	}

	// Create notifications for new proposals
	if len(result.Proposals) > 0 {
		if err := j.em.NotifyNewProposals(result.Proposals); err != nil {
			j.em.logger.Warn("Failed to create notifications", zap.Error(err))
		}
	}

	// Update last run time
	j.setLastRunTime(time.Now())

	j.em.logger.Info("Weekly analysis completed",
		zap.Int("patterns_found", result.PatternsFound),
		zap.Int("proposals_created", len(result.Proposals)),
	)
}

// getLastRunTime retrieves the last run time from config
func (j *WeeklyAnalysisJob) getLastRunTime() (*time.Time, error) {
	var cfg store.Config
	err := j.em.store.DB().Where("`key` = ?", "evolution_last_run").First(&cfg).Error
	if err != nil {
		return nil, err
	}

	t, err := time.Parse(time.RFC3339, cfg.Value)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// setLastRunTime stores the last run time
func (j *WeeklyAnalysisJob) setLastRunTime(t time.Time) error {
	return j.em.store.DB().Save(&store.Config{
		Key:       "evolution_last_run",
		Value:     t.Format(time.RFC3339),
		UpdatedAt: time.Now(),
	}).Error
}

// MigrateEvolutionTables runs database migrations for evolution-related tables
func MigrateEvolutionTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&EvolutionProposal{},
		&DetectedPattern{},
		&SkillUsageRecord{},
		&PersonaVersion{},
		&Notification{},
	)
}

// Integration functions for PersonaManager

// SetupEvolution initializes the evolution system on a PersonaManager
func (pm *PersonaManager) SetupEvolution(store *store.Store, llmClient *llm.Client) (*EvolutionManager, error) {
	em := NewEvolutionManager(pm, store, llmClient, pm.logger)

	if err := em.Initialize(); err != nil {
		return nil, err
	}

	return em, nil
}

// RunWeeklyAnalysis manually triggers a weekly analysis
func (pm *PersonaManager) RunWeeklyAnalysis(ctx context.Context) (*AnalysisResult, error) {
	if pm.evolutionEngine == nil {
		return nil, fmt.Errorf("evolution not initialized")
	}

	return pm.evolutionEngine.AnalyzePatterns(ctx)
}

// AutoApplyProposal applies a proposal if it meets the auto-apply threshold
func (pm *PersonaManager) AutoApplyProposal(proposal *EvolutionProposal) error {
	if !pm.evolutionConfig.AutoApplyHighConfidence {
		return fmt.Errorf("auto-apply is disabled")
	}

	if !proposal.CanAutoApply(pm.evolutionConfig) {
		return fmt.Errorf("proposal does not meet auto-apply threshold")
	}

	if err := pm.ApplyProposalChange(proposal); err != nil {
		return err
	}

	if pm.proposalManager != nil {
		return pm.proposalManager.Apply(proposal.ID)
	}

	return nil
}
