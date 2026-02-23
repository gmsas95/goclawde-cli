package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Sort options
const (
	SortByRelevance = "relevance"
	SortByRating    = "rating"
	SortByInstalls  = "installs"
	SortByRecent    = "recent"
	SortByName      = "name"
)

// Category constants
const (
	CategoryAll           = "all"
	CategoryProductivity  = "productivity"
	CategoryDevelopment   = "development"
	CategoryData          = "data"
	CategoryCommunication = "communication"
	CategoryAutomation    = "automation"
	CategoryEntertainment = "entertainment"
	CategoryUtilities     = "utilities"
)

// Client provides marketplace operations
type Client struct {
	db           *gorm.DB
	githubClient *GitHubClient
	manager      *Manager
	reviews      *ReviewsManager
	logger       *zap.Logger
}

// SearchOptions contains search parameters
type SearchOptions struct {
	Query    string
	Category string
	Sort     string
	Limit    int
	Offset   int
	Verified bool
	Tags     []string
	Author   string
}

// SearchResult contains search results
type SearchResult struct {
	Agents  []*AgentPackage
	Total   int64
	HasMore bool
}

// NewClient creates a new marketplace client
func NewClient(db *gorm.DB, githubClient *GitHubClient, manager *Manager, reviews *ReviewsManager, logger *zap.Logger) *Client {
	return &Client{
		db:           db,
		githubClient: githubClient,
		manager:      manager,
		reviews:      reviews,
		logger:       logger,
	}
}

// Search searches for agents in the marketplace
func (c *Client) Search(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	c.logger.Debug("Searching marketplace",
		zap.String("query", opts.Query),
		zap.String("category", opts.Category),
		zap.String("sort", opts.Sort))

	// First try GitHub for fresh results
	if opts.Query != "" && c.githubClient != nil {
		return c.searchGitHub(ctx, opts)
	}

	// Otherwise search local database
	return c.searchLocal(ctx, opts)
}

// searchGitHub searches GitHub repositories
func (c *Client) searchGitHub(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	repos, err := c.githubClient.SearchRepositories(ctx, opts.Query)
	if err != nil {
		// Fall back to local search on error
		c.logger.Warn("GitHub search failed, falling back to local", zap.Error(err))
		return c.searchLocal(ctx, opts)
	}

	var agents []*AgentPackage
	for _, repo := range repos {
		// Parse agent from each repo
		pkg, err := c.githubClient.ParseAgentFromRepo(ctx, repo.Name)
		if err != nil {
			c.logger.Debug("Failed to parse agent from repo",
				zap.String("repo", repo.Name),
				zap.Error(err))
			continue
		}

		// Filter by category if specified
		if opts.Category != "" && opts.Category != CategoryAll {
			if !c.matchesCategory(pkg, opts.Category) {
				continue
			}
		}

		// Filter by tags
		if len(opts.Tags) > 0 && !c.hasAnyTag(pkg, opts.Tags) {
			continue
		}

		// Filter by author
		if opts.Author != "" && !strings.EqualFold(pkg.Author, opts.Author) {
			continue
		}

		agents = append(agents, pkg)
	}

	// Sort results
	agents = c.sortAgents(agents, opts.Sort)

	// Apply limit and offset
	total := int64(len(agents))
	start := opts.Offset
	end := start + opts.Limit

	if start > len(agents) {
		start = len(agents)
	}
	if end > len(agents) {
		end = len(agents)
	}

	return &SearchResult{
		Agents:  agents[start:end],
		Total:   total,
		HasMore: end < len(agents),
	}, nil
}

// searchLocal searches the local database
func (c *Client) searchLocal(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	query := c.db.Model(&MarketplaceAgent{})

	// Apply text search
	if opts.Query != "" {
		searchTerm := "%" + opts.Query + "%"
		query = query.Where(
			"name LIKE ? OR description LIKE ? OR author LIKE ?",
			searchTerm, searchTerm, searchTerm)
	}

	// Apply verified filter
	if opts.Verified {
		query = query.Where("verified = ?", true)
	}

	// Apply author filter
	if opts.Author != "" {
		query = query.Where("author = ?", opts.Author)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count results: %w", err)
	}

	// Apply sorting
	switch opts.Sort {
	case SortByRating:
		query = query.Order("rating DESC, review_count DESC")
	case SortByInstalls:
		query = query.Order("install_count DESC")
	case SortByRecent:
		query = query.Order("updated_at DESC")
	case SortByName:
		query = query.Order("name ASC")
	default:
		query = query.Order("rating DESC, install_count DESC")
	}

	// Apply pagination
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}

	// Execute query
	var dbAgents []*MarketplaceAgent
	if err := query.Find(&dbAgents).Error; err != nil {
		return nil, fmt.Errorf("failed to search agents: %w", err)
	}

	// Convert to AgentPackage
	agents := make([]*AgentPackage, len(dbAgents))
	for i, dbAgent := range dbAgents {
		pkg, err := dbAgent.ToAgentPackage()
		if err != nil {
			c.logger.Warn("Failed to convert agent", zap.Error(err))
			continue
		}
		agents[i] = pkg
	}

	return &SearchResult{
		Agents:  agents,
		Total:   total,
		HasMore: int64(opts.Offset+len(agents)) < total,
	}, nil
}

