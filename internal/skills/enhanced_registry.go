package skills

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/store"
)

// SkillSource represents where a skill came from
type SkillSource string

const (
	SourceBuiltin SkillSource = "builtin"
	SourceGitHub  SkillSource = "github"
	SourceLocal   SkillSource = "local"
	SourceMCP     SkillSource = "mcp"
)

// SkillStatus represents the current status of a skill
type SkillStatus string

const (
	StatusEnabled  SkillStatus = "enabled"
	StatusDisabled SkillStatus = "disabled"
	StatusError    SkillStatus = "error"
	StatusLoading  SkillStatus = "loading"
)

// RuntimeSkill represents an enhanced skill with metadata
type RuntimeSkill struct {
	Manifest   *SkillManifest
	Tools      []Tool
	Status     SkillStatus
	Source     SkillSource
	SourceURL  string
	ErrorMsg   string
	LoadedAt   time.Time
	UpdatedAt  time.Time
	LastUsedAt *time.Time
	UseCount   int
}

// IsEnabled returns true if the skill is enabled
func (s *RuntimeSkill) IsEnabled() bool {
	return s.Status == StatusEnabled
}

// Enable enables the skill
func (s *RuntimeSkill) Enable() error {
	if s.Status == StatusError && s.ErrorMsg != "" {
		return fmt.Errorf("cannot enable skill with errors: %s", s.ErrorMsg)
	}
	s.Status = StatusEnabled
	s.UpdatedAt = time.Now()
	return nil
}

// Disable disables the skill
func (s *RuntimeSkill) Disable() error {
	s.Status = StatusDisabled
	s.UpdatedAt = time.Now()
	return nil
}

// SetError sets the skill to error status
func (s *RuntimeSkill) SetError(err error) {
	s.Status = StatusError
	s.ErrorMsg = err.Error()
	s.UpdatedAt = time.Now()
}

// RecordUse updates usage statistics
func (s *RuntimeSkill) RecordUse() {
	now := time.Now()
	s.LastUsedAt = &now
	s.UseCount++
}

// GetTool returns a tool by name from the skill
func (s *RuntimeSkill) GetTool(name string) (*Tool, bool) {
	for i := range s.Tools {
		if s.Tools[i].Name == name {
			return &s.Tools[i], true
		}
	}
	return nil, false
}

// MCPTool represents a tool from an MCP server
type MCPTool struct {
	Name        string                                                                      `json:"name"`
	Description string                                                                      `json:"description"`
	InputSchema map[string]interface{}                                                      `json:"inputSchema"`
	ServerName  string                                                                      `json:"serverName"`
	Handler     func(ctx context.Context, args map[string]interface{}) (interface{}, error) `json:"-"`
}

// EnhancedRegistry extends the base Registry with Phase 2 features
type EnhancedRegistry struct {
	*Registry
	runtimeSkills map[string]*RuntimeSkill
	mcpTools      map[string]*MCPTool
	skillOrder    []string // Maintain registration order
}

// NewEnhancedRegistry creates a new enhanced registry
func NewEnhancedRegistry(store *store.Store) *EnhancedRegistry {
	return &EnhancedRegistry{
		Registry:      NewRegistry(store),
		runtimeSkills: make(map[string]*RuntimeSkill),
		mcpTools:      make(map[string]*MCPTool),
		skillOrder:    []string{},
	}
}

// RegisterSkill adds a runtime skill to the registry
func (r *EnhancedRegistry) RegisterSkill(skill *RuntimeSkill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := skill.Manifest.Name
	if _, exists := r.runtimeSkills[name]; exists {
		return fmt.Errorf("skill %s already registered", name)
	}

	// Validate manifest
	if err := skill.Manifest.Validate(); err != nil {
		skill.SetError(err)
		return fmt.Errorf("skill validation failed: %w", err)
	}

	r.runtimeSkills[name] = skill
	r.skillOrder = append(r.skillOrder, name)

	// Register tools if skill is enabled
	if skill.IsEnabled() {
		for _, tool := range skill.Tools {
			r.tools[tool.Name] = tool
		}
	}

	return nil
}

// Enable enables a skill by name
func (r *EnhancedRegistry) Enable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	skill, exists := r.runtimeSkills[name]
	if !exists {
		return fmt.Errorf("skill not found: %s", name)
	}

	if err := skill.Enable(); err != nil {
		return err
	}

	// Register tools
	for _, tool := range skill.Tools {
		r.tools[tool.Name] = tool
	}

	return nil
}

