// Package runtime provides the skill runtime engine for loading and executing skills
package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gmsas95/myrai-cli/internal/skills"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// SkillManifest represents the frontmatter in SKILL.md
type SkillManifest struct {
	Name         string                 `yaml:"name"`
	Version      string                 `yaml:"version"`
	Description  string                 `yaml:"description"`
	Author       string                 `yaml:"author"`
	Tags         []string               `yaml:"tags,omitempty"`
	Dependencies []string               `yaml:"dependencies,omitempty"`
	Tools        []ToolDefinition       `yaml:"tools,omitempty"`
	Config       map[string]interface{} `yaml:"config,omitempty"`
	EntryPoint   string                 `yaml:"entry_point,omitempty"`
	Sandbox      SandboxConfig          `yaml:"sandbox,omitempty"`
	CreatedAt    time.Time              `yaml:"created_at,omitempty"`
	UpdatedAt    time.Time              `yaml:"updated_at,omitempty"`
}

// ToolDefinition defines a tool provided by a skill
type ToolDefinition struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Parameters  map[string]interface{} `yaml:"parameters"`
}

// SandboxConfig defines sandbox settings for a skill
type SandboxConfig struct {
	Enabled     bool     `yaml:"enabled"`
	AllowFS     bool     `yaml:"allow_fs,omitempty"`
	AllowNet    bool     `yaml:"allow_net,omitempty"`
	AllowExec   bool     `yaml:"allow_exec,omitempty"`
	AllowedDirs []string `yaml:"allowed_dirs,omitempty"`
}

// Skill represents a loaded skill with its manifest and runtime state
type Skill struct {
	Manifest    SkillManifest
	Path        string
	Content     string
	RawManifest string
	Status      SkillStatus
	LoadedAt    time.Time
	LastError   error
}

// SkillStatus represents the current status of a skill
type SkillStatus int

const (
	SkillStatusUnknown SkillStatus = iota
	SkillStatusLoading
	SkillStatusLoaded
	SkillStatusValidating
	SkillStatusValidated
	SkillStatusError
	SkillStatusDisabled
)

func (s SkillStatus) String() string {
	switch s {
	case SkillStatusLoading:
		return "loading"
	case SkillStatusLoaded:
		return "loaded"
	case SkillStatusValidating:
		return "validating"
	case SkillStatusValidated:
		return "validated"
	case SkillStatusError:
		return "error"
	case SkillStatusDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}

// Loader manages loading skills from the filesystem
type Loader struct {
	skills map[string]*Skill
	paths  []string
	logger *zap.Logger
	mu     sync.RWMutex
}

// NewLoader creates a new skill loader
func NewLoader(logger *zap.Logger) *Loader {
	return &Loader{
		skills: make(map[string]*Skill),
		paths:  []string{},
		logger: logger,
	}
}

// AddPath adds a search path for skills
func (l *Loader) AddPath(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.paths = append(l.paths, path)
}

// LoadSkill loads a skill from a manifest file path
func (l *Loader) LoadSkill(manifestPath string) (*Skill, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if already loaded
	if skill, exists := l.skills[manifestPath]; exists {
		return skill, nil
	}

	skill := &Skill{
		Path:     manifestPath,
		Status:   SkillStatusLoading,
		LoadedAt: time.Now(),
	}

	l.logger.Info("Loading skill", zap.String("path", manifestPath))

	// Read the SKILL.md file
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		skill.Status = SkillStatusError
		skill.LastError = fmt.Errorf("failed to read skill file: %w", err)
		return nil, skill.LastError
	}

	skill.Content = string(content)

	// Parse frontmatter
	manifest, rawManifest, err := parseFrontmatter(skill.Content)
	if err != nil {
		skill.Status = SkillStatusError
		skill.LastError = fmt.Errorf("failed to parse frontmatter: %w", err)
		return nil, skill.LastError
	}

	skill.Manifest = *manifest
	skill.RawManifest = rawManifest
	skill.Status = SkillStatusLoaded

	// Store in cache
	l.skills[manifestPath] = skill
	l.skills[manifest.Name] = skill

	l.logger.Info("Skill loaded successfully",
		zap.String("name", manifest.Name),
		zap.String("version", manifest.Version),
	)

	return skill, nil
}

// LoadSkillFromDir loads a skill from a directory containing SKILL.md
func (l *Loader) LoadSkillFromDir(dirPath string) (*Skill, error) {
	manifestPath := filepath.Join(dirPath, "SKILL.md")
	return l.LoadSkill(manifestPath)
}

// parseFrontmatter extracts YAML frontmatter from SKILL.md content
func parseFrontmatter(content string) (*SkillManifest, string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return nil, "", fmt.Errorf("empty skill file")
	}

	// Check for frontmatter delimiter
	if !strings.HasPrefix(lines[0], "---") {
		return nil, "", fmt.Errorf("no frontmatter found")
	}

	// Find end of frontmatter
	var endIdx int
	for i := 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "---") {
			endIdx = i
			break
		}
	}

	if endIdx == 0 {
		return nil, "", fmt.Errorf("unclosed frontmatter")
	}

	// Extract YAML content
	yamlContent := strings.Join(lines[1:endIdx], "\n")

	var manifest SkillManifest
	if err := yaml.Unmarshal([]byte(yamlContent), &manifest); err != nil {
		return nil, "", fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	return &manifest, yamlContent, nil
}

