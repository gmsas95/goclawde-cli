package marketplace

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// MarketplaceAgent represents an agent in the marketplace database
type MarketplaceAgent struct {
	ID          string          `gorm:"primaryKey" json:"id"`
	Name        string          `gorm:"index:idx_name,unique" json:"name"`
	Version     string          `json:"version"`
	Author      string          `gorm:"index" json:"author"`
	Description string          `json:"description"`
	Tags        json.RawMessage `json:"tags" gorm:"type:text"`
	Icon        string          `json:"icon"`
	Homepage    string          `json:"homepage"`
	Repository  string          `json:"repository"`
	License     string          `json:"license"`

	// Requirements stored as JSON
	Requirements json.RawMessage `json:"requirements" gorm:"type:text"`
	Knowledge    json.RawMessage `json:"knowledge" gorm:"type:text"`
	Skills       json.RawMessage `json:"skills" gorm:"type:text"`
	Chains       json.RawMessage `json:"chains" gorm:"type:text"`
	MCPServers   json.RawMessage `json:"mcp_servers" gorm:"type:text"`
	Persona      json.RawMessage `json:"persona" gorm:"type:text"`
	Pricing      json.RawMessage `json:"pricing" gorm:"type:text"`
	Support      json.RawMessage `json:"support" gorm:"type:text"`

	// Verification and badges
	Verified      bool            `json:"verified"`
	Badges        json.RawMessage `json:"badges" gorm:"type:text"`
	SecurityScore int             `json:"security_score"`
	QualityScore  int             `json:"quality_score"`

	// Stats
	Rating       float64 `json:"rating"`
	ReviewCount  int     `json:"review_count"`
	InstallCount int     `json:"install_count"`

	// GitHub metadata
	GitHubOrg       string     `json:"github_org"`
	GitHubRepo      string     `json:"github_repo"`
	GitHubStars     int        `json:"github_stars"`
	GitHubUpdatedAt *time.Time `json:"github_updated_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name
func (MarketplaceAgent) TableName() string {
	return "marketplace_agents"
}

// BeforeCreate hook
func (a *MarketplaceAgent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = generateMarketplaceID("agent")
	}
	return nil
}

// ToAgentPackage converts database model to AgentPackage
func (a *MarketplaceAgent) ToAgentPackage() (*AgentPackage, error) {
	pkg := &AgentPackage{
		ID:           a.ID,
		Name:         a.Name,
		Version:      a.Version,
		Author:       a.Author,
		Description:  a.Description,
		Icon:         a.Icon,
		Homepage:     a.Homepage,
		Repository:   a.Repository,
		License:      a.License,
		Verified:     a.Verified,
		Rating:       a.Rating,
		ReviewCount:  a.ReviewCount,
		InstallCount: a.InstallCount,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}

	if len(a.Tags) > 0 {
		json.Unmarshal(a.Tags, &pkg.Tags)
	}
	if len(a.Badges) > 0 {
		json.Unmarshal(a.Badges, &pkg.Badges)
	}
	if len(a.Requirements) > 0 {
		json.Unmarshal(a.Requirements, &pkg.Requirements)
	}
	if len(a.Knowledge) > 0 {
		json.Unmarshal(a.Knowledge, &pkg.Knowledge)
	}
	if len(a.Skills) > 0 {
		json.Unmarshal(a.Skills, &pkg.Skills)
	}
	if len(a.Chains) > 0 {
		json.Unmarshal(a.Chains, &pkg.Chains)
	}
	if len(a.MCPServers) > 0 {
		json.Unmarshal(a.MCPServers, &pkg.MCPServers)
	}
	if len(a.Persona) > 0 {
		json.Unmarshal(a.Persona, &pkg.Persona)
	}
	if len(a.Pricing) > 0 {
		json.Unmarshal(a.Pricing, &pkg.Pricing)
	}
	if len(a.Support) > 0 {
		json.Unmarshal(a.Support, &pkg.Support)
	}

	return pkg, nil
}

// FromAgentPackage converts AgentPackage to database model
func FromAgentPackage(pkg *AgentPackage) *MarketplaceAgent {
	a := &MarketplaceAgent{
		ID:           pkg.ID,
		Name:         pkg.Name,
		Version:      pkg.Version,
		Author:       pkg.Author,
		Description:  pkg.Description,
		Icon:         pkg.Icon,
		Homepage:     pkg.Homepage,
		Repository:   pkg.Repository,
		License:      pkg.License,
		Verified:     pkg.Verified,
		Rating:       pkg.Rating,
		ReviewCount:  pkg.ReviewCount,
		InstallCount: pkg.InstallCount,
	}

	if len(pkg.Tags) > 0 {
		a.Tags, _ = json.Marshal(pkg.Tags)
	}
	if len(pkg.Badges) > 0 {
		a.Badges, _ = json.Marshal(pkg.Badges)
	}

	a.Requirements, _ = json.Marshal(pkg.Requirements)
	a.Knowledge, _ = json.Marshal(pkg.Knowledge)
	a.Skills, _ = json.Marshal(pkg.Skills)
	if len(pkg.Chains) > 0 {
		a.Chains, _ = json.Marshal(pkg.Chains)
	}
	if len(pkg.MCPServers) > 0 {
		a.MCPServers, _ = json.Marshal(pkg.MCPServers)
	}
	a.Persona, _ = json.Marshal(pkg.Persona)
	a.Pricing, _ = json.Marshal(pkg.Pricing)
	a.Support, _ = json.Marshal(pkg.Support)

	return a
}

// InstalledAgent represents an installed agent instance
type InstalledAgent struct {
	ID      string `gorm:"primaryKey" json:"id"`
	AgentID string `gorm:"index:idx_agent_user,unique" json:"agent_id"`
	UserID  string `gorm:"index:idx_agent_user" json:"user_id"`

	// Installation metadata
	Version     string `json:"version"`
	InstallPath string `json:"install_path"`
	ConfigPath  string `json:"config_path"`

	// Status
	IsActive  bool `json:"is_active"`
	IsEnabled bool `json:"is_enabled"`

	// Configuration overrides
	ConfigOverrides json.RawMessage `json:"config_overrides" gorm:"type:text"`

	// Timestamps
	InstalledAt time.Time  `json:"installed_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`

	// Relations
	Agent *MarketplaceAgent `json:"agent,omitempty" gorm:"foreignKey:AgentID"`
}

