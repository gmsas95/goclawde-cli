package persona

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	maxActiveProjects = 10
	projectIndexFile  = "project-index.json"
)

// Project represents a project with its context
type Project struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // coding, writing, research, business
	Description string                 `json:"description"`
	Context     map[string]string      `json:"context"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	LastAccessed time.Time             `json:"last_accessed"`
	Position    int                    `json:"position"` // LRU position (1 = most recent)
	IsArchived  bool                   `json:"is_archived"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ProjectManager manages projects with LRU tracking
type ProjectManager struct {
	basePath string
	logger   *zap.Logger
	projects map[string]*Project
	mu       sync.RWMutex
}

// NewProjectManager creates a new project manager
func NewProjectManager(basePath string, logger *zap.Logger) *ProjectManager {
	pm := &ProjectManager{
		basePath: basePath,
		logger:   logger,
		projects: make(map[string]*Project),
	}

	// Ensure directories exist
	os.MkdirAll(filepath.Join(basePath, "active"), 0755)
	os.MkdirAll(filepath.Join(basePath, "archived"), 0755)

	// Load existing projects
	pm.loadIndex()

	return pm
}

// CreateProject creates a new project
func (pm *ProjectManager) CreateProject(name, projectType, description string) (*Project, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Sanitize name for filesystem
	safeName := sanitizeProjectName(name)

	// Check if project exists
	if _, exists := pm.projects[safeName]; exists {
		return nil, fmt.Errorf("project '%s' already exists", name)
	}

	// Check if we need to archive oldest project
	pm.enforceProjectLimit()

	project := &Project{
		Name:         name,
		Type:         projectType,
		Description:  description,
		Context:      make(map[string]string),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastAccessed: time.Now(),
		Position:     1,
		Metadata:     make(map[string]interface{}),
	}

	// Shift all existing projects down
	for _, p := range pm.projects {
		if !p.IsArchived {
			p.Position++
		}
	}

	pm.projects[safeName] = project

	// Save project
	if err := pm.saveProjectInternal(project); err != nil {
		return nil, err
	}

	// Save index
	if err := pm.saveIndex(); err != nil {
		return nil, err
	}

	pm.logger.Info("Created project",
		zap.String("name", name),
		zap.String("type", projectType),
	)

	return project, nil
}

// LoadProject loads a project by name and updates its LRU position
func (pm *ProjectManager) LoadProject(name string) (*Project, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	safeName := sanitizeProjectName(name)

	project, exists := pm.projects[safeName]
	if !exists {
		return nil, fmt.Errorf("project '%s' not found", name)
	}

	// If archived, move to active
	if project.IsArchived {
		pm.unarchiveProject(project)
	}

	// Update LRU position
	pm.updateLRU(project)

	// Load full project data from disk
	if err := pm.loadProjectData(project); err != nil {
		return nil, err
	}

	project.LastAccessed = time.Now()
	project.UpdatedAt = time.Now()

	// Save updated project
	pm.saveProjectInternal(project)
	pm.saveIndex()

	return project, nil
}

// GetCurrentProject returns the most recently used active project
func (pm *ProjectManager) GetCurrentProject() *Project {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var current *Project
	for _, p := range pm.projects {
		if !p.IsArchived && (current == nil || p.Position < current.Position) {
			current = p
		}
	}

	return current
}

// ListProjects returns all projects sorted by position
func (pm *ProjectManager) ListProjects() ([]*Project, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var active, archived []*Project

	for _, p := range pm.projects {
		if p.IsArchived {
			archived = append(archived, p)
		} else {
			active = append(active, p)
		}
	}

	// Sort active by position
	sort.Slice(active, func(i, j int) bool {
		return active[i].Position < active[j].Position
	})

	// Sort archived by last accessed
	sort.Slice(archived, func(i, j int) bool {
		return archived[i].LastAccessed.After(archived[j].LastAccessed)
	})

	return append(active, archived...), nil
}