// Disable disables a skill by name
func (r *EnhancedRegistry) Disable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	skill, exists := r.runtimeSkills[name]
	if !exists {
		return fmt.Errorf("skill not found: %s", name)
	}

	if err := skill.Disable(); err != nil {
		return err
	}

	// Unregister tools
	for _, tool := range skill.Tools {
		delete(r.tools, tool.Name)
	}

	return nil
}

// GetRuntimeSkill retrieves a runtime skill by name
func (r *EnhancedRegistry) GetRuntimeSkill(name string) (*RuntimeSkill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	skill, ok := r.runtimeSkills[name]
	return skill, ok
}

// ListRuntimeSkills returns all runtime skills
func (r *EnhancedRegistry) ListRuntimeSkills() []*RuntimeSkill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*RuntimeSkill, 0, len(r.runtimeSkills))
	for _, name := range r.skillOrder {
		if skill, ok := r.runtimeSkills[name]; ok {
			skills = append(skills, skill)
		}
	}
	return skills
}

// ListSkillsBySource returns skills filtered by source
func (r *EnhancedRegistry) ListSkillsBySource(source SkillSource) []*RuntimeSkill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*RuntimeSkill
	for _, skill := range r.runtimeSkills {
		if skill.Source == source {
			result = append(result, skill)
		}
	}
	return result
}

// ListSkillsByStatus returns skills filtered by status
func (r *EnhancedRegistry) ListSkillsByStatus(status SkillStatus) []*RuntimeSkill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*RuntimeSkill
	for _, skill := range r.runtimeSkills {
		if skill.Status == status {
			result = append(result, skill)
		}
	}
	return result
}

// RegisterMCPTool registers an MCP tool
func (r *EnhancedRegistry) RegisterMCPTool(tool *MCPTool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.mcpTools[tool.Name]; exists {
		return fmt.Errorf("MCP tool %s already registered", tool.Name)
	}

	r.mcpTools[tool.Name] = tool

	// Also add to regular tools
	r.tools[tool.Name] = Tool{
		Name:        tool.Name,
		Description: tool.Description,
		Parameters:  tool.InputSchema,
		Handler:     tool.Handler,
	}

	return nil
}

// GetMCPTool retrieves an MCP tool by name
func (r *EnhancedRegistry) GetMCPTool(name string) (*MCPTool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.mcpTools[name]
	return tool, ok
}

// ListMCPTools returns all MCP tools
func (r *EnhancedRegistry) ListMCPTools() []*MCPTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*MCPTool, 0, len(r.mcpTools))
	for _, tool := range r.mcpTools {
		tools = append(tools, tool)
	}
	return tools
}

// UnregisterSkill removes a skill from the registry
func (r *EnhancedRegistry) UnregisterSkill(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	skill, exists := r.runtimeSkills[name]
	if !exists {
		return fmt.Errorf("skill not found: %s", name)
	}

	// Unregister all tools
	for _, tool := range skill.Tools {
		delete(r.tools, tool.Name)
	}

	delete(r.runtimeSkills, name)

	// Remove from order
	newOrder := make([]string, 0, len(r.skillOrder)-1)
	for _, n := range r.skillOrder {
		if n != name {
			newOrder = append(newOrder, n)
		}
	}
	r.skillOrder = newOrder

	return nil
}

// UpdateSkill updates an existing skill
func (r *EnhancedRegistry) UpdateSkill(name string, skill *RuntimeSkill) error {
	// Unregister old skill
	if err := r.UnregisterSkill(name); err != nil {
		return err
	}

	// Register new skill
	return r.RegisterSkill(skill)
}

// GetStats returns registry statistics
func (r *EnhancedRegistry) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	enabled := 0
	disabled := 0
	errors := 0

	for _, skill := range r.runtimeSkills {
		switch skill.Status {
		case StatusEnabled:
			enabled++
		case StatusDisabled:
			disabled++
		case StatusError:
			errors++
		}
	}

	return map[string]interface{}{
		"total_skills": len(r.runtimeSkills),
		"enabled":      enabled,
		"disabled":     disabled,
		"errors":       errors,
		"total_tools":  len(r.tools),
		"mcp_tools":    len(r.mcpTools),
	}
}

// SearchSkills searches skills by name, description, or tags
func (r *EnhancedRegistry) SearchSkills(query string) []*RuntimeSkill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(query)
	var results []*RuntimeSkill

	for _, skill := range r.runtimeSkills {
		if strings.Contains(strings.ToLower(skill.Manifest.Name), query) ||
			strings.Contains(strings.ToLower(skill.Manifest.Description), query) {
			results = append(results, skill)
			continue
		}

		// Search tags
		for _, tag := range skill.Manifest.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, skill)
				break
			}
		}
	}

	return results
}
