// Package runtime provides skill loading and hot-reload functionality
package runtime

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gmsas95/myrai-cli/internal/errors"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"gopkg.in/yaml.v3"
)

// SkillLoader handles loading skills from various sources and hot-reloading
type SkillLoader struct {
	registry *skills.Registry
	watchers map[string]*fsnotify.Watcher
	mu       sync.RWMutex
	loaded   map[string]*LoadedSkill
	stopCh   chan struct{}
}

// LoadedSkill represents a skill loaded into the runtime
type LoadedSkill struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Source      string          `json:"source"`
	SourceType  SkillSourceType `json:"source_type"`
	Path        string          `json:"path,omitempty"`
	Tools       []skills.Tool   `json:"tools"`
	Manifest    *SkillManifest  `json:"manifest,omitempty"`
	LoadedAt    string          `json:"loaded_at"`
	LastUpdated string          `json:"last_updated"`
}

// SkillSourceType represents the source of a skill
type SkillSourceType string

const (
	SourceGitHub  SkillSourceType = "github"
	SourceLocal   SkillSourceType = "local"
	SourceBuiltIn SkillSourceType = "builtin"
)

// SkillManifest represents the SKILL.md manifest structure
type SkillManifest struct {
	Name            string         `yaml:"name" json:"name"`
	Version         string         `yaml:"version" json:"version"`
	Description     string         `yaml:"description" json:"description"`
	Author          string         `yaml:"author" json:"author"`
	Tags            []string       `yaml:"tags" json:"tags"`
	MinMyraiVersion string         `yaml:"min_myrai_version" json:"min_myrai_version"`
	MCP             *MCPConfig     `yaml:"mcp,omitempty" json:"mcp,omitempty"`
	Tools           []ManifestTool `yaml:"tools" json:"tools"`
}

// MCPConfig represents MCP server requirements
type MCPConfig struct {
	Server   string `yaml:"server" json:"server"`
	Required bool   `yaml:"required" json:"required"`
}

// ManifestTool represents a tool definition in the manifest
type ManifestTool struct {
	Name        string              `yaml:"name" json:"name"`
	Description string              `yaml:"description" json:"description"`
	Parameters  []ManifestParameter `yaml:"parameters" json:"parameters"`
}

// ManifestParameter represents a parameter definition
type ManifestParameter struct {
	Name        string      `yaml:"name" json:"name"`
	Type        string      `yaml:"type" json:"type"`
	Required    bool        `yaml:"required" json:"required"`
	Default     interface{} `yaml:"default,omitempty" json:"default,omitempty"`
	Description string      `yaml:"description" json:"description"`
	Enum        []string    `yaml:"enum,omitempty" json:"enum,omitempty"`
}

// NewSkillLoader creates a new skill loader
func NewSkillLoader(registry *skills.Registry) *SkillLoader {
	return &SkillLoader{
		registry: registry,
		watchers: make(map[string]*fsnotify.Watcher),
		loaded:   make(map[string]*LoadedSkill),
		stopCh:   make(chan struct{}),
	}
}

// LoadFromGitHub loads a skill from a GitHub repository
func (sl *SkillLoader) LoadFromGitHub(repo string) (*LoadedSkill, error) {
	// Parse repository string (e.g., "github.com/user/repo" or "user/repo")
	repo = strings.TrimPrefix(repo, "github.com/")
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return nil, errors.New("SKILL_001", fmt.Sprintf("invalid GitHub repository format: %s", repo))
	}

	owner, repoName := parts[0], parts[1]

	// Construct raw URL for SKILL.md
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/SKILL.md", owner, repoName)

	// Download manifest
	manifest, err := sl.downloadManifest(rawURL)
	if err != nil {
		// Try master branch as fallback
		rawURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/SKILL.md", owner, repoName)
		manifest, err = sl.downloadManifest(rawURL)
		if err != nil {
			return nil, errors.Wrap(err, "SKILL_002", fmt.Sprintf("failed to load skill from GitHub: %s", repo))
		}
	}

	skill := &LoadedSkill{
		Name:        manifest.Name,
		Version:     manifest.Version,
		Description: manifest.Description,
		Source:      repo,
		SourceType:  SourceGitHub,
		Manifest:    manifest,
		Tools:       sl.convertManifestTools(manifest),
	}

	if err := sl.registerSkill(skill); err != nil {
		return nil, err
	}

	return skill, nil
}

