package marketplace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AgentPackage represents a complete agent package from AGENT.yaml
type AgentPackage struct {
	ID          string   `yaml:"id,omitempty" json:"id"`
	Name        string   `yaml:"name" json:"name" validate:"required"`
	Version     string   `yaml:"version" json:"version" validate:"required,semver"`
	Author      string   `yaml:"author" json:"author" validate:"required"`
	Description string   `yaml:"description" json:"description" validate:"required,max=500"`
	Tags        []string `yaml:"tags" json:"tags"`
	Icon        string   `yaml:"icon" json:"icon"`
	Homepage    string   `yaml:"homepage,omitempty" json:"homepage"`
	Repository  string   `yaml:"repository,omitempty" json:"repository"`
	License     string   `yaml:"license" json:"license" validate:"required"`

	Requirements AgentRequirements `yaml:"requirements" json:"requirements"`
	Knowledge    AgentKnowledge    `yaml:"knowledge" json:"knowledge"`
	Skills       AgentSkills       `yaml:"skills" json:"skills"`
	Chains       []AgentChain      `yaml:"chains,omitempty" json:"chains"`
	MCPServers   []MCPServer       `yaml:"mcp_servers,omitempty" json:"mcp_servers"`
	Persona      AgentPersona      `yaml:"persona" json:"persona"`
	Pricing      AgentPricing      `yaml:"pricing" json:"pricing"`
	Support      AgentSupport      `yaml:"support" json:"support"`
	Stats        AgentStats        `yaml:"stats,omitempty" json:"stats"`

	// Internal fields (not in YAML)
	Verified     bool      `yaml:"-" json:"verified"`
	Badges       []string  `yaml:"-" json:"badges"`
	Rating       float64   `yaml:"-" json:"rating"`
	ReviewCount  int       `yaml:"-" json:"review_count"`
	InstallCount int       `yaml:"-" json:"install_count"`
	CreatedAt    time.Time `yaml:"-" json:"created_at"`
	UpdatedAt    time.Time `yaml:"-" json:"updated_at"`

	// Installation metadata
	InstallPath string `yaml:"-" json:"-"`
	IsActive    bool   `yaml:"-" json:"is_active"`
}

// AgentRequirements defines system requirements
type AgentRequirements struct {
	MinMyraiVersion string            `yaml:"min_myrai_version" json:"min_myrai_version" validate:"required,semver"`
	Memory          string            `yaml:"memory" json:"memory"`
	CPU             string            `yaml:"cpu,omitempty" json:"cpu"`
	Disk            string            `yaml:"disk,omitempty" json:"disk"`
	Dependencies    []string          `yaml:"dependencies,omitempty" json:"dependencies"`
	EnvVars         map[string]string `yaml:"env_vars,omitempty" json:"env_vars"`
}

// AgentKnowledge defines knowledge cluster configuration
type AgentKnowledge struct {
	NeuralClusters []NeuralCluster `yaml:"neural_clusters" json:"neural_clusters"`
}

// NeuralCluster represents a knowledge cluster
type NeuralCluster struct {
	Name        string   `yaml:"name" json:"name" validate:"required"`
	Description string   `yaml:"description" json:"description"`
	Documents   []string `yaml:"documents" json:"documents"`
	Sources     []string `yaml:"sources,omitempty" json:"sources"`
}

// AgentSkills defines skill configuration
type AgentSkills struct {
	Builtin  []BuiltinSkill  `yaml:"builtin" json:"builtin"`
	External []ExternalSkill `yaml:"external,omitempty" json:"external"`
}

// BuiltinSkill represents a built-in skill to enable
type BuiltinSkill struct {
	Name    string                 `yaml:"name" json:"name" validate:"required"`
	Config  map[string]interface{} `yaml:"config,omitempty" json:"config"`
	Enabled bool                   `yaml:"enabled" json:"enabled"`
}

// ExternalSkill represents an external skill dependency
type ExternalSkill struct {
	Name     string `yaml:"name" json:"name" validate:"required"`
	Version  string `yaml:"version" json:"version" validate:"semver"`
	Source   string `yaml:"source" json:"source"`
	Required bool   `yaml:"required" json:"required"`
}