// ArchiveProject archives a project
func (pm *ProjectManager) ArchiveProject(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	safeName := sanitizeProjectName(name)

	project, exists := pm.projects[safeName]
	if !exists {
		return fmt.Errorf("project '%s' not found", name)
	}

	project.IsArchived = true
	project.UpdatedAt = time.Now()

	// Move file to archived directory
	src := filepath.Join(pm.basePath, "active", safeName+".json")
	dst := filepath.Join(pm.basePath, "archived", safeName+".json")

	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("failed to archive project: %w", err)
	}

	// Reorder remaining active projects
	pm.reorderPositions()

	return pm.saveIndex()
}

// UpdateProjectContext updates project context
func (pm *ProjectManager) UpdateProjectContext(name string, key, value string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	safeName := sanitizeProjectName(name)

	project, exists := pm.projects[safeName]
	if !exists {
		return fmt.Errorf("project '%s' not found", name)
	}

	if project.Context == nil {
		project.Context = make(map[string]string)
	}

	project.Context[key] = value
	project.UpdatedAt = time.Now()

	return pm.saveProjectInternal(project)
}

// DeleteProject permanently deletes a project
func (pm *ProjectManager) DeleteProject(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	safeName := sanitizeProjectName(name)

	project, exists := pm.projects[safeName]
	if !exists {
		return fmt.Errorf("project '%s' not found", name)
	}

	// Delete file
	dir := "active"
	if project.IsArchived {
		dir = "archived"
	}
	path := filepath.Join(pm.basePath, dir, safeName+".json")
	os.Remove(path)

	// Remove from map
	delete(pm.projects, safeName)

	// Reorder if it was active
	if !project.IsArchived {
		pm.reorderPositions()
	}

	return pm.saveIndex()
}

// SaveProject saves current project state
func (pm *ProjectManager) SaveProject(project *Project) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	project.UpdatedAt = time.Now()
	return pm.saveProjectInternal(project)
}

// ArchiveProject archives a project by name
func (pm *ProjectManager) ArchiveProjectByName(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	safeName := sanitizeProjectName(name)
	project, exists := pm.projects[safeName]
	if !exists {
		return fmt.Errorf("project '%s' not found", name)
	}

	if project.IsArchived {
		return fmt.Errorf("project '%s' is already archived", name)
	}

	pm.archiveProjectInternal(project)
	return pm.saveIndex()
}

// DeleteProjectByName deletes a project by name
func (pm *ProjectManager) DeleteProjectByName(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	safeName := sanitizeProjectName(name)
	project, exists := pm.projects[safeName]
	if !exists {
		return fmt.Errorf("project '%s' not found", name)
	}

	// Delete file
	dir := "active"
	if project.IsArchived {
		dir = "archived"
	}
	path := filepath.Join(pm.basePath, dir, safeName+".json")
	os.Remove(path)

	// Remove from map
	delete(pm.projects, safeName)

	// Reorder if it was active
	if !project.IsArchived {
		pm.reorderPositions()
	}

	return pm.saveIndex()
}

// Internal methods

func (pm *ProjectManager) enforceProjectLimit() {
	active := 0
	for _, p := range pm.projects {
		if !p.IsArchived {
			active++
		}
	}

	if active >= maxActiveProjects {
		// Find oldest (highest position) project and archive it
		var oldest *Project
		for _, p := range pm.projects {
			if !p.IsArchived && (oldest == nil || p.Position > oldest.Position) {
				oldest = p
			}
		}

		if oldest != nil {
			pm.logger.Info("Archiving oldest project due to limit",
				zap.String("project", oldest.Name),
			)
			pm.archiveProjectInternal(oldest)
		}
	}
}

func (pm *ProjectManager) archiveProjectInternal(project *Project) {
	project.IsArchived = true
	project.UpdatedAt = time.Now()

	src := filepath.Join(pm.basePath, "active", sanitizeProjectName(project.Name)+".json")
	dst := filepath.Join(pm.basePath, "archived", sanitizeProjectName(project.Name)+".json")

	os.Rename(src, dst)
	pm.reorderPositions()
}

