// Package persona implements MemoryCore-inspired persona and memory system
package persona

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// PersonaManager manages AI persona, user preferences, and project contexts
type PersonaManager struct {
	workspacePath string
	logger        *zap.Logger

	// Core files
	identity *Identity
	user     *UserProfile
	tools    string
	agents   string

	// Runtime state
	currentProject *Project
	projects       *ProjectManager
	timeAwareness  *TimeAwareness

	// Caching
	systemPromptCache string
	cacheValid        bool
	cacheMu           sync.RWMutex

	mu sync.RWMutex
}

// Identity represents the AI's personality and characteristics
type Identity struct {
	Name        string
	Personality string
	Voice       string
	Values      []string
	Expertise   []string
}

// UserProfile represents learned user preferences
type UserProfile struct {
	Name               string
	CommunicationStyle string
	Preferences        map[string]string
	Expertise          []string
	Goals              []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// NewPersonaManager creates a new persona manager
func NewPersonaManager(workspacePath string, logger *zap.Logger) (*PersonaManager, error) {
	pm := &PersonaManager{
		workspacePath: workspacePath,
		logger:        logger,
		identity:      &Identity{Name: "Myrai"},
		user: &UserProfile{
			Preferences: make(map[string]string),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	// Ensure workspace exists
	if err := pm.ensureWorkspace(); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Load existing files
	if err := pm.Load(); err != nil {
		logger.Warn("Failed to load persona files, using defaults", zap.Error(err))
	}

	// Initialize subsystems
	pm.projects = NewProjectManager(filepath.Join(workspacePath, "projects"), logger)
	pm.timeAwareness = NewTimeAwareness()

	return pm, nil
}

// Load loads all persona files from workspace
func (pm *PersonaManager) Load() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Load IDENTITY.md
	identityPath := filepath.Join(pm.workspacePath, "IDENTITY.md")
	if data, err := os.ReadFile(identityPath); err == nil {
		pm.identity = parseIdentity(string(data))
	}

	// Load USER.md
	userPath := filepath.Join(pm.workspacePath, "USER.md")
	if data, err := os.ReadFile(userPath); err == nil {
		pm.user = parseUserProfile(string(data))
	}

	// Load TOOLS.md
	toolsPath := filepath.Join(pm.workspacePath, "TOOLS.md")
	if data, err := os.ReadFile(toolsPath); err == nil {
		pm.tools = string(data)
	}

	// Load AGENTS.md
	agentsPath := filepath.Join(pm.workspacePath, "AGENTS.md")
	if data, err := os.ReadFile(agentsPath); err == nil {
		pm.agents = string(data)
	}

	// Load current project
	if pm.projects != nil {
		pm.currentProject = pm.projects.GetCurrentProject()
	}

	return nil
}

// Save saves all persona files to workspace
func (pm *PersonaManager) Save() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Save IDENTITY.md
	identityPath := filepath.Join(pm.workspacePath, "IDENTITY.md")
	if err := os.WriteFile(identityPath, []byte(pm.identity.String()), 0644); err != nil {
		return fmt.Errorf("failed to save IDENTITY.md: %w", err)
	}

	// Save USER.md
	userPath := filepath.Join(pm.workspacePath, "USER.md")
	if err := os.WriteFile(userPath, []byte(pm.user.String()), 0644); err != nil {
		return fmt.Errorf("failed to save USER.md: %w", err)
	}

	return nil
}

// GetSystemPrompt builds the complete system prompt with all context
func (pm *PersonaManager) GetSystemPrompt() string {
	// Build new prompt
	pm.mu.RLock()
	var parts []string

	// Time awareness context (ALWAYS fresh, never cached)
	parts = append(parts, pm.timeAwareness.GetContext())

	// Check cache for the rest
	pm.cacheMu.RLock()
	if pm.cacheValid && pm.systemPromptCache != "" {
		cached := pm.systemPromptCache
		pm.cacheMu.RUnlock()
		pm.mu.RUnlock()
		// Prepend fresh time context to cached content
		return parts[0] + "\n\n" + cached
	}
	pm.cacheMu.RUnlock()

	// Identity context
	parts = append(parts, pm.getIdentityContext())

	// User profile context
	parts = append(parts, pm.getUserContext())

	// Current project context
	if pm.currentProject != nil {
		parts = append(parts, pm.getProjectContext())
	}

	// Tools context
	if pm.tools != "" {
		parts = append(parts, "## Available Tools\n"+pm.tools)
	}

	// Agents guidelines
	if pm.agents != "" {
		parts = append(parts, pm.agents)
	}
	pm.mu.RUnlock()

	// Join all parts except time (which is parts[0])
	cachedParts := strings.Join(parts[1:], "\n\n")

	// Cache only the non-time-sensitive parts
	pm.cacheMu.Lock()
	pm.systemPromptCache = cachedParts
	pm.cacheValid = true
	pm.cacheMu.Unlock()

	// Return full prompt with fresh time context
	return parts[0] + "\n\n" + cachedParts
}

// InvalidateCache invalidates the system prompt cache
func (pm *PersonaManager) InvalidateCache() {
	pm.cacheMu.Lock()
	pm.cacheValid = false
	pm.systemPromptCache = ""
	pm.cacheMu.Unlock()
}

// GetIdentity returns the AI's identity
func (pm *PersonaManager) GetIdentity() *Identity {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.identity
}

// SetIdentity updates the AI's identity
func (pm *PersonaManager) SetIdentity(identity *Identity) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.identity = identity
	pm.InvalidateCache()
	return pm.Save()
}