// LoadFromLocal loads a skill from a local directory
func (sl *SkillLoader) LoadFromLocal(path string) (*LoadedSkill, error) {
	// Validate path exists
	info, err := os.Stat(path)
	if err != nil {
		return nil, errors.Wrap(err, "SKILL_003", fmt.Sprintf("skill path not found: %s", path))
	}

	var manifestPath string
	if info.IsDir() {
		manifestPath = filepath.Join(path, "SKILL.md")
	} else {
		manifestPath = path
		path = filepath.Dir(path)
	}

	// Read manifest
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, errors.Wrap(err, "SKILL_004", fmt.Sprintf("failed to read SKILL.md: %s", manifestPath))
	}

	manifest, err := sl.parseManifest(data)
	if err != nil {
		return nil, errors.Wrap(err, "SKILL_005", "failed to parse skill manifest")
	}

	skill := &LoadedSkill{
		Name:        manifest.Name,
		Version:     manifest.Version,
		Description: manifest.Description,
		Source:      path,
		SourceType:  SourceLocal,
		Path:        path,
		Manifest:    manifest,
		Tools:       sl.convertManifestTools(manifest),
	}

	if err := sl.registerSkill(skill); err != nil {
		return nil, err
	}

	return skill, nil
}

// Watch starts watching a directory for skill changes
func (sl *SkillLoader) Watch(path string) error {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// Check if already watching this path
	if _, exists := sl.watchers[path]; exists {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Wrap(err, "SKILL_006", "failed to create file watcher")
	}

	// Add the directory and all subdirectories
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(p)
		}
		return nil
	})

	if err != nil {
		watcher.Close()
		return errors.Wrap(err, "SKILL_007", "failed to watch directory")
	}

	sl.watchers[path] = watcher

	// Start watching in a goroutine
	go sl.watchLoop(watcher, path)

	return nil
}

// Stop stops all watchers and cleans up
func (sl *SkillLoader) Stop() {
	close(sl.stopCh)

	sl.mu.Lock()
	defer sl.mu.Unlock()

	for path, watcher := range sl.watchers {
		watcher.Close()
		delete(sl.watchers, path)
	}
}

// HotReload reloads a specific skill by name
func (sl *SkillLoader) HotReload(skillName string) error {
	sl.mu.RLock()
	skill, exists := sl.loaded[skillName]
	sl.mu.RUnlock()

	if !exists {
		return errors.New("SKILL_008", fmt.Sprintf("skill not found: %s", skillName))
	}

	switch skill.SourceType {
	case SourceLocal:
		_, err := sl.LoadFromLocal(skill.Path)
		return err
	case SourceGitHub:
		_, err := sl.LoadFromGitHub(skill.Source)
		return err
	default:
		return errors.New("SKILL_009", fmt.Sprintf("cannot hot-reload skill from source: %s", skill.SourceType))
	}
}

// ListLoaded returns all loaded skills
func (sl *SkillLoader) ListLoaded() []*LoadedSkill {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	skills := make([]*LoadedSkill, 0, len(sl.loaded))
	for _, skill := range sl.loaded {
		skills = append(skills, skill)
	}
	return skills
}

// GetSkill returns a loaded skill by name
func (sl *SkillLoader) GetSkill(name string) (*LoadedSkill, bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	skill, ok := sl.loaded[name]
	return skill, ok
}

// ============ Private Methods ============

func (sl *SkillLoader) watchLoop(watcher *fsnotify.Watcher, basePath string) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			sl.handleEvent(event)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			// Log error
			_ = err

		case <-sl.stopCh:
			return
		}
	}
}