// GetAgent retrieves an agent by ID
func (c *Client) GetAgent(ctx context.Context, id string) (*AgentPackage, error) {
	// Try database first
	var dbAgent MarketplaceAgent
	err := c.db.First(&dbAgent, "id = ? OR name = ?", id, id).Error
	if err == nil {
		return dbAgent.ToAgentPackage()
	}

	// Try GitHub
	if c.githubClient != nil {
		pkg, err := c.githubClient.ParseAgentFromRepo(ctx, id)
		if err == nil {
			return pkg, nil
		}
	}

	return nil, fmt.Errorf("agent not found: %s", id)
}

// GetAgentByName retrieves an agent by name
func (c *Client) GetAgentByName(ctx context.Context, name string) (*AgentPackage, error) {
	return c.GetAgent(ctx, name)
}

// DownloadAgent downloads and installs an agent
func (c *Client) DownloadAgent(ctx context.Context, repo, version, userID string) (*InstalledAgent, error) {
	return c.manager.Install(ctx, repo, version, userID)
}

// ListCategories returns available categories
func (c *Client) ListCategories(ctx context.Context) ([]string, error) {
	// This could come from database or be hardcoded
	categories := []string{
		CategoryAll,
		CategoryProductivity,
		CategoryDevelopment,
		CategoryData,
		CategoryCommunication,
		CategoryAutomation,
		CategoryEntertainment,
		CategoryUtilities,
	}
	return categories, nil
}

// GetFeatured returns featured agents
func (c *Client) GetFeatured(ctx context.Context, limit int) ([]*AgentPackage, error) {
	var dbAgents []*MarketplaceAgent
	err := c.db.Where("verified = ?", true).
		Order("install_count DESC, rating DESC").
		Limit(limit).
		Find(&dbAgents).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get featured agents: %w", err)
	}

	agents := make([]*AgentPackage, 0, len(dbAgents))
	for _, dbAgent := range dbAgents {
		pkg, err := dbAgent.ToAgentPackage()
		if err != nil {
			continue
		}
		agents = append(agents, pkg)
	}

	return agents, nil
}

// GetTrending returns trending agents
func (c *Client) GetTrending(ctx context.Context, limit int) ([]*AgentPackage, error) {
	var dbAgents []*MarketplaceAgent
	// Trending = high install count in last 30 days
	// For now, just use recent installs
	err := c.db.Order("install_count DESC").
		Limit(limit).
		Find(&dbAgents).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get trending agents: %w", err)
	}

	agents := make([]*AgentPackage, 0, len(dbAgents))
	for _, dbAgent := range dbAgents {
		pkg, err := dbAgent.ToAgentPackage()
		if err != nil {
			continue
		}
		pkg.Badges = append(pkg.Badges, BadgeTrending)
		agents = append(agents, pkg)
	}

	return agents, nil
}

// GetNew returns new arrivals
func (c *Client) GetNew(ctx context.Context, limit int) ([]*AgentPackage, error) {
	var dbAgents []*MarketplaceAgent
	err := c.db.Order("created_at DESC").
		Limit(limit).
		Find(&dbAgents).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get new agents: %w", err)
	}

	agents := make([]*AgentPackage, 0, len(dbAgents))
	for _, dbAgent := range dbAgents {
		pkg, err := dbAgent.ToAgentPackage()
		if err != nil {
			continue
		}
		agents = append(agents, pkg)
	}

	return agents, nil
}