// ValidateManifest validates a skill manifest
func ValidateManifest(skill *Skill) error {
	if skill == nil {
		return fmt.Errorf("skill is nil")
	}

	skill.Status = SkillStatusValidating

	// Required fields
	if strings.TrimSpace(skill.Manifest.Name) == "" {
		skill.Status = SkillStatusError
		skill.LastError = fmt.Errorf("skill name is required")
		return skill.LastError
	}

	if strings.TrimSpace(skill.Manifest.Version) == "" {
		skill.Status = SkillStatusError
		skill.LastError = fmt.Errorf("skill version is required")
		return skill.LastError
	}

	if strings.TrimSpace(skill.Manifest.Description) == "" {
		skill.Status = SkillStatusError
		skill.LastError = fmt.Errorf("skill description is required")
		return skill.LastError
	}

	// Validate version format (semantic versioning)
	if !isValidVersion(skill.Manifest.Version) {
		skill.Status = SkillStatusError
		skill.LastError = fmt.Errorf("invalid version format: %s", skill.Manifest.Version)
		return skill.LastError
	}

	// Validate tool definitions
	for _, tool := range skill.Manifest.Tools {
		if strings.TrimSpace(tool.Name) == "" {
			skill.Status = SkillStatusError
			skill.LastError = fmt.Errorf("tool name cannot be empty")
			return skill.LastError
		}
	}

	skill.Status = SkillStatusValidated
	return nil
}

// isValidVersion checks if a version string follows semantic versioning
func isValidVersion(version string) bool {
	// Basic semver check: x.y.z
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}

	for _, part := range parts {
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

// GetSkill retrieves a loaded skill by name or path
func (l *Loader) GetSkill(name string) *Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.skills[name]
}

// ListSkills returns all loaded skills
func (l *Loader) ListSkills() []*Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()

	skills := make([]*Skill, 0, len(l.skills))
	seen := make(map[string]bool)

	for _, skill := range l.skills {
		if !seen[skill.Manifest.Name] {
			skills = append(skills, skill)
			seen[skill.Manifest.Name] = true
		}
	}

	return skills
}

// ListSkillsByStatus returns skills filtered by status
func (l *Loader) ListSkillsByStatus(status SkillStatus) []*Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()

	skills := make([]*Skill, 0)
	seen := make(map[string]bool)

	for _, skill := range l.skills {
		if skill.Status == status && !seen[skill.Manifest.Name] {
			skills = append(skills, skill)
			seen[skill.Manifest.Name] = true
		}
	}

	return skills
}

// UnloadSkill removes a skill from the loader
func (l *Loader) UnloadSkill(name string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if skill, exists := l.skills[name]; exists {
		delete(l.skills, name)
		delete(l.skills, skill.Path)
		return true
	}
	return false
}

// ScanDirectory scans a directory for SKILL.md files and loads them
func (l *Loader) ScanDirectory(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.ToLower(info.Name()) == "skill.md" {
			_, err := l.LoadSkill(path)
			if err != nil {
				l.logger.Warn("Failed to load skill",
					zap.String("path", path),
					zap.Error(err),
				)
			}
		}

		return nil
	})
}

// ScanDirectories scans all configured paths for skills
func (l *Loader) ScanDirectories() error {
	l.mu.RLock()
	paths := make([]string, len(l.paths))
	copy(paths, l.paths)
	l.mu.RUnlock()

	for _, path := range paths {
		if err := l.ScanDirectory(path); err != nil {
			l.logger.Error("Failed to scan directory",
				zap.String("path", path),
				zap.Error(err),
			)
		}
	}

	return nil
}

// ParseSkillContent parses SKILL.md content without loading from file
func ParseSkillContent(content string) (*Skill, error) {
	manifest, rawManifest, err := parseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	skill := &Skill{
		Manifest:    *manifest,
		Content:     content,
		RawManifest: rawManifest,
		Status:      SkillStatusLoaded,
		LoadedAt:    time.Now(),
	}

	return skill, nil
}

// ToRegistrySkill converts a runtime.Skill to a skills.Skill for registry compatibility
func (s *Skill) ToRegistrySkill(handler skills.ToolHandler) skills.Skill {
	tools := make([]skills.Tool, len(s.Manifest.Tools))
	for i, tool := range s.Manifest.Tools {
		tools[i] = skills.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
			Handler:     handler,
		}
	}

	return &runtimeSkill{
		name:        s.Manifest.Name,
		description: s.Manifest.Description,
		version:     s.Manifest.Version,
		tools:       tools,
		enabled:     s.Status == SkillStatusValidated,
	}
}

// runtimeSkill implements skills.Skill interface
type runtimeSkill struct {
	name        string
	description string
	version     string
	tools       []skills.Tool
	enabled     bool
}

func (r *runtimeSkill) Name() string         { return r.name }
func (r *runtimeSkill) Description() string  { return r.description }
func (r *runtimeSkill) Version() string      { return r.version }
func (r *runtimeSkill) Tools() []skills.Tool { return r.tools }
func (r *runtimeSkill) IsEnabled() bool      { return r.enabled }
func (r *runtimeSkill) Enable() error        { r.enabled = true; return nil }
func (r *runtimeSkill) Disable() error       { r.enabled = false; return nil }