// GetUserProfile returns the user profile
func (pm *PersonaManager) GetUserProfile() *UserProfile {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.user
}

// UpdateUserPreference updates a user preference
func (pm *PersonaManager) UpdateUserPreference(key, value string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.user.Preferences[key] = value
	pm.user.UpdatedAt = time.Now()
	pm.InvalidateCache()
	return pm.Save()
}

// GetCurrentProject returns the current active project
func (pm *PersonaManager) GetCurrentProject() *Project {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.currentProject
}

// SwitchProject switches to a different project
func (pm *PersonaManager) SwitchProject(name string) error {
	project, err := pm.projects.LoadProject(name)
	if err != nil {
		return err
	}

	pm.mu.Lock()
	pm.currentProject = project
	pm.mu.Unlock()

	pm.InvalidateCache()
	return nil
}

// CreateProject creates a new project
func (pm *PersonaManager) CreateProject(name, projectType, description string) (*Project, error) {
	return pm.projects.CreateProject(name, projectType, description)
}

// ListProjects returns all projects
func (pm *PersonaManager) ListProjects() ([]*Project, error) {
	return pm.projects.ListProjects()
}

// ArchiveProject archives a project
func (pm *PersonaManager) ArchiveProject(name string) error {
	return pm.projects.ArchiveProjectByName(name)
}

// DeleteProject deletes a project
func (pm *PersonaManager) DeleteProject(name string) error {
	return pm.projects.DeleteProjectByName(name)
}

// GetWorkspacePath returns the workspace path
func (pm *PersonaManager) GetWorkspacePath() string {
	return pm.workspacePath
}

// Internal helper methods

