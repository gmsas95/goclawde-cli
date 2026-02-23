package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Manager handles agent installation, updates, and removal
type Manager struct {
	db           *gorm.DB
	store        *store.Store
	githubClient *GitHubClient
	verifier     *Verifier
	installDir   string
	logger       *zap.Logger
}

// NewManager creates a new agent manager
func NewManager(s *store.Store, githubClient *GitHubClient, installDir string, logger *zap.Logger) *Manager {
	if installDir == "" {
		// Default to ~/.myrai/agents
		home, _ := os.UserHomeDir()
		installDir = filepath.Join(home, ".myrai", "agents")
	}

	return &Manager{
		db:           s.DB(),
		store:        s,
		githubClient: githubClient,
		verifier:     NewVerifier(logger),
		installDir:   installDir,
		logger:       logger,
	}
}

// Install installs an agent from a GitHub repository
func (m *Manager) Install(ctx context.Context, repo, version, userID string) (*InstalledAgent, error) {
	m.logger.Info("Installing agent",
		zap.String("repo", repo),
		zap.String("version", version))

	// Parse agent from GitHub
	pkg, err := m.githubClient.ParseAgentFromRepo(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse agent: %w", err)
	}

	// If version specified, check if it matches
	if version != "" && version != "latest" {
		if !pkg.MatchesVersion(version) {
			// Try to get specific release
			release, err := m.getReleaseByVersion(ctx, repo, version)
			if err != nil {
				return nil, fmt.Errorf("version %s not found: %w", version, err)
			}
			// Use release tag as version
			pkg.Version = release.TagName
		}
	}

	// Check if already installed
	existing, err := m.getInstalledByName(ctx, pkg.Name, userID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("agent %s is already installed (version %s). Use 'update' to upgrade",
			pkg.Name, existing.Version)
	}

	// Download release
	downloadDir, err := m.githubClient.DownloadRelease(ctx, repo, pkg.Version, m.installDir)
	if err != nil {
		return nil, fmt.Errorf("failed to download agent: %w", err)
	}

	// Load bundle for verification
	bundle, err := LoadFromDirectory(downloadDir)
	if err != nil {
		os.RemoveAll(downloadDir)
		return nil, fmt.Errorf("failed to load agent bundle: %w", err)
	}

	// Verify package
	verifyResult, err := m.verifier.Verify(ctx, pkg, bundle)
	if err != nil {
		m.logger.Warn("Verification failed", zap.Error(err))
	}

	if !verifyResult.Passed {
		// Log warnings but continue installation
		m.logger.Warn("Agent verification failed",
			zap.Strings("errors", verifyResult.Errors),
			zap.Strings("warnings", verifyResult.Warnings))
	}

	// Save or update marketplace agent record
	marketplaceAgent := FromAgentPackage(pkg)
	marketplaceAgent.Verified = verifyResult.Passed
	marketplaceAgent.Badges, _ = json.Marshal(verifyResult.Badges)
	marketplaceAgent.SecurityScore = verifyResult.SecurityScore
	marketplaceAgent.QualityScore = verifyResult.QualityScore
	marketplaceAgent.GitHubOrg = m.githubClient.GetOrg()
	marketplaceAgent.GitHubRepo = repo

	var dbAgent MarketplaceAgent
	err = m.db.Where("name = ?", pkg.Name).First(&dbAgent).Error
	if err == gorm.ErrRecordNotFound {
		// Create new
		if err := m.db.Create(marketplaceAgent).Error; err != nil {
			return nil, fmt.Errorf("failed to save agent record: %w", err)
		}
		dbAgent = *marketplaceAgent
	} else if err != nil {
		return nil, fmt.Errorf("failed to check existing agent: %w", err)
	} else {
		// Update existing
		marketplaceAgent.ID = dbAgent.ID
		if err := m.db.Save(marketplaceAgent).Error; err != nil {
			return nil, fmt.Errorf("failed to update agent record: %w", err)
		}
	}

	// Create installed agent record
	installed := &InstalledAgent{
		AgentID:     dbAgent.ID,
		UserID:      userID,
		Version:     pkg.Version,
		InstallPath: downloadDir,
		ConfigPath:  filepath.Join(downloadDir, "config"),
		IsActive:    true,
		IsEnabled:   true,
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := m.db.Create(installed).Error; err != nil {
		os.RemoveAll(downloadDir)
		return nil, fmt.Errorf("failed to save installation record: %w", err)
	}

	// Update install count
	m.db.Model(&MarketplaceAgent{}).Where("id = ?", dbAgent.ID).
		UpdateColumn("install_count", gorm.Expr("install_count + 1"))

	m.logger.Info("Agent installed successfully",
		zap.String("agent", pkg.Name),
		zap.String("version", pkg.Version),
		zap.String("path", downloadDir))

	return installed, nil
}

// Update updates an installed agent to the latest version
func (m *Manager) Update(ctx context.Context, agentName, userID string) (*InstalledAgent, error) {
	m.logger.Info("Updating agent", zap.String("agent", agentName))

	// Get installed agent
	installed, err := m.getInstalledByName(ctx, agentName, userID)
	if err != nil {
		return nil, fmt.Errorf("agent not installed: %w", err)
	}

	// Get marketplace agent
	var agent MarketplaceAgent
	if err := m.db.First(&agent, "id = ?", installed.AgentID).Error; err != nil {
		return nil, fmt.Errorf("failed to find agent record: %w", err)
	}

	// Get latest version from GitHub
	pkg, err := m.githubClient.ParseAgentFromRepo(ctx, agent.GitHubRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest version: %w", err)
	}

	// Check if update is needed
	if pkg.Version == installed.Version {
		m.logger.Info("Agent is already up to date",
			zap.String("agent", agentName),
			zap.String("version", installed.Version))
		return installed, nil
	}

	// Backup current installation
	backupDir := installed.InstallPath + ".backup"
	if err := os.Rename(installed.InstallPath, backupDir); err != nil {
		return nil, fmt.Errorf("failed to backup current installation: %w", err)
	}

	// Download new version
	downloadDir, err := m.githubClient.DownloadRelease(ctx, agent.GitHubRepo, pkg.Version, m.installDir)
	if err != nil {
		// Restore backup
		os.Rename(backupDir, installed.InstallPath)
		return nil, fmt.Errorf("failed to download update: %w", err)
	}

	// Verify new version
	bundle, err := LoadFromDirectory(downloadDir)
	if err != nil {
		os.RemoveAll(downloadDir)
		os.Rename(backupDir, installed.InstallPath)
		return nil, fmt.Errorf("failed to load updated bundle: %w", err)
	}

	verifyResult, err := m.verifier.Verify(ctx, pkg, bundle)
	if err != nil {
		m.logger.Warn("Verification warning", zap.Error(err))
	}

	// Update agent record
	marketplaceAgent := FromAgentPackage(pkg)
	marketplaceAgent.ID = agent.ID
	marketplaceAgent.Verified = verifyResult.Passed
	marketplaceAgent.Badges, _ = json.Marshal(verifyResult.Badges)
	marketplaceAgent.SecurityScore = verifyResult.SecurityScore
	marketplaceAgent.QualityScore = verifyResult.QualityScore

	if err := m.db.Save(marketplaceAgent).Error; err != nil {
		// Restore backup
		os.RemoveAll(downloadDir)
		os.Rename(backupDir, installed.InstallPath)
		return nil, fmt.Errorf("failed to update agent record: %w", err)
	}

	// Update installed record
	installed.Version = pkg.Version
	installed.InstallPath = downloadDir
	installed.UpdatedAt = time.Now()

	if err := m.db.Save(installed).Error; err != nil {
		// Restore backup
		os.RemoveAll(downloadDir)
		os.Rename(backupDir, installed.InstallPath)
		return nil, fmt.Errorf("failed to update installation record: %w", err)
	}

	// Remove backup
	os.RemoveAll(backupDir)

	m.logger.Info("Agent updated successfully",
		zap.String("agent", agentName),
		zap.String("from_version", installed.Version),
		zap.String("to_version", pkg.Version))

	return installed, nil
}

// Remove uninstalls an agent
func (m *Manager) Remove(ctx context.Context, agentName, userID string) error {
	m.logger.Info("Removing agent", zap.String("agent", agentName))

	// Get installed agent
	installed, err := m.getInstalledByName(ctx, agentName, userID)
	if err != nil {
		return fmt.Errorf("agent not installed: %w", err)
	}

	// Remove installation directory
	if err := os.RemoveAll(installed.InstallPath); err != nil {
		m.logger.Warn("Failed to remove installation directory",
			zap.String("path", installed.InstallPath),
			zap.Error(err))
	}

	// Remove from database
	if err := m.db.Delete(installed).Error; err != nil {
		return fmt.Errorf("failed to remove installation record: %w", err)
	}

	m.logger.Info("Agent removed successfully", zap.String("agent", agentName))
	return nil
}

// ListInstalled lists all installed agents for a user
func (m *Manager) ListInstalled(ctx context.Context, userID string) ([]*InstalledAgent, error) {
	var installed []*InstalledAgent
	err := m.db.Where("user_id = ?", userID).
		Preload("Agent").
		Order("installed_at DESC").
		Find(&installed).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list installed agents: %w", err)
	}

	return installed, nil
}