func (sl *SkillLoader) handleEvent(event fsnotify.Event) {
	// Only care about SKILL.md files
	if !strings.HasSuffix(event.Name, "SKILL.md") {
		return
	}

	switch event.Op {
	case fsnotify.Write:
		// Reload the skill
		sl.mu.RLock()
		for name, skill := range sl.loaded {
			if skill.Path == filepath.Dir(event.Name) || skill.Path == event.Name {
				sl.mu.RUnlock()
				sl.HotReload(name)
				return
			}
		}
		sl.mu.RUnlock()

	case fsnotify.Create:
		// Load new skill
		sl.LoadFromLocal(event.Name)

	case fsnotify.Remove:
		// Unregister skill
		sl.mu.Lock()
		for name, skill := range sl.loaded {
			if skill.Path == filepath.Dir(event.Name) || skill.Path == event.Name {
				delete(sl.loaded, name)
				break
			}
		}
		sl.mu.Unlock()
	}
}

func (sl *SkillLoader) registerSkill(skill *LoadedSkill) error {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// Unregister existing skill if present
	if existing, ok := sl.loaded[skill.Name]; ok {
		// TODO: Unregister from skills.Registry
		_ = existing
	}

	// Register with skills registry
	skillImpl := &RuntimeSkill{
		BaseSkill: skills.NewBaseSkill(skill.Name, skill.Description, skill.Version),
		tools:     skill.Tools,
	}

	for _, tool := range skill.Tools {
		skillImpl.AddTool(tool)
	}

	if err := sl.registry.Register(skillImpl); err != nil {
		return errors.Wrap(err, "SKILL_010", fmt.Sprintf("failed to register skill: %s", skill.Name))
	}

	sl.loaded[skill.Name] = skill
	return nil
}

func (sl *SkillLoader) downloadManifest(url string) (*SkillManifest, error) {
	// Simple HTTP GET (in production, use proper HTTP client with retries)
	resp, err := httpGet(url)
	if err != nil {
		return nil, err
	}

	return sl.parseManifest(resp)
}

func (sl *SkillLoader) parseManifest(data []byte) (*SkillManifest, error) {
	// Parse YAML front matter from markdown
	content := string(data)

	var yamlContent string
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			yamlContent = strings.TrimSpace(parts[1])
		}
	}

	if yamlContent == "" {
		return nil, fmt.Errorf("no YAML front matter found")
	}

	// Parse YAML using proper YAML library
	manifest := &SkillManifest{}
	if err := yaml.Unmarshal([]byte(yamlContent), manifest); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return manifest, nil
}

func (sl *SkillLoader) convertManifestTools(manifest *SkillManifest) []skills.Tool {
	tools := make([]skills.Tool, 0, len(manifest.Tools))

	for _, mt := range manifest.Tools {
		params := make(map[string]interface{})

		if len(mt.Parameters) > 0 {
			properties := make(map[string]interface{})
			required := make([]string, 0)

			for _, p := range mt.Parameters {
				prop := map[string]interface{}{
					"type":        p.Type,
					"description": p.Description,
				}
				if len(p.Enum) > 0 {
					prop["enum"] = p.Enum
				}
				if p.Default != nil {
					prop["default"] = p.Default
				}
				properties[p.Name] = prop

				if p.Required {
					required = append(required, p.Name)
				}
			}

			params["type"] = "object"
			params["properties"] = properties
			params["required"] = required
		}

		tool := skills.Tool{
			Name:        mt.Name,
			Description: mt.Description,
			Parameters:  params,
			Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
				// In production, this would call the actual skill implementation
				return map[string]string{
					"status": "executed",
					"tool":   mt.Name,
				}, nil
			},
		}
		tools = append(tools, tool)
	}

	return tools
}

// RuntimeSkill implements the skills.Skill interface for runtime-loaded skills
type RuntimeSkill struct {
	*skills.BaseSkill
	tools []skills.Tool
}

func (s *RuntimeSkill) Tools() []skills.Tool {
	return s.tools
}

// httpGet performs an HTTP GET request with retries and timeouts
func httpGet(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var lastErr error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", "Myrai-Skill-Loader/1.0")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}
			return body, nil
		}

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("URL not found: %s", url)
		}

		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