func (pm *PersonaManager) ensureWorkspace() error {
	dirs := []string{
		pm.workspacePath,
		filepath.Join(pm.workspacePath, "projects"),
		filepath.Join(pm.workspacePath, "diary"),
		filepath.Join(pm.workspacePath, "memory"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (pm *PersonaManager) getIdentityContext() string {
	var parts []string
	parts = append(parts, "## Your Identity")
	parts = append(parts, fmt.Sprintf("You are %s, a helpful AI assistant.", pm.identity.Name))

	if pm.identity.Personality != "" {
		parts = append(parts, fmt.Sprintf("Personality: %s", pm.identity.Personality))
	}

	if pm.identity.Voice != "" {
		parts = append(parts, fmt.Sprintf("Communication style: %s", pm.identity.Voice))
	}

	if len(pm.identity.Values) > 0 {
		parts = append(parts, fmt.Sprintf("Values: %s", strings.Join(pm.identity.Values, ", ")))
	}

	if len(pm.identity.Expertise) > 0 {
		parts = append(parts, fmt.Sprintf("Areas of expertise: %s", strings.Join(pm.identity.Expertise, ", ")))
	}

	return strings.Join(parts, "\n")
}

func (pm *PersonaManager) getUserContext() string {
	var parts []string
	parts = append(parts, "## User Profile")

	if pm.user.Name != "" {
		parts = append(parts, fmt.Sprintf("Name: %s", pm.user.Name))
	}

	if pm.user.CommunicationStyle != "" {
		parts = append(parts, fmt.Sprintf("Preferred communication: %s", pm.user.CommunicationStyle))
	}

	if len(pm.user.Expertise) > 0 {
		parts = append(parts, fmt.Sprintf("User expertise: %s", strings.Join(pm.user.Expertise, ", ")))
	}

	if len(pm.user.Goals) > 0 {
		parts = append(parts, fmt.Sprintf("User goals: %s", strings.Join(pm.user.Goals, ", ")))
	}

	if len(pm.user.Preferences) > 0 {
		parts = append(parts, "Preferences:")
		for k, v := range pm.user.Preferences {
			parts = append(parts, fmt.Sprintf("  - %s: %s", k, v))
		}
	}

	return strings.Join(parts, "\n")
}

func (pm *PersonaManager) getProjectContext() string {
	if pm.currentProject == nil {
		return ""
	}

	var parts []string
	parts = append(parts, "## Current Project Context")
	parts = append(parts, fmt.Sprintf("Project: %s", pm.currentProject.Name))
	parts = append(parts, fmt.Sprintf("Type: %s", pm.currentProject.Type))

	if pm.currentProject.Description != "" {
		parts = append(parts, fmt.Sprintf("Description: %s", pm.currentProject.Description))
	}

	if len(pm.currentProject.Context) > 0 {
		parts = append(parts, "Context:")
		for k, v := range pm.currentProject.Context {
			parts = append(parts, fmt.Sprintf("  - %s: %s", k, v))
		}
	}

	return strings.Join(parts, "\n")
}

// String methods for serialization

func (i *Identity) String() string {
	var parts []string
	parts = append(parts, "# Identity")
	parts = append(parts, "")
	parts = append(parts, fmt.Sprintf("Name: %s", i.Name))
	parts = append(parts, "")

	if i.Personality != "" {
		parts = append(parts, "## Personality")
		parts = append(parts, i.Personality)
		parts = append(parts, "")
	}

	if i.Voice != "" {
		parts = append(parts, "## Voice")
		parts = append(parts, i.Voice)
		parts = append(parts, "")
	}

	if len(i.Values) > 0 {
		parts = append(parts, "## Values")
		for _, v := range i.Values {
			parts = append(parts, fmt.Sprintf("- %s", v))
		}
		parts = append(parts, "")
	}

	if len(i.Expertise) > 0 {
		parts = append(parts, "## Expertise")
		for _, e := range i.Expertise {
			parts = append(parts, fmt.Sprintf("- %s", e))
		}
	}

	return strings.Join(parts, "\n")
}

func (u *UserProfile) String() string {
	var parts []string
	parts = append(parts, "# User Profile")
	parts = append(parts, "")

	if u.Name != "" {
		parts = append(parts, fmt.Sprintf("Name: %s", u.Name))
		parts = append(parts, "")
	}

	if u.CommunicationStyle != "" {
		parts = append(parts, "## Communication Style")
		parts = append(parts, u.CommunicationStyle)
		parts = append(parts, "")
	}

	if len(u.Expertise) > 0 {
		parts = append(parts, "## Expertise")
		for _, e := range u.Expertise {
			parts = append(parts, fmt.Sprintf("- %s", e))
		}
		parts = append(parts, "")
	}

	if len(u.Goals) > 0 {
		parts = append(parts, "## Goals")
		for _, g := range u.Goals {
			parts = append(parts, fmt.Sprintf("- %s", g))
		}
		parts = append(parts, "")
	}

	if len(u.Preferences) > 0 {
		parts = append(parts, "## Preferences")
		for k, v := range u.Preferences {
			parts = append(parts, fmt.Sprintf("- %s: %s", k, v))
		}
		parts = append(parts, "")
	}

	parts = append(parts, fmt.Sprintf("Updated: %s", u.UpdatedAt.Format(time.RFC3339)))

	return strings.Join(parts, "\n")
}

// Parse functions for deserialization

func parseIdentity(data string) *Identity {
	i := &Identity{Name: "Myrai"}
	lines := strings.Split(data, "\n")
	var currentSection string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "# ") && strings.Contains(line, "Identity") {
			continue
		}

		if strings.HasPrefix(line, "Name:") {
			i.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
			continue
		}

		if strings.HasPrefix(line, "## ") {
			currentSection = strings.ToLower(strings.TrimPrefix(line, "## "))
			continue
		}

		if strings.HasPrefix(line, "-") {
			item := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			switch currentSection {
			case "values":
				i.Values = append(i.Values, item)
			case "expertise":
				i.Expertise = append(i.Expertise, item)
			}
		} else if currentSection == "personality" {
			i.Personality += line + " "
		} else if currentSection == "voice" {
			i.Voice += line + " "
		}
	}

	i.Personality = strings.TrimSpace(i.Personality)
	i.Voice = strings.TrimSpace(i.Voice)

	return i
}

func parseUserProfile(data string) *UserProfile {
	u := &UserProfile{
		Preferences: make(map[string]string),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	lines := strings.Split(data, "\n")
	var currentSection string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "# ") {
			continue
		}

		if strings.HasPrefix(line, "Name:") {
			u.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
			continue
		}

		if strings.HasPrefix(line, "## ") {
			currentSection = strings.ToLower(strings.TrimPrefix(line, "## "))
			continue
		}

		if strings.HasPrefix(line, "-") {
			item := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			switch currentSection {
			case "expertise":
				u.Expertise = append(u.Expertise, item)
			case "goals":
				u.Goals = append(u.Goals, item)
			case "preferences":
				if idx := strings.Index(item, ":"); idx > 0 {
					key := strings.TrimSpace(item[:idx])
					value := strings.TrimSpace(item[idx+1:])
					u.Preferences[key] = value
				}
			}
		} else if currentSection == "communication style" {
			u.CommunicationStyle += line + " "
		}
	}

	u.CommunicationStyle = strings.TrimSpace(u.CommunicationStyle)

	return u
}