// GetInstalled gets a specific installed agent
func (m *Manager) GetInstalled(ctx context.Context, agentID, userID string) (*InstalledAgent, error) {
	var installed InstalledAgent
	err := m.db.Where("agent_id = ? AND user_id = ?", agentID, userID).
		Preload("Agent").
		First(&installed).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("agent not installed")
		}
		return nil, fmt.Errorf("failed to get installed agent: %w", err)
	}

	return &installed, nil
}

// Activate activates an installed agent
func (m *Manager) Activate(ctx context.Context, agentName, userID string) error {
	installed, err := m.getInstalledByName(ctx, agentName, userID)
	if err != nil {
		return err
	}

	installed.IsActive = true
	installed.UpdatedAt = time.Now()

	return m.db.Save(installed).Error
}

// Deactivate deactivates an installed agent
func (m *Manager) Deactivate(ctx context.Context, agentName, userID string) error {
	installed, err := m.getInstalledByName(ctx, agentName, userID)
	if err != nil {
		return err
	}

	installed.IsActive = false
	installed.UpdatedAt = time.Now()

	return m.db.Save(installed).Error
}

// Enable enables an installed agent
func (m *Manager) Enable(ctx context.Context, agentName, userID string) error {
	installed, err := m.getInstalledByName(ctx, agentName, userID)
	if err != nil {
		return err
	}

	installed.IsEnabled = true
	installed.UpdatedAt = time.Now()

	return m.db.Save(installed).Error
}