// AgentChain represents a predefined chain/workflow
type AgentChain struct {
	Name        string   `yaml:"name" json:"name" validate:"required"`
	Description string   `yaml:"description" json:"description"`
	Steps       []string `yaml:"steps" json:"steps" validate:"required,min=1"`
	Triggers    []string `yaml:"triggers,omitempty" json:"triggers"`
}

// MCPServer represents an MCP server configuration
type MCPServer struct {
	Name        string            `yaml:"name" json:"name" validate:"required"`
	Command     string            `yaml:"command" json:"command" validate:"required"`
	Args        []string          `yaml:"args,omitempty" json:"args"`
	Env         map[string]string `yaml:"env,omitempty" json:"env"`
	Disabled    bool              `yaml:"disabled" json:"disabled"`
	AutoInstall bool              `yaml:"auto_install" json:"auto_install"`
}

// AgentPersona defines the agent's persona configuration
type AgentPersona struct {
	Name         string   `yaml:"name" json:"name"`
	Personality  string   `yaml:"personality" json:"personality"`
	Voice        string   `yaml:"voice" json:"voice"`
	Values       []string `yaml:"values,omitempty" json:"values"`
	Expertise    []string `yaml:"expertise,omitempty" json:"expertise"`
	SystemPrompt string   `yaml:"system_prompt,omitempty" json:"system_prompt"`
}

// AgentPricing defines pricing configuration
type AgentPricing struct {
	Model     string   `yaml:"model" json:"model" validate:"oneof=free paid subscription"`
	Price     float64  `yaml:"price,omitempty" json:"price"`
	Currency  string   `yaml:"currency,omitempty" json:"currency"`
	TrialDays int      `yaml:"trial_days,omitempty" json:"trial_days"`
	Features  []string `yaml:"features,omitempty" json:"features"`
}

// AgentSupport defines support information
type AgentSupport struct {
	Email         string `yaml:"email,omitempty" json:"email"`
	URL           string `yaml:"url,omitempty" json:"url"`
	Issues        string `yaml:"issues,omitempty" json:"issues"`
	Documentation string `yaml:"documentation,omitempty" json:"documentation"`
}

// AgentStats contains marketplace statistics
type AgentStats struct {
	InstallCount int       `yaml:"install_count,omitempty" json:"install_count"`
	Rating       float64   `yaml:"rating,omitempty" json:"rating"`
	ReviewCount  int       `yaml:"review_count,omitempty" json:"review_count"`
	LastUpdated  time.Time `yaml:"last_updated,omitempty" json:"last_updated"`
}

// ParseAgentYAML parses an AGENT.yaml file into an AgentPackage
func ParseAgentYAML(r io.Reader) (*AgentPackage, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read AGENT.yaml: %w", err)
	}

	var pkg AgentPackage
	if err := yaml.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse AGENT.yaml: %w", err)
	}

	// Set defaults
	if pkg.Pricing.Model == "" {
		pkg.Pricing.Model = "free"
	}
	if pkg.License == "" {
		pkg.License = "MIT"
	}

	return &pkg, nil
}

// ParseAgentYAMLFile parses an AGENT.yaml file from disk
func ParseAgentYAMLFile(path string) (*AgentPackage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	return ParseAgentYAML(file)
}

// Validate validates the agent package
func (p *AgentPackage) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if p.Version == "" {
		return fmt.Errorf("agent version is required")
	}
	if !IsValidSemver(p.Version) {
		return fmt.Errorf("invalid semantic version: %s", p.Version)
	}
	if p.Author == "" {
		return fmt.Errorf("agent author is required")
	}
	if p.Description == "" {
		return fmt.Errorf("agent description is required")
	}
	if p.Requirements.MinMyraiVersion == "" {
		return fmt.Errorf("minimum myrai version is required")
	}
	if !IsValidSemver(p.Requirements.MinMyraiVersion) {
		return fmt.Errorf("invalid minimum myrai version: %s", p.Requirements.MinMyraiVersion)
	}

	// Validate skills
	for i, skill := range p.Skills.Builtin {
		if skill.Name == "" {
			return fmt.Errorf("builtin skill at index %d is missing name", i)
		}
	}

	for i, skill := range p.Skills.External {
		if skill.Name == "" {
			return fmt.Errorf("external skill at index %d is missing name", i)
		}
	}

	// Validate MCP servers
	for i, server := range p.MCPServers {
		if server.Name == "" {
			return fmt.Errorf("MCP server at index %d is missing name", i)
		}
		if server.Command == "" {
			return fmt.Errorf("MCP server '%s' is missing command", server.Name)
		}
	}

	return nil
}