func (pm *ProjectManager) unarchiveProject(project *Project) {
	pm.enforceProjectLimit()

	project.IsArchived = false
	project.Position = 1
	project.UpdatedAt = time.Now()

	src := filepath.Join(pm.basePath, "archived", sanitizeProjectName(project.Name)+".json")
	dst := filepath.Join(pm.basePath, "active", sanitizeProjectName(project.Name)+".json")

	os.Rename(src, dst)

	// Shift others down
	for _, p := range pm.projects {
		if !p.IsArchived && p.Name != project.Name {
			p.Position++
		}
	}
}

func (pm *ProjectManager) updateLRU(project *Project) {
	oldPos := project.Position
	project.Position = 1
	project.LastAccessed = time.Now()

	// Shift others down
	for _, p := range pm.projects {
		if !p.IsArchived && p.Name != project.Name && p.Position < oldPos {
			p.Position++
		}
	}
}

func (pm *ProjectManager) reorderPositions() {
	// Get active projects sorted by last accessed
	var active []*Project
	for _, p := range pm.projects {
		if !p.IsArchived {
			active = append(active, p)
		}
	}

	sort.Slice(active, func(i, j int) bool {
		return active[i].LastAccessed.After(active[j].LastAccessed)
	})

	// Reassign positions
	for i, p := range active {
		p.Position = i + 1
	}
}

func (pm *ProjectManager) saveProjectInternal(project *Project) error {
	dir := "active"
	if project.IsArchived {
		dir = "archived"
	}

	path := filepath.Join(pm.basePath, dir, sanitizeProjectName(project.Name)+".json")
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func (pm *ProjectManager) loadProjectData(project *Project) error {
	dir := "active"
	if project.IsArchived {
		dir = "archived"
	}

	path := filepath.Join(pm.basePath, dir, sanitizeProjectName(project.Name)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read project: %w", err)
	}

	return json.Unmarshal(data, project)
}

func (pm *ProjectManager) saveIndex() error {
	path := filepath.Join(pm.basePath, projectIndexFile)
	data, err := json.MarshalIndent(pm.projects, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func (pm *ProjectManager) loadIndex() error {
	path := filepath.Join(pm.basePath, projectIndexFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &pm.projects)
}

func sanitizeProjectName(name string) string {
	// Replace spaces and special chars with underscores
	name = strings.ToLower(name)
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		} else if r == ' ' {
			result.WriteRune('_')
		}
	}
	return result.String()
}

// GetProjectTemplate returns a template for a project type
func GetProjectTemplate(projectType string) map[string]string {
	templates := map[string]map[string]string{
		"coding": {
			"language":     "Primary programming language",
			"framework":    "Framework or libraries being used",
			"repository":   "Git repository URL",
			"environment":  "Development environment setup",
			"objective":    "What are you building?",
		},
		"writing": {
			"topic":        "What are you writing about?",
			"audience":     "Target audience",
			"format":       "Format (blog, essay, documentation, etc.)",
			"tone":         "Desired tone (formal, casual, technical, etc.)",
			"objective":    "Main goal of the piece",
		},
		"research": {
			"topic":        "Research topic or question",
			"field":        "Field of study",
			"methodology":  "Research approach",
			"sources":      "Key sources or references",
			"objective":    "What do you hope to discover?",
		},
		"business": {
			"company":      "Company or organization name",
			"industry":     "Industry or sector",
			"objective":    "Business objective",
			"stakeholders": "Key stakeholders",
			"metrics":      "Success metrics or KPIs",
		},
	}

	if template, ok := templates[projectType]; ok {
		return template
	}

	return map[string]string{
		"objective": "What is the goal of this project?",
	}
}

// AvailableProjectTypes returns list of available project types
func AvailableProjectTypes() []string {
	return []string{"coding", "writing", "research", "business"}
}