// Disable disables an installed agent
func (m *Manager) Disable(ctx context.Context, agentName, userID string) error {
	installed, err := m.getInstalledByName(ctx, agentName, userID)
	if err != nil {
		return err
	}

	installed.IsEnabled = false
	installed.UpdatedAt = time.Now()

	return m.db.Save(installed).Error
}

// GetAgentPackage loads the agent package for an installed agent
func (m *Manager) GetAgentPackage(ctx context.Context, agentName, userID string) (*AgentPackage, error) {
	installed, err := m.getInstalledByName(ctx, agentName, userID)
	if err != nil {
		return nil, err
	}

	// Load from installation directory
	bundle, err := LoadFromDirectory(installed.InstallPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load agent package: %w", err)
	}

	return bundle.Package, nil
}

// IsInstalled checks if an agent is installed
func (m *Manager) IsInstalled(ctx context.Context, agentName, userID string) bool {
	_, err := m.getInstalledByName(ctx, agentName, userID)
	return err == nil
}

// GetInstallPath returns the installation path for an agent
func (m *Manager) GetInstallPath(agentName, userID string) (string, error) {
	installed, err := m.getInstalledByName(context.Background(), agentName, userID)
	if err != nil {
		return "", err
	}
	return installed.InstallPath, nil
}

// Helper methods

func (m *Manager) getInstalledByName(ctx context.Context, agentName, userID string) (*InstalledAgent, error) {
	var installed InstalledAgent
	err := m.db.Joins("JOIN marketplace_agents ON marketplace_agents.id = installed_agents.agent_id").
		Where("marketplace_agents.name = ? AND installed_agents.user_id = ?",
			agentName, userID).
		Preload("Agent").
		First(&installed).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("agent not installed: %s", agentName)
		}
		return nil, err
	}

	return &installed, nil
}

func (m *Manager) getReleaseByVersion(ctx context.Context, repo, version string) (*GitHubRelease, error) {
	releases, err := m.githubClient.ListReleases(ctx, repo)
	if err != nil {
		return nil, err
	}

	for _, release := range releases {
		if release.TagName == version || release.TagName == "v"+version {
			return &release, nil
		}
	}

	return nil, fmt.Errorf("version %s not found", version)
}

// GetInstallDir returns the installation directory
func (m *Manager) GetInstallDir() string {
	return m.installDir
}

// SetInstallDir sets the installation directory
func (m *Manager) SetInstallDir(dir string) {
	m.installDir = dir
}