// Save writes the agent package to an AGENT.yaml file
func (p *AgentPackage) Save(path string) error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal agent package: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write AGENT.yaml: %w", err)
	}

	return nil
}

// GetID returns a unique identifier for the agent
func (p *AgentPackage) GetID() string {
	if p.ID != "" {
		return p.ID
	}
	return fmt.Sprintf("%s@%s", p.Name, p.Version)
}

// GetFullName returns the agent name with version
func (p *AgentPackage) GetFullName() string {
	return fmt.Sprintf("%s@%s", p.Name, p.Version)
}

// MatchesVersion checks if the agent matches a version constraint
func (p *AgentPackage) MatchesVersion(constraint string) bool {
	return MatchesVersion(p.Version, constraint)
}

// IsFree returns true if the agent is free to use
func (p *AgentPackage) IsFree() bool {
	return p.Pricing.Model == "free" || p.Pricing.Price == 0
}

// GetRequiredSkills returns all required skill names
func (p *AgentPackage) GetRequiredSkills() []string {
	var skills []string
	for _, s := range p.Skills.Builtin {
		if s.Enabled {
			skills = append(skills, s.Name)
		}
	}
	for _, s := range p.Skills.External {
		if s.Required {
			skills = append(skills, s.Name)
		}
	}
	return skills
}

// GetInstallDir returns the installation directory name
func (p *AgentPackage) GetInstallDir() string {
	return fmt.Sprintf("%s-%s", sanitizeName(p.Name), p.Version)
}

// sanitizeName sanitizes a name for use in filesystem paths
func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

// IsValidSemver checks if a string is a valid semantic version
func IsValidSemver(version string) bool {
	// Simple semver validation (major.minor.patch[-prerelease][+build])
	parts := strings.Split(version, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}

	for _, part := range parts {
		// Handle pre-release and build metadata
		if idx := strings.IndexAny(part, "-+"); idx != -1 {
			part = part[:idx]
		}
		if part == "" {
			return false
		}
		// Check if numeric
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}

	return true
}

// MatchesVersion checks if a version matches a constraint
// Supports: exact version, "latest", or partial version prefix
func MatchesVersion(version, constraint string) bool {
	if constraint == "latest" {
		return true
	}
	if constraint == version {
		return true
	}
	if strings.HasPrefix(version, constraint) {
		return true
	}
	return false
}

// AgentPackageBundle represents the complete agent package including files
type AgentPackageBundle struct {
	Package  *AgentPackage
	RootPath string
	Files    map[string][]byte
}

// LoadFromDirectory loads an agent package from a directory
func LoadFromDirectory(path string) (*AgentPackageBundle, error) {
	yamlPath := filepath.Join(path, "AGENT.yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		yamlPath = filepath.Join(path, "agent.yaml")
		if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("AGENT.yaml not found in %s", path)
		}
	}

	pkg, err := ParseAgentYAMLFile(yamlPath)
	if err != nil {
		return nil, err
	}

	bundle := &AgentPackageBundle{
		Package:  pkg,
		RootPath: path,
		Files:    make(map[string][]byte),
	}

	return bundle, nil
}

// GetFile retrieves a file from the bundle
func (b *AgentPackageBundle) GetFile(relPath string) ([]byte, error) {
	if data, ok := b.Files[relPath]; ok {
		return data, nil
	}

	fullPath := filepath.Join(b.RootPath, relPath)
	return os.ReadFile(fullPath)
}

// ListFiles lists all files in the bundle directory
func (b *AgentPackageBundle) ListFiles() ([]string, error) {
	var files []string
	err := filepath.Walk(b.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, err := filepath.Rel(b.RootPath, path)
			if err != nil {
				return err
			}
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}