// TableName specifies the table name
func (InstalledAgent) TableName() string {
	return "installed_agents"
}

// BeforeCreate hook
func (i *InstalledAgent) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = generateMarketplaceID("installed")
	}
	if i.UserID == "" {
		i.UserID = "default"
	}
	return nil
}

// AgentReview represents a user review for an agent
type AgentReview struct {
	ID      string `gorm:"primaryKey" json:"id"`
	AgentID string `gorm:"index:idx_agent_review" json:"agent_id"`
	UserID  string `gorm:"index" json:"user_id"`

	// Review content
	Rating  int    `json:"rating" gorm:"check:rating >= 1 AND rating <= 5"`
	Title   string `json:"title"`
	Content string `json:"content" gorm:"type:text"`

	// Review metadata
	Version  string `json:"version"`  // Version being reviewed
	Verified bool   `json:"verified"` // User actually installed this version
	Helpful  int    `json:"helpful"`  // Number of users who found this helpful

	// Status
	IsPublished bool `json:"is_published"`
	IsDeleted   bool `json:"is_deleted"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	Agent *MarketplaceAgent `json:"agent,omitempty" gorm:"foreignKey:AgentID"`
}

// TableName specifies the table name
func (AgentReview) TableName() string {
	return "agent_reviews"
}

// BeforeCreate hook
func (r *AgentReview) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = generateMarketplaceID("review")
	}
	if r.UserID == "" {
		r.UserID = "default"
	}
	return nil
}

// AgentVersion tracks available versions
type AgentVersion struct {
	ID      string `gorm:"primaryKey" json:"id"`
	AgentID string `gorm:"index:idx_agent_version,unique" json:"agent_id"`
	Version string `json:"version"`

	// Release info
	ReleaseNotes string `json:"release_notes" gorm:"type:text"`
	DownloadURL  string `json:"download_url"`
	Checksum     string `json:"checksum"`
	SizeBytes    int64  `json:"size_bytes"`

	// Status
	IsStable     bool `json:"is_stable"`
	IsPrerelease bool `json:"is_prerelease"`
	Deprecated   bool `json:"deprecated"`

	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the table name
func (AgentVersion) TableName() string {
	return "agent_versions"
}

// BeforeCreate hook
func (v *AgentVersion) BeforeCreate(tx *gorm.DB) error {
	if v.ID == "" {
		v.ID = generateMarketplaceID("version")
	}
	return nil
}

// AgentCategory represents agent categories/tags for searching
type AgentCategory struct {
	ID          string `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"uniqueIndex" json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	SortOrder   int    `json:"sort_order"`

	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the table name
func (AgentCategory) TableName() string {
	return "agent_categories"
}

// generateMarketplaceID generates a unique ID for marketplace entities
func generateMarketplaceID(prefix string) string {
	return prefix + "_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a random string
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}

// MigrationOrder returns the order in which tables should be migrated
func MigrationOrder() []interface{} {
	return []interface{}{
		&MarketplaceAgent{},
		&InstalledAgent{},
		&AgentReview{},
		&AgentVersion{},
		&AgentCategory{},
	}
}