// SubmitAgent submits an agent to the marketplace (for publishing)
func (c *Client) SubmitAgent(ctx context.Context, pkg *AgentPackage, bundle *AgentPackageBundle) error {
	// Verify the package
	verifier := NewVerifier(c.logger)
	result, err := verifier.Verify(ctx, pkg, bundle)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	if !result.Passed {
		return fmt.Errorf("agent failed verification: %v", result.Errors)
	}

	// Save to database
	marketplaceAgent := FromAgentPackage(pkg)
	marketplaceAgent.Verified = true
	badgesJSON, _ := json.Marshal(result.Badges)
	marketplaceAgent.Badges = badgesJSON
	marketplaceAgent.SecurityScore = result.SecurityScore
	marketplaceAgent.QualityScore = result.QualityScore

	var existing MarketplaceAgent
	err = c.db.Where("name = ?", pkg.Name).First(&existing).Error
	if err == nil {
		// Update existing
		marketplaceAgent.ID = existing.ID
		return c.db.Save(marketplaceAgent).Error
	}

	// Create new
	return c.db.Create(marketplaceAgent).Error
}

// Helper methods

func (c *Client) matchesCategory(pkg *AgentPackage, category string) bool {
	category = strings.ToLower(category)

	// Check tags
	for _, tag := range pkg.Tags {
		if strings.ToLower(tag) == category {
			return true
		}
	}

	// Check name and description
	if strings.Contains(strings.ToLower(pkg.Name), category) ||
		strings.Contains(strings.ToLower(pkg.Description), category) {
		return true
	}

	return false
}

func (c *Client) hasAnyTag(pkg *AgentPackage, tags []string) bool {
	for _, searchTag := range tags {
		searchTag = strings.ToLower(searchTag)
		for _, tag := range pkg.Tags {
			if strings.ToLower(tag) == searchTag {
				return true
			}
		}
	}
	return false
}

func (c *Client) sortAgents(agents []*AgentPackage, sortBy string) []*AgentPackage {
	switch sortBy {
	case SortByRating:
		sort.Slice(agents, func(i, j int) bool {
			return agents[i].Rating > agents[j].Rating
		})
	case SortByInstalls:
		sort.Slice(agents, func(i, j int) bool {
			return agents[i].InstallCount > agents[j].InstallCount
		})
	case SortByRecent:
		sort.Slice(agents, func(i, j int) bool {
			return agents[i].UpdatedAt.After(agents[j].UpdatedAt)
		})
	case SortByName:
		sort.Slice(agents, func(i, j int) bool {
			return strings.ToLower(agents[i].Name) < strings.ToLower(agents[j].Name)
		})
	default:
		// Default: sort by weighted score
		sort.Slice(agents, func(i, j int) bool {
			scoreI := agents[i].Rating*0.4 + float64(agents[i].InstallCount)*0.01*0.6
			scoreJ := agents[j].Rating*0.4 + float64(agents[j].InstallCount)*0.01*0.6
			return scoreI > scoreJ
		})
	}
	return agents
}

// GetManager returns the agent manager
func (c *Client) GetManager() *Manager {
	return c.manager
}

// GetReviewsManager returns the reviews manager
func (c *Client) GetReviewsManager() *ReviewsManager {
	return c.reviews
}

// SyncWithGitHub syncs local database with GitHub repositories
func (c *Client) SyncWithGitHub(ctx context.Context) error {
	if c.githubClient == nil {
		return fmt.Errorf("GitHub client not configured")
	}

	repos, err := c.githubClient.ListRepositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	for _, repo := range repos {
		// Parse agent from repo
		pkg, err := c.githubClient.ParseAgentFromRepo(ctx, repo.Name)
		if err != nil {
			c.logger.Debug("Failed to parse agent",
				zap.String("repo", repo.Name),
				zap.Error(err))
			continue
		}

		// Update or create in database
		marketplaceAgent := FromAgentPackage(pkg)
		marketplaceAgent.GitHubOrg = c.githubClient.GetOrg()
		marketplaceAgent.GitHubRepo = repo.Name
		marketplaceAgent.GitHubStars = repo.Stars
		marketplaceAgent.GitHubUpdatedAt = &repo.PushedAt

		var existing MarketplaceAgent
		err = c.db.Where("name = ?", pkg.Name).First(&existing).Error
		if err == nil {
			marketplaceAgent.ID = existing.ID
			if err := c.db.Save(marketplaceAgent).Error; err != nil {
				c.logger.Warn("Failed to update agent",
					zap.String("name", pkg.Name),
					zap.Error(err))
			}
		} else {
			if err := c.db.Create(marketplaceAgent).Error; err != nil {
				c.logger.Warn("Failed to create agent",
					zap.String("name", pkg.Name),
					zap.Error(err))
			}
		}
	}

	return nil
}
